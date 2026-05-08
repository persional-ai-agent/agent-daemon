package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type CodexClient struct {
	BaseURL      string
	APIKey       string
	Model        string
	HTTPClient   *http.Client
	UseStreaming bool
}

type codexRequest struct {
	Model  string      `json:"model"`
	Input  []any       `json:"input"`
	Tools  []codexTool `json:"tools,omitempty"`
	Stream bool        `json:"stream,omitempty"`
}

type codexTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type codexResponse struct {
	Output []map[string]any `json:"output"`
}

func NewCodexClient(baseURL, apiKey, modelName string) *CodexClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(modelName) == "" {
		modelName = "gpt-5-codex"
	}
	return &CodexClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Model:      modelName,
		HTTPClient: &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *CodexClient) ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error) {
	return c.ChatCompletionWithEvents(ctx, messages, tools, nil)
}

func (c *CodexClient) ChatCompletionWithEvents(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	if c.UseStreaming {
		return c.chatCompletionStream(ctx, messages, tools, sink)
	}
	reqBody := codexRequest{
		Model: c.Model,
		Input: toCodexInput(messages),
		Tools: toCodexTools(tools),
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return core.Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/responses", bytes.NewReader(b))
	if err != nil {
		return core.Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return core.Message{}, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return core.Message{}, fmt.Errorf("codex responses api error (%d): %s", resp.StatusCode, string(data))
	}
	var out codexResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return core.Message{}, err
	}
	return fromCodexResponse(out), nil
}

type codexStreamBuilder struct {
	ID   string
	Type string
	Role string
	Name string
	Text strings.Builder
	Args strings.Builder
}

func (c *CodexClient) chatCompletionStream(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	reqBody := codexRequest{
		Model:  c.Model,
		Input:  toCodexInput(messages),
		Tools:  toCodexTools(tools),
		Stream: true,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return core.Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/responses", bytes.NewReader(b))
	if err != nil {
		return core.Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return core.Message{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return core.Message{}, fmt.Errorf("codex responses api error (%d): %s", resp.StatusCode, string(data))
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		data, _ := io.ReadAll(resp.Body)
		var out codexResponse
		if err := json.Unmarshal(data, &out); err != nil {
			return core.Message{}, err
		}
		return fromCodexResponse(out), nil
	}
	emitStreamEvent(sink, StreamEvent{
		Provider: "codex",
		Type:     "message_start",
		Data:     map[string]any{},
	})

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	builders := map[string]*codexStreamBuilder{}
	order := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		if errObj, ok := event["error"].(map[string]any); ok {
			return core.Message{}, fmt.Errorf("codex stream error: %s", asString(errObj["message"]))
		}
		// Completed envelope may already contain full response.
		if respObj, ok := event["response"].(map[string]any); ok {
			responseID := asString(respObj["id"])
			if usage, ok := respObj["usage"].(map[string]any); ok && len(usage) > 0 {
				emitStreamEvent(sink, StreamEvent{
					Provider: "codex",
					Type:     "usage",
					Data:     usage,
				})
			}
			if output, ok := respObj["output"].([]any); ok {
				msg := fromCodexResponse(codexResponse{Output: toMapSlice(output)})
				finishReason := "stop"
				if reason := asString(respObj["finish_reason"]); reason != "" {
					finishReason = reason
				}
				if len(msg.ToolCalls) > 0 {
					finishReason = "tool_calls"
				}
				doneData := map[string]any{
					"text":            msg.Content,
					"tool_call_count": len(msg.ToolCalls),
					"finish_reason":   finishReason,
				}
				if incomplete, ok := respObj["incomplete_details"].(map[string]any); ok {
					if s := asString(incomplete["reason"]); s != "" {
						doneData["incomplete_reason"] = s
					}
				}
				if responseID != "" {
					doneData["response_id"] = responseID
				}
				emitStreamEvent(sink, StreamEvent{
					Provider: "codex",
					Type:     "message_done",
					Data:     doneData,
				})
				return msg, nil
			}
		}
		typ := strings.ToLower(strings.TrimSpace(asString(event["type"])))
		switch typ {
		case "response.output_item.added":
			item, _ := event["item"].(map[string]any)
			id := asString(item["id"])
			if id == "" {
				id = "item-" + strconv.Itoa(len(builders))
			}
			if _, ok := builders[id]; !ok {
				order = append(order, id)
			}
			b := &codexStreamBuilder{
				ID:   id,
				Type: strings.ToLower(strings.TrimSpace(asString(item["type"]))),
				Role: asString(item["role"]),
				Name: asString(item["name"]),
			}
			if text := extractCodexText(item["content"]); text != "" {
				b.Text.WriteString(text)
			}
			if args := asString(item["arguments"]); args != "" {
				b.Args.WriteString(args)
			}
			builders[id] = b
			if b.Type == "function_call" {
				emitStreamEvent(sink, StreamEvent{
					Provider: "codex",
					Type:     "tool_call_start",
					Data: map[string]any{
						"tool_call_id": id,
						"tool_name":    b.Name,
					},
				})
				emitStreamEvent(sink, StreamEvent{
					Provider: "codex",
					Type:     "tool_args_start",
					Data: map[string]any{
						"tool_call_id": id,
						"tool_name":    b.Name,
					},
				})
			}
		case "response.output_text.delta":
			id := asString(event["item_id"])
			if id == "" {
				id = asString(event["output_item_id"])
			}
			if id == "" {
				continue
			}
			b := ensureCodexBuilder(builders, &order, id)
			if b.Type == "" {
				b.Type = "message"
			}
			if b.Role == "" {
				b.Role = "assistant"
			}
			b.Text.WriteString(asString(event["delta"]))
			emitStreamEvent(sink, StreamEvent{
				Provider: "codex",
				Type:     "text_delta",
				Data:     map[string]any{"text": asString(event["delta"])},
			})
		case "response.function_call_arguments.delta":
			id := asString(event["item_id"])
			if id == "" {
				id = asString(event["output_item_id"])
			}
			if id == "" {
				continue
			}
			b := ensureCodexBuilder(builders, &order, id)
			if b.Type == "" {
				b.Type = "function_call"
			}
			if name := asString(event["name"]); name != "" {
				b.Name = name
			}
			b.Args.WriteString(asString(event["delta"]))
			emitStreamEvent(sink, StreamEvent{
				Provider: "codex",
				Type:     "tool_args_delta",
				Data: map[string]any{
					"tool_name":       b.Name,
					"arguments_delta": asString(event["delta"]),
				},
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return core.Message{}, err
	}
	msg := buildCodexMessageFromStream(builders, order, sink)
	emitStreamEvent(sink, StreamEvent{
		Provider: "codex",
		Type:     "message_done",
		Data: map[string]any{
			"text":            msg.Content,
			"tool_call_count": len(msg.ToolCalls),
			"finish_reason": func() string {
				if len(msg.ToolCalls) > 0 {
					return "tool_calls"
				}
				return "stop"
			}(),
		},
	})
	return msg, nil
}

func ensureCodexBuilder(builders map[string]*codexStreamBuilder, order *[]string, id string) *codexStreamBuilder {
	if b, ok := builders[id]; ok {
		return b
	}
	b := &codexStreamBuilder{ID: id}
	builders[id] = b
	*order = append(*order, id)
	return b
}

func buildCodexMessageFromStream(builders map[string]*codexStreamBuilder, order []string, sink StreamEventSink) core.Message {
	// Keep deterministic ordering for stable outputs.
	indexed := make([]string, 0, len(order))
	seen := map[string]struct{}{}
	for _, id := range order {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		indexed = append(indexed, id)
	}
	sort.Strings(indexed)
	texts := make([]string, 0, len(indexed))
	calls := make([]core.ToolCall, 0, len(indexed))
	for _, id := range indexed {
		b := builders[id]
		switch b.Type {
		case "message":
			if strings.EqualFold(b.Role, "assistant") {
				if t := strings.TrimSpace(b.Text.String()); t != "" {
					texts = append(texts, t)
				}
			}
		case "function_call":
			if strings.TrimSpace(b.Name) == "" {
				continue
			}
			args := strings.TrimSpace(b.Args.String())
			if args == "" {
				args = "{}"
			}
			calls = append(calls, core.ToolCall{
				ID:   id,
				Type: "function",
				Function: core.ToolFunction{
					Name:      b.Name,
					Arguments: args,
				},
			})
			emitStreamEvent(sink, StreamEvent{
				Provider: "codex",
				Type:     "tool_call_done",
				Data: map[string]any{
					"tool_call_id": id,
					"tool_name":    b.Name,
					"arguments":    args,
				},
			})
			emitStreamEvent(sink, StreamEvent{
				Provider: "codex",
				Type:     "tool_args_done",
				Data: map[string]any{
					"tool_call_id": id,
					"tool_name":    b.Name,
					"arguments":    args,
				},
			})
		}
	}
	return core.Message{
		Role:      "assistant",
		Content:   strings.Join(texts, "\n"),
		ToolCalls: calls,
	}
}

func toMapSlice(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, x := range items {
		m, _ := x.(map[string]any)
		if m != nil {
			out = append(out, m)
		}
	}
	return out
}

func toCodexTools(tools []core.ToolSchema) []codexTool {
	out := make([]codexTool, 0, len(tools))
	for _, t := range tools {
		out = append(out, codexTool{
			Type:        "function",
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  t.Function.Parameters,
		})
	}
	return out
}

func toCodexInput(messages []core.Message) []any {
	items := make([]any, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			items = append(items, map[string]any{"type": "message", "role": "system", "content": msg.Content})
		case "user":
			items = append(items, map[string]any{"type": "message", "role": "user", "content": msg.Content})
		case "assistant":
			if strings.TrimSpace(msg.Content) != "" {
				items = append(items, map[string]any{"type": "message", "role": "assistant", "content": msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				items = append(items, map[string]any{
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				})
			}
		case "tool":
			items = append(items, map[string]any{
				"type":    "function_call_output",
				"call_id": msg.ToolCallID,
				"output":  msg.Content,
			})
		}
	}
	return items
}

func fromCodexResponse(resp codexResponse) core.Message {
	texts := make([]string, 0, 2)
	calls := make([]core.ToolCall, 0, 2)
	for _, item := range resp.Output {
		switch strings.ToLower(strings.TrimSpace(asString(item["type"]))) {
		case "message":
			if strings.EqualFold(asString(item["role"]), "assistant") {
				if text := strings.TrimSpace(extractCodexText(item["content"])); text != "" {
					texts = append(texts, text)
				}
			}
		case "function_call":
			callID := asString(item["call_id"])
			if callID == "" {
				callID = asString(item["id"])
			}
			name := asString(item["name"])
			if callID == "" || name == "" {
				continue
			}
			args := asString(item["arguments"])
			if args == "" {
				if input, ok := item["input"].(map[string]any); ok {
					b, _ := json.Marshal(input)
					args = string(b)
				} else {
					args = "{}"
				}
			}
			calls = append(calls, core.ToolCall{
				ID:   callID,
				Type: "function",
				Function: core.ToolFunction{
					Name:      name,
					Arguments: args,
				},
			})
		}
	}
	return core.Message{
		Role:      "assistant",
		Content:   strings.Join(texts, "\n"),
		ToolCalls: calls,
	}
}

func extractCodexText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, 0, len(v))
		for _, p := range v {
			m, _ := p.(map[string]any)
			if m == nil {
				continue
			}
			if strings.EqualFold(asString(m["type"]), "output_text") || strings.EqualFold(asString(m["type"]), "text") {
				if t := asString(m["text"]); strings.TrimSpace(t) != "" {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
