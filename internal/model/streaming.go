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
			}
		}
	case "message_done":
		if _, ok := evt.Data["message_id"]; !ok {
			if s := asAnyString(evt.Data["id"]); s != "" {
				evt.Data["message_id"] = s
			}
		}
		if _, ok := evt.Data["finish_reason"]; !ok {
			if s := asAnyString(evt.Data["reason"]); s != "" {
				evt.Data["finish_reason"] = s
			} else {
				evt.Data["finish_reason"] = "stop"
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
	}
	return evt
}

func asAnyString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
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
