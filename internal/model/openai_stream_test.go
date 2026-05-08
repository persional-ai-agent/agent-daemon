package model

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestOpenAIChatCompletionStreamingText(t *testing.T) {
	var seenStream bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		stream, _ := req["stream"].(bool)
		seenStream = stream
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello \"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client := NewOpenAIClient(srv.URL, "", "gpt-test")
	client.UseStreaming = true
	msg, err := client.ChatCompletion(context.Background(), []core.Message{
		{Role: "user", Content: "say hi"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !seenStream {
		t.Fatal("expected stream=true in request")
	}
	if msg.Content != "hello world" {
		t.Fatalf("unexpected streamed content: %+v", msg)
	}
}

func TestOpenAIChatCompletionStreamingToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"REA\"}}]}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"DME.md\\\"}\"}}]}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client := NewOpenAIClient(srv.URL, "", "gpt-test")
	client.UseStreaming = true
	msg, err := client.ChatCompletion(context.Background(), []core.Message{
		{Role: "user", Content: "read readme"},
	}, []core.ToolSchema{
		{Type: "function", Function: core.ToolSchemaDetail{Name: "read_file", Parameters: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected one streamed tool call, got %+v", msg.ToolCalls)
	}
	tc := msg.ToolCalls[0]
	if tc.Function.Name != "read_file" {
		t.Fatalf("unexpected tool call name: %+v", tc)
	}
	if !strings.Contains(tc.Function.Arguments, "README.md") {
		t.Fatalf("unexpected tool call args: %+v", tc)
	}
}

func TestOpenAIChatCompletionStreamingUsageEvent(t *testing.T) {
	var includeUsage bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if opts, ok := req["stream_options"].(map[string]any); ok {
			includeUsage, _ = opts["include_usage"].(bool)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5},\"choices\":[]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client := NewOpenAIClient(srv.URL, "", "gpt-test")
	client.UseStreaming = true
	seenUsage := false
	_, err := client.ChatCompletionWithEvents(context.Background(), []core.Message{
		{Role: "user", Content: "say hi"},
	}, nil, func(evt StreamEvent) {
		if evt.Type == "usage" {
			seenUsage = true
			if evt.Data["prompt_tokens"] != float64(3) && evt.Data["prompt_tokens"] != 3 {
				t.Fatalf("unexpected usage payload: %+v", evt)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !includeUsage {
		t.Fatal("expected stream_options.include_usage=true in request")
	}
	if !seenUsage {
		t.Fatal("expected usage stream event")
	}
}
