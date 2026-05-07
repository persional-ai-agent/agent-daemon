package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	Workdir       string
	MaxIterations int
}

type SessionStore interface {
	AppendMessage(sessionID string, msg core.Message) error
	LoadMessages(sessionID string, limit int) ([]core.Message, error)
}

func (e *Engine) Run(ctx context.Context, sessionID, userInput, systemPrompt string, existing []core.Message) (*core.RunResult, error) {
	if e.MaxIterations <= 0 {
		e.MaxIterations = 30
	}
	messages := core.CloneMessages(existing)
	if len(messages) == 0 && strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, core.Message{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, core.Message{Role: "user", Content: userInput})
	_ = e.persist(sessionID, core.Message{Role: "user", Content: userInput})

	for turn := 0; turn < e.MaxIterations; turn++ {
		assistant, err := e.callWithRetry(ctx, messages)
		if err != nil {
			return nil, err
		}
		messages = append(messages, assistant)
		_ = e.persist(sessionID, assistant)

		if len(assistant.ToolCalls) == 0 {
			return &core.RunResult{SessionID: sessionID, FinalResponse: assistant.Content, Messages: messages, TurnsUsed: turn + 1, FinishedNaturally: true}, nil
		}

		for _, tc := range assistant.ToolCalls {
			args := tools.ParseJSONArgs(tc.Function.Arguments)
			result := e.Registry.Dispatch(ctx, tc.Function.Name, args, tools.ToolContext{SessionID: sessionID, SessionStore: e.SearchStore, MemoryStore: e.MemoryStore, TodoStore: e.TodoStore, Workdir: e.Workdir})
			toolMsg := core.Message{Role: "tool", ToolCallID: tc.ID, Name: tc.Function.Name, Content: result}
			messages = append(messages, toolMsg)
			_ = e.persist(sessionID, toolMsg)
		}
	}
	return &core.RunResult{SessionID: sessionID, FinalResponse: "", Messages: messages, TurnsUsed: e.MaxIterations, FinishedNaturally: false}, nil
}

func (e *Engine) callWithRetry(ctx context.Context, messages []core.Message) (core.Message, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		msg, err := e.Client.ChatCompletion(ctx, messages, e.Registry.Schemas())
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
