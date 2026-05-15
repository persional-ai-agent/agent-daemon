package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/slashcmd"
	clitools "github.com/dingjingmaster/agent-daemon/internal/tools"
)

type sessionLister interface {
	ListRecentSessions(limit int) ([]map[string]any, error)
}

type sessionDetailer interface {
	LoadMessagesPage(sessionID string, offset, limit int) ([]core.Message, error)
	SessionStats(sessionID string) (map[string]any, error)
}

type sessionCompactor interface {
	CompactSession(sessionID string, keepLastN int) (before int, after int, err error)
}

type chatState struct {
	SessionID    string
	SystemPrompt string
	History      []core.Message
}

func printCLIEnvelope(ok bool, payload map[string]any, errCode, errMsg string) {
	out := map[string]any{
		"ok":          ok,
		"api_version": "v1",
		"compat":      "2026-05-13",
	}
	if ok {
		for k, v := range payload {
			out[k] = v
		}
	} else {
		out["error"] = map[string]any{
			"code":    errCode,
			"message": errMsg,
		}
	}
	bs, _ := json.Marshal(out)
	fmt.Println(string(bs))
}

func RunChat(ctx context.Context, eng *agent.Engine, sessionID, firstMessage, preloadSkills string) error {
	reader := bufio.NewReader(os.Stdin)
	var history []core.Message
	if eng.SessionStore != nil {
		history, _ = eng.SessionStore.LoadMessages(sessionID, 500)
	}
	systemPrompt := agent.DefaultSystemPrompt()

	if strings.TrimSpace(preloadSkills) != "" {
		block := buildPreloadedSkillsBlock(eng.Workdir, preloadSkills)
		if block != "" {
			systemPrompt = systemPrompt + "\n\n" + block
		}
	}
	state := &chatState{SessionID: sessionID, SystemPrompt: systemPrompt, History: history}

	if strings.TrimSpace(firstMessage) != "" {
		res, err := runWithContextRecovery(ctx, eng, state, firstMessage)
		if err != nil {
			return err
		}
		state.History = res.Messages
		fmt.Println(res.FinalResponse)
	}
	printChatWelcome(state.SessionID)
	for {
		fmt.Print("agent> ")
		eofRead := false
		line, err := reader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			eofRead = true
			line = strings.TrimSpace(line)
			if line == "" {
				return nil
			}
		} else {
			line = strings.TrimSpace(line)
		}
		if line == "" {
			continue
		}
		line = slashcmd.NormalizeInput(line)
		if strings.EqualFold(line, "/exit") || strings.EqualFold(line, "/quit") {
			return nil
		}
		if strings.HasPrefix(line, "/") {
			handled, err := handleSlashCommandState(ctx, line, state, eng)
			if err != nil {
				return err
			}
			if handled {
				continue
			}
		}
		res, err := runWithContextRecovery(ctx, eng, state, line)
		if err != nil {
			return err
		}
		state.History = append([]core.Message(nil), res.Messages...)
		fmt.Println(res.FinalResponse)
		if eofRead {
			return nil
		}
	}
}

func printChatWelcome(sessionID string) {
	fmt.Printf("session: %s\n", sessionID)
	fmt.Println("输入 /help 查看可用命令，/quit 退出。")
}

func handleSlashCommand(line, sessionID, systemPrompt string, history []core.Message, eng *agent.Engine) ([]core.Message, string, bool, error) {
	state := &chatState{SessionID: sessionID, SystemPrompt: systemPrompt, History: history}
	handled, err := handleSlashCommandState(context.Background(), line, state, eng)
	return state.History, state.SystemPrompt, handled, err
}

func handleSlashCommandState(ctx context.Context, line string, state *chatState, eng *agent.Engine) (bool, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return true, nil
	}
	cmd := strings.ToLower(strings.TrimSpace(fields[0]))
	switch cmd {
	case "/help", "/commands":
		printSlashHelp()
		return true, nil
	case "/session", "/status":
		printCLIEnvelope(true, map[string]any{"session_id": state.SessionID, "messages_in_context": len(state.History), "tools": len(eng.Registry.Names())}, "", "")
		return true, nil
	case "/new", "/reset":
		nextID := uuid.NewString()
		if len(fields) > 1 {
			nextID = strings.TrimSpace(fields[1])
		}
		state.SessionID = nextID
		state.History = nil
		printCLIEnvelope(true, map[string]any{"session_id": state.SessionID, "reset": true}, "", "")
		return true, nil
	case "/resume":
		if len(fields) != 2 {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /resume <session_id>")
			return true, nil
		}
		if eng.SessionStore == nil {
			printCLIEnvelope(false, nil, "session_store_unavailable", "session store unavailable")
			return true, nil
		}
		loaded, err := eng.SessionStore.LoadMessages(fields[1], 500)
		if err != nil {
			return true, err
		}
		state.SessionID = fields[1]
		state.History = loaded
		printCLIEnvelope(true, map[string]any{"session_id": state.SessionID, "loaded_messages": len(loaded)}, "", "")
		return true, nil
	case "/tools":
		return handleToolsSlash(fields, eng), nil
	case "/toolsets":
		return handleToolsetsSlash(fields), nil
	case "/history":
		limit := 10
		if len(fields) > 1 {
			v, err := strconv.Atoi(fields[1])
			if err != nil || v <= 0 {
				printCLIEnvelope(false, nil, "invalid_argument", "用法: /history [n]  (n 必须是正整数)")
				return true, nil
			}
			limit = v
		}
		printHistoryPreview(state.History, limit)
		return true, nil
	case "/sessions":
		limit := 10
		if len(fields) > 1 {
			v, err := strconv.Atoi(fields[1])
			if err != nil || v <= 0 {
				printCLIEnvelope(false, nil, "invalid_argument", "用法: /sessions [n]  (n 必须是正整数)")
				return true, nil
			}
			limit = v
		}
		if eng.SessionStore == nil {
			printCLIEnvelope(false, nil, "session_store_unavailable", "session store unavailable")
			return true, nil
		}
		lister, ok := eng.SessionStore.(sessionLister)
		if !ok {
			printCLIEnvelope(false, nil, "not_supported", "当前会话存储不支持会话列表。")
			return true, nil
		}
		rows, err := lister.ListRecentSessions(limit)
		if err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"count": len(rows), "sessions": rows}, "", "")
		return true, nil
	case "/stats":
		target := state.SessionID
		if len(fields) > 1 {
			target = strings.TrimSpace(fields[1])
		}
		if eng.SessionStore == nil {
			printCLIEnvelope(false, nil, "session_store_unavailable", "session store unavailable")
			return true, nil
		}
		detailer, ok := eng.SessionStore.(sessionDetailer)
		if !ok {
			printCLIEnvelope(false, nil, "not_supported", "当前会话存储不支持统计信息。")
			return true, nil
		}
		stats, err := detailer.SessionStats(target)
		if err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"stats": stats}, "", "")
		return true, nil
	case "/usage":
		target := state.SessionID
		if len(fields) > 1 {
			target = strings.TrimSpace(fields[1])
		}
		if eng.SessionStore == nil {
			printCLIEnvelope(false, nil, "session_store_unavailable", "session store unavailable")
			return true, nil
		}
		detailer, ok := eng.SessionStore.(sessionDetailer)
		if !ok {
			printCLIEnvelope(false, nil, "not_supported", "当前会话存储不支持统计信息。")
			return true, nil
		}
		stats, err := detailer.SessionStats(target)
		if err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"usage": stats}, "", "")
		return true, nil
	case "/show":
		target := state.SessionID
		offset := 0
		limit := 20
		if len(fields) > 1 {
			target = strings.TrimSpace(fields[1])
		}
		if len(fields) > 2 {
			if v, err := strconv.Atoi(fields[2]); err == nil && v >= 0 {
				offset = v
			}
		}
		if len(fields) > 3 {
			if v, err := strconv.Atoi(fields[3]); err == nil && v > 0 {
				limit = v
			}
		}
		if eng.SessionStore == nil {
			printCLIEnvelope(false, nil, "session_store_unavailable", "session store unavailable")
			return true, nil
		}
		detailer, ok := eng.SessionStore.(sessionDetailer)
		if !ok {
			printCLIEnvelope(false, nil, "not_supported", "当前会话存储不支持消息分页查看。")
			return true, nil
		}
		msgs, err := detailer.LoadMessagesPage(target, offset, limit)
		if err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"session_id": target, "offset": offset, "limit": limit, "count": len(msgs), "messages": msgs}, "", "")
		return true, nil
	case "/reload":
		if eng.SessionStore == nil {
			printCLIEnvelope(false, nil, "session_store_unavailable", "session store unavailable")
			return true, nil
		}
		loaded, err := eng.SessionStore.LoadMessages(state.SessionID, 500)
		if err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"count": len(loaded), "messages": loaded}, "", "")
		state.History = loaded
		return true, nil
	case "/clear":
		// 仅清空当前进程内上下文；不会删除持久化会话。
		printCLIEnvelope(true, map[string]any{"cleared": true}, "", "")
		state.History = nil
		return true, nil
	case "/undo":
		next, removed := removeLastTurn(state.History)
		state.History = next
		printCLIEnvelope(true, map[string]any{"removed_messages": removed, "messages_in_context": len(state.History)}, "", "")
		return true, nil
	case "/retry":
		idx := lastUserMessageIndex(state.History)
		if idx < 0 {
			printCLIEnvelope(false, nil, "not_available", "没有可重试的上一条用户消息。")
			return true, nil
		}
		userInput := state.History[idx].Content
		base := core.CloneMessages(state.History[:idx])
		old := state.History
		state.History = base
		res, err := runWithContextRecovery(ctx, eng, state, userInput)
		if err != nil {
			state.History = old
			return true, err
		}
		state.History = append([]core.Message(nil), res.Messages...)
		fmt.Println(res.FinalResponse)
		return true, nil
	case "/recover":
		if len(fields) != 2 || strings.ToLower(strings.TrimSpace(fields[1])) != "context" {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /recover context")
			return true, nil
		}
		prev := state.SessionID
		state.SessionID = uuid.NewString()
		state.History = nil
		printCLIEnvelope(true, map[string]any{"recovered": true, "previous_session_id": prev, "session_id": state.SessionID}, "", "")
		return true, nil
	case "/compress":
		tail := 20
		if len(fields) > 1 {
			v, err := strconv.Atoi(fields[1])
			if err != nil || v <= 0 {
				printCLIEnvelope(false, nil, "invalid_argument", "用法: /compress [tail_messages]")
				return true, nil
			}
			tail = v
		}
		next, meta := compactHistory(state.History, tail)
		state.History = next
		printCLIEnvelope(true, meta, "", "")
		return true, nil
	case "/save":
		path := ""
		if len(fields) > 1 {
			path = strings.TrimSpace(fields[1])
		}
		saved, err := saveHistory(eng.Workdir, state.SessionID, state.History, path)
		if err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"path": saved, "messages": len(state.History)}, "", "")
		return true, nil
	case "/sethome":
		if len(fields) == 1 || len(fields) > 3 {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /sethome <platform> <chat_id> | /sethome <platform:chat_id>")
			return true, nil
		}
		p, cid, err := clitools.ParseSetHomeArgs(fields[1:])
		if err != nil {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /sethome <platform> <chat_id> | /sethome <platform:chat_id>")
			return true, nil
		}
		env := clitools.HomeTargetEnvVar(p)
		_ = os.Setenv(env, cid)
		_ = clitools.SetHomeTarget(eng.Workdir, p, cid)
		printCLIEnvelope(true, map[string]any{"platform": p, "chat_id": cid, "home_target": p + ":" + cid, "env": env}, "", "")
		return true, nil
	case "/targets":
		filter := ""
		if len(fields) > 2 {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /targets [platform]")
			return true, nil
		}
		if len(fields) == 2 {
			filter = strings.ToLower(strings.TrimSpace(fields[1]))
		}
		_, out, err := clitools.BuildDeliveryTargets(eng.Workdir, filter)
		if err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"count": len(out), "platform": filter, "targets": out}, "", "")
		return true, nil
	case "/continuity":
		if len(fields) > 2 {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /continuity [off|user_id|user_name]")
			return true, nil
		}
		if len(fields) == 1 {
			mode, err := clitools.ResolveGatewayContinuityMode(eng.Workdir)
			if err != nil {
				return true, err
			}
			printCLIEnvelope(true, map[string]any{"continuity_mode": mode}, "", "")
			return true, nil
		}
		mode, parseErr := clitools.ParseGatewayContinuityModeArg(fields[1:])
		if parseErr != nil {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /continuity [off|user_id|user_name]")
			return true, nil
		}
		if err := clitools.SetGatewaySetting(eng.Workdir, "continuity_mode", mode); err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"continuity_mode": mode}, "", "")
		return true, nil
	case "/whoami":
		ref, parseErr := clitools.ParseGatewayIdentityRefArgs(fields[1:])
		if parseErr != nil {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /whoami <platform> <user_id>")
			return true, nil
		}
		globalID, err := clitools.ResolveGatewayIdentity(eng.Workdir, ref.Platform, ref.UserID)
		if err != nil {
			return true, err
		}
		mode, _ := clitools.ResolveGatewayContinuityMode(eng.Workdir)
		printCLIEnvelope(true, map[string]any{
			"platform":        ref.Platform,
			"user_id":         ref.UserID,
			"global_id":       globalID,
			"continuity_mode": mode,
		}, "", "")
		return true, nil
	case "/resolve":
		resolvedArgs, parseErr := clitools.ParseGatewayResolveArgs(fields[1:])
		if parseErr != nil {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /resolve <platform> <chat_type> <chat_id> <user_id> [user_name]")
			return true, nil
		}
		resolved, err := clitools.ResolveGatewaySessionMapping(eng.Workdir, resolvedArgs.Platform, resolvedArgs.ChatType, resolvedArgs.ChatID, resolvedArgs.UserID, resolvedArgs.UserName)
		if err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{
			"platform":         resolved.Platform,
			"chat_type":        resolved.ChatType,
			"chat_id":          resolved.ChatID,
			"user_id":          resolved.UserID,
			"user_name":        resolved.UserName,
			"route_session":    resolved.RouteSession,
			"mapped_session":   resolved.MappedSession,
			"resolved_session": resolved.ResolvedSession,
			"global_id":        resolved.GlobalID,
			"global_source":    resolved.GlobalSource,
			"continuity_mode":  resolved.ContinuityMode,
		}, "", "")
		return true, nil
	case "/setid":
		setArgs, parseErr := clitools.ParseGatewaySetIdentityArgs(fields[1:])
		if parseErr != nil {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /setid <platform> <user_id> <global_user_id>")
			return true, nil
		}
		if err := clitools.UpsertGatewayIdentity(eng.Workdir, setArgs.Platform, setArgs.UserID, setArgs.GlobalID); err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"platform": setArgs.Platform, "user_id": setArgs.UserID, "global_id": setArgs.GlobalID, "updated": true}, "", "")
		return true, nil
	case "/unsetid":
		ref, parseErr := clitools.ParseGatewayIdentityRefArgs(fields[1:])
		if parseErr != nil {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /unsetid <platform> <user_id>")
			return true, nil
		}
		if err := clitools.DeleteGatewayIdentity(eng.Workdir, ref.Platform, ref.UserID); err != nil {
			return true, err
		}
		printCLIEnvelope(true, map[string]any{"platform": ref.Platform, "user_id": ref.UserID, "deleted": true}, "", "")
		return true, nil
	case "/todo":
		if eng.TodoStore == nil {
			printCLIEnvelope(false, nil, "not_available", "todo store unavailable")
			return true, nil
		}
		items := eng.TodoStore.List(state.SessionID)
		printCLIEnvelope(true, map[string]any{"count": len(items), "todos": items}, "", "")
		return true, nil
	case "/memory":
		if eng.MemoryStore == nil {
			printCLIEnvelope(false, nil, "not_available", "memory store unavailable")
			return true, nil
		}
		snapshot, err := eng.MemoryStore.Snapshot()
		if err != nil {
			return true, err
		}
		if len(fields) > 1 {
			target := strings.ToLower(strings.TrimSpace(fields[1]))
			printCLIEnvelope(true, map[string]any{"target": target, "content": snapshot[target]}, "", "")
			return true, nil
		}
		printCLIEnvelope(true, map[string]any{"memory": snapshot}, "", "")
		return true, nil
	case "/model":
		if len(fields) > 3 {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /model [provider:model|provider model]")
			return true, nil
		}
		provider, _ := clitools.GetGatewaySetting(eng.Workdir, "model_provider")
		modelName, _ := clitools.GetGatewaySetting(eng.Workdir, "model_name")
		baseURL, _ := clitools.GetGatewaySetting(eng.Workdir, "model_base_url")
		if strings.TrimSpace(provider) == "" {
			provider = strings.TrimSpace(os.Getenv("AGENT_MODEL_PROVIDER"))
		}
		if strings.TrimSpace(modelName) == "" {
			modelName = strings.TrimSpace(os.Getenv("AGENT_MODEL"))
		}
		if strings.TrimSpace(baseURL) == "" {
			baseURL = strings.TrimSpace(os.Getenv("AGENT_BASE_URL"))
		}
		if len(fields) > 1 {
			nextProvider, nextModel, ok := parseCLIModelSpec(fields[1:])
			if !ok {
				printCLIEnvelope(false, nil, "invalid_argument", "用法: /model [provider:model|provider model]")
				return true, nil
			}
			if err := clitools.SetGatewaySetting(eng.Workdir, "model_provider", nextProvider); err != nil {
				return true, err
			}
			if err := clitools.SetGatewaySetting(eng.Workdir, "model_name", nextModel); err != nil {
				return true, err
			}
			_ = os.Setenv("AGENT_MODEL_PROVIDER", nextProvider)
			_ = os.Setenv("AGENT_MODEL", nextModel)
			printCLIEnvelope(true, map[string]any{
				"updated":  true,
				"provider": nextProvider,
				"model":    nextModel,
				"note":     "model preference updated; takes effect for newly started daemon/model client",
			}, "", "")
			return true, nil
		}
		if provider == "" {
			provider = "openai"
		}
		if modelName == "" {
			modelName = "(default)"
		}
		if baseURL == "" {
			baseURL = "(default)"
		}
		printCLIEnvelope(true, map[string]any{"client": fmt.Sprintf("%T", eng.Client), "provider": provider, "model": modelName, "base_url": baseURL}, "", "")
		return true, nil
	case "/personality":
		if len(fields) == 1 || strings.EqualFold(strings.TrimSpace(fields[1]), "show") {
			printCLIEnvelope(true, map[string]any{"system_prompt": state.SystemPrompt}, "", "")
			return true, nil
		}
		if strings.EqualFold(strings.TrimSpace(fields[1]), "reset") {
			state.SystemPrompt = agent.DefaultSystemPrompt()
			printCLIEnvelope(true, map[string]any{"reset": true, "system_prompt": state.SystemPrompt}, "", "")
			return true, nil
		}
		state.SystemPrompt = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		printCLIEnvelope(true, map[string]any{"updated": true, "system_prompt": state.SystemPrompt}, "", "")
		return true, nil
	case "/cancel", "/stop":
		// Interactive CLI currently has no async running task handle to cancel.
		printCLIEnvelope(false, nil, "not_supported", "当前 CLI 模式不支持 /cancel；请使用 Ctrl+C 中断当前轮。")
		return true, nil
	case "/tui":
		printTUIStatus(eng)
		return true, nil
	default:
		printCLIEnvelope(false, nil, "unknown_command", fmt.Sprintf("未知命令: %s（输入 /help 查看命令）", fields[0]))
		return true, nil
	}
}

func runWithContextRecovery(ctx context.Context, eng *agent.Engine, state *chatState, userInput string) (*core.RunResult, error) {
	res, err := eng.Run(ctx, state.SessionID, userInput, state.SystemPrompt, state.History)
	if err == nil {
		return res, nil
	}
	if !isContextLimitError(err) {
		return nil, err
	}
	fmt.Println("[context] 上下文超限，正在压缩当前会话并重试...")
	if compactor, ok := eng.SessionStore.(sessionCompactor); ok && compactor != nil {
		_, _, _ = compactor.CompactSession(state.SessionID, 20)
	}
	state.History, _ = compactHistory(state.History, 20)
	res, err = eng.Run(ctx, state.SessionID, userInput, state.SystemPrompt, state.History)
	if err == nil {
		fmt.Println("[context] 重试成功。")
		return res, nil
	}
	if !isContextLimitError(err) {
		return nil, err
	}
	prev := state.SessionID
	state.SessionID = uuid.NewString()
	state.History = nil
	fmt.Printf("[context] 压缩后仍超限，切换新会话重试：%s -> %s\n", prev, state.SessionID)
	res, err = eng.Run(ctx, state.SessionID, userInput, state.SystemPrompt, state.History)
	if err != nil {
		return nil, err
	}
	fmt.Println("[context] 新会话重试成功。")
	return res, nil
}

func isContextLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "exceed_context_size_error") ||
		strings.Contains(msg, "exceeds the available context size") ||
		(strings.Contains(msg, "context size") && strings.Contains(msg, "exceed"))
}

func printSlashHelp() {
	lines := slashcmd.CLIHelpLines()
	fmt.Println(strings.Join(lines, "\n"))
}

func handleToolsSlash(fields []string, eng *agent.Engine) bool {
	sub := "list"
	if len(fields) > 1 {
		sub = strings.ToLower(strings.TrimSpace(fields[1]))
	}
	switch sub {
	case "list":
		names := eng.Registry.Names()
		printCLIEnvelope(true, map[string]any{"count": len(names), "tools": names}, "", "")
	case "show":
		if len(fields) != 3 {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /tools show <tool_name>")
			return true
		}
		for _, schema := range eng.Registry.Schemas() {
			if schema.Function.Name == fields[2] {
				printCLIEnvelope(true, map[string]any{"schema": schema}, "", "")
				return true
			}
		}
		printCLIEnvelope(false, nil, "not_found", "tool not found: "+fields[2])
	case "schemas":
		schemas := eng.Registry.Schemas()
		printCLIEnvelope(true, map[string]any{"count": len(schemas), "schemas": schemas}, "", "")
	default:
		printCLIEnvelope(false, nil, "invalid_argument", "用法: /tools [list|show <tool>|schemas]")
	}
	return true
}

func handleToolsetsSlash(fields []string) bool {
	sub := "list"
	if len(fields) > 1 {
		sub = strings.ToLower(strings.TrimSpace(fields[1]))
	}
	switch sub {
	case "list":
		items := clitools.ListToolsets()
		printCLIEnvelope(true, map[string]any{"count": len(items), "toolsets": items}, "", "")
	case "show":
		if len(fields) != 3 {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /toolsets show <name>")
			return true
		}
		ts, ok := clitools.GetToolset(fields[2])
		if !ok {
			printCLIEnvelope(false, nil, "not_found", "toolset not found: "+fields[2])
			return true
		}
		printCLIEnvelope(true, map[string]any{"toolset": ts}, "", "")
	case "resolve":
		if len(fields) != 3 {
			printCLIEnvelope(false, nil, "invalid_argument", "用法: /toolsets resolve <name[,name]>")
			return true
		}
		allowed, err := clitools.ResolveToolset(splitCSV(fields[2]))
		if err != nil {
			printCLIEnvelope(false, nil, "invalid_argument", err.Error())
			return true
		}
		names := make([]string, 0, len(allowed))
		for name := range allowed {
			names = append(names, name)
		}
		sort.Strings(names)
		printCLIEnvelope(true, map[string]any{"count": len(names), "tools": names}, "", "")
	default:
		printCLIEnvelope(false, nil, "invalid_argument", "用法: /toolsets [list|show <name>|resolve <name[,name]>]")
	}
	return true
}

func printHistoryPreview(history []core.Message, limit int) {
	if len(history) == 0 {
		fmt.Println("历史为空。")
		return
	}
	if limit > len(history) {
		limit = len(history)
	}
	start := len(history) - limit
	for i := start; i < len(history); i++ {
		msg := history[i]
		content := strings.TrimSpace(msg.Content)
		if len(content) > 120 {
			content = content[:120] + "..."
		}
		if content == "" {
			content = "(empty)"
		}
		fmt.Printf("%d. [%s] %s\n", i+1, msg.Role, content)
	}
}

func printTUIStatus(eng *agent.Engine) {
	tools := eng.Registry.Names()
	sort.Strings(tools)
	fmt.Println("CLI/TUI 状态")
	fmt.Printf("- 运行模式: 交互式 CLI（slash 命令已启用）\n")
	fmt.Printf("- 当前工具数量: %d\n", len(tools))
	fmt.Printf("- 会话命令: /new, /resume, /retry, /undo, /compress, /save\n")
	fmt.Printf("- 管理命令: /tools, /toolsets, /todo, /memory, /model\n")
	fmt.Printf("- Lite TUI: agentd tui -mode lite 会显示实时运行事件\n")
}

func buildPreloadedSkillsBlock(workdir, skillsCSV string) string {
	names := strings.Split(skillsCSV, ",")
	var parts []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path := filepath.Join(workdir, "skills", name, "SKILL.md")
		bs, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parts = append(parts, "## Preloaded Skill: "+name+"\n"+string(bs))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[IMPORTANT: The user has preloaded the following skill(s) for this session. Follow their instructions carefully.]\n\n" + strings.Join(parts, "\n\n")
}

func removeLastTurn(history []core.Message) ([]core.Message, int) {
	idx := lastUserMessageIndex(history)
	if idx < 0 {
		return history, 0
	}
	return core.CloneMessages(history[:idx]), len(history) - idx
}

func lastUserMessageIndex(history []core.Message) int {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			return i
		}
	}
	return -1
}

func compactHistory(history []core.Message, tail int) ([]core.Message, map[string]any) {
	if tail <= 0 {
		tail = 20
	}
	if len(history) <= tail+1 {
		return history, map[string]any{"compacted": false, "before": len(history), "after": len(history), "reason": "history shorter than tail"}
	}
	head := history[:len(history)-tail]
	recent := history[len(history)-tail:]
	summary := summarizeForCLI(head, 6000)
	next := make([]core.Message, 0, len(recent)+1)
	next = append(next, core.Message{Role: "assistant", Content: "[Context summary created by /compress]\n" + summary})
	next = append(next, core.CloneMessages(recent)...)
	return next, map[string]any{"compacted": true, "before": len(history), "after": len(next), "summarized_messages": len(head), "tail_messages": tail}
}

func parseCLIModelSpec(args []string) (provider, modelName string, ok bool) {
	if len(args) == 1 {
		spec := strings.TrimSpace(args[0])
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		provider = strings.ToLower(strings.TrimSpace(parts[0]))
		modelName = strings.TrimSpace(parts[1])
		if provider == "" || modelName == "" {
			return "", "", false
		}
		return provider, modelName, true
	}
	if len(args) == 2 {
		provider = strings.ToLower(strings.TrimSpace(args[0]))
		modelName = strings.TrimSpace(args[1])
		if provider == "" || modelName == "" {
			return "", "", false
		}
		return provider, modelName, true
	}
	return "", "", false
}

func summarizeForCLI(messages []core.Message, budget int) string {
	var b strings.Builder
	for _, msg := range messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" && len(msg.ToolCalls) > 0 {
			content = fmt.Sprintf("%d tool call(s)", len(msg.ToolCalls))
		}
		if content == "" {
			continue
		}
		line := fmt.Sprintf("- %s: %s\n", msg.Role, oneLine(content, 500))
		if b.Len()+len(line) > budget {
			b.WriteString("- ... older context truncated by CLI compression\n")
			break
		}
		b.WriteString(line)
	}
	if b.Len() == 0 {
		return "(empty)"
	}
	return b.String()
}

func saveHistory(workdir, sessionID string, history []core.Message, requested string) (string, error) {
	path := strings.TrimSpace(requested)
	if path == "" {
		path = "agent-session-" + safeFilePart(sessionID) + "-" + time.Now().Format("20060102-150405") + ".json"
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

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func oneLine(s string, limit int) string {
	s = strings.Join(strings.Fields(s), " ")
	if limit > 0 && len(s) > limit {
		return s[:limit] + "..."
	}
	return s
}

func safeFilePart(s string) string {
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
