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
	if evt.Data["usage_consistency_status"] != "derived" {
		t.Fatalf("expected usage_consistency_status=derived, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsagePromptCacheTokens(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "usage",
		Data: map[string]any{
			"input_tokens":               20,
			"output_tokens":              4,
			"cache_creation_input_tokens": 12,
			"cache_read_input_tokens":     7,
		},
	})
	if evt.Data["prompt_cache_write_tokens"] != 12 || evt.Data["prompt_cache_read_tokens"] != 7 {
		t.Fatalf("expected normalized prompt cache tokens, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsagePromptCacheTokensFromNestedDetails(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 3,
			"prompt_tokens_details": map[string]any{
				"cached_tokens": 5,
			},
		},
	})
	if evt.Data["prompt_cache_read_tokens"] != 5 {
		t.Fatalf("expected prompt_cache_read_tokens from prompt_tokens_details.cached_tokens, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageReasoningTokens(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 8,
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": 6,
			},
		},
	})
	if evt.Data["reasoning_tokens"] != 6 {
		t.Fatalf("expected reasoning_tokens from completion_tokens_details.reasoning_tokens, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageReasoningTokensAliases(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "usage",
		Data: map[string]any{
			"input_tokens":  5,
			"output_tokens": 4,
			"output_tokens_details": map[string]any{
				"reasoning_tokens": 3,
			},
		},
	})
	if evt.Data["reasoning_tokens"] != 3 {
		t.Fatalf("expected reasoning_tokens from output_tokens_details.reasoning_tokens, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageTotalTokensDerivedWhenMissing(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"prompt_tokens":     11,
			"completion_tokens": 9,
		},
	})
	if evt.Data["total_tokens"] != 20 {
		t.Fatalf("expected derived total_tokens=20, got %+v", evt)
	}
	if evt.Data["usage_consistency_status"] != "derived" {
		t.Fatalf("expected usage_consistency_status=derived, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageTotalTokensAdjustedWhenTooSmall(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 7,
			"total_tokens":      12,
		},
	})
	if evt.Data["total_tokens"] != 17 {
		t.Fatalf("expected adjusted total_tokens=17, got %+v", evt)
	}
	if evt.Data["total_tokens_adjusted"] != true {
		t.Fatalf("expected total_tokens_adjusted=true, got %+v", evt)
	}
	if evt.Data["usage_consistency_status"] != "adjusted" {
		t.Fatalf("expected usage_consistency_status=adjusted, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageTotalTokensKeepsConsistentSource(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"prompt_tokens":     6,
			"completion_tokens": 5,
			"total_tokens":      11,
		},
	})
	if evt.Data["usage_consistency_status"] != "ok" {
		t.Fatalf("expected usage_consistency_status=ok, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageStringNumbers(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"prompt_tokens":     "6",
			"completion_tokens": "4",
		},
	})
	if evt.Data["total_tokens"] != 10 {
		t.Fatalf("expected total_tokens=10 from string numbers, got %+v", evt)
	}
	if evt.Data["usage_consistency_status"] != "derived" {
		t.Fatalf("expected usage_consistency_status=derived, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageInvalidNumbers(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"prompt_tokens":     "abc",
			"completion_tokens": "4x",
		},
	})
	if _, ok := evt.Data["total_tokens"]; ok {
		t.Fatalf("did not expect total_tokens for invalid numeric fields, got %+v", evt)
	}
	if evt.Data["usage_consistency_status"] != "invalid" {
		t.Fatalf("expected usage_consistency_status=invalid, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageConsistencyStatusOpenAIOk(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"prompt_tokens":     8,
			"completion_tokens": 2,
			"total_tokens":      10,
		},
	})
	if evt.Data["usage_consistency_status"] != "ok" {
		t.Fatalf("expected usage_consistency_status=ok for openai, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageConsistencyStatusAnthropicDerived(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "usage",
		Data: map[string]any{
			"input_tokens":  14,
			"output_tokens": 3,
		},
	})
	if evt.Data["usage_consistency_status"] != "derived" {
		t.Fatalf("expected usage_consistency_status=derived for anthropic, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageConsistencyStatusCodexInvalid(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "usage",
		Data: map[string]any{
			"tokens": "NaN",
		},
	})
	if evt.Data["usage_consistency_status"] != "invalid" {
		t.Fatalf("expected usage_consistency_status=invalid for codex, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageConsistencyStatusOpenAISourceOnly(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "usage",
		Data: map[string]any{
			"total_tokens": 21,
		},
	})
	if evt.Data["usage_consistency_status"] != "source_only" {
		t.Fatalf("expected usage_consistency_status=source_only for openai, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageConsistencyStatusAnthropicSourceOnly(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Anthropic",
		Type:     "usage",
		Data: map[string]any{
			"total_tokens": 17,
		},
	})
	if evt.Data["usage_consistency_status"] != "source_only" {
		t.Fatalf("expected usage_consistency_status=source_only for anthropic, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageConsistencyStatusCodexSourceOnly(t *testing.T) {
	evt := normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "usage",
		Data: map[string]any{
			"total_tokens": 9,
		},
	})
	if evt.Data["usage_consistency_status"] != "source_only" {
		t.Fatalf("expected usage_consistency_status=source_only for codex, got %+v", evt)
	}
}

func TestNormalizeStreamEventUsageConsistencyStatusTableDriven(t *testing.T) {
	cases := []struct {
		name           string
		provider       string
		data           map[string]any
		wantStatus     string
		wantAdjusted   bool
		wantTotalToken int
	}{
		{
			name:           "openai-ok",
			provider:       "openai",
			data:           map[string]any{"prompt_tokens": 4, "completion_tokens": 3, "total_tokens": 7},
			wantStatus:     "ok",
			wantAdjusted:   false,
			wantTotalToken: 7,
		},
		{
			name:           "anthropic-derived",
			provider:       "anthropic",
			data:           map[string]any{"input_tokens": 5, "output_tokens": 2},
			wantStatus:     "derived",
			wantAdjusted:   false,
			wantTotalToken: 7,
		},
		{
			name:           "codex-source-only",
			provider:       "codex",
			data:           map[string]any{"total_tokens": 9},
			wantStatus:     "source_only",
			wantAdjusted:   false,
			wantTotalToken: 9,
		},
		{
			name:           "openai-adjusted",
			provider:       "openai",
			data:           map[string]any{"prompt_tokens": 8, "completion_tokens": 4, "total_tokens": 5},
			wantStatus:     "adjusted",
			wantAdjusted:   true,
			wantTotalToken: 12,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evt := normalizeStreamEvent(StreamEvent{
				Provider: tc.provider,
				Type:     "usage",
				Data:     tc.data,
			})
			gotStatus, _ := evt.Data["usage_consistency_status"].(string)
			if gotStatus != tc.wantStatus {
				t.Fatalf("unexpected usage_consistency_status: got=%q want=%q evt=%+v", gotStatus, tc.wantStatus, evt)
			}
			if tc.wantAdjusted {
				if b, _ := evt.Data["total_tokens_adjusted"].(bool); !b {
					t.Fatalf("expected total_tokens_adjusted=true, got evt=%+v", evt)
				}
			}
			if tc.wantTotalToken > 0 {
				if n, ok := evt.Data["total_tokens"].(int); !ok || n != tc.wantTotalToken {
					t.Fatalf("unexpected total_tokens: got=%v want=%d evt=%+v", evt.Data["total_tokens"], tc.wantTotalToken, evt)
				}
			}
		})
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
	if done.Data["incomplete_reason"] != "length" {
		t.Fatalf("expected canonical incomplete_reason=length from incomplete_details.reason, got %+v", done)
	}
}

func TestNormalizeStreamEventMessageDoneLengthCompletesIncompleteReason(t *testing.T) {
	done := normalizeStreamEvent(StreamEvent{
		Provider: "Codex",
		Type:     "message_done",
		Data: map[string]any{
			"id":            "msg-9",
			"finish_reason": "max_output_tokens",
		},
	})
	if done.Data["finish_reason"] != "length" {
		t.Fatalf("expected canonical finish_reason=length, got %+v", done)
	}
	if done.Data["incomplete_reason"] != "length" {
		t.Fatalf("expected inferred incomplete_reason=length, got %+v", done)
	}
}

func TestNormalizeStreamEventMessageDoneCanonicalIncompleteReason(t *testing.T) {
	done := normalizeStreamEvent(StreamEvent{
		Provider: "OpenAI",
		Type:     "message_done",
		Data: map[string]any{
			"id":                "msg-10",
			"finish_reason":     "length",
			"incomplete_reason": "max_tokens",
		},
	})
	if done.Data["incomplete_reason"] != "length" {
		t.Fatalf("expected canonical incomplete_reason=length, got %+v", done)
	}
}
