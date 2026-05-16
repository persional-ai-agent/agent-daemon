package gateway

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/platform"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
	"github.com/google/uuid"
)

type Runner struct {
	adapters   []PlatformAdapter
	engine     *agent.Engine
	allowedFor func(platform string) string
	mu         sync.Mutex
	wg         sync.WaitGroup

	sessionsMu sync.Mutex
	sessions   map[string]*sessionWorker

	pairMu   sync.Mutex
	pairings map[string]map[string]bool // platform -> user_id -> paired
	pairCode string
	pairFile string

	hookSpoolMu   sync.Mutex
	hookSpoolPath string
	hookSpoolSeen map[string]time.Time
	identityStore *identityStore
}

func NewRunner(adapters []PlatformAdapter, engine *agent.Engine, allowedFor func(platform string) string) *Runner {
	r := &Runner{
		adapters:      adapters,
		engine:        engine,
		allowedFor:    allowedFor,
		sessions:      map[string]*sessionWorker{},
		pairings:      map[string]map[string]bool{},
		pairCode:      strings.TrimSpace(os.Getenv("AGENT_GATEWAY_PAIR_CODE")),
		pairFile:      filepath.Join(engine.Workdir, ".agent-daemon", "gateway_pairs.json"),
		hookSpoolPath: filepath.Join(engine.Workdir, ".agent-daemon", "gateway_hooks_spool.jsonl"),
		hookSpoolSeen: map[string]time.Time{},
		identityStore: newIdentityStore(engine.Workdir),
	}
	r.loadPairings()
	r.loadHookSpoolSeen()
	return r
}

func (r *Runner) Start(ctx context.Context) error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SPOOL")), "true") {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.hookSpoolReplayLoop(ctx)
		}()
	}
	for _, a := range r.adapters {
		adapter := a
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			adapterCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			adapter.OnMessage(adapterCtx, func(msgCtx context.Context, event MessageEvent) {
				r.enqueueMessage(msgCtx, adapter, event)
			})

			if err := adapter.Connect(adapterCtx); err != nil {
				log.Printf("[gateway:%s] connect failed: %v", adapter.Name(), err)
				return
			}
			platform.Register(adapter)

			<-adapterCtx.Done()
			platform.Unregister(adapter.Name())
			if err := adapter.Disconnect(context.Background()); err != nil {
				log.Printf("[gateway:%s] disconnect error: %v", adapter.Name(), err)
			}
		}()
	}
	return nil
}

func (r *Runner) Stop() {
	r.wg.Wait()
}

type sessionWorker struct {
	key     string
	adapter PlatformAdapter
	engine  *agent.Engine
	allowed string
	runner  *Runner

	queue chan MessageEvent

	mu                   sync.Mutex
	activeSessionID      string
	systemPrompt         string
	lastUserID           string
	lastUserInputByUser  map[string]string
	lastShowByUser       map[string]showCursorState
	lastSessionIDsByUser map[string][]string
	cancelCurrent        context.CancelFunc
	running              bool
	lastApprovalID       string
	lastApprovalCommand  string
	lastApprovalReason   string
}

type showCursorState struct {
	sessionID string
	offset    int
	limit     int
}

type gatewayCommand struct {
	raw     string
	head    string
	args    []string
	isSlash bool
}

type gatewaySessionCompactor interface {
	CompactSession(sessionID string, keepLastN int) (before int, after int, err error)
}

type gatewaySessionStatsStore interface {
	SessionStats(sessionID string) (map[string]any, error)
}

type gatewaySessionDetailStore interface {
	LoadMessagesPage(sessionID string, offset, limit int) ([]core.Message, error)
}

type gatewaySessionLister interface {
	ListRecentSessions(limit int) ([]map[string]any, error)
}

func parseGatewayCommand(platformName, text string) gatewayCommand {
	raw := normalizeGatewayCommand(platformName, strings.TrimSpace(text))
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return gatewayCommand{raw: raw}
	}
	head := strings.ToLower(strings.TrimSpace(parts[0]))
	args := make([]string, 0, len(parts)-1)
	if len(parts) > 1 {
		args = append(args, parts[1:]...)
	}
	return gatewayCommand{
		raw:     raw,
		head:    head,
		args:    args,
		isSlash: strings.HasPrefix(head, "/"),
	}
}

func deliveryHooksEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_DELIVERY")), "true")
}

func mergePayloadMeta(base map[string]any, extra map[string]any) map[string]any {
	for k, v := range extra {
		base[k] = v
	}
	return base
}

func (w *sessionWorker) sendText(ctx context.Context, chatID, content, replyTo string, meta map[string]any) (SendResult, error) {
	var (
		res SendResult
		err error
	)
	if rich, ok := w.adapter.(platform.RichTextSender); ok {
		res, err = rich.SendText(ctx, chatID, content, replyTo, meta)
	} else {
		res, err = w.adapter.Send(ctx, chatID, content, replyTo)
	}
	if w.runner != nil && deliveryHooksEnabled() {
		data := map[string]any{
			"platform":   w.adapter.Name(),
			"chat_id":    chatID,
			"reply_to":   replyTo,
			"success":    res.Success,
			"message_id": res.MessageID,
			"error":      res.Error,
			"kind":       "text",
			"at":         time.Now().Format(time.RFC3339Nano),
		}
		mergePayloadMeta(data, meta)
		w.runner.emitHook("gateway.delivery.send", data)
	}
	return res, err
}

func (w *sessionWorker) sendSlashText(ctx context.Context, event MessageEvent, content, slash string) {
	_, _ = w.sendText(ctx, event.ChatID, content, event.MessageID, tools.BuildSlashPayload(slash))
}

func (w *sessionWorker) sendSlashModeText(ctx context.Context, event MessageEvent, content, slash, mode string) {
	_, _ = w.sendText(ctx, event.ChatID, content, event.MessageID, tools.BuildSlashModePayload(slash, mode))
}

func (w *sessionWorker) sendSlashSubcommandText(ctx context.Context, event MessageEvent, content, slash, subcommand string) {
	_, _ = w.sendText(ctx, event.ChatID, content, event.MessageID, tools.BuildSlashSubcommandPayload(slash, subcommand))
}

func (w *sessionWorker) sendApprovalText(ctx context.Context, event MessageEvent, content, slash, approvalID string) {
	_, _ = w.sendText(ctx, event.ChatID, content, event.MessageID, tools.BuildApprovalCommandPayload(slash, approvalID))
}

func (w *sessionWorker) sendMetaText(ctx context.Context, event MessageEvent, content string, meta map[string]any) {
	_, _ = w.sendText(ctx, event.ChatID, content, event.MessageID, meta)
}

func (w *sessionWorker) sendAuthText(ctx context.Context, event MessageEvent, content, status string) {
	w.sendMetaText(ctx, event, content, tools.BuildAuthPayload(status))
}

func (w *sessionWorker) editText(ctx context.Context, chatID, messageID, content string, meta map[string]any) error {
	err := w.adapter.EditMessage(ctx, chatID, messageID, content)
	if w.runner != nil && deliveryHooksEnabled() {
		data := map[string]any{
			"platform":   w.adapter.Name(),
			"chat_id":    chatID,
			"message_id": messageID,
			"success":    err == nil,
			"error":      "",
			"kind":       "text",
			"at":         time.Now().Format(time.RFC3339Nano),
		}
		if err != nil {
			data["error"] = err.Error()
		}
		mergePayloadMeta(data, meta)
		w.runner.emitHook("gateway.delivery.edit", data)
	}
	return err
}

func (w *sessionWorker) sendMedia(ctx context.Context, chatID, path, caption, replyTo string, meta map[string]any) (SendResult, error) {
	ms, ok := w.adapter.(platform.MediaSender)
	if !ok {
		res := SendResult{Success: false, Error: "adapter does not support media"}
		if w.runner != nil && deliveryHooksEnabled() {
			data := map[string]any{
				"platform": w.adapter.Name(),
				"chat_id":  chatID,
				"reply_to": replyTo,
				"success":  false,
				"error":    res.Error,
				"path":     path,
				"at":       time.Now().Format(time.RFC3339Nano),
			}
			mergePayloadMeta(data, meta)
			w.runner.emitHook("gateway.delivery.media", data)
		}
		return res, nil
	}
	res, err := ms.SendMedia(ctx, chatID, path, caption, replyTo)
	if w.runner != nil && deliveryHooksEnabled() {
		data := map[string]any{
			"platform":   w.adapter.Name(),
			"chat_id":    chatID,
			"reply_to":   replyTo,
			"success":    res.Success,
			"message_id": res.MessageID,
			"error":      res.Error,
			"path":       path,
			"at":         time.Now().Format(time.RFC3339Nano),
		}
		mergePayloadMeta(data, meta)
		if err != nil && data["error"] == "" {
			data["error"] = err.Error()
		}
		w.runner.emitHook("gateway.delivery.media", data)
	}
	return res, err
}

func (r *Runner) enqueueMessage(ctx context.Context, adapter PlatformAdapter, event MessageEvent) {
	allowed := ""
	if r.allowedFor != nil {
		allowed = r.allowedFor(adapter.Name())
	}
	sessionKey := BuildSessionKey(adapter.Name(), event.ChatType, event.ChatID)

	r.sessionsMu.Lock()
	w := r.sessions[sessionKey]
	if w == nil {
		engCopy := *r.engine
		w = &sessionWorker{
			key:                  sessionKey,
			adapter:              adapter,
			engine:               &engCopy,
			allowed:              allowed,
			runner:               r,
			queue:                make(chan MessageEvent, 32),
			activeSessionID:      sessionKey,
			systemPrompt:         agent.DefaultSystemPrompt(),
			lastUserInputByUser:  map[string]string{},
			lastShowByUser:       map[string]showCursorState{},
			lastSessionIDsByUser: map[string][]string{},
		}
		r.sessions[sessionKey] = w
		go w.run(ctx)
	}
	r.sessionsMu.Unlock()

	if mapped := r.resolveMappedSessionID(adapter.Name(), event.UserID, event.UserName); strings.TrimSpace(mapped) != "" {
		w.activateSession(mapped, event.UserID)
	}

	select {
	case w.queue <- event:
	default:
		// Drop oldest by draining one, then enqueue.
		select {
		case <-w.queue:
		default:
		}
		select {
		case w.queue <- event:
		default:
		}
	}
}

func (r *Runner) resolveMappedSessionID(platformName, userID, userName string) string {
	if r == nil || r.engine == nil {
		return ""
	}
	globalID, err := tools.ResolveGatewayIdentity(r.engine.Workdir, platformName, userID)
	if err == nil && strings.TrimSpace(globalID) != "" {
		return BuildSessionKey("global", "user", globalID)
	}
	if auto := tools.AutoGlobalIdentity(r.continuityMode(), userID, userName); strings.TrimSpace(auto) != "" {
		return BuildSessionKey("global", "user", auto)
	}
	return ""
}

func (r *Runner) continuityMode() string {
	if r == nil || r.engine == nil {
		return "off"
	}
	if v, err := tools.ResolveGatewayContinuityMode(r.engine.Workdir); err == nil {
		return v
	}
	return "off"
}

func (w *sessionWorker) run(parent context.Context) {
	for {
		select {
		case <-parent.Done():
			return
		case event := <-w.queue:
			w.handleEvent(parent, event)
		}
	}
}

func (w *sessionWorker) handleEvent(ctx context.Context, event MessageEvent) {
	w.setLastUserID(event.UserID)
	globalID := ""
	if w.runner != nil && w.runner.identityStore != nil {
		if v, err := w.runner.identityStore.resolve(w.adapter.Name(), event.UserID); err == nil {
			globalID = strings.TrimSpace(v)
		}
	}
	_ = tools.UpsertChannelDirectory(w.engine.Workdir, tools.ChannelDirectoryEntry{
		Platform:   w.adapter.Name(),
		ChatID:     event.ChatID,
		ChatType:   event.ChatType,
		UserID:     event.UserID,
		UserName:   event.UserName,
		GlobalID:   globalID,
		HomeTarget: tools.ResolveHomeTarget(w.engine.Workdir, w.adapter.Name()),
	})
	allowed := ""
	if w != nil {
		allowed = w.allowed
	}
	authorized := CheckAuthorization(allowed, event.UserID)

	// Minimal slash commands for Hermes parity.
	parsed := parseGatewayCommand(w.adapter.Name(), event.Text)
	if parsed.isSlash {
		if GatewayCommandRequiresAuthorization(parsed.head) && !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
			_, _ = w.adapter.Send(ctx, event.ChatID, tools.AccessDeniedEN(), event.MessageID)
			return
		}
		switch parsed.head {
		case "/pair":
			if w.runner == nil {
				_, _ = w.adapter.Send(ctx, event.ChatID, tools.PairingUnavailableEN(), event.MessageID)
				return
			}
			code := ""
			if len(parsed.args) >= 1 {
				code = strings.TrimSpace(parsed.args[0])
			}
			if w.runner.tryPair(w.adapter.Name(), event.UserID, code) {
				_, _ = w.adapter.Send(ctx, event.ChatID, tools.PairSucceededEN(), event.MessageID)
			} else {
				_, _ = w.adapter.Send(ctx, event.ChatID, tools.FailedEN("Pair"), event.MessageID)
			}
			return
		case "/unpair":
			if w.runner == nil {
				_, _ = w.adapter.Send(ctx, event.ChatID, tools.UnpairUnavailableEN(), event.MessageID)
				return
			}
			if w.runner.unpair(w.adapter.Name(), event.UserID) {
				_, _ = w.adapter.Send(ctx, event.ChatID, tools.UnpairedEN(), event.MessageID)
			} else {
				_, _ = w.adapter.Send(ctx, event.ChatID, tools.NotPairedEN(), event.MessageID)
			}
			return
		case "/session":
			if len(parsed.args) == 0 {
				active := w.currentSessionID()
				meta := tools.AttachSlashPayload(tools.BuildSessionOverviewPayload(active, w.key, -1, -1), "/session")
				w.sendMetaText(ctx, event, "_Route session: "+escapeMarkdown(w.key)+"\\nActive session: "+escapeMarkdown(active)+"_", meta)
				return
			}
			if len(parsed.args) != 1 || strings.TrimSpace(parsed.args[0]) == "" {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandSessionUsage), "/session")
				return
			}
			target := strings.TrimSpace(parsed.args[0])
			prev := w.currentSessionID()
			w.activateSession(target, event.UserID)
			meta := tools.AttachSlashPayload(tools.BuildSessionSwitchPayload(prev, target, false, -1), "/session")
			w.sendMetaText(ctx, event, tools.SessionSwitchedEN(escapeMarkdown(prev), escapeMarkdown(target)), meta)
			return
		case "/whoami":
			globalID := ""
			if w.runner != nil && w.runner.identityStore != nil {
				if v, err := w.runner.identityStore.resolve(w.adapter.Name(), event.UserID); err == nil {
					globalID = strings.TrimSpace(v)
				}
			}
			mode := "off"
			if w.runner != nil {
				mode = w.runner.continuityMode()
			}
			autoID := tools.AutoGlobalIdentity(mode, event.UserID, event.UserName)
			whoami := tools.GatewayWhoamiResult{
				Platform:       w.adapter.Name(),
				UserID:         event.UserID,
				UserName:       event.UserName,
				ActiveSession:  w.currentSessionID(),
				GlobalID:       globalID,
				ContinuityMode: mode,
				AutoGlobalID:   autoID,
			}
			reply := tools.RenderGatewayWhoamiText(whoami)
			meta := tools.AttachSlashPayload(tools.BuildGatewayWhoamiPayload(whoami), "/whoami")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/resolve":
			resolveArgs, parseErr := tools.ParseGatewayResolveArgsWithDefaults(parsed.args, tools.GatewayResolveArgs{
				Platform: w.adapter.Name(),
				ChatType: event.ChatType,
				ChatID:   event.ChatID,
				UserID:   event.UserID,
				UserName: event.UserName,
			})
			if parseErr != nil {
				w.sendSlashText(ctx, event, tools.GatewayResolveUsageEN(), "/resolve")
				return
			}
			resolved, err := tools.ResolveGatewaySessionMapping(w.engine.Workdir, resolveArgs.Platform, resolveArgs.ChatType, resolveArgs.ChatID, resolveArgs.UserID, resolveArgs.UserName)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Resolve", escapeMarkdown(err.Error())), "/resolve")
				return
			}
			reply := tools.RenderGatewaySessionResolveText(resolved)
			meta := tools.AttachSlashPayload(tools.BuildGatewaySessionResolvePayload(resolved), "/resolve")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/continuity":
			if len(parsed.args) > 1 {
				w.sendSlashText(ctx, event, tools.GatewayContinuityUsageEN(), "/continuity")
				return
			}
			if len(parsed.args) == 0 {
				mode := "off"
				if w.runner != nil {
					mode = w.runner.continuityMode()
				}
				meta := tools.AttachSlashPayload(tools.BuildGatewayContinuityPayload(mode), "/continuity")
				w.sendMetaText(ctx, event, "_Continuity mode: "+escapeMarkdown(mode)+"_", meta)
				return
			}
			mode, pErr := tools.ParseGatewayContinuityModeArg(parsed.args)
			if pErr != nil {
				w.sendSlashText(ctx, event, tools.GatewayContinuityUsageEN(), "/continuity")
				return
			}
			updatedMode, uErr := tools.UpdateGatewayContinuityMode(w.engine.Workdir, mode)
			if uErr != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Continuity update", escapeMarkdown(uErr.Error())), "/continuity")
				return
			}
			meta := tools.AttachSlashPayload(tools.BuildGatewayContinuityPayload(updatedMode), "/continuity")
			w.sendMetaText(ctx, event, "_Continuity mode updated: "+escapeMarkdown(updatedMode)+"_", meta)
			return
		case "/setid":
			globalID, pErr := tools.ParseGatewayGlobalIDArg(parsed.args)
			if pErr != nil {
				w.sendSlashText(ctx, event, tools.GatewaySetIDGatewayUsageEN(), "/setid")
				return
			}
			if w.runner == nil || w.runner.identityStore == nil {
				w.sendSlashText(ctx, event, tools.IdentityStoreUnavailableEN(), "/setid")
				return
			}
			if err := w.runner.identityStore.bind(w.adapter.Name(), event.UserID, globalID); err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Setid", escapeMarkdown(err.Error())), "/setid")
				return
			}
			targetSession := BuildSessionKey("global", "user", globalID)
			prev := w.currentSessionID()
			w.activateSession(targetSession, event.UserID)
			_ = tools.UpsertChannelDirectory(w.engine.Workdir, tools.ChannelDirectoryEntry{
				Platform:   w.adapter.Name(),
				ChatID:     event.ChatID,
				ChatType:   event.ChatType,
				UserID:     event.UserID,
				UserName:   event.UserName,
				GlobalID:   globalID,
				HomeTarget: tools.ResolveHomeTarget(w.engine.Workdir, w.adapter.Name()),
			})
			meta := tools.AttachSlashPayload(tools.BuildGatewayIdentityBindPayload(w.adapter.Name(), event.UserID, globalID, prev, targetSession), "/setid")
			w.sendMetaText(ctx, event, "_Identity bound: "+escapeMarkdown(event.UserID)+" -> "+escapeMarkdown(globalID)+"; session "+escapeMarkdown(prev)+" -> "+escapeMarkdown(targetSession)+"_", meta)
			return
		case "/unsetid":
			if len(parsed.args) > 0 {
				w.sendSlashText(ctx, event, tools.GatewayUnsetIDGatewayUsageEN(), "/unsetid")
				return
			}
			if w.runner == nil || w.runner.identityStore == nil {
				w.sendSlashText(ctx, event, tools.IdentityStoreUnavailableEN(), "/unsetid")
				return
			}
			if err := w.runner.identityStore.unbind(w.adapter.Name(), event.UserID); err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Unsetid", escapeMarkdown(err.Error())), "/unsetid")
				return
			}
			_ = tools.ClearChannelDirectoryGlobalID(w.engine.Workdir, w.adapter.Name(), event.ChatID)
			meta := tools.AttachSlashPayload(tools.BuildGatewayIdentityPayload(w.adapter.Name(), event.UserID, "", false, true), "/unsetid")
			w.sendMetaText(ctx, event, "_Identity unbound for user: "+escapeMarkdown(event.UserID)+"_", meta)
			return
		case "/history":
			limit := 10
			if len(parsed.args) > 1 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandHistoryUsage), "/history")
				return
			}
			if len(parsed.args) == 1 {
				n, err := strconv.Atoi(strings.TrimSpace(parsed.args[0]))
				if err != nil || n <= 0 {
					w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandHistoryUsage), "/history")
					return
				}
				limit = n
			}
			active := w.currentSessionID()
			msgs, err := w.engine.SessionStore.LoadMessages(active, 500)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("History", escapeMarkdown(err.Error())), "/history")
				return
			}
			reply := renderGatewayHistory(msgs, limit)
			meta := tools.AttachSlashPayload(tools.BuildSessionHistoryPayload(active, len(msgs), limit), "/history")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/show":
			target, offset, limit, parseErr := parseGatewayShowArgs(parsed.args, w.currentSessionID())
			if parseErr != nil {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandShowUsage), "/show")
				return
			}
			detailStore, ok := w.engine.SessionStore.(gatewaySessionDetailStore)
			if !ok || detailStore == nil {
				w.sendSlashText(ctx, event, tools.NotSupportedBySessionStoreEN("Show"), "/show")
				return
			}
			msgs, err := detailStore.LoadMessagesPage(target, offset, limit)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Show", escapeMarkdown(err.Error())), "/show")
				return
			}
			w.setShowCursor(event.UserID, target, offset, limit)
			reply := renderGatewayShow(target, offset, limit, msgs)
			meta := tools.AttachSlashPayload(tools.BuildSessionShowPayload(target, offset, limit, msgs), "/show")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/next", "/prev":
			if len(parsed.args) > 0 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandNextPrevUsage), parsed.head)
				return
			}
			target, offset, limit := w.showCursor(event.UserID)
			if strings.TrimSpace(target) == "" {
				target = w.currentSessionID()
				offset = 0
				limit = 20
			}
			if parsed.head == "/next" {
				offset += limit
			} else {
				offset -= limit
				if offset < 0 {
					offset = 0
				}
			}
			detailStore, ok := w.engine.SessionStore.(gatewaySessionDetailStore)
			if !ok || detailStore == nil {
				w.sendSlashText(ctx, event, tools.NotSupportedBySessionStoreEN("Show pagination"), parsed.head)
				return
			}
			msgs, err := detailStore.LoadMessagesPage(target, offset, limit)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedFromSlashWithEscapedErrorEN(parsed.head, escapeMarkdown(err.Error())), parsed.head)
				return
			}
			w.setShowCursor(event.UserID, target, offset, limit)
			reply := renderGatewayShow(target, offset, limit, msgs)
			meta := tools.AttachSlashPayload(tools.BuildSessionShowPayload(target, offset, limit, msgs), parsed.head)
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/sessions":
			limit := 10
			if len(parsed.args) > 1 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandSessionsUsage), "/sessions")
				return
			}
			if len(parsed.args) == 1 {
				v, err := strconv.Atoi(strings.TrimSpace(parsed.args[0]))
				if err != nil || v <= 0 {
					w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandSessionsUsage), "/sessions")
					return
				}
				limit = v
			}
			lister, ok := w.engine.SessionStore.(gatewaySessionLister)
			if !ok || lister == nil {
				w.sendSlashText(ctx, event, tools.NotSupportedBySessionStoreEN("Sessions"), "/sessions")
				return
			}
			items, err := lister.ListRecentSessions(limit)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Sessions", escapeMarkdown(err.Error())), "/sessions")
				return
			}
			ids := make([]string, 0, len(items))
			for _, item := range items {
				sid, _ := item["session_id"].(string)
				if strings.TrimSpace(sid) != "" {
					ids = append(ids, strings.TrimSpace(sid))
				}
			}
			w.setLastSessionIDs(event.UserID, ids)
			reply := renderGatewaySessions(w.currentSessionID(), items)
			meta := tools.AttachSlashPayload(tools.BuildSessionListPayload(limit, items), "/sessions")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/pick":
			if len(parsed.args) != 1 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandPickUsage), "/pick")
				return
			}
			idx, err := strconv.Atoi(strings.TrimSpace(parsed.args[0]))
			if err != nil || idx <= 0 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandPickUsage), "/pick")
				return
			}
			list := w.getLastSessionIDs(event.UserID)
			if len(list) == 0 {
				w.sendSlashText(ctx, event, tools.SessionsListRequiredForPickEN(), "/pick")
				return
			}
			if idx > len(list) {
				w.sendSlashText(ctx, event, tools.PickIndexOutOfRangeEN(len(list)), "/pick")
				return
			}
			target := list[idx-1]
			prev := w.currentSessionID()
			w.activateSession(target, event.UserID)
			meta := tools.AttachSlashPayload(tools.BuildSessionPickPayload(prev, target, idx), "/pick")
			w.sendMetaText(ctx, event, "_Session picked: "+escapeMarkdown(prev)+" -> "+escapeMarkdown(target)+"_", meta)
			return
		case "/stats":
			target := w.currentSessionID()
			if len(parsed.args) > 1 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandStatsUsage), "/stats")
				return
			}
			if len(parsed.args) == 1 && strings.TrimSpace(parsed.args[0]) != "" {
				target = strings.TrimSpace(parsed.args[0])
			}
			statsStore, ok := w.engine.SessionStore.(gatewaySessionStatsStore)
			if !ok || statsStore == nil {
				w.sendSlashText(ctx, event, tools.NotSupportedBySessionStoreEN("Stats"), "/stats")
				return
			}
			stats, err := statsStore.SessionStats(target)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Stats", escapeMarkdown(err.Error())), "/stats")
				return
			}
			reply := renderGatewayStats(target, stats)
			meta := tools.AttachSlashPayload(tools.BuildSessionStatsPayload(target, stats), "/stats")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/cancel":
			w.mu.Lock()
			cancel := w.cancelCurrent
			running := w.running
			w.mu.Unlock()
			if cancel != nil && running {
				cancel()
				meta := tools.AttachSlashPayload(tools.BuildSessionCancelPayload(w.currentSessionID(), true, "cancelled"), "/cancel")
				w.sendMetaText(ctx, event, tools.CancelledEN(), meta)
			} else {
				meta := tools.AttachSlashPayload(tools.BuildSessionCancelPayload(w.currentSessionID(), false, "no_active_task"), "/cancel")
				w.sendMetaText(ctx, event, tools.NoActiveTaskEN(), meta)
			}
			return
		case "/new", "/reset":
			next := ""
			if parsed.head == "/new" && len(parsed.args) == 1 {
				next = strings.TrimSpace(parsed.args[0])
			} else if len(parsed.args) > 0 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandNewResetUsage), parsed.head)
				return
			}
			if next == "" {
				next = uuid.NewString()
			}
			prev := w.currentSessionID()
			w.activateSession(next, event.UserID)
			meta := tools.AttachSlashPayload(tools.BuildSessionSwitchPayload(prev, next, true, 0), parsed.head)
			w.sendMetaText(ctx, event, tools.SessionSwitchedEN(escapeMarkdown(prev), escapeMarkdown(next)), meta)
			return
		case "/resume":
			if len(parsed.args) != 1 || strings.TrimSpace(parsed.args[0]) == "" {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandResumeUsage), "/resume")
				return
			}
			target := strings.TrimSpace(parsed.args[0])
			if _, err := w.engine.SessionStore.LoadMessages(target, 1); err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Resume", escapeMarkdown(err.Error())), "/resume")
				return
			}
			prev := w.currentSessionID()
			w.activateSession(target, event.UserID)
			meta := tools.AttachSlashPayload(tools.BuildSessionSwitchPayload(prev, target, false, -1), "/resume")
			w.sendMetaText(ctx, event, tools.SessionResumedEN(escapeMarkdown(prev), escapeMarkdown(target)), meta)
			return
		case "/recover":
			if len(parsed.args) != 1 || !strings.EqualFold(strings.TrimSpace(parsed.args[0]), "context") {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandRecoverUsage), "/recover")
				return
			}
			lastInput := w.getLastUserInput(event.UserID)
			if strings.TrimSpace(lastInput) == "" {
				w.sendSlashText(ctx, event, tools.NoRecentUserInputToReplayEN(), "/recover")
				return
			}
			prev := w.currentSessionID()
			next := uuid.NewString()
			w.activateSession(next, event.UserID)
			meta := tools.AttachSlashPayload(tools.BuildSessionRecoverPayload(prev, next, true), "/recover")
			w.sendMetaText(ctx, event, tools.ContextRecoveredEN(escapeMarkdown(prev), escapeMarkdown(next)), meta)
			replay := event
			replay.Text = lastInput
			select {
			case w.queue <- replay:
			default:
				w.sendSlashText(ctx, event, tools.RecoverReplayQueueFullEN(), "/recover")
			}
			return
		case "/retry":
			lastInput := w.getLastUserInput(event.UserID)
			if strings.TrimSpace(lastInput) == "" {
				active := w.currentSessionID()
				if msgs, err := w.engine.SessionStore.LoadMessages(active, 500); err == nil {
					lastInput = latestUserInputFromMessages(msgs)
				}
			}
			if strings.TrimSpace(lastInput) == "" {
				w.sendSlashText(ctx, event, tools.NoRecentUserInputToReplayEN(), "/retry")
				return
			}
			w.sendSlashText(ctx, event, tools.RetryReplayingEN(), "/retry")
			replay := event
			replay.Text = lastInput
			select {
			case w.queue <- replay:
			default:
				w.sendSlashText(ctx, event, tools.RetryQueueFullEN(), "/retry")
			}
			return
		case "/undo":
			active := w.currentSessionID()
			msgs, err := w.engine.SessionStore.LoadMessages(active, 500)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Undo", escapeMarkdown(err.Error())), "/undo")
				return
			}
			nextMsgs, removed := removeLastTurnFromMessages(msgs)
			if removed == 0 {
				w.sendSlashText(ctx, event, tools.NoTurnToUndoEN(), "/undo")
				return
			}
			nextSession := uuid.NewString()
			for _, m := range nextMsgs {
				if err := w.engine.SessionStore.AppendMessage(nextSession, m); err != nil {
					w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Undo", escapeMarkdown(err.Error())), "/undo")
					return
				}
			}
			w.activateSession(nextSession, event.UserID)
			meta := tools.AttachSlashPayload(tools.BuildSessionUndoPayload(nextSession, removed, len(nextMsgs)), "/undo")
			w.sendMetaText(ctx, event, tools.UndoCompleteEN(removed, escapeMarkdown(nextSession)), meta)
			return
		case "/clear":
			if len(parsed.args) > 0 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandClearUsage), "/clear")
				return
			}
			prev := w.currentSessionID()
			next := uuid.NewString()
			w.activateSession(next, event.UserID)
			meta := tools.AttachSlashPayload(tools.BuildSessionClearPayload(prev, next, true), "/clear")
			w.sendMetaText(ctx, event, tools.ContextClearedEN(escapeMarkdown(prev), escapeMarkdown(next)), meta)
			return
		case "/reload":
			if len(parsed.args) > 0 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandReloadUsage), "/reload")
				return
			}
			active := w.currentSessionID()
			msgs, err := w.engine.SessionStore.LoadMessages(active, 500)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Reload", escapeMarkdown(err.Error())), "/reload")
				return
			}
			w.setLastUserInput(event.UserID, latestUserInputFromMessages(msgs))
			meta := tools.AttachSlashPayload(tools.BuildSessionReloadPayload(active, msgs), "/reload")
			w.sendMetaText(ctx, event, tools.SessionReloadedEN(escapeMarkdown(active), len(msgs)), meta)
			return
		case "/save":
			if len(parsed.args) > 1 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandSaveUsage), "/save")
				return
			}
			active := w.currentSessionID()
			msgs, err := w.engine.SessionStore.LoadMessages(active, 500)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Save", escapeMarkdown(err.Error())), "/save")
				return
			}
			requested := ""
			if len(parsed.args) == 1 {
				requested = strings.TrimSpace(parsed.args[0])
			}
			path, err := saveGatewayHistory(w.engine.Workdir, active, msgs, requested)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Save", escapeMarkdown(err.Error())), "/save")
				return
			}
			meta := tools.AttachSlashPayload(tools.BuildSessionSavePayload(active, path, len(msgs)), "/save")
			w.sendMetaText(ctx, event, tools.SessionSavedEN(escapeMarkdown(active), escapeMarkdown(path)), meta)
			return
		case "/sethome":
			if len(parsed.args) == 0 || len(parsed.args) > 2 {
				w.sendSlashText(ctx, event, tools.GatewaySetHomeUsageEN(), "/sethome")
				return
			}
			homePlatform, homeChatID, err := tools.ParseSetHomeArgs(parsed.args)
			if err != nil {
				w.sendSlashText(ctx, event, tools.GatewaySetHomeUsageEN(), "/sethome")
				return
			}
			envKey := tools.HomeTargetEnvVar(homePlatform)
			_ = os.Setenv(envKey, homeChatID)
			_ = tools.SetHomeTarget(w.engine.Workdir, homePlatform, homeChatID)
			_ = tools.UpsertChannelDirectory(w.engine.Workdir, tools.ChannelDirectoryEntry{
				Platform:   homePlatform,
				ChatID:     homeChatID,
				ChatType:   event.ChatType,
				UserID:     event.UserID,
				UserName:   event.UserName,
				GlobalID:   globalID,
				HomeTarget: homeChatID,
			})
			meta := tools.AttachSlashPayload(tools.BuildSetHomePayload(homePlatform, homeChatID), "/sethome")
			_, _ = w.sendText(
				ctx,
				event.ChatID,
				"_Home target updated: "+escapeMarkdown(homePlatform)+":"+escapeMarkdown(homeChatID)+" ("+escapeMarkdown(envKey)+" )_",
				event.MessageID,
				meta,
			)
			return
		case "/targets":
			if len(parsed.args) > 1 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandTargetsUsage), "/targets")
				return
			}
			filter := ""
			if len(parsed.args) == 1 {
				filter = strings.ToLower(strings.TrimSpace(parsed.args[0]))
			}
			platforms, items, err := tools.BuildDeliveryTargets(w.engine.Workdir, filter)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Targets", escapeMarkdown(err.Error())), "/targets")
				return
			}
			reply := renderGatewayTargets(items, filter)
			meta := tools.AttachSlashPayload(tools.BuildTargetsPayload(filter, platforms, items), "/targets")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/skills":
			root := filepath.Join(w.engine.Workdir, "skills")
			if len(parsed.args) == 0 {
				reply, err := renderGatewaySkillsList(root)
				if err != nil {
					w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Skills", escapeMarkdown(err.Error())), "/skills")
					return
				}
				w.sendSlashSubcommandText(ctx, event, escapeMarkdown(reply), "/skills", "list")
				return
			}
			if len(parsed.args) == 1 {
				name := strings.TrimSpace(parsed.args[0])
				reply, err := renderGatewaySkillView(root, name)
				if err != nil {
					meta := tools.BuildSlashModePayloadWithExtra("/skills", "show", map[string]any{"name": name})
					w.sendMetaText(ctx, event, "_"+escapeMarkdown(tools.NotFoundEN("skill", name))+"_", meta)
					return
				}
				meta := tools.BuildSlashSubcommandPayloadWithExtra("/skills", "show", map[string]any{"name": name})
				w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
				return
			}
			w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandSkillsUsage), "/skills")
			return
		case "/tools":
			sub := "list"
			if len(parsed.args) >= 1 {
				sub = strings.ToLower(strings.TrimSpace(parsed.args[0]))
			}
			switch sub {
			case "list":
				if len(parsed.args) > 1 {
					w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandToolsUsage), "/tools")
					return
				}
				reply := renderGatewayToolsList(w.engine.Registry.Names())
				w.sendSlashSubcommandText(ctx, event, escapeMarkdown(reply), "/tools", "list")
				return
			case "show":
				if len(parsed.args) != 2 {
					w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandToolsShowUsage), "/tools")
					return
				}
				name := strings.TrimSpace(parsed.args[1])
				for _, schema := range w.engine.Registry.Schemas() {
					if schema.Function.Name == name {
						reply := renderGatewayToolSchema(schema)
						meta := tools.BuildSlashSubcommandPayloadWithExtra("/tools", "show", map[string]any{"tool": name})
						w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
						return
					}
				}
				meta := tools.BuildSlashModePayloadWithExtra("/tools", "show", map[string]any{"tool": name})
				w.sendMetaText(ctx, event, "_"+escapeMarkdown(tools.NotFoundEN("tool", name))+"_", meta)
				return
			case "schemas":
				if len(parsed.args) > 1 {
					w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandToolsSchemasUsage), "/tools")
					return
				}
				reply := renderGatewayToolSchemas(w.engine.Registry.Schemas())
				w.sendSlashSubcommandText(ctx, event, escapeMarkdown(reply), "/tools", "schemas")
				return
			default:
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandToolsUsage), "/tools")
				return
			}
		case "/compress":
			keepLastN := 20
			if len(parsed.args) > 1 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandCompressUsage), "/compress")
				return
			}
			if len(parsed.args) == 1 {
				n, err := strconv.Atoi(strings.TrimSpace(parsed.args[0]))
				if err != nil || n <= 0 {
					w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandCompressUsage), "/compress")
					return
				}
				keepLastN = n
			}
			activeSessionID := w.currentSessionID()
			if compactor, ok := w.engine.SessionStore.(gatewaySessionCompactor); ok && compactor != nil {
				before, after, err := compactor.CompactSession(activeSessionID, keepLastN)
				if err != nil {
					w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Compress", escapeMarkdown(err.Error())), "/compress")
					return
				}
				meta := tools.AttachSlashPayload(tools.BuildSessionCompressPayload(activeSessionID, before, after, keepLastN, before-after, true, ""), "/compress")
				w.sendMetaText(ctx, event, tools.SessionCompressedEN(before, after), meta)
				return
			}
			w.sendSlashText(ctx, event, tools.NotSupportedBySessionStoreEN("Compress"), "/compress")
			return
		case "/usage":
			if len(parsed.args) > 1 {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandUsageUsage), "/usage")
				return
			}
			statsStore, ok := w.engine.SessionStore.(gatewaySessionStatsStore)
			if !ok || statsStore == nil {
				w.sendSlashText(ctx, event, tools.NotSupportedBySessionStoreEN("Usage"), "/usage")
				return
			}
			target := w.currentSessionID()
			if len(parsed.args) == 1 && strings.TrimSpace(parsed.args[0]) != "" {
				target = strings.TrimSpace(parsed.args[0])
			}
			stats, err := statsStore.SessionStats(target)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Usage", escapeMarkdown(err.Error())), "/usage")
				return
			}
			reply := renderGatewayStats(target, stats)
			meta := tools.AttachSlashPayload(tools.BuildSessionUsagePayload(target, stats), "/usage")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/model":
			if len(parsed.args) > 2 {
				w.sendSlashText(ctx, event, tools.GatewayModelUsageEN(), "/model")
				return
			}
			modelPref, err := tools.ResolveGatewayModelPreference(w.engine.Workdir)
			if err != nil {
				w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Model query", escapeMarkdown(err.Error())), "/model")
				return
			}
			provider := modelPref.Provider
			modelName := modelPref.Model
			baseURL := modelPref.BaseURL
			if len(parsed.args) > 0 {
				next, pErr := tools.ParseGatewayModelSpecArgs(parsed.args)
				if pErr != nil {
					w.sendSlashText(ctx, event, tools.GatewayModelUsageEN(), "/model")
					return
				}
				if err := tools.UpdateGatewayModelPreference(w.engine.Workdir, next); err != nil {
					w.sendSlashText(ctx, event, tools.FailedWithEscapedErrorEN("Model update", escapeMarkdown(err.Error())), "/model")
					return
				}
				provider = next.Provider
				modelName = next.Model
				reply := "_Model preference updated: " + escapeMarkdown(provider) + ":" + escapeMarkdown(modelName) + "_\nTakes effect for newly started daemon/model client."
				meta := tools.AttachSlashPayload(tools.BuildGatewayModelUpdatePayload(tools.GatewayModelSpec{Provider: provider, Model: modelName}), "/model")
				w.sendMetaText(ctx, event, reply, meta)
				return
			}
			displayPref := tools.DisplayGatewayModelPreference(tools.GatewayModelPreference{Provider: provider, Model: modelName, BaseURL: baseURL})
			reply := "Model client: " + fmt.Sprintf("%T", w.engine.Client) +
				"\nProvider: " + displayPref.Provider +
				"\nModel: " + displayPref.Model +
				"\nBase URL: " + displayPref.BaseURL
			meta := tools.AttachSlashPayload(tools.BuildGatewayModelPayload(fmt.Sprintf("%T", w.engine.Client), displayPref), "/model")
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/personality":
			if len(parsed.args) == 0 || strings.EqualFold(strings.TrimSpace(parsed.args[0]), "show") {
				reply := "System prompt:\n" + w.currentSystemPrompt()
				w.sendSlashModeText(ctx, event, escapeMarkdown(reply), "/personality", "show")
				return
			}
			if len(parsed.args) == 1 && strings.EqualFold(strings.TrimSpace(parsed.args[0]), "reset") {
				w.setSystemPrompt(agent.DefaultSystemPrompt())
				w.sendSlashModeText(ctx, event, "_Personality reset to default._", "/personality", "reset")
				return
			}
			next := strings.TrimSpace(strings.TrimPrefix(parsed.raw, parsed.head))
			if next == "" {
				w.sendSlashText(ctx, event, tools.UsageEN(tools.CommandPersonalityUsage), "/personality")
				return
			}
			w.setSystemPrompt(next)
			w.sendSlashModeText(ctx, event, "_Personality updated._", "/personality", "set")
			return
		case "/queue":
			qLen := len(w.queue)
			w.sendSlashText(ctx, event, "_Queue length: "+itoa(qLen)+"_", "/queue")
			return
		case "/status":
			snapshot := w.gatewayStatusSnapshot()
			reply := tools.RenderGatewayStatusText(snapshot)
			meta := mergePayloadMeta(tools.BuildSlashPayload("/status"), tools.BuildGatewayStatusPayload(snapshot))
			w.sendMetaText(ctx, event, escapeMarkdown(reply), meta)
			return
		case "/approve", "/deny":
			approve := parsed.head == "/approve"
			approvalID := w.resolveApprovalID(parsed.args)
			if approvalID == "" {
				w.sendSlashText(ctx, event, tools.UsageENEither(GatewayCommandUsage("approve"), GatewayCommandUsage("deny")), parsed.head)
				return
			}
			reply := w.confirmApproval(ctx, approvalID, approve)
			w.sendApprovalText(ctx, event, escapeMarkdown(reply), parsed.head, approvalID)
			return
		case "/approvals":
			reply := w.approvalStatus(ctx)
			w.sendApprovalText(ctx, event, escapeMarkdown(reply), parsed.head, "")
			return
		case "/pending":
			reply := w.pendingApprovalStatus()
			if w.adapter.Name() == "yuanbao" && !strings.Contains(reply, tools.NoPendingApprovalEN()) {
				reply += "\nquick_reply: 批准 / 拒绝"
			}
			w.sendApprovalText(ctx, event, escapeMarkdown(reply), "/pending", w.resolveApprovalID(nil))
			return
		case "/grant":
			reply := w.grantApproval(ctx, parsed)
			w.sendApprovalText(ctx, event, escapeMarkdown(reply), "/grant", "")
			return
		case "/revoke":
			reply := w.revokeApproval(ctx, parsed)
			w.sendApprovalText(ctx, event, escapeMarkdown(reply), "/revoke", "")
			return
		case "/help":
			helpText := GatewayHelpText(w.adapter.Name() == "yuanbao")
			w.sendSlashText(ctx, event, helpText, "/help")
			return
		}
	}

	if !authorized {
		if w.runner != nil && w.runner.isPaired(w.adapter.Name(), event.UserID) {
			authorized = true
		}
	}
	if !authorized {
		w.sendAuthText(ctx, event, tools.AccessDeniedEN(), "denied")
		return
	}
	w.setLastUserInput(event.UserID, event.Text)

	sessionKey := w.currentSessionID()

	history, err := w.engine.SessionStore.LoadMessages(sessionKey, 500)
	if err != nil {
		log.Printf("[gateway:%s] load history: %v", w.adapter.Name(), err)
		history = nil
	}

	collector := NewStreamCollector()
	streamMsgID := ""

	eng := *w.engine
	eng.GatewayPlatform = w.adapter.Name()
	eng.GatewayChatID = event.ChatID
	eng.GatewayChatType = event.ChatType
	eng.GatewayUserID = event.UserID
	eng.GatewayUserName = event.UserName
	eng.GatewayMessageID = event.MessageID
	eng.GatewayThreadID = event.ThreadID

	hookVerbose := strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_VERBOSE")), "true")
	if w.runner != nil {
		w.runner.emitHook("gateway.started", map[string]any{
			"platform":    w.adapter.Name(),
			"chat_id":     event.ChatID,
			"chat_type":   event.ChatType,
			"user_id":     event.UserID,
			"user_name":   event.UserName,
			"message_id":  event.MessageID,
			"thread_id":   event.ThreadID,
			"session_key": sessionKey,
			"text":        truncateString(event.Text, 2000),
			"at":          time.Now().Format(time.RFC3339Nano),
		})
	}

	// Hermes-style slow-response hint: if the agent produces no visible output for a while, send a waiting message.
	slowTimeout := parseIntEnv("AGENT_GATEWAY_SLOW_RESPONSE_TIMEOUT_SECONDS", 120)
	slowMessage := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_SLOW_RESPONSE_MESSAGE"))
	if slowMessage == "" {
		slowMessage = "任务有点复杂，正在努力处理中，请耐心等待..."
	}
	lastOutputMu := sync.Mutex{}
	lastOutputAt := time.Now()
	slowSent := false

	eng.EventSink = func(evt core.AgentEvent) {
		if hookVerbose && w.runner != nil {
			switch evt.Type {
			case "tool_started":
				w.runner.emitHook("gateway.tool_started", map[string]any{
					"platform":    w.adapter.Name(),
					"session_key": sessionKey,
					"tool_name":   evt.ToolName,
					"turn":        evt.Turn,
					"data":        evt.Data,
					"at":          time.Now().Format(time.RFC3339Nano),
				})
			case "tool_finished":
				parsed := tools.ParseJSONArgs(evt.Content)
				if approvalID, command, reason, ok := pendingApprovalDetails(parsed); ok {
					w.setPendingApproval(approvalID, command, reason)
					pendingText := "Pending approval: /approve " + approvalID + " or /deny " + approvalID
					if w.adapter.Name() == "yuanbao" {
						pendingText += "\nQuick reply: 批准 / 拒绝"
					}
					_, _ = w.sendText(context.Background(), event.ChatID, pendingText, event.MessageID, map[string]any{
						"phase":       "approval",
						"approval_id": approvalID,
						"quick_reply": w.adapter.Name() == "yuanbao",
					})
				}
				w.runner.emitHook("gateway.tool_finished", map[string]any{
					"platform":    w.adapter.Name(),
					"session_key": sessionKey,
					"tool_name":   evt.ToolName,
					"turn":        evt.Turn,
					"result":      truncateString(evt.Content, 4000),
					"data":        evt.Data,
					"at":          time.Now().Format(time.RFC3339Nano),
				})
			case "error":
				w.runner.emitHook("gateway.error", map[string]any{
					"platform":    w.adapter.Name(),
					"session_key": sessionKey,
					"turn":        evt.Turn,
					"error":       truncateString(evt.Content, 2000),
					"data":        evt.Data,
					"at":          time.Now().Format(time.RFC3339Nano),
				})
			}
		}
		collector.Ingest(evt)
		if collector.ShouldEdit() {
			content := collector.Content()
			if content == "" {
				return
			}
			lastOutputMu.Lock()
			lastOutputAt = time.Now()
			lastOutputMu.Unlock()
			if streamMsgID == "" {
				_ = w.adapter.SendTyping(context.Background(), event.ChatID)
				result, sendErr := w.sendText(context.Background(), event.ChatID, escapeMarkdown(content), event.MessageID, map[string]any{
					"phase": "stream",
					"turn":  evt.Turn,
				})
				if sendErr == nil {
					streamMsgID = result.MessageID
				}
			} else {
				_ = w.editText(context.Background(), event.ChatID, streamMsgID, escapeMarkdown(content)+"…", map[string]any{
					"phase": "stream",
					"turn":  evt.Turn,
				})
			}
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.cancelCurrent = cancel
	w.running = true
	w.mu.Unlock()

	// Start slow-response notifier (one-shot).
	if slowTimeout > 0 {
		go func() {
			t := time.NewTicker(1 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-runCtx.Done():
					return
				case <-t.C:
					lastOutputMu.Lock()
					since := time.Since(lastOutputAt)
					already := slowSent
					if since >= time.Duration(slowTimeout)*time.Second && !slowSent {
						slowSent = true
					}
					lastOutputMu.Unlock()
					if already {
						continue
					}
					if since >= time.Duration(slowTimeout)*time.Second {
						_, _ = w.sendText(context.Background(), event.ChatID, escapeMarkdown(slowMessage), event.MessageID, map[string]any{"phase": "slow_hint"})
						return
					}
				}
			}
		}()
	}

	userInput := gatewayUserInput(event)
	res, runErr := eng.Run(runCtx, sessionKey, userInput, w.currentSystemPrompt(), history)
	if runErr != nil && isGatewayContextLimitError(runErr) {
		_, _ = w.sendText(context.Background(), event.ChatID, escapeMarkdown("上下文过长，正在自动压缩并重试一次..."), event.MessageID, map[string]any{"phase": "context_recovery"})
		if compactor, ok := w.engine.SessionStore.(gatewaySessionCompactor); ok && compactor != nil {
			_, _, _ = compactor.CompactSession(sessionKey, 20)
		}
		history = compactGatewayHistory(history, 20)
		res, runErr = eng.Run(runCtx, sessionKey, userInput, w.currentSystemPrompt(), history)
		if runErr == nil {
			_, _ = w.sendText(context.Background(), event.ChatID, escapeMarkdown("上下文恢复完成，继续执行。"), event.MessageID, map[string]any{"phase": "context_recovery"})
		} else if isGatewayContextLimitError(runErr) {
			_, _ = w.sendText(context.Background(), event.ChatID, escapeMarkdown("压缩后仍超限，正在使用空上下文重试一次..."), event.MessageID, map[string]any{"phase": "context_recovery"})
			history = nil
			res, runErr = eng.Run(runCtx, sessionKey, userInput, w.currentSystemPrompt(), history)
			if runErr == nil {
				_, _ = w.sendText(context.Background(), event.ChatID, escapeMarkdown("已使用空上下文恢复并继续执行。"), event.MessageID, map[string]any{"phase": "context_recovery"})
			}
		}
	}

	w.mu.Lock()
	w.cancelCurrent = nil
	w.running = false
	w.mu.Unlock()
	cancel()

	finalContent := collector.Content()
	if finalContent == "" && res != nil {
		finalContent = res.FinalResponse
	}
	if finalContent == "" && runErr != nil {
		finalContent = "Error: " + runErr.Error()
	}

	lastOutputMu.Lock()
	lastOutputAt = time.Now()
	lastOutputMu.Unlock()

	// Hermes-style: if the final assistant content is "MEDIA: <path>", attempt to deliver it as an attachment.
	// This avoids requiring an explicit send_message call for common artifact flows (e.g. TTS).
	// Safety: only allow files under engine.Workdir or /tmp, and only if the adapter supports platform.MediaSender.
	if mediaPath, ok := extractMediaPath(finalContent, eng.Workdir); ok {
		if _, ok := w.adapter.(platform.MediaSender); ok {
			if streamMsgID != "" {
				_ = w.editText(context.Background(), event.ChatID, streamMsgID, escapeMarkdown("Sending media…"), map[string]any{"phase": "media"})
			}
			res, err := w.sendMedia(context.Background(), event.ChatID, mediaPath, "", event.MessageID, map[string]any{"phase": "media"})
			if err == nil && (res.Success || strings.TrimSpace(res.Error) == "") {
				if streamMsgID != "" {
					_ = w.editText(context.Background(), event.ChatID, streamMsgID, escapeMarkdown("Sent media."), map[string]any{"phase": "media"})
				}
				return
			}
			// Fall back to sending the original content if delivery failed.
		}
	}

	if streamMsgID != "" {
		_ = w.editText(context.Background(), event.ChatID, streamMsgID, escapeMarkdown(finalContent), map[string]any{"phase": "final"})
	} else {
		_, _ = w.sendText(context.Background(), event.ChatID, escapeMarkdown(finalContent), event.MessageID, map[string]any{"phase": "final"})
	}

	if w.runner != nil {
		if runErr != nil {
			w.runner.emitHook("gateway.failed", map[string]any{
				"platform":    w.adapter.Name(),
				"chat_id":     event.ChatID,
				"chat_type":   event.ChatType,
				"user_id":     event.UserID,
				"user_name":   event.UserName,
				"message_id":  event.MessageID,
				"thread_id":   event.ThreadID,
				"session_key": sessionKey,
				"error":       truncateString(runErr.Error(), 4000),
				"final":       truncateString(finalContent, 4000),
				"at":          time.Now().Format(time.RFC3339Nano),
			})
		}
		w.runner.emitHook("gateway.completed", map[string]any{
			"platform":    w.adapter.Name(),
			"chat_id":     event.ChatID,
			"chat_type":   event.ChatType,
			"user_id":     event.UserID,
			"user_name":   event.UserName,
			"message_id":  event.MessageID,
			"thread_id":   event.ThreadID,
			"session_key": sessionKey,
			"final":       truncateString(finalContent, 12000),
			"at":          time.Now().Format(time.RFC3339Nano),
		})
	}
}

func gatewayUserInput(event MessageEvent) string {
	text := strings.TrimSpace(event.Text)
	if len(event.MediaURLs) == 0 {
		return text
	}
	lines := make([]string, 0, 1+len(event.MediaURLs))
	if text != "" {
		lines = append(lines, text)
	} else {
		lines = append(lines, "[gateway_media_message]")
	}
	lines = append(lines, "Media URLs:")
	for _, mediaURL := range event.MediaURLs {
		mediaURL = strings.TrimSpace(mediaURL)
		if mediaURL == "" {
			continue
		}
		lines = append(lines, "- "+mediaURL)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func pendingApprovalDetails(parsed map[string]any) (string, string, string, bool) {
	if len(parsed) == 0 {
		return "", "", "", false
	}
	status, _ := parsed["status"].(string)
	if strings.TrimSpace(status) != "pending_approval" {
		return "", "", "", false
	}
	id, _ := parsed["approval_id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", "", false
	}
	command, _ := parsed["command"].(string)
	reason, _ := parsed["reason"].(string)
	return id, strings.TrimSpace(command), strings.TrimSpace(reason), true
}

func normalizeGatewayCommand(platformName, text string) string {
	cmd := strings.TrimSpace(text)
	if cmd == "" {
		return ""
	}
	if strings.HasPrefix(cmd, "/") {
		return CanonicalizeGatewaySlashText(cmd)
	}
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return cmd
	}
	if canonical, ok := ResolveGatewayCommand(parts[0]); ok {
		return "/" + canonical + withTail(parts)
	}
	if !strings.EqualFold(strings.TrimSpace(platformName), "yuanbao") {
		return cmd
	}
	if slash, ok := ResolveYuanbaoQuickReplyCommand(parts[0]); ok {
		return slash + withTail(parts)
	}
	return cmd
}

func isGatewayContextLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "exceed_context_size_error") ||
		strings.Contains(msg, "exceeds the available context size") ||
		(strings.Contains(msg, "context size") && strings.Contains(msg, "exceed"))
}

func compactGatewayHistory(history []core.Message, tail int) []core.Message {
	if tail <= 0 {
		tail = 20
	}
	if len(history) <= tail {
		return core.CloneMessages(history)
	}
	return core.CloneMessages(history[len(history)-tail:])
}

func latestUserInputFromMessages(history []core.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.Role != "user" {
			continue
		}
		text := strings.TrimSpace(m.Content)
		if text == "" || strings.HasPrefix(text, "/") {
			continue
		}
		return text
	}
	return ""
}

func removeLastTurnFromMessages(history []core.Message) ([]core.Message, int) {
	idx := -1
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			idx = i
			break
		}
	}
	if idx < 0 {
		return core.CloneMessages(history), 0
	}
	return core.CloneMessages(history[:idx]), len(history) - idx
}

func renderGatewayHistory(history []core.Message, limit int) string {
	if len(history) == 0 {
		return "History is empty."
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > len(history) {
		limit = len(history)
	}
	start := len(history) - limit
	lines := make([]string, 0, limit+1)
	lines = append(lines, "Recent history:")
	for i := start; i < len(history); i++ {
		msg := history[i]
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			content = "(empty)"
		}
		if len(content) > 120 {
			content = content[:120] + "..."
		}
		lines = append(lines, itoa(i+1)+". ["+msg.Role+"] "+content)
	}
	return strings.Join(lines, "\n")
}

func renderGatewayStats(sessionID string, stats map[string]any) string {
	if len(stats) == 0 {
		return "Stats are empty."
	}
	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys)+1)
	lines = append(lines, "Session stats: "+sessionID)
	for _, k := range keys {
		lines = append(lines, k+": "+fmt.Sprintf("%v", stats[k]))
	}
	return strings.Join(lines, "\n")
}

func renderGatewayShow(sessionID string, offset, limit int, msgs []core.Message) string {
	lines := []string{
		"Session messages: " + sessionID,
		"offset=" + itoa(offset) + " limit=" + itoa(limit) + " count=" + itoa(len(msgs)),
	}
	if len(msgs) == 0 {
		lines = append(lines, "(empty)")
		return strings.Join(lines, "\n")
	}
	for i, msg := range msgs {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			content = "(empty)"
		}
		if len(content) > 120 {
			content = content[:120] + "..."
		}
		lines = append(lines, itoa(offset+i+1)+". ["+msg.Role+"] "+content)
	}
	return strings.Join(lines, "\n")
}

func parseGatewayShowArgs(args []string, defaultSessionID string) (sessionID string, offset int, limit int, err error) {
	sessionID = strings.TrimSpace(defaultSessionID)
	offset = 0
	limit = 20
	if len(args) == 0 {
		return sessionID, offset, limit, nil
	}
	if len(args) > 3 {
		return "", 0, 0, fmt.Errorf("too many args")
	}
	first := strings.TrimSpace(args[0])
	if first == "" {
		return "", 0, 0, fmt.Errorf("empty arg")
	}
	if n, convErr := strconv.Atoi(first); convErr == nil {
		if n < 0 {
			return "", 0, 0, fmt.Errorf("invalid offset")
		}
		offset = n
		if len(args) >= 2 {
			v, e := strconv.Atoi(strings.TrimSpace(args[1]))
			if e != nil || v <= 0 {
				return "", 0, 0, fmt.Errorf("invalid limit")
			}
			limit = v
		}
		if len(args) == 3 {
			return "", 0, 0, fmt.Errorf("too many args")
		}
		return sessionID, offset, limit, nil
	}
	sessionID = first
	if len(args) >= 2 {
		v, e := strconv.Atoi(strings.TrimSpace(args[1]))
		if e != nil || v < 0 {
			return "", 0, 0, fmt.Errorf("invalid offset")
		}
		offset = v
	}
	if len(args) == 3 {
		v, e := strconv.Atoi(strings.TrimSpace(args[2]))
		if e != nil || v <= 0 {
			return "", 0, 0, fmt.Errorf("invalid limit")
		}
		limit = v
	}
	return sessionID, offset, limit, nil
}

func renderGatewaySessions(activeSessionID string, items []map[string]any) string {
	lines := []string{"Recent sessions:"}
	if len(items) == 0 {
		lines = append(lines, "(empty)")
		return strings.Join(lines, "\n")
	}
	for i, item := range items {
		sid, _ := item["session_id"].(string)
		lastSeen, _ := item["last_seen"].(string)
		line := itoa(i+1) + ". " + sid
		if strings.TrimSpace(lastSeen) != "" {
			line += " last_seen=" + lastSeen
		}
		if strings.TrimSpace(sid) != "" && strings.TrimSpace(sid) == strings.TrimSpace(activeSessionID) {
			line += " [active]"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func saveGatewayHistory(workdir, sessionID string, history []core.Message, requested string) (string, error) {
	path := strings.TrimSpace(requested)
	if path == "" {
		path = filepath.Join(".agent-daemon", "gateway-exports", "gateway-session-"+safeGatewayFilePart(sessionID)+"-"+time.Now().Format("20060102-150405")+".json")
	}
	if !filepath.IsAbs(path) {
		base := strings.TrimSpace(workdir)
		if base == "" {
			base = "."
		}
		path = filepath.Join(base, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	payload := map[string]any{"session_id": sessionID, "messages": history}
	bs, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, bs, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func safeGatewayFilePart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "session"
	}
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func renderGatewayToolsList(names []string) string {
	if len(names) == 0 {
		return "Tools: (empty)"
	}
	sort.Strings(names)
	lines := make([]string, 0, len(names)+1)
	lines = append(lines, "Tools ("+itoa(len(names))+"):")
	for i, n := range names {
		lines = append(lines, itoa(i+1)+". "+n)
	}
	return strings.Join(lines, "\n")
}

func renderGatewaySkillsList(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "Skills (0): " + tools.SkillsDirectoryNotFoundEN() + ".", nil
		}
		return "", err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, e.Name(), "SKILL.md")); err == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "Skills (0): no local skills found.", nil
	}
	return "Skills (" + itoa(len(names)) + "): " + strings.Join(names, ", "), nil
}

func renderGatewaySkillView(root, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty skill name")
	}
	path := filepath.Join(root, name, "SKILL.md")
	bs, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(bs))
	if content == "" {
		return "Skill: " + name + "\n(empty)", nil
	}
	lines := strings.Split(content, "\n")
	if len(lines) > 12 {
		lines = lines[:12]
		lines = append(lines, "...(truncated)")
	}
	return "Skill: " + name + "\n" + strings.Join(lines, "\n"), nil
}

func renderGatewayTargets(items []map[string]any, platformFilter string) string {
	filter := strings.ToLower(strings.TrimSpace(platformFilter))
	lines := make([]string, 0, len(items)+2)
	title := "Known targets"
	if filter != "" {
		title += " (" + filter + ")"
	}
	lines = append(lines, title+":")
	count := 0
	for _, it := range items {
		platformName, _ := it["platform"].(string)
		if filter != "" && platformName != filter {
			continue
		}
		count++
		target, _ := it["target"].(string)
		if strings.TrimSpace(target) == "" {
			chatID, _ := it["chat_id"].(string)
			target = platformName + ":" + chatID
		}
		line := target
		if home, _ := it["home_target"].(string); strings.TrimSpace(home) != "" {
			line += " [home=" + home + "]"
		}
		if userID, _ := it["user_id"].(string); strings.TrimSpace(userID) != "" {
			line += " user=" + userID
		}
		if globalID, _ := it["global_id"].(string); strings.TrimSpace(globalID) != "" {
			line += " global=" + globalID
		}
		if last, _ := it["last_seen_at"].(string); strings.TrimSpace(last) != "" {
			line += " last=" + last
		}
		if connected, ok := it["connected"].(bool); ok {
			if connected {
				line += " connected=yes"
			} else {
				line += " connected=no"
			}
		}
		lines = append(lines, line)
		if count >= 20 {
			lines = append(lines, "...(truncated)")
			break
		}
	}
	if count == 0 {
		lines = append(lines, "(empty)")
	}
	return strings.Join(lines, "\n")
}

func renderGatewayToolSchema(schema core.ToolSchema) string {
	bs, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return tools.MarshalFailedEN("Schema", err)
	}
	return "Tool schema:\n" + string(bs)
}

func renderGatewayToolSchemas(schemas []core.ToolSchema) string {
	bs, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return tools.MarshalFailedEN("Schemas", err)
	}
	return "Tool schemas (" + itoa(len(schemas)) + "):\n" + string(bs)
}

func withTail(parts []string) string {
	if len(parts) <= 1 {
		return ""
	}
	return " " + strings.Join(parts[1:], " ")
}

func (w *sessionWorker) setPendingApproval(id, command, reason string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastApprovalID = strings.TrimSpace(id)
	w.lastApprovalCommand = strings.TrimSpace(command)
	w.lastApprovalReason = strings.TrimSpace(reason)
}

func (w *sessionWorker) resolveApprovalID(args []string) string {
	if len(args) >= 1 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0])
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.TrimSpace(w.lastApprovalID)
}

func (w *sessionWorker) clearApprovalID(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(id) == "" || strings.TrimSpace(w.lastApprovalID) == strings.TrimSpace(id) {
		w.lastApprovalID = ""
		w.lastApprovalCommand = ""
		w.lastApprovalReason = ""
	}
}

func (w *sessionWorker) confirmApproval(ctx context.Context, approvalID string, approve bool) string {
	raw := w.engine.Registry.Dispatch(ctx, "approval", map[string]any{
		"action":      "confirm",
		"approval_id": approvalID,
		"approve":     approve,
	}, w.approvalToolContext())
	parsed := tools.ParseJSONArgs(raw)
	if errText, _ := parsed["error"].(string); strings.TrimSpace(errText) != "" {
		return tools.ApprovalFailedEN(errText)
	}
	approved, _ := parsed["approved"].(bool)
	command, _ := parsed["command"].(string)
	if !approved {
		w.clearApprovalID(approvalID)
		return tools.ApprovalDeniedEN(command)
	}
	w.clearApprovalID(approvalID)
	output, _ := parsed["output"].(string)
	output = strings.TrimSpace(output)
	if output != "" {
		return tools.ApprovalApprovedExecutedEN(truncateString(output, 1500))
	}
	return tools.ApprovalApprovedEN(command)
}

func (w *sessionWorker) approvalStatus(ctx context.Context) string {
	raw := w.engine.Registry.Dispatch(ctx, "approval", map[string]any{
		"action": "status",
	}, w.approvalToolContext())
	parsed := tools.ParseJSONArgs(raw)
	if errText, _ := parsed["error"].(string); strings.TrimSpace(errText) != "" {
		return tools.ApprovalStatusFailedEN(errText)
	}
	approvals, _ := parsed["approvals"].([]any)
	if len(approvals) == 0 {
		return tools.NoActiveApprovalsEN()
	}
	lines := make([]string, 0, len(approvals)+1)
	if approved, _ := parsed["approved"].(bool); approved {
		lines = append(lines, tools.SessionApprovalActiveEN())
	}
	for _, item := range approvals {
		m, _ := item.(map[string]any)
		if len(m) == 0 {
			continue
		}
		scope, _ := m["scope"].(string)
		pattern, _ := m["pattern"].(string)
		expiresAt, _ := m["expires_at"].(string)
		line := scope
		if strings.TrimSpace(pattern) != "" {
			line += ":" + pattern
		}
		if strings.TrimSpace(expiresAt) != "" {
			line += " until " + expiresAt
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return tools.NoActiveApprovalsEN()
	}
	return strings.Join(lines, "\n")
}

func (w *sessionWorker) gatewayStatusSnapshot() tools.GatewayStatusSnapshot {
	activeSessionID := w.currentSessionID()
	lastUserID := w.getLastUserID()
	snapshot := tools.GatewayStatusSnapshot{
		Platform:      w.adapter.Name(),
		RouteSession:  w.key,
		ActiveSession: activeSessionID,
		QueueLen:      len(w.queue),
		Running:       w.isRunning(),
	}
	if w.runner != nil {
		paired := w.runner.isPaired(w.adapter.Name(), lastUserID)
		snapshot.Paired = paired
		mode := w.runner.continuityMode()
		snapshot.ContinuityMode = mode
		if mapped := w.runner.resolveMappedSessionID(w.adapter.Name(), lastUserID, ""); strings.TrimSpace(mapped) != "" {
			snapshot.MappedSession = mapped
		}
	}
	if statsStore, ok := w.engine.SessionStore.(gatewaySessionStatsStore); ok && statsStore != nil {
		if stats, err := statsStore.SessionStats(activeSessionID); err == nil {
			if n, ok := stats["message_count"]; ok {
				snapshot.MessageCount = n
			}
		}
	}
	if last := w.resolveApprovalID(nil); strings.TrimSpace(last) != "" {
		snapshot.LastApprovalID = last
	}
	return snapshot
}

func (w *sessionWorker) gatewayStatusText() string {
	return tools.RenderGatewayStatusText(w.gatewayStatusSnapshot())
}

func (w *sessionWorker) pendingApprovalStatus() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(w.lastApprovalID) == "" {
		return tools.NoPendingApprovalEN()
	}
	lines := []string{"approval_id: " + w.lastApprovalID}
	if strings.TrimSpace(w.lastApprovalCommand) != "" {
		lines = append(lines, "command: "+truncateString(w.lastApprovalCommand, 500))
	}
	if strings.TrimSpace(w.lastApprovalReason) != "" {
		lines = append(lines, "reason: "+truncateString(w.lastApprovalReason, 500))
	}
	lines = append(lines, "actions: /approve "+w.lastApprovalID+" or /deny "+w.lastApprovalID)
	return strings.Join(lines, "\n")
}

func (w *sessionWorker) isRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

func (w *sessionWorker) currentSessionID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(w.activeSessionID) == "" {
		return w.key
	}
	return strings.TrimSpace(w.activeSessionID)
}

func (w *sessionWorker) currentSystemPrompt() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(w.systemPrompt) == "" {
		return agent.DefaultSystemPrompt()
	}
	return w.systemPrompt
}

func (w *sessionWorker) setSystemPrompt(prompt string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(prompt) == "" {
		w.systemPrompt = agent.DefaultSystemPrompt()
		return
	}
	w.systemPrompt = prompt
}

func (w *sessionWorker) setActiveSessionID(sessionID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = w.key
	}
	w.activeSessionID = sessionID
}

func (w *sessionWorker) activateSession(sessionID, userID string) {
	w.setActiveSessionID(sessionID)
	active := w.currentSessionID()
	w.setShowCursor(userID, active, 0, 20)
	if w.engine == nil || w.engine.SessionStore == nil {
		w.clearLastUserInput(userID)
		return
	}
	msgs, err := w.engine.SessionStore.LoadMessages(active, 500)
	if err != nil {
		w.clearLastUserInput(userID)
		return
	}
	w.setLastUserInput(userID, latestUserInputFromMessages(msgs))
}

func (w *sessionWorker) setLastUserInput(userID, text string) {
	userID = strings.TrimSpace(userID)
	text = strings.TrimSpace(text)
	if userID == "" || text == "" || strings.HasPrefix(text, "/") {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastUserInputByUser == nil {
		w.lastUserInputByUser = map[string]string{}
	}
	w.lastUserInputByUser[userID] = text
}

func (w *sessionWorker) getLastUserInput(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.TrimSpace(w.lastUserInputByUser[userID])
}

func (w *sessionWorker) setLastUserID(userID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastUserID = strings.TrimSpace(userID)
}

func (w *sessionWorker) getLastUserID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.TrimSpace(w.lastUserID)
}

func (w *sessionWorker) clearLastUserInput(userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.lastUserInputByUser, userID)
}

func (w *sessionWorker) setShowCursor(userID, sessionID string, offset, limit int) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastShowByUser == nil {
		w.lastShowByUser = map[string]showCursorState{}
	}
	w.lastShowByUser[userID] = showCursorState{sessionID: strings.TrimSpace(sessionID), offset: offset, limit: limit}
}

func (w *sessionWorker) showCursor(userID string) (sessionID string, offset, limit int) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", 0, 0
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	c, ok := w.lastShowByUser[userID]
	if !ok {
		return "", 0, 0
	}
	return strings.TrimSpace(c.sessionID), c.offset, c.limit
}

func (w *sessionWorker) setLastSessionIDs(userID string, ids []string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastSessionIDsByUser == nil {
		w.lastSessionIDsByUser = map[string][]string{}
	}
	w.lastSessionIDsByUser[userID] = append([]string(nil), ids...)
}

func (w *sessionWorker) getLastSessionIDs(userID string) []string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.lastSessionIDsByUser[userID]...)
}

func (w *sessionWorker) grantApproval(ctx context.Context, parsed gatewayCommand) string {
	args, usageErr := parseApprovalManageCommand(parsed.head, parsed.args)
	if usageErr != "" {
		return usageErr
	}
	args["action"] = "grant"
	raw := w.engine.Registry.Dispatch(ctx, "approval", args, w.approvalToolContext())
	out := tools.ParseJSONArgs(raw)
	if errText, _ := out["error"].(string); strings.TrimSpace(errText) != "" {
		return tools.GrantFailedEN(errText)
	}
	scope, _ := out["scope"].(string)
	pattern, _ := out["pattern"].(string)
	expiresAt, _ := out["expires_at"].(string)
	if strings.TrimSpace(scope) == "pattern" && strings.TrimSpace(pattern) != "" {
		return tools.GrantedPatternApprovalEN(pattern, expiresAt)
	}
	return tools.GrantedSessionApprovalEN(expiresAt)
}

func (w *sessionWorker) revokeApproval(ctx context.Context, parsed gatewayCommand) string {
	args, usageErr := parseApprovalManageCommand(parsed.head, parsed.args)
	if usageErr != "" {
		return usageErr
	}
	args["action"] = "revoke"
	raw := w.engine.Registry.Dispatch(ctx, "approval", args, w.approvalToolContext())
	out := tools.ParseJSONArgs(raw)
	if errText, _ := out["error"].(string); strings.TrimSpace(errText) != "" {
		return tools.RevokeFailedEN(errText)
	}
	scope, _ := out["scope"].(string)
	pattern, _ := out["pattern"].(string)
	revoked, _ := out["revoked"].(bool)
	if strings.TrimSpace(scope) == "pattern" && strings.TrimSpace(pattern) != "" {
		if revoked {
			return tools.RevokedPatternApprovalEN(pattern)
		}
		return tools.NotFoundEN("pattern approval", pattern)
	}
	if revoked {
		return tools.RevokedSessionApprovalEN()
	}
	return tools.NoActiveSessionApprovalEN()
}

func parseApprovalManageCommand(head string, argsIn []string) (map[string]any, string) {
	if strings.TrimSpace(head) == "" {
		return nil, GatewayGrantRevokeCombinedUsage()
	}
	canonical := strings.ToLower(strings.TrimSpace(head))
	if !strings.HasPrefix(canonical, "/") {
		canonical = "/" + canonical
	}
	argsIn = append([]string(nil), argsIn...)
	parts := make([]string, 0, len(argsIn)+1)
	parts = append(parts, canonical)
	parts = append(parts, argsIn...)
	args := map[string]any{}
	if len(parts) <= 1 {
		return args, ""
	}
	if strings.EqualFold(parts[1], "pattern") {
		if len(parts) < 3 || strings.TrimSpace(parts[2]) == "" {
			return nil, GatewayGrantPatternOrRevokePatternUsage()
		}
		args["scope"] = "pattern"
		args["pattern"] = strings.TrimSpace(parts[2])
		if len(parts) >= 4 {
			if ttl, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil && ttl >= 0 {
				args["ttl_seconds"] = ttl
			} else if strings.HasPrefix(parts[0], "/grant") {
				return nil, tools.UsageEN(GatewayGrantPatternUsage())
			}
		}
		return args, ""
	}
	if ttl, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && ttl >= 0 {
		args["ttl_seconds"] = ttl
		return args, ""
	}
	return nil, GatewayGrantRevokeCombinedUsage()
}

func (w *sessionWorker) approvalToolContext() tools.ToolContext {
	return tools.ToolContext{
		SessionID:      w.currentSessionID(),
		SessionStore:   w.engine.SearchStore,
		MemoryStore:    w.engine.MemoryStore,
		TodoStore:      w.engine.TodoStore,
		ApprovalStore:  w.engine.ApprovalStore,
		DelegateRunner: w.engine,
		Workdir:        w.engine.Workdir,
	}
}

func parseIntEnv(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func truncateString(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func extractMediaPath(finalContent, workdir string) (string, bool) {
	s := strings.TrimSpace(finalContent)
	if s == "" {
		return "", false
	}
	up := strings.ToUpper(s)
	if !strings.HasPrefix(up, "MEDIA:") {
		return "", false
	}
	p := strings.TrimSpace(s[len("MEDIA:"):])
	if p == "" {
		return "", false
	}
	// Only consider the first line as the path.
	if i := strings.IndexByte(p, '\n'); i >= 0 {
		p = strings.TrimSpace(p[:i])
	}
	if p == "" {
		return "", false
	}

	abs := p
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(workdir, abs)
	}
	abs = filepath.Clean(abs)

	// Allow list: workdir subtree and /tmp subtree.
	if !isWithin(abs, workdir) && !isWithin(abs, "/tmp") {
		return "", false
	}
	info, err := os.Stat(abs)
	if err != nil || info == nil || !info.Mode().IsRegular() {
		return "", false
	}
	return abs, true
}

func isWithin(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if root == "" {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "~", "\\~")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, ">", "\\>")
	s = strings.ReplaceAll(s, "#", "\\#")
	s = strings.ReplaceAll(s, "+", "\\+")
	s = strings.ReplaceAll(s, "-", "\\-")
	s = strings.ReplaceAll(s, "=", "\\=")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	s = strings.ReplaceAll(s, ".", "\\.")
	s = strings.ReplaceAll(s, "!", "\\!")
	return s
}

func (r *Runner) isPaired(platformName, userID string) bool {
	platformName = strings.ToLower(strings.TrimSpace(platformName))
	userID = strings.TrimSpace(userID)
	if platformName == "" || userID == "" {
		return false
	}
	r.pairMu.Lock()
	defer r.pairMu.Unlock()
	m := r.pairings[platformName]
	return m != nil && m[userID]
}

func (r *Runner) tryPair(platformName, userID, code string) bool {
	platformName = strings.ToLower(strings.TrimSpace(platformName))
	userID = strings.TrimSpace(userID)
	code = strings.TrimSpace(code)
	if platformName == "" || userID == "" {
		return false
	}
	// If no pair code configured, pairing is disabled.
	if strings.TrimSpace(r.pairCode) == "" {
		return false
	}
	if code == "" || code != r.pairCode {
		return false
	}
	r.pairMu.Lock()
	defer r.pairMu.Unlock()
	m := r.pairings[platformName]
	if m == nil {
		m = map[string]bool{}
		r.pairings[platformName] = m
	}
	m[userID] = true
	_ = r.savePairingsLocked()
	return true
}

func (r *Runner) unpair(platformName, userID string) bool {
	platformName = strings.ToLower(strings.TrimSpace(platformName))
	userID = strings.TrimSpace(userID)
	if platformName == "" || userID == "" {
		return false
	}
	r.pairMu.Lock()
	defer r.pairMu.Unlock()
	m := r.pairings[platformName]
	if m == nil || !m[userID] {
		return false
	}
	delete(m, userID)
	if len(m) == 0 {
		delete(r.pairings, platformName)
	}
	_ = r.savePairingsLocked()
	return true
}

func (r *Runner) loadPairings() {
	if strings.TrimSpace(r.pairFile) == "" {
		return
	}
	bs, err := os.ReadFile(r.pairFile)
	if err != nil || len(bs) == 0 {
		return
	}
	var raw map[string][]string
	if err := json.Unmarshal(bs, &raw); err != nil {
		return
	}
	r.pairMu.Lock()
	defer r.pairMu.Unlock()
	for p, ids := range raw {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" || len(ids) == 0 {
			continue
		}
		m := r.pairings[p]
		if m == nil {
			m = map[string]bool{}
			r.pairings[p] = m
		}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				m[id] = true
			}
		}
	}
}

func (r *Runner) savePairingsLocked() error {
	if strings.TrimSpace(r.pairFile) == "" {
		return nil
	}
	out := map[string][]string{}
	for p, m := range r.pairings {
		if len(m) == 0 {
			continue
		}
		ids := make([]string, 0, len(m))
		for id, ok := range m {
			if ok && strings.TrimSpace(id) != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			out[p] = ids
		}
	}
	bs, _ := json.MarshalIndent(out, "", "  ")
	if err := os.MkdirAll(filepath.Dir(r.pairFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.pairFile, bs, 0o644)
}

func itoa(n int) string {
	// No strconv import needed for this file.
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + (n % 10))
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

var _ = time.Second

func (r *Runner) emitHook(eventType string, payload map[string]any) {
	hookURL := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_URL"))
	if hookURL == "" {
		return
	}
	timeout := parseIntEnv("AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS", 4)
	if timeout <= 0 {
		timeout = 4
	}
	retries := parseIntEnv("AGENT_GATEWAY_HOOK_RETRIES", 2)
	if retries < 0 {
		retries = 0
	}
	backoffMs := parseIntEnv("AGENT_GATEWAY_HOOK_BACKOFF_MS", 250)
	if backoffMs < 0 {
		backoffMs = 0
	}
	secret := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SECRET"))
	eventID := uuid.NewString()
	body := map[string]any{
		"id":   eventID,
		"type": eventType,
		"at":   time.Now().Format(time.RFC3339Nano),
		"data": payload,
	}
	bs, _ := json.Marshal(body)
	go func() {
		client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
		ok := false
		for attempt := 0; attempt <= retries; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			ts := strconv.FormatInt(time.Now().Unix(), 10)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, hookURL, bytes.NewReader(bs))
			if err != nil {
				cancel()
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Agent-Event", eventType)
			req.Header.Set("X-Agent-Event-Id", eventID)
			req.Header.Set("X-Agent-Timestamp", ts)
			if secret != "" {
				req.Header.Set("X-Agent-Signature", signHook(secret, ts, bs))
			}
			resp, err := client.Do(req)
			if resp != nil {
				_ = resp.Body.Close()
			}
			cancel()
			if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				ok = true
				return
			}
			if attempt < retries && backoffMs > 0 {
				time.Sleep(time.Duration(backoffMs*(attempt+1)) * time.Millisecond)
			}
		}
		if !ok && strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SPOOL")), "true") {
			r.appendHookSpool(eventType, eventID, bs)
		}
	}()
}

func signHook(secret, ts string, body []byte) string {
	// Signature scheme: hex(hmac_sha256(secret, ts + "." + bodyBytes))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(ts))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

type hookSpoolEntry struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Body      string `json:"body"` // raw JSON string of hook envelope
	CreatedAt string `json:"created_at"`
}

func (r *Runner) appendHookSpool(eventType, eventID string, body []byte) {
	r.hookSpoolMu.Lock()
	defer r.hookSpoolMu.Unlock()
	if shouldDedupSpool() && strings.TrimSpace(eventID) != "" {
		if r.hookSpoolSeen == nil {
			r.hookSpoolSeen = map[string]time.Time{}
		}
		if _, ok := r.hookSpoolSeen[eventID]; ok {
			return
		}
		r.hookSpoolSeen[eventID] = time.Now()
		r.trimHookSpoolSeenLocked()
	}
	path := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SPOOL_PATH"))
	if path == "" {
		path = r.hookSpoolPath
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	r.rotateHookSpoolLocked(path)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	entry := hookSpoolEntry{
		Type:      eventType,
		ID:        eventID,
		Body:      string(body),
		CreatedAt: time.Now().Format(time.RFC3339Nano),
	}
	bs, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = f.Write(append(bs, '\n'))
}

func (r *Runner) rotateHookSpoolLocked(path string) {
	// Rotate when spool exceeds a size threshold.
	maxBytes := int64(parseIntEnv("AGENT_GATEWAY_HOOK_SPOOL_MAX_BYTES", 5<<20)) // default 5MB
	if maxBytes <= 0 {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info == nil {
		return
	}
	if info.Size() < maxBytes {
		return
	}
	keep := parseIntEnv("AGENT_GATEWAY_HOOK_SPOOL_ROTATE_KEEP", 3)
	if keep < 0 {
		keep = 0
	}
	ts := time.Now().Format("20060102_150405")
	rotated := path + "." + ts
	_ = os.Rename(path, rotated)
	// Best-effort: delete older rotated files beyond keep.
	if keep == 0 {
		return
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path) + "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var olds []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, base) {
			olds = append(olds, filepath.Join(dir, name))
		}
	}
	sort.Strings(olds)
	if len(olds) <= keep {
		return
	}
	for _, p := range olds[:len(olds)-keep] {
		_ = os.Remove(p)
	}
}

func (r *Runner) hookSpoolReplayLoop(ctx context.Context) {
	interval := parseIntEnv("AGENT_GATEWAY_HOOK_SPOOL_REPLAY_SECONDS", 10)
	if interval <= 0 {
		interval = 10
	}
	t := time.NewTicker(time.Duration(interval) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.replayHookSpoolOnce(ctx)
		}
	}
}

func (r *Runner) replayHookSpoolOnce(ctx context.Context) {
	hookURL := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_URL"))
	if hookURL == "" {
		return
	}
	path := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SPOOL_PATH"))
	if path == "" {
		path = r.hookSpoolPath
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	r.hookSpoolMu.Lock()
	defer r.hookSpoolMu.Unlock()
	r.rotateHookSpoolLocked(path)

	bs, err := os.ReadFile(path)
	if err != nil || len(bs) == 0 {
		return
	}
	lines := strings.Split(string(bs), "\n")
	if len(lines) == 0 {
		return
	}
	maxLines := parseIntEnv("AGENT_GATEWAY_HOOK_SPOOL_MAX_LINES", 2000)
	if maxLines <= 0 {
		maxLines = 2000
	}
	remaining := make([]string, 0, len(lines))
	seen := map[string]bool{}

	timeout := parseIntEnv("AGENT_GATEWAY_HOOK_TIMEOUT_SECONDS", 4)
	if timeout <= 0 {
		timeout = 4
	}
	secret := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SECRET"))
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	sent := 0
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		var e hookSpoolEntry
		if err := json.Unmarshal([]byte(ln), &e); err != nil || strings.TrimSpace(e.Body) == "" {
			continue
		}
		if shouldDedupSpool() && strings.TrimSpace(e.ID) != "" {
			if seen[e.ID] {
				continue
			}
			seen[e.ID] = true
		}
		bodyBytes := []byte(e.Body)
		reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, hookURL, bytes.NewReader(bodyBytes))
		if err != nil {
			cancel()
			remaining = append(remaining, ln)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Agent-Event", strings.TrimSpace(e.Type))
		if strings.TrimSpace(e.ID) != "" {
			req.Header.Set("X-Agent-Event-Id", strings.TrimSpace(e.ID))
		}
		req.Header.Set("X-Agent-Timestamp", ts)
		if secret != "" {
			req.Header.Set("X-Agent-Signature", signHook(secret, ts, bodyBytes))
		}
		resp, err := client.Do(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		cancel()
		if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			sent++
			if sent >= 200 {
				// Limit work per tick.
				continue
			}
			continue
		}
		remaining = append(remaining, ln)
		if len(remaining) >= maxLines {
			break
		}
	}

	// Rewrite spool with remaining.
	out := strings.Join(remaining, "\n")
	if len(remaining) > 0 && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	_ = os.WriteFile(path, []byte(out), 0o644)
}

func shouldDedupSpool() bool {
	v := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SPOOL_DEDUP"))
	if v == "" {
		return true
	}
	return !strings.EqualFold(v, "false")
}

func (r *Runner) loadHookSpoolSeen() {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SPOOL")), "true") {
		return
	}
	path := strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_SPOOL_PATH"))
	if path == "" {
		path = r.hookSpoolPath
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	bs, err := os.ReadFile(path)
	if err != nil || len(bs) == 0 {
		return
	}
	lines := strings.Split(string(bs), "\n")
	limit := 5000
	if r.hookSpoolSeen == nil {
		r.hookSpoolSeen = map[string]time.Time{}
	}
	now := time.Now()
	for i := len(lines) - 1; i >= 0 && limit > 0; i-- {
		ln := strings.TrimSpace(lines[i])
		if ln == "" {
			continue
		}
		var e hookSpoolEntry
		if err := json.Unmarshal([]byte(ln), &e); err != nil {
			continue
		}
		if id := strings.TrimSpace(e.ID); id != "" {
			if _, ok := r.hookSpoolSeen[id]; !ok {
				r.hookSpoolSeen[id] = now
				limit--
			}
		}
	}
	r.trimHookSpoolSeenLocked()
}

func (r *Runner) trimHookSpoolSeenLocked() {
	// Cap the in-memory set to avoid unbounded growth.
	max := parseIntEnv("AGENT_GATEWAY_HOOK_SPOOL_DEDUP_MAX", 5000)
	if max <= 0 {
		max = 5000
	}
	if len(r.hookSpoolSeen) <= max {
		return
	}
	// Remove oldest entries (O(n log n) but max is small).
	type kv struct {
		id string
		t  time.Time
	}
	items := make([]kv, 0, len(r.hookSpoolSeen))
	for id, t := range r.hookSpoolSeen {
		items = append(items, kv{id: id, t: t})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].t.Before(items[j].t) })
	cut := len(items) - max
	for i := 0; i < cut; i++ {
		delete(r.hookSpoolSeen, items[i].id)
	}
}
