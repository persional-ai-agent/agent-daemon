package model

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestAnthropicClientStreamingText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"hello\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	client := NewAnthropicClient(srv.URL, "k", "claude-test")
	client.UseStreaming = true
	msg, err := client.ChatCompletion(context.Background(), []core.Message{
		{Role: "user", Content: "say hi"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "hello world" {
		t.Fatalf("unexpected streaming text: %+v", msg)
	}
}

func TestAnthropicClientStreamingToolUse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_1\",\"name\":\"read_file\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"path\\\":\\\"REA\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"DME.md\\\"}\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	client := NewAnthropicClient(srv.URL, "k", "claude-test")
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
		t.Fatalf("unexpected tool call args: %+v", tc.Function.Arguments)
	}
}

func TestAnthropicClientStreamingUsageEvent(t *testing.T) {
	client := NewAnthropicClient("", "k", "claude-test")
	client.UseStreaming = true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"message_delta\",\"usage\":{\"input_tokens\":11,\"output_tokens\":5}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()
	client.BaseURL = srv.URL

	seenUsage := false
	_, err := client.ChatCompletionWithEvents(context.Background(), []core.Message{
		{Role: "user", Content: "count"},
	}, nil, func(evt StreamEvent) {
		if evt.Type == "usage" {
			seenUsage = true
			if evt.Data["input_tokens"] != float64(11) {
				t.Fatalf("unexpected usage payload: %+v", evt)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !seenUsage {
		t.Fatal("expected usage stream event")
	}
}
