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

type AnthropicClient struct {
	BaseURL      string
	APIKey       string
	Model        string
	HTTPClient   *http.Client
	MaxTokens    int
	UseStreaming bool
}

type anthropicRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []anthropicTurn `json:"messages"`
	Tools     []anthropicTool `json:"tools,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
}

type anthropicTurn struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}

type anthropicResponse struct {
	Content []anthropicBlock `json:"content"`
}

type anthropicBlock struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"`
}

func NewAnthropicClient(baseURL, apiKey, modelName string) *AnthropicClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	if strings.TrimSpace(modelName) == "" {
		modelName = "claude-3-5-haiku-latest"
	}
	return &AnthropicClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Model:      modelName,
		HTTPClient: &http.Client{Timeout: 180 * time.Second},
		MaxTokens:  4096,
	}
}

func (c *AnthropicClient) ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error) {
	return c.ChatCompletionWithEvents(ctx, messages, tools, nil)
}

func (c *AnthropicClient) ChatCompletionWithEvents(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	if c.UseStreaming {
		return c.chatCompletionStream(ctx, messages, tools, sink)
	}
	systemText, turns := toAnthropicTurns(messages)
	reqBody := anthropicRequest{
		Model:     c.Model,
		MaxTokens: c.MaxTokens,
		System:    systemText,
		Messages:  turns,
		Tools:     toAnthropicTools(tools),
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return core.Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/messages", bytes.NewReader(b))
	if err != nil {
		return core.Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if c.APIKey != "" {
		req.Header.Set("x-api-key", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return core.Message{}, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return core.Message{}, fmt.Errorf("anthropic api error (%d): %s", resp.StatusCode, string(data))
	}
	var out anthropicResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return core.Message{}, err
	}
	return fromAnthropicResponse(out), nil
}

type anthropicStreamBlockBuilder struct {
	Type  string
	ID    string
	Name  string
	Text  strings.Builder
	Input strings.Builder
}

func (c *AnthropicClient) chatCompletionStream(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	systemText, turns := toAnthropicTurns(messages)
	reqBody := anthropicRequest{
		Model:     c.Model,
		MaxTokens: c.MaxTokens,
		System:    systemText,
		Messages:  turns,
		Tools:     toAnthropicTools(tools),
		Stream:    true,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return core.Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/messages", bytes.NewReader(b))
	if err != nil {
		return core.Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if c.APIKey != "" {
		req.Header.Set("x-api-key", c.APIKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return core.Message{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return core.Message{}, fmt.Errorf("anthropic api error (%d): %s", resp.StatusCode, string(data))
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		data, _ := io.ReadAll(resp.Body)
		var out anthropicResponse
		if err := json.Unmarshal(data, &out); err != nil {
			return core.Message{}, err
		}
		return fromAnthropicResponse(out), nil
	}
	emitStreamEvent(sink, StreamEvent{
		Provider: "anthropic",
		Type:     "message_start",
		Data:     map[string]any{},
	})

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	blocks := map[int]*anthropicStreamBlockBuilder{}
	messageID := ""
	finishReasonRaw := ""
	stopSequenceRaw := ""
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
			return core.Message{}, fmt.Errorf("anthropic stream error: %s", asString(errObj["message"]))
		}
		index := intFromAny(event["index"])
		typ := strings.ToLower(strings.TrimSpace(asString(event["type"])))
		switch typ {
		case "message_start":
			msg, _ := event["message"].(map[string]any)
			if msgID := asString(msg["id"]); msgID != "" {
				messageID = msgID
				emitStreamEvent(sink, StreamEvent{
					Provider: "anthropic",
					Type:     "message_start",
					Data:     map[string]any{"message_id": msgID},
				})
			}
		case "content_block_start":
			block, _ := event["content_block"].(map[string]any)
			b := &anthropicStreamBlockBuilder{
				Type: strings.ToLower(strings.TrimSpace(asString(block["type"]))),
				ID:   asString(block["id"]),
				Name: asString(block["name"]),
			}
			if input, ok := block["input"].(map[string]any); ok {
				raw, _ := json.Marshal(input)
				b.Input.Write(raw)
			}
			if text := asString(block["text"]); text != "" {
				b.Text.WriteString(text)
			}
			blocks[index] = b
			if b.Type == "tool_use" {
				emitStreamEvent(sink, StreamEvent{
					Provider: "anthropic",
					Type:     "tool_call_start",
					Data: map[string]any{
						"tool_call_id": b.ID,
						"tool_name":    b.Name,
					},
				})
				emitStreamEvent(sink, StreamEvent{
					Provider: "anthropic",
					Type:     "tool_args_start",
					Data: map[string]any{
						"tool_call_id": b.ID,
						"tool_name":    b.Name,
					},
				})
			}
		case "content_block_delta":
			b := blocks[index]
			if b == nil {
				b = &anthropicStreamBlockBuilder{}
				blocks[index] = b
			}
			delta, _ := event["delta"].(map[string]any)
			deltaType := strings.ToLower(strings.TrimSpace(asString(delta["type"])))
			switch deltaType {
			case "text_delta":
				b.Type = "text"
				b.Text.WriteString(asString(delta["text"]))
				emitStreamEvent(sink, StreamEvent{
					Provider: "anthropic",
					Type:     "text_delta",
					Data:     map[string]any{"text": asString(delta["text"])},
				})
			case "input_json_delta":
				if b.Type == "" {
					b.Type = "tool_use"
				}
				b.Input.WriteString(asString(delta["partial_json"]))
				emitStreamEvent(sink, StreamEvent{
					Provider: "anthropic",
					Type:     "tool_args_delta",
					Data: map[string]any{
						"tool_name":       b.Name,
						"arguments_delta": asString(delta["partial_json"]),
					},
				})
			}
		case "message_delta":
			if stopReason := asString(event["stop_reason"]); stopReason != "" {
				finishReasonRaw = stopReason
			}
			if stopSequence := asString(event["stop_sequence"]); stopSequence != "" {
				stopSequenceRaw = stopSequence
			}
			usage, _ := event["usage"].(map[string]any)
			if len(usage) > 0 {
				emitStreamEvent(sink, StreamEvent{
					Provider: "anthropic",
					Type:     "usage",
					Data:     usage,
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return core.Message{}, err
	}

	indexes := make([]int, 0, len(blocks))
	for idx := range blocks {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	texts := make([]string, 0, len(indexes))
	calls := make([]core.ToolCall, 0, len(indexes))
	for _, idx := range indexes {
		b := blocks[idx]
		switch b.Type {
		case "text":
			if t := strings.TrimSpace(b.Text.String()); t != "" {
				texts = append(texts, t)
			}
		case "tool_use":
			if strings.TrimSpace(b.Name) == "" {
				continue
			}
			callID := b.ID
			if callID == "" {
				callID = "toolu_stream_" + strconv.Itoa(idx)
			}
			args := strings.TrimSpace(b.Input.String())
			if args == "" {
				args = "{}"
			}
			calls = append(calls, core.ToolCall{
				ID:   callID,
				Type: "function",
				Function: core.ToolFunction{
					Name:      b.Name,
					Arguments: args,
				},
			})
			emitStreamEvent(sink, StreamEvent{
				Provider: "anthropic",
				Type:     "tool_call_done",
				Data: map[string]any{
					"tool_call_id": callID,
					"tool_name":    b.Name,
					"arguments":    args,
				},
			})
			emitStreamEvent(sink, StreamEvent{
				Provider: "anthropic",
				Type:     "tool_args_done",
				Data: map[string]any{
					"tool_call_id": callID,
					"tool_name":    b.Name,
					"arguments":    args,
				},
			})
		}
	}
	finishReason := finishReasonRaw
	if strings.TrimSpace(finishReason) == "" {
		finishReason = "stop"
		if len(calls) > 0 {
			finishReason = "tool_calls"
		}
	}
	doneData := map[string]any{
		"text":            strings.Join(texts, "\n"),
		"tool_call_count": len(calls),
		"finish_reason":   finishReason,
	}
	if strings.TrimSpace(stopSequenceRaw) != "" {
		doneData["stop_sequence"] = stopSequenceRaw
	}
	if strings.TrimSpace(messageID) != "" {
		doneData["message_id"] = messageID
	}
	if finishReason == "max_tokens" {
		doneData["incomplete_reason"] = "length"
	}
	emitStreamEvent(sink, StreamEvent{
		Provider: "anthropic",
		Type:     "message_done",
		Data:     doneData,
	})
	return core.Message{
		Role:      "assistant",
		Content:   strings.Join(texts, "\n"),
		ToolCalls: calls,
	}, nil
}

func intFromAny(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	default:
		return 0
	}
}

func toAnthropicTools(tools []core.ToolSchema) []anthropicTool {
	out := make([]anthropicTool, 0, len(tools))
	for _, t := range tools {
		out = append(out, anthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}
	return out
}

func toAnthropicTurns(messages []core.Message) (string, []anthropicTurn) {
	systemParts := make([]string, 0, 2)
	turns := make([]anthropicTurn, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			if strings.TrimSpace(msg.Content) != "" {
				systemParts = append(systemParts, msg.Content)
			}
		case "user":
			turns = append(turns, anthropicTurn{
				Role: "user",
				Content: []map[string]any{
					{"type": "text", "text": msg.Content},
				},
			})
		case "assistant":
			blocks := make([]map[string]any, 0, len(msg.ToolCalls)+1)
			if strings.TrimSpace(msg.Content) != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": parseToolArgs(tc.Function.Arguments),
				})
			}
			if len(blocks) == 0 {
				continue
			}
			turns = append(turns, anthropicTurn{Role: "assistant", Content: blocks})
		case "tool":
			turns = append(turns, anthropicTurn{
				Role: "user",
				Content: []map[string]any{
					{
						"type":        "tool_result",
						"tool_use_id": msg.ToolCallID,
						"content":     msg.Content,
					},
				},
			})
		}
	}
	return strings.Join(systemParts, "\n\n"), turns
}

func parseToolArgs(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{"_raw": raw}
	}
	return out
}

func fromAnthropicResponse(resp anthropicResponse) core.Message {
	texts := make([]string, 0, 2)
	calls := make([]core.ToolCall, 0, 2)
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if strings.TrimSpace(block.Text) != "" {
				texts = append(texts, block.Text)
			}
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			calls = append(calls, core.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: core.ToolFunction{
					Name:      block.Name,
					Arguments: string(args),
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
