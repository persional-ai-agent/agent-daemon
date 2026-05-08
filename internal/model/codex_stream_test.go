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

func TestCodexClientStreamingText(t *testing.T) {
	var seenStream bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		stream, _ := req["stream"].(bool)
		seenStream = stream
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\"hello \"}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_1\",\"delta\":\"codex\"}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client := NewCodexClient(srv.URL, "", "gpt-5-codex")
	client.UseStreaming = true
	msg, err := client.ChatCompletion(context.Background(), []core.Message{
		{Role: "user", Content: "say hi"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !seenStream {
		t.Fatal("expected stream=true in codex request")
	}
	if msg.Content != "hello codex" {
		t.Fatalf("unexpected streamed content: %+v", msg)
	}
}

func TestCodexClientStreamingFunctionCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.output_item.added\",\"item\":{\"id\":\"call_1\",\"type\":\"function_call\",\"name\":\"read_file\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"call_1\",\"delta\":\"{\\\"path\\\":\\\"REA\"}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.function_call_arguments.delta\",\"item_id\":\"call_1\",\"delta\":\"DME.md\\\"}\"}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client := NewCodexClient(srv.URL, "", "gpt-5-codex")
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

func TestCodexClientStreamingUsageEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"response\":{\"usage\":{\"input_tokens\":8,\"output_tokens\":4,\"total_tokens\":12},\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":\"ok\"}]}}\n\n")
	}))
	defer srv.Close()

	client := NewCodexClient(srv.URL, "", "gpt-5-codex")
	client.UseStreaming = true
	seenUsage := false
	_, err := client.ChatCompletionWithEvents(context.Background(), []core.Message{
		{Role: "user", Content: "hello"},
	}, nil, func(evt StreamEvent) {
		if evt.Type == "usage" {
			seenUsage = true
			if evt.Data["input_tokens"] != float64(8) {
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

func TestCodexClientStreamingCompletedEnvelopeCarriesResponseID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"response\":{\"id\":\"resp_777\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"},\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":\"ok\"}]}}\n\n")
	}))
	defer srv.Close()

	client := NewCodexClient(srv.URL, "", "gpt-5-codex")
	client.UseStreaming = true
	seenDoneWithResponseID := false
	seenIncompleteReason := false
	_, err := client.ChatCompletionWithEvents(context.Background(), []core.Message{
		{Role: "user", Content: "hello"},
	}, nil, func(evt StreamEvent) {
		if evt.Type == "message_done" && evt.Data["response_id"] == "resp_777" {
			seenDoneWithResponseID = true
		}
		if evt.Type == "message_done" && evt.Data["incomplete_reason"] == "max_output_tokens" {
			seenIncompleteReason = true
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !seenDoneWithResponseID {
		t.Fatal("expected message_done response_id from completed envelope")
	}
	if !seenIncompleteReason {
		t.Fatal("expected message_done incomplete_reason from completed envelope")
	}
}

func TestCodexStreamingUsageStatusInvalidE2E(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"response\":{\"usage\":{\"tokens\":\"NaN\"},\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":\"ok\"}]}}\n\n")
	}))
	defer srv.Close()

	client := NewCodexClient(srv.URL, "", "gpt-5-codex")
	client.UseStreaming = true
	gotStatus := ""
	_, err := CompleteWithEvents(context.Background(), client, []core.Message{
		{Role: "user", Content: "hello"},
	}, nil, func(evt StreamEvent) {
		if evt.Type == "usage" {
			if s, _ := evt.Data["usage_consistency_status"].(string); s != "" {
				gotStatus = s
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotStatus != "invalid" {
		t.Fatalf("expected normalized usage_consistency_status=invalid, got %q", gotStatus)
	}
}

func TestCodexStreamingUsageStatusAdjustedE2E(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"response\":{\"usage\":{\"input_tokens\":6,\"output_tokens\":4,\"total_tokens\":7},\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":\"ok\"}]}}\n\n")
	}))
	defer srv.Close()

	client := NewCodexClient(srv.URL, "", "gpt-5-codex")
	client.UseStreaming = true
	gotStatus := ""
	gotAdjusted := false
	gotTotal := 0
	_, err := CompleteWithEvents(context.Background(), client, []core.Message{
		{Role: "user", Content: "hello"},
	}, nil, func(evt StreamEvent) {
		if evt.Type == "usage" {
			if s, _ := evt.Data["usage_consistency_status"].(string); s != "" {
				gotStatus = s
			}
			if b, _ := evt.Data["total_tokens_adjusted"].(bool); b {
				gotAdjusted = true
			}
			if n, ok := evt.Data["total_tokens"].(int); ok {
				gotTotal = n
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotStatus != "adjusted" {
		t.Fatalf("expected normalized usage_consistency_status=adjusted, got %q", gotStatus)
	}
	if !gotAdjusted || gotTotal != 10 {
		t.Fatalf("expected total_tokens_adjusted=true and total_tokens=10, got adjusted=%v total=%d", gotAdjusted, gotTotal)
	}
}
