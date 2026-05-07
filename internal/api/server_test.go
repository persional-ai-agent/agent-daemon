package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type fakeModelClient struct {
	response core.Message
}

func (f fakeModelClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	return f.response, nil
}

type scriptedModelClient struct {
	mu        sync.Mutex
	responses []core.Message
}

func (c *scriptedModelClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.responses) == 0 {
		return core.Message{}, nil
	}
	msg := c.responses[0]
	c.responses = c.responses[1:]
	return msg, nil
}

type apiTestTool struct {
	name string
	call func(context.Context, map[string]any, tools.ToolContext) (map[string]any, error)
}

func (t apiTestTool) Name() string { return t.name }

func (t apiTestTool) Schema() core.ToolSchema {
	return core.ToolSchema{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:        t.name,
			Description: "api test tool",
			Parameters:  map[string]any{"type": "object"},
		},
	}
}

func (t apiTestTool) Call(ctx context.Context, args map[string]any, tc tools.ToolContext) (map[string]any, error) {
	return t.call(ctx, args, tc)
}

type blockingModelClient struct {
	started chan struct{}
	once    sync.Once
}

func (b *blockingModelClient) ChatCompletion(ctx context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	return core.Message{}, ctx.Err()
}

type stubSessionStore struct{}

func (s *stubSessionStore) AppendMessage(string, core.Message) error {
	return nil
}

func (s *stubSessionStore) LoadMessages(string, int) ([]core.Message, error) {
	return nil, nil
}

func TestHandleChatStream(t *testing.T) {
	srv := &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "hello from stream"}},
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", got)
	}

	body := rec.Body.String()
	parts := []string{
		"event: session",
		"event: user_message",
		"event: turn_started",
		"event: assistant_message",
		"event: completed",
		"event: result",
		`"final_response":"hello from stream"`,
	}
	last := -1
	for _, part := range parts {
		idx := strings.Index(body, part)
		if idx < 0 {
			t.Fatalf("expected response to contain %q, body=%s", part, body)
		}
		if idx < last {
			t.Fatalf("expected %q to appear after previous event, body=%s", part, body)
		}
		last = idx
	}
	if !strings.Contains(body, `"session_id":"`) {
		t.Fatalf("expected response to contain session id, body=%s", body)
	}
}

func TestHandleChatIncludesSummary(t *testing.T) {
	srv := &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "hello summary"}},
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewBufferString(`{"session_id":"s-summary","message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["final_response"] != "hello summary" {
		t.Fatalf("expected final response to remain top-level, got %+v", body)
	}
	summary, ok := body["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got %+v", body)
	}
	if summary["status"] != "completed" || summary["tool_call_count"] != float64(0) || summary["assistant_message_count"] != float64(1) {
		t.Fatalf("unexpected summary payload: %+v", summary)
	}
}

func TestHandleChatStreamCancel(t *testing.T) {
	client := &blockingModelClient{started: make(chan struct{})}
	srv := &Server{
		Engine: &agent.Engine{
			Client:       client,
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-cancel","message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")

	done := make(chan struct{})
	go func() {
		srv.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-client.started:
	case <-time.After(2 * time.Second):
		t.Fatal("streaming request did not start model call in time")
	}

	cancelReq := httptest.NewRequest(http.MethodPost, "/v1/chat/cancel", bytes.NewBufferString(`{"session_id":"s-cancel"}`))
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(cancelRec, cancelReq)

	if cancelRec.Code != http.StatusOK {
		t.Fatalf("unexpected cancel status: %d, body=%s", cancelRec.Code, cancelRec.Body.String())
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("streaming request did not finish after cancellation")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: cancelled") {
		t.Fatalf("expected cancelled event, body=%s", body)
	}
	if strings.Contains(body, "event: result") {
		t.Fatalf("did not expect result event after cancellation, body=%s", body)
	}
	if !strings.Contains(cancelRec.Body.String(), `"cancelled":true`) {
		t.Fatalf("expected cancel response payload, body=%s", cancelRec.Body.String())
	}
}

func TestHandleChatStreamDelegateEvents(t *testing.T) {
	args := `{"goal":"child-task","context":"child-context","max_iterations":2}`
	client := &scriptedModelClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "delegate_task",
						Arguments: args,
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "child completed",
			},
			{
				Role:    "assistant",
				Content: "parent completed",
			},
		},
	}
	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, nil)
	srv := &Server{
		Engine: &agent.Engine{
			Client:       client,
			Registry:     registry,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-delegate","message":"delegate now"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	parts := []string{
		"event: delegate_started",
		`"goal":"child-task"`,
		`"status":"running"`,
		"event: delegate_finished",
		`"status":"completed"`,
		`"success":true`,
		`"final_response":"child completed"`,
		`"final_response":"parent completed"`,
	}
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected stream body to contain %q, body=%s", part, body)
		}
	}
}

func TestHandleChatIncludesToolSummary(t *testing.T) {
	client := &scriptedModelClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "echo_tool",
						Arguments: `{"value":"ping"}`,
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "tool finished",
			},
		},
	}
	registry := tools.NewRegistry()
	registry.Register(apiTestTool{
		name: "echo_tool",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"echo": args["value"]}, nil
		},
	})
	srv := &Server{
		Engine: &agent.Engine{
			Client:       client,
			Registry:     registry,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewBufferString(`{"session_id":"s-tool-summary","message":"run tool"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["final_response"] != "tool finished" {
		t.Fatalf("expected top-level final response, got %+v", body)
	}
	summary, ok := body["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got %+v", body)
	}
	if summary["tool_call_count"] != float64(1) || summary["delegate_count"] != float64(0) {
		t.Fatalf("unexpected tool summary counts: %+v", summary)
	}
	toolNames, ok := summary["tool_names"].([]any)
	if !ok || len(toolNames) != 1 || toolNames[0] != "echo_tool" {
		t.Fatalf("unexpected tool_names summary: %+v", summary)
	}
}

func TestHandleChatStreamToolFinishedStructuredData(t *testing.T) {
	client := &scriptedModelClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "echo_tool",
						Arguments: `{"value":"ping"}`,
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "tool done",
			},
		},
	}
	registry := tools.NewRegistry()
	registry.Register(apiTestTool{
		name: "echo_tool",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"echo": args["value"]}, nil
		},
	})
	srv := &Server{
		Engine: &agent.Engine{
			Client:       client,
			Registry:     registry,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-tool","message":"run tool"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	parts := []string{
		"event: tool_started",
		`"tool_call_id":"call-1"`,
		`"tool_name":"echo_tool"`,
		`"status":"running"`,
		`"arguments":{"value":"ping"}`,
		"event: tool_finished",
		`"tool_call_id":"call-1"`,
		`"status":"completed"`,
		`"success":true`,
		`"echo":"ping"`,
	}
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected stream body to contain %q, body=%s", part, body)
		}
	}
}
