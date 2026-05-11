package model

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestCodexClientParsesFunctionCall(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"output":[{"type":"message","role":"assistant","content":"thinking"},{"type":"function_call","call_id":"call_1","name":"read_file","arguments":"{\"path\":\"README.md\"}"}]}`))
	}))
	defer srv.Close()

	client := NewCodexClient(srv.URL, "k", "gpt-5-codex")
	msg, err := client.ChatCompletion(context.Background(), []core.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "read file"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Role != "assistant" || !strings.Contains(msg.Content, "thinking") {
		t.Fatalf("unexpected assistant message: %+v", msg)
	}
	if len(msg.ToolCalls) != 1 || msg.ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("unexpected tool calls: %+v", msg.ToolCalls)
	}
}

func TestCodexClientBuildsFunctionCallOutputInput(t *testing.T) {
	var reqBody map[string]any
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		_, _ = w.Write([]byte(`{"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer srv.Close()

	client := NewCodexClient(srv.URL, "k", "gpt-5-codex")
	_, err := client.ChatCompletion(context.Background(), []core.Message{
		{Role: "system", Content: "system"},
		{Role: "assistant", ToolCalls: []core.ToolCall{{
			ID:   "call_1",
			Type: "function",
			Function: core.ToolFunction{
				Name:      "read_file",
				Arguments: `{"path":"README.md"}`,
			},
		}}},
		{Role: "tool", ToolCallID: "call_1", Name: "read_file", Content: `{"content":"x"}`},
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
	items, ok := reqBody["input"].([]any)
	if !ok || len(items) < 2 {
		t.Fatalf("expected input items in codex request, got %+v", reqBody["input"])
	}
	foundOutput := false
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		if strings.EqualFold(asString(m["type"]), "function_call_output") && asString(m["call_id"]) == "call_1" {
			foundOutput = true
		}
	}
	if !foundOutput {
		t.Fatalf("expected function_call_output input item, got %+v", items)
	}
}
