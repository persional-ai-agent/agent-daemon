package model

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestAnthropicClientParsesToolUse(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "k" {
			t.Fatalf("missing api key header, got %q", got)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"doing tool call"},{"type":"tool_use","id":"toolu_1","name":"read_file","input":{"path":"README.md"}}]}`))
	}))
	defer srv.Close()

	client := NewAnthropicClient(srv.URL, "k", "claude-test")
	msg, err := client.ChatCompletion(context.Background(), []core.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "read file"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Role != "assistant" || !strings.Contains(msg.Content, "doing tool call") {
		t.Fatalf("unexpected assistant message: %+v", msg)
	}
	if len(msg.ToolCalls) != 1 || msg.ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("unexpected tool calls: %+v", msg.ToolCalls)
	}
	if !strings.Contains(msg.ToolCalls[0].Function.Arguments, `"path":"README.md"`) {
		t.Fatalf("unexpected tool arguments: %+v", msg.ToolCalls[0].Function.Arguments)
	}
}

func TestAnthropicClientBuildsToolResultTurn(t *testing.T) {
	var reqBody map[string]any
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	}))
	defer srv.Close()

	client := NewAnthropicClient(srv.URL, "k", "claude-test")
	_, err := client.ChatCompletion(context.Background(), []core.Message{
		{Role: "system", Content: "system rule"},
		{Role: "assistant", ToolCalls: []core.ToolCall{{
			ID:   "toolu_1",
			Type: "function",
			Function: core.ToolFunction{
				Name:      "read_file",
				Arguments: `{"path":"README.md"}`,
			},
		}}},
		{Role: "tool", ToolCallID: "toolu_1", Name: "read_file", Content: `{"content":"x"}`},
		{Role: "user", Content: "continue"},
	}, []core.ToolSchema{{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:       "read_file",
			Parameters: map[string]any{"type": "object"},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if reqBody["system"] != "system rule" {
		t.Fatalf("expected system text in request, got %+v", reqBody["system"])
	}
	msgs, ok := reqBody["messages"].([]any)
	if !ok || len(msgs) < 2 {
		t.Fatalf("expected anthropic messages, got %+v", reqBody["messages"])
	}
	foundToolResult := false
	for _, raw := range msgs {
		m, _ := raw.(map[string]any)
		content, _ := m["content"].([]any)
		for _, c := range content {
			block, _ := c.(map[string]any)
			if block["type"] == "tool_result" {
				foundToolResult = true
			}
		}
	}
	if !foundToolResult {
		t.Fatalf("expected tool_result block, request=%+v", reqBody)
	}
}
