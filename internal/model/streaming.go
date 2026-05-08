package model

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type StreamEvent struct {
	Provider string
	Type     string
	Data     map[string]any
}

type StreamEventSink func(StreamEvent)

type EventClient interface {
	ChatCompletionWithEvents(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error)
}

func CompleteWithEvents(ctx context.Context, client Client, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	if ec, ok := client.(EventClient); ok {
		wrapped := sink
		if sink != nil {
			wrapped = func(evt StreamEvent) {
				sink(normalizeStreamEvent(evt))
			}
		}
		return ec.ChatCompletionWithEvents(ctx, messages, tools, wrapped)
	}
	return client.ChatCompletion(ctx, messages, tools)
}

func normalizeStreamEvent(evt StreamEvent) StreamEvent {
	evt.Provider = strings.ToLower(strings.TrimSpace(evt.Provider))
	evt.Type = strings.ToLower(strings.TrimSpace(evt.Type))
	switch evt.Type {
	case "tool_arguments_start":
		evt.Type = "tool_args_start"
	case "tool_arguments_delta":
		evt.Type = "tool_args_delta"
	case "tool_arguments_done":
		evt.Type = "tool_args_done"
	}
	if evt.Data == nil {
		evt.Data = map[string]any{}
	}
	switch evt.Type {
	case "message_start":
		if _, ok := evt.Data["message_id"]; !ok {
			if s := asAnyString(evt.Data["id"]); s != "" {
				evt.Data["message_id"] = s
			} else if s := asAnyString(evt.Data["response_id"]); s != "" {
				evt.Data["message_id"] = s
			} else if msg := asAnyMap(evt.Data["message"]); msg != nil {
				if s := asAnyString(msg["id"]); s != "" {
					evt.Data["message_id"] = s
				}
			}
		}
	case "message_done":
		if _, ok := evt.Data["message_id"]; !ok {
			if s := asAnyString(evt.Data["id"]); s != "" {
				evt.Data["message_id"] = s
			} else if s := asAnyString(evt.Data["response_id"]); s != "" {
				evt.Data["message_id"] = s
			} else if msg := asAnyMap(evt.Data["message"]); msg != nil {
				if s := asAnyString(msg["id"]); s != "" {
					evt.Data["message_id"] = s
				}
			}
		}
		if _, ok := evt.Data["finish_reason"]; !ok {
			if s := asAnyString(evt.Data["reason"]); s != "" {
				evt.Data["finish_reason"] = s
			} else {
				evt.Data["finish_reason"] = "stop"
			}
		}
		evt.Data["finish_reason"] = canonicalFinishReason(asAnyString(evt.Data["finish_reason"]))
		if _, ok := evt.Data["stop_sequence"]; !ok {
			if s := asAnyString(evt.Data["stop"]); s != "" {
				evt.Data["stop_sequence"] = s
			}
		}
		if _, ok := evt.Data["incomplete_reason"]; !ok {
			if s := asAnyString(evt.Data["reason_detail"]); s != "" {
				evt.Data["incomplete_reason"] = s
			} else if details := asAnyMap(evt.Data["incomplete_details"]); details != nil {
				if s := asAnyString(details["reason"]); s != "" {
					evt.Data["incomplete_reason"] = s
				}
			}
		}
		if s := asAnyString(evt.Data["incomplete_reason"]); s != "" {
			evt.Data["incomplete_reason"] = canonicalIncompleteReason(s)
		}
		if evt.Data["finish_reason"] == "length" {
			if s := asAnyString(evt.Data["incomplete_reason"]); s == "" {
				evt.Data["incomplete_reason"] = "length"
			}
		}
	case "text_delta":
		if _, ok := evt.Data["text"]; !ok {
			if s := asAnyString(evt.Data["delta"]); s != "" {
				evt.Data["text"] = s
			} else if s := asAnyString(evt.Data["content"]); s != "" {
				evt.Data["text"] = s
			}
		}
	case "tool_args_delta":
		if _, ok := evt.Data["tool_name"]; !ok {
			if s := asAnyString(evt.Data["name"]); s != "" {
				evt.Data["tool_name"] = s
			} else if s := asAnyString(evt.Data["function_name"]); s != "" {
				evt.Data["tool_name"] = s
			}
		}
		if _, ok := evt.Data["arguments_delta"]; !ok {
			if s := asAnyString(evt.Data["delta"]); s != "" {
				evt.Data["arguments_delta"] = s
			} else if s := asAnyString(evt.Data["partial_json"]); s != "" {
				evt.Data["arguments_delta"] = s
			}
		}
	case "tool_args_start":
		if _, ok := evt.Data["tool_name"]; !ok {
			if s := asAnyString(evt.Data["name"]); s != "" {
				evt.Data["tool_name"] = s
			}
		}
		if _, ok := evt.Data["tool_call_id"]; !ok {
			if s := asAnyString(evt.Data["call_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["tool_use_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["item_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["output_item_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["id"]); s != "" {
				evt.Data["tool_call_id"] = s
			}
		}
	case "tool_args_done":
		if _, ok := evt.Data["tool_name"]; !ok {
			if s := asAnyString(evt.Data["name"]); s != "" {
				evt.Data["tool_name"] = s
			}
		}
		if _, ok := evt.Data["tool_call_id"]; !ok {
			if s := asAnyString(evt.Data["call_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["tool_use_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["item_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["output_item_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["id"]); s != "" {
				evt.Data["tool_call_id"] = s
			}
		}
		if _, ok := evt.Data["arguments"]; !ok {
			if s := asAnyString(evt.Data["input"]); s != "" {
				evt.Data["arguments"] = s
			}
		}
	case "tool_call_start":
		if _, ok := evt.Data["tool_name"]; !ok {
			if s := asAnyString(evt.Data["name"]); s != "" {
				evt.Data["tool_name"] = s
			}
		}
		if _, ok := evt.Data["tool_call_id"]; !ok {
			if s := asAnyString(evt.Data["call_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["tool_use_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["item_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["output_item_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["id"]); s != "" {
				evt.Data["tool_call_id"] = s
			}
		}
	case "tool_call_done":
		if _, ok := evt.Data["tool_name"]; !ok {
			if s := asAnyString(evt.Data["name"]); s != "" {
				evt.Data["tool_name"] = s
			}
		}
		if _, ok := evt.Data["tool_call_id"]; !ok {
			if s := asAnyString(evt.Data["call_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["tool_use_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["item_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["output_item_id"]); s != "" {
				evt.Data["tool_call_id"] = s
			} else if s := asAnyString(evt.Data["id"]); s != "" {
				evt.Data["tool_call_id"] = s
			}
		}
		if _, ok := evt.Data["arguments"]; !ok {
			if s := asAnyString(evt.Data["input"]); s != "" {
				evt.Data["arguments"] = s
			}
		}
	case "usage":
		if _, ok := evt.Data["prompt_tokens"]; !ok {
			if n, ok := asAnyInt(evt.Data["input_tokens"]); ok {
				evt.Data["prompt_tokens"] = n
			}
		}
		if _, ok := evt.Data["completion_tokens"]; !ok {
			if n, ok := asAnyInt(evt.Data["output_tokens"]); ok {
				evt.Data["completion_tokens"] = n
			}
		}
		if _, ok := evt.Data["total_tokens"]; !ok {
			prompt, hasPrompt := asAnyInt(evt.Data["prompt_tokens"])
			completion, hasCompletion := asAnyInt(evt.Data["completion_tokens"])
			if hasPrompt && hasCompletion {
				evt.Data["total_tokens"] = prompt + completion
			} else if n, ok := asAnyInt(evt.Data["tokens"]); ok {
				evt.Data["total_tokens"] = n
			}
		}
		// Prompt cache read tokens:
		// - Anthropic: cache_read_input_tokens
		// - OpenAI/Codex (nested): prompt_tokens_details.cached_tokens / input_tokens_details.cached_tokens
		if _, ok := evt.Data["prompt_cache_read_tokens"]; !ok {
			if n, ok := asAnyInt(evt.Data["cache_read_input_tokens"]); ok {
				evt.Data["prompt_cache_read_tokens"] = n
			} else if n, ok := asAnyInt(evt.Data["cached_tokens"]); ok {
				evt.Data["prompt_cache_read_tokens"] = n
			} else if details := asAnyMap(evt.Data["prompt_tokens_details"]); details != nil {
				if n, ok := asAnyInt(details["cached_tokens"]); ok {
					evt.Data["prompt_cache_read_tokens"] = n
				}
			} else if details := asAnyMap(evt.Data["input_tokens_details"]); details != nil {
				if n, ok := asAnyInt(details["cached_tokens"]); ok {
					evt.Data["prompt_cache_read_tokens"] = n
				}
			}
		}
		// Prompt cache write tokens:
		// - Anthropic: cache_creation_input_tokens
		if _, ok := evt.Data["prompt_cache_write_tokens"]; !ok {
			if n, ok := asAnyInt(evt.Data["cache_creation_input_tokens"]); ok {
				evt.Data["prompt_cache_write_tokens"] = n
			}
		}
	}
	return evt
}

func asAnyString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func asAnyMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asAnyInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case json.Number:
		n, err := strconv.Atoi(string(x))
		if err != nil {
			return 0, false
		}
		return n, true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func canonicalFinishReason(reason string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "", "stop", "end_turn":
		return "stop"
	case "tool_calls", "tool_use":
		return "tool_calls"
	case "max_tokens", "max_output_tokens", "length":
		return "length"
	default:
		return strings.ToLower(strings.TrimSpace(reason))
	}
}

func canonicalIncompleteReason(reason string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "", "none":
		return ""
	case "max_tokens", "max_output_tokens", "length":
		return "length"
	default:
		return strings.ToLower(strings.TrimSpace(reason))
	}
}
