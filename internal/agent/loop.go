package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/model"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type Engine struct {
	Client        model.Client
	Registry      *tools.Registry
	SessionStore  SessionStore
	SearchStore   tools.SessionSearchStore
	MemoryStore   tools.MemoryStore
	TodoStore     *tools.TodoStore
	ApprovalStore *tools.ApprovalStore
	Workdir       string
	SystemPrompt  string
	MaxIterations int
	MaxContextChars int
	CompressionTailMessages int
	EventSink     func(core.AgentEvent)

	// Optional gateway context (set by gateway runner; empty for CLI/HTTP).
	GatewayPlatform  string
	GatewayChatID    string
	GatewayChatType  string
	GatewayUserID    string
	GatewayUserName  string
	GatewayMessageID string
	GatewayThreadID  string
}

type SessionStore interface {
	AppendMessage(sessionID string, msg core.Message) error
	LoadMessages(sessionID string, limit int) ([]core.Message, error)
}

func (e *Engine) Run(ctx context.Context, sessionID, userInput, systemPrompt string, existing []core.Message) (*core.RunResult, error) {
	if e.MaxIterations <= 0 {
		e.MaxIterations = 30
	}
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = e.SystemPrompt
	}
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = DefaultSystemPrompt()
	}
		systemPrompt = buildRuntimeSystemPrompt(systemPrompt, e.Workdir, e.MemoryStore, e.Registry)

	messages := core.CloneMessages(existing)
	messages = withSystemPrompt(messages, systemPrompt)
	messages = append(messages, core.Message{Role: "user", Content: userInput})
	_ = e.persist(sessionID, core.Message{Role: "user", Content: userInput})
	e.emit(core.AgentEvent{Type: "user_message", SessionID: sessionID, Content: userInput})

	for turn := 0; turn < e.MaxIterations; turn++ {
		e.emit(core.AgentEvent{Type: "turn_started", SessionID: sessionID, Turn: turn + 1})
		var compressed map[string]any
		messages, compressed = compressMessages(messages, e.MaxContextChars, e.CompressionTailMessages)
		if compressed != nil {
			e.emit(core.AgentEvent{Type: "context_compacted", SessionID: sessionID, Turn: turn + 1, Data: compressed})
		}
		assistant, err := e.callWithRetry(ctx, messages, turn+1, sessionID)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				e.emit(core.AgentEvent{Type: "cancelled", SessionID: sessionID, Turn: turn + 1, Content: err.Error(), Data: terminalEventData("cancelled", err, turn+1)})
				return nil, err
			}
			e.emit(core.AgentEvent{Type: "error", SessionID: sessionID, Turn: turn + 1, Content: err.Error(), Data: terminalEventData("error", err, turn+1)})
			return nil, err
		}
		messages = append(messages, assistant)
		_ = e.persist(sessionID, assistant)
		e.emit(core.AgentEvent{Type: "assistant_message", SessionID: sessionID, Turn: turn + 1, Content: assistant.Content, Data: assistantEventData(assistant)})

		if len(assistant.ToolCalls) == 0 {
			e.emit(core.AgentEvent{Type: "completed", SessionID: sessionID, Turn: turn + 1, Content: assistant.Content, Data: completedEventData(assistant)})
			return &core.RunResult{SessionID: sessionID, FinalResponse: assistant.Content, Messages: messages, TurnsUsed: turn + 1, FinishedNaturally: true}, nil
		}

		for _, tc := range assistant.ToolCalls {
			args := tools.ParseJSONArgs(tc.Function.Arguments)
			e.emit(core.AgentEvent{Type: "tool_started", SessionID: sessionID, Turn: turn + 1, ToolName: tc.Function.Name, Data: toolStartedEventData(tc.ID, tc.Function.Name, args)})
			result := e.Registry.Dispatch(ctx, tc.Function.Name, args, tools.ToolContext{
				SessionID:      sessionID,
				SessionStore:   e.SearchStore,
				MemoryStore:    e.MemoryStore,
				TodoStore:      e.TodoStore,
				ApprovalStore:  e.ApprovalStore,
				DelegateRunner: e,
				Workdir:        e.Workdir,
				GatewayPlatform:  e.GatewayPlatform,
				GatewayChatID:    e.GatewayChatID,
				GatewayChatType:  e.GatewayChatType,
				GatewayUserID:    e.GatewayUserID,
				GatewayUserName:  e.GatewayUserName,
				GatewayMessageID: e.GatewayMessageID,
				GatewayThreadID:  e.GatewayThreadID,
				ToolEventSink: func(eventType string, data map[string]any) {
					e.emit(core.AgentEvent{
						Type:      "mcp_stream_event",
						SessionID: sessionID,
						Turn:      turn + 1,
						ToolName:  tc.Function.Name,
						Data: map[string]any{
							"tool_name":  tc.Function.Name,
							"event_type": eventType,
							"event_data": data,
						},
					})
				},
			})
			toolMsg := core.Message{Role: "tool", ToolCallID: tc.ID, Name: tc.Function.Name, Content: result}
			messages = append(messages, toolMsg)
			_ = e.persist(sessionID, toolMsg)
			e.emit(core.AgentEvent{Type: "tool_finished", SessionID: sessionID, Turn: turn + 1, ToolName: tc.Function.Name, Content: result, Data: toolFinishedEventData(tc.ID, tc.Function.Name, args, result)})
		}
	}
	e.emit(core.AgentEvent{Type: "max_iterations_reached", SessionID: sessionID, Turn: e.MaxIterations, Data: maxIterationsEventData(e.MaxIterations)})
	return &core.RunResult{SessionID: sessionID, FinalResponse: "", Messages: messages, TurnsUsed: e.MaxIterations, FinishedNaturally: false}, nil
}

func (e *Engine) RunSubtask(ctx context.Context, parentSessionID, goal, taskContext string, maxIterations int) (map[string]any, error) {
	childSessionID := parentSessionID + ":sub:" + uuid.NewString()
	child := *e
	child.TodoStore = tools.NewTodoStore()
	if maxIterations > 0 {
		child.MaxIterations = maxIterations
	}
	userInput := strings.TrimSpace(goal)
	if strings.TrimSpace(taskContext) != "" {
		userInput = userInput + "\n\nContext:\n" + strings.TrimSpace(taskContext)
	}
	e.emit(core.AgentEvent{Type: "delegate_started", SessionID: childSessionID, Content: goal, Data: map[string]any{"parent_session_id": parentSessionID, "goal": goal, "status": "running"}})
	res, err := child.Run(ctx, childSessionID, userInput, child.SystemPrompt, nil)
	if err != nil {
		e.emit(core.AgentEvent{Type: "delegate_failed", SessionID: childSessionID, Content: err.Error(), Data: map[string]any{"parent_session_id": parentSessionID, "goal": goal, "status": delegateStatusFromError(err), "success": false, "error": err.Error()}})
		return nil, err
	}
	result := map[string]any{"session_id": childSessionID, "parent_session_id": parentSessionID, "goal": goal, "final_response": res.FinalResponse, "turns_used": res.TurnsUsed, "finished_naturally": res.FinishedNaturally, "message_count": len(res.Messages)}
	e.emit(core.AgentEvent{Type: "delegate_finished", SessionID: childSessionID, Content: res.FinalResponse, Data: map[string]any{"parent_session_id": parentSessionID, "goal": goal, "status": "completed", "success": true, "result": result}})
	return result, nil
}

func (e *Engine) callWithRetry(ctx context.Context, messages []core.Message, turn int, sessionID string) (core.Message, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		msg, err := model.CompleteWithEvents(ctx, e.Client, messages, e.Registry.Schemas(), func(evt model.StreamEvent) {
			e.emit(core.AgentEvent{
				Type:      "model_stream_event",
				SessionID: sessionID,
				Turn:      turn,
				Data: map[string]any{
					"provider":   evt.Provider,
					"event_type": evt.Type,
					"event_data": evt.Data,
				},
			})
		})
		if err == nil {
			return msg, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return core.Message{}, ctx.Err()
		case <-time.After(time.Duration(i+1) * time.Second):
		}
	}
	return core.Message{}, fmt.Errorf("model call failed after retries: %w", lastErr)
}

func (e *Engine) persist(sessionID string, msg core.Message) error {
	if e.SessionStore == nil {
		return nil
	}
	return e.SessionStore.AppendMessage(sessionID, msg)
}

func (e *Engine) emit(event core.AgentEvent) {
	if e.EventSink != nil {
		e.EventSink(event)
	}
}

func delegateStatusFromError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	default:
		return "failed"
	}
}

func toolStartedEventData(toolCallID, toolName string, args map[string]any) map[string]any {
	return map[string]any{
		"status":       "running",
		"tool_call_id": toolCallID,
		"tool_name":    toolName,
		"arguments":    args,
	}
}

func toolFinishedEventData(toolCallID, toolName string, args map[string]any, raw string) map[string]any {
	parsed := tools.ParseJSONArgs(raw)
	if len(parsed) == 0 {
		return map[string]any{
			"status":       "completed",
			"success":      true,
			"tool_call_id": toolCallID,
			"tool_name":    toolName,
			"arguments":    args,
			"result":       raw,
		}
	}
	success := true
	status := "completed"
	if errText, ok := parsed["error"].(string); ok && strings.TrimSpace(errText) != "" {
		success = false
		status = "failed"
	}
	return map[string]any{
		"status":       status,
		"success":      success,
		"tool_call_id": toolCallID,
		"tool_name":    toolName,
		"arguments":    args,
		"result":       parsed,
	}
}

func assistantEventData(msg core.Message) map[string]any {
	return map[string]any{
		"status":          "completed",
		"message_role":    "assistant",
		"content_length":  len(msg.Content),
		"tool_call_count": len(msg.ToolCalls),
		"has_tool_calls":  len(msg.ToolCalls) > 0,
	}
}

func completedEventData(msg core.Message) map[string]any {
	data := assistantEventData(msg)
	data["finished_naturally"] = true
	return data
}

func terminalEventData(status string, err error, turn int) map[string]any {
	data := map[string]any{
		"status": status,
		"turn":   turn,
	}
	if err != nil {
		data["error"] = err.Error()
	}
	return data
}

func maxIterationsEventData(limit int) map[string]any {
	return map[string]any{
		"status":         "max_iterations_reached",
		"max_iterations": limit,
		"finished":       false,
	}
}
