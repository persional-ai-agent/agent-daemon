package model

import "testing"

func TestNormalizeStreamEventTextDelta(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "text_delta",
		Data:     map[string]any{"delta": "hello"},
	})
	if evt.Provider != "openai" {
		t.Fatalf("expected lower-case provider, got %+v", evt)
	}
	if evt.Data["text"] != "hello" {
		t.Fatalf("expected normalized text field, got %+v", evt)
	}
}

func TestNormalizeStreamEventToolArgumentsDelta(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "tool_arguments_delta",
		Data: map[string]any{
			"name":         "read_file",
			"partial_json": "{\"path\":\"README.md\"}",
		},
	})
	if evt.Type != "tool_args_delta" {
		t.Fatalf("expected normalized type tool_args_delta, got %+v", evt)
	}
	if evt.Data["tool_name"] != "read_file" || evt.Data["arguments_delta"] != "{\"path\":\"README.md\"}" {
		t.Fatalf("expected normalized tool args event, got %+v", evt)
	}
}

func TestNormalizeStreamEventToolCallStart(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "tool_call_start",
		Data: map[string]any{
			"id":   "call-1",
			"name": "read_file",
		},
	})
	if evt.Data["tool_name"] != "read_file" || evt.Data["tool_call_id"] != "call-1" {
		t.Fatalf("expected normalized tool_call_start event, got %+v", evt)
	}
}

func TestNormalizeStreamEventToolCallDone(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "tool_call_done",
		Data: map[string]any{
			"call_id": "call-2",
			"name":    "write_file",
			"input":   "{\"path\":\"a.txt\"}",
		},
	})
	if evt.Data["tool_name"] != "write_file" || evt.Data["tool_call_id"] != "call-2" || evt.Data["arguments"] != "{\"path\":\"a.txt\"}" {
		t.Fatalf("expected normalized tool_call_done event, got %+v", evt)
	}
}

func TestNormalizeStreamEventMessageStartDone(t *testing.T) {
	start := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "message_start",
		Data:     map[string]any{"id": "msg-1"},
	})
	done := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "message_done",
		Data:     map[string]any{"id": "msg-1"},
	})
	if start.Data["message_id"] != "msg-1" || done.Data["message_id"] != "msg-1" {
		t.Fatalf("expected normalized message_start/message_done events, got start=%+v done=%+v", start, done)
	}
	if done.Data["finish_reason"] != "stop" {
		t.Fatalf("expected default finish_reason=stop, got %+v", done)
	}
}

func TestNormalizeStreamEventToolArgumentsStartDoneAliases(t *testing.T) {
	start := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "tool_arguments_start",
		Data: map[string]any{
			"id":   "call-9",
			"name": "read_file",
		},
	})
	done := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "tool_arguments_done",
		Data: map[string]any{
			"call_id": "call-9",
			"name":    "read_file",
			"input":   "{\"path\":\"x\"}",
		},
	})
	if start.Type != "tool_args_start" || done.Type != "tool_args_done" {
		t.Fatalf("expected normalized alias types, got start=%+v done=%+v", start, done)
	}
	if start.Data["tool_call_id"] != "call-9" || done.Data["tool_call_id"] != "call-9" || done.Data["arguments"] != "{\"path\":\"x\"}" {
		t.Fatalf("expected normalized alias payloads, got start=%+v done=%+v", start, done)
	}
}

func TestNormalizeStreamEventUsage(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "usage",
		Data: map[string]any{
			"input_tokens":  10,
			"output_tokens": 7,
		},
	})
	if evt.Data["prompt_tokens"] != 10 || evt.Data["completion_tokens"] != 7 || evt.Data["total_tokens"] != 17 {
		t.Fatalf("expected normalized usage payload, got %+v", evt)
	}
}
