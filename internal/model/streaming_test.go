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

func TestNormalizeStreamEventMessageDoneFinishReasonCanonical(t *testing.T) {
	done := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "message_done",
		Data: map[string]any{
			"id":            "msg-2",
			"finish_reason": "end_turn",
		},
	})
	if done.Data["finish_reason"] != "stop" {
		t.Fatalf("expected canonical finish_reason=stop, got %+v", done)
	}

	done = normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "message_done",
		Data: map[string]any{
			"id":            "msg-3",
			"finish_reason": "tool_use",
		},
	})
	if done.Data["finish_reason"] != "tool_calls" {
		t.Fatalf("expected canonical finish_reason=tool_calls, got %+v", done)
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

func TestNormalizeStreamEventToolCallIDFromToolUseID(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "tool_call_start",
		Data: map[string]any{
			"tool_use_id": "toolu_123",
			"name":        "read_file",
		},
	})
	if evt.Data["tool_call_id"] != "toolu_123" {
		t.Fatalf("expected tool_call_id from tool_use_id, got %+v", evt)
	}
}

func TestNormalizeStreamEventMessageIDAliases(t *testing.T) {
	start := normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "message_start",
		Data:     map[string]any{"response_id": "resp_1"},
	})
	if start.Data["message_id"] != "resp_1" {
		t.Fatalf("expected message_id from response_id, got %+v", start)
	}

	done := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "message_done",
		Data: map[string]any{
			"message": map[string]any{"id": "msg_nested_1"},
		},
	})
	if done.Data["message_id"] != "msg_nested_1" {
		t.Fatalf("expected message_id from nested message.id, got %+v", done)
	}
}

func TestNormalizeStreamEventToolCallIDAliasesFromItems(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "tool_args_start",
		Data: map[string]any{
			"output_item_id": "item_101",
			"name":           "read_file",
		},
	})
	if evt.Data["tool_call_id"] != "item_101" {
		t.Fatalf("expected tool_call_id from output_item_id, got %+v", evt)
	}
}

func TestNormalizeStreamEventMessageDoneTerminationMetadata(t *testing.T) {
	done := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "message_done",
		Data: map[string]any{
			"id":   "msg-7",
			"stop": "</END>",
		},
	})
	if done.Data["stop_sequence"] != "</END>" {
		t.Fatalf("expected stop_sequence from stop alias, got %+v", done)
	}

	done = normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "message_done",
		Data: map[string]any{
			"id": "msg-8",
			"incomplete_details": map[string]any{
				"reason": "max_output_tokens",
			},
		},
	})
	if done.Data["incomplete_reason"] != "max_output_tokens" {
		t.Fatalf("expected incomplete_reason from incomplete_details.reason, got %+v", done)
	}
}
