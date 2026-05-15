package gateway

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

	mu                  sync.Mutex
	cancelCurrent       context.CancelFunc
	running             bool
	lastApprovalID      string
	lastApprovalCommand string
	lastApprovalReason  string
}

func deliveryHooksEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("AGENT_GATEWAY_HOOK_DELIVERY")), "true")
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
		for k, v := range meta {
			data[k] = v
		}
		w.runner.emitHook("gateway.delivery.send", data)
	}
	return res, err
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
		for k, v := range meta {
			data[k] = v
		}
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
			for k, v := range meta {
				data[k] = v
			}
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
		for k, v := range meta {
			data[k] = v
		}
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
			key:     sessionKey,
			adapter: adapter,
			engine:  &engCopy,
			allowed: allowed,
			runner:  r,
			queue:   make(chan MessageEvent, 32),
		}
		r.sessions[sessionKey] = w
		go w.run(ctx)
	}
	r.sessionsMu.Unlock()

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
	allowed := ""
	if w != nil {
		allowed = w.allowed
	}
	authorized := CheckAuthorization(allowed, event.UserID)

	// Minimal slash commands for Hermes parity.
	cmd := normalizeGatewayCommand(w.adapter.Name(), strings.TrimSpace(event.Text))
	if strings.HasPrefix(cmd, "/") {
		switch strings.ToLower(strings.Fields(cmd)[0]) {
		case "/pair":
			if w.runner == nil {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Pairing unavailable._", event.MessageID)
				return
			}
			code := ""
			parts := strings.Fields(cmd)
			if len(parts) >= 2 {
				code = strings.TrimSpace(parts[1])
			}
			if w.runner.tryPair(w.adapter.Name(), event.UserID, code) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Paired successfully._", event.MessageID)
			} else {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Pair failed._", event.MessageID)
			}
			return
		case "/unpair":
			if w.runner == nil {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Unpair unavailable._", event.MessageID)
				return
			}
			if w.runner.unpair(w.adapter.Name(), event.UserID) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Unpaired._", event.MessageID)
			} else {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Not paired._", event.MessageID)
			}
			return
		case "/cancel", "/stop":
			if !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
				return
			}
			w.mu.Lock()
			cancel := w.cancelCurrent
			running := w.running
			w.mu.Unlock()
			if cancel != nil && running {
				cancel()
				_, _ = w.sendText(ctx, event.ChatID, "_Cancelled._", event.MessageID, map[string]any{"slash": "/cancel"})
			} else {
				_, _ = w.sendText(ctx, event.ChatID, "_No active task._", event.MessageID, map[string]any{"slash": "/cancel"})
			}
			return
		case "/queue":
			if !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
				return
			}
			qLen := len(w.queue)
			_, _ = w.sendText(ctx, event.ChatID, "_Queue length: "+itoa(qLen)+"_", event.MessageID, map[string]any{"slash": "/queue"})
			return
		case "/status":
			if !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
				return
			}
			reply := w.gatewayStatusText()
			_, _ = w.sendText(ctx, event.ChatID, escapeMarkdown(reply), event.MessageID, map[string]any{"slash": "/status"})
			return
		case "/approve", "/deny":
			if !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
				return
			}
			approve := strings.EqualFold(strings.Fields(cmd)[0], "/approve")
			approvalID := w.resolveApprovalID(cmd)
			if approvalID == "" {
				_, _ = w.sendText(ctx, event.ChatID, "Usage: /approve <approval_id> or /deny <approval_id>", event.MessageID, map[string]any{"slash": strings.Fields(cmd)[0]})
				return
			}
			reply := w.confirmApproval(ctx, approvalID, approve)
			_, _ = w.sendText(ctx, event.ChatID, escapeMarkdown(reply), event.MessageID, map[string]any{"slash": strings.Fields(cmd)[0], "approval_id": approvalID})
			return
		case "/approvals", "/approval":
			if !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
				return
			}
			reply := w.approvalStatus(ctx)
			_, _ = w.sendText(ctx, event.ChatID, escapeMarkdown(reply), event.MessageID, map[string]any{"slash": strings.Fields(cmd)[0]})
			return
		case "/pending":
			if !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
				return
			}
			reply := w.pendingApprovalStatus()
			if w.adapter.Name() == "yuanbao" && !strings.Contains(reply, "No pending approval.") {
				reply += "\nquick_reply: 批准 / 拒绝"
			}
			meta := map[string]any{"slash": "/pending"}
			if approvalID := w.resolveApprovalID(""); strings.TrimSpace(approvalID) != "" {
				meta["approval_id"] = approvalID
			}
			_, _ = w.sendText(ctx, event.ChatID, escapeMarkdown(reply), event.MessageID, meta)
			return
		case "/grant":
			if !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
				return
			}
			reply := w.grantApproval(ctx, cmd)
			_, _ = w.sendText(ctx, event.ChatID, escapeMarkdown(reply), event.MessageID, map[string]any{"slash": "/grant"})
			return
		case "/revoke":
			if !authorized && (w.runner == nil || !w.runner.isPaired(w.adapter.Name(), event.UserID)) {
				_, _ = w.adapter.Send(ctx, event.ChatID, "_Access denied._", event.MessageID)
				return
			}
			reply := w.revokeApproval(ctx, cmd)
			_, _ = w.sendText(ctx, event.ChatID, escapeMarkdown(reply), event.MessageID, map[string]any{"slash": "/revoke"})
			return
		case "/help":
			helpText := "Commands: /pair <code>, /unpair, /cancel, /queue, /status, /pending, /approvals, /grant [ttl], /grant pattern <name> [ttl], /revoke, /revoke pattern <name>, /approve <id>, /deny <id>, /help"
			if w.adapter.Name() == "yuanbao" {
				helpText += "\nQuick reply aliases: 状态, 待审批, 审批, 批准, 拒绝, 帮助"
			}
			_, _ = w.sendText(ctx, event.ChatID, helpText, event.MessageID, map[string]any{"slash": "/help"})
			return
		}
	}

	if !authorized {
		if w.runner != nil && w.runner.isPaired(w.adapter.Name(), event.UserID) {
			authorized = true
		}
	}
	if !authorized {
		_, _ = w.sendText(ctx, event.ChatID, "_Access denied._", event.MessageID, map[string]any{"auth": "denied"})
		return
	}

	sessionKey := w.key

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
	res, runErr := eng.Run(runCtx, sessionKey, userInput, agent.DefaultSystemPrompt(), history)

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
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return ""
		}
		head := strings.TrimPrefix(parts[0], "/")
		if canonical, ok := ResolveGatewayCommand(head); ok {
			return "/" + canonical + withTail(parts)
		}
		if len(parts) == 1 {
			return "/" + strings.ToLower(head)
		}
		return "/" + strings.ToLower(head) + withTail(parts)
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
	switch strings.TrimSpace(parts[0]) {
	case "批准", "同意", "通过":
		return "/approve" + withTail(parts)
	case "拒绝", "驳回":
		return "/deny" + withTail(parts)
	case "状态":
		return "/status"
	case "待审批":
		return "/pending"
	case "审批":
		return "/approvals"
	case "帮助":
		return "/help"
	default:
		return cmd
	}
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

func (w *sessionWorker) resolveApprovalID(cmd string) string {
	parts := strings.Fields(strings.TrimSpace(cmd))
	if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[1])
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
		return "Approval failed: " + errText
	}
	approved, _ := parsed["approved"].(bool)
	command, _ := parsed["command"].(string)
	if !approved {
		w.clearApprovalID(approvalID)
		if strings.TrimSpace(command) == "" {
			return "Denied."
		}
		return "Denied: " + command
	}
	w.clearApprovalID(approvalID)
	output, _ := parsed["output"].(string)
	output = strings.TrimSpace(output)
	if output != "" {
		return "Approved and executed.\n" + truncateString(output, 1500)
	}
	if strings.TrimSpace(command) != "" {
		return "Approved: " + command
	}
	return "Approved."
}

func (w *sessionWorker) approvalStatus(ctx context.Context) string {
	raw := w.engine.Registry.Dispatch(ctx, "approval", map[string]any{
		"action": "status",
	}, w.approvalToolContext())
	parsed := tools.ParseJSONArgs(raw)
	if errText, _ := parsed["error"].(string); strings.TrimSpace(errText) != "" {
		return "Approval status failed: " + errText
	}
	approvals, _ := parsed["approvals"].([]any)
	if len(approvals) == 0 {
		return "No active approvals."
	}
	lines := make([]string, 0, len(approvals)+1)
	if approved, _ := parsed["approved"].(bool); approved {
		lines = append(lines, "Session approval: active")
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
		return "No active approvals."
	}
	return strings.Join(lines, "\n")
}

func (w *sessionWorker) gatewayStatusText() string {
	lines := []string{
		"platform: " + w.adapter.Name(),
		"session: " + w.key,
		"queue: " + itoa(len(w.queue)),
	}
	if w.runner != nil {
		paired := w.runner.isPaired(w.adapter.Name(), w.engine.GatewayUserID)
		if paired {
			lines = append(lines, "paired: yes")
		} else {
			lines = append(lines, "paired: no")
		}
	}
	lines = append(lines, "running: "+map[bool]string{true: "yes", false: "no"}[w.isRunning()])
	if last := w.resolveApprovalID(""); strings.TrimSpace(last) != "" {
		lines = append(lines, "last_approval_id: "+last)
	}
	return strings.Join(lines, "\n")
}

func (w *sessionWorker) pendingApprovalStatus() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(w.lastApprovalID) == "" {
		return "No pending approval."
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

func (w *sessionWorker) grantApproval(ctx context.Context, cmd string) string {
	args, usageErr := parseApprovalManageCommand(cmd)
	if usageErr != "" {
		return usageErr
	}
	args["action"] = "grant"
	raw := w.engine.Registry.Dispatch(ctx, "approval", args, w.approvalToolContext())
	parsed := tools.ParseJSONArgs(raw)
	if errText, _ := parsed["error"].(string); strings.TrimSpace(errText) != "" {
		return "Grant failed: " + errText
	}
	scope, _ := parsed["scope"].(string)
	pattern, _ := parsed["pattern"].(string)
	expiresAt, _ := parsed["expires_at"].(string)
	if strings.TrimSpace(scope) == "pattern" && strings.TrimSpace(pattern) != "" {
		if strings.TrimSpace(expiresAt) != "" {
			return "Granted pattern approval: " + pattern + " until " + expiresAt
		}
		return "Granted pattern approval: " + pattern
	}
	if strings.TrimSpace(expiresAt) != "" {
		return "Granted session approval until " + expiresAt
	}
	return "Granted session approval."
}

func (w *sessionWorker) revokeApproval(ctx context.Context, cmd string) string {
	args, usageErr := parseApprovalManageCommand(cmd)
	if usageErr != "" {
		return usageErr
	}
	args["action"] = "revoke"
	raw := w.engine.Registry.Dispatch(ctx, "approval", args, w.approvalToolContext())
	parsed := tools.ParseJSONArgs(raw)
	if errText, _ := parsed["error"].(string); strings.TrimSpace(errText) != "" {
		return "Revoke failed: " + errText
	}
	scope, _ := parsed["scope"].(string)
	pattern, _ := parsed["pattern"].(string)
	revoked, _ := parsed["revoked"].(bool)
	if strings.TrimSpace(scope) == "pattern" && strings.TrimSpace(pattern) != "" {
		if revoked {
			return "Revoked pattern approval: " + pattern
		}
		return "Pattern approval not found: " + pattern
	}
	if revoked {
		return "Revoked session approval."
	}
	return "No active session approval."
}

func parseApprovalManageCommand(cmd string) (map[string]any, string) {
	parts := strings.Fields(strings.TrimSpace(cmd))
	args := map[string]any{}
	if len(parts) <= 1 {
		return args, ""
	}
	if strings.EqualFold(parts[1], "pattern") {
		if len(parts) < 3 || strings.TrimSpace(parts[2]) == "" {
			return nil, "Usage: /grant pattern <name> [ttl] or /revoke pattern <name>"
		}
		args["scope"] = "pattern"
		args["pattern"] = strings.TrimSpace(parts[2])
		if len(parts) >= 4 {
			if ttl, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil && ttl >= 0 {
				args["ttl_seconds"] = ttl
			} else if strings.HasPrefix(parts[0], "/grant") {
				return nil, "Usage: /grant pattern <name> [ttl]"
			}
		}
		return args, ""
	}
	if ttl, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && ttl >= 0 {
		args["ttl_seconds"] = ttl
		return args, ""
	}
	return nil, "Usage: /grant [ttl], /grant pattern <name> [ttl], /revoke, or /revoke pattern <name>"
}

func (w *sessionWorker) approvalToolContext() tools.ToolContext {
	return tools.ToolContext{
		SessionID:      w.key,
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
