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

type Client interface {
	ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error)
}

type OpenAIClient struct {
	BaseURL      string
	APIKey       string
	Model        string
	HTTPClient   *http.Client
	UseStreaming bool
}

type chatRequest struct {
	Model       string            `json:"model"`
	Messages    []core.Message    `json:"messages"`
	Tools       []core.ToolSchema `json:"tools,omitempty"`
	Temperature float64           `json:"temperature"`
	N           int               `json:"n"`
	Stream      bool              `json:"stream,omitempty"`
	StreamOpts  map[string]any    `json:"stream_options,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message core.Message `json:"message"`
	} `json:"choices"`
}

func NewOpenAIClient(baseURL, apiKey, modelName string) *OpenAIClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIClient{BaseURL: baseURL, APIKey: apiKey, Model: modelName, HTTPClient: &http.Client{Timeout: 180 * time.Second}}
}

func (c *OpenAIClient) ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error) {
	return c.ChatCompletionWithEvents(ctx, messages, tools, nil)
}

func (c *OpenAIClient) ChatCompletionWithEvents(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	if c.UseStreaming {
		return c.chatCompletionStream(ctx, messages, tools, sink)
	}
	reqBody := chatRequest{Model: c.Model, Messages: messages, Tools: tools, Temperature: 0.2, N: 1}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return core.Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/chat/completions", bytes.NewReader(b))
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
		return core.Message{}, fmt.Errorf("openai api error (%d): %s", resp.StatusCode, string(data))
	}
	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return core.Message{}, err
	}
	if len(out.Choices) == 0 {
		return core.Message{}, fmt.Errorf("empty choices")
	}
	return out.Choices[0].Message, nil
}

type openAIStreamChunk struct {
	Usage   map[string]any `json:"usage,omitempty"`
	Choices []struct {
		FinishReason *string `json:"finish_reason,omitempty"`
		Delta        struct {
			Content   string `json:"content,omitempty"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id,omitempty"`
				Type     string `json:"type,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
}

type streamToolCallBuilder struct {
	ID   string
	Type string
	Name string
	Args strings.Builder
}

func (c *OpenAIClient) chatCompletionStream(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	reqBody := chatRequest{
		Model:       c.Model,
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.2,
		N:           1,
		Stream:      true,
		StreamOpts:  map[string]any{"include_usage": true},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return core.Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/chat/completions", bytes.NewReader(b))
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
		return core.Message{}, fmt.Errorf("openai api error (%d): %s", resp.StatusCode, string(data))
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		data, _ := io.ReadAll(resp.Body)
		var out chatResponse
		if err := json.Unmarshal(data, &out); err != nil {
			return core.Message{}, err
		}
		if len(out.Choices) == 0 {
			return core.Message{}, fmt.Errorf("empty choices")
		}
		return out.Choices[0].Message, nil
	}
	emitStreamEvent(sink, StreamEvent{
		Provider: "openai",
		Type:     "message_start",
		Data:     map[string]any{},
	})
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	content := strings.Builder{}
	toolBuilders := map[int]*streamToolCallBuilder{}
	var streamFinishReason string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			break
		}
		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Usage) > 0 {
			emitStreamEvent(sink, StreamEvent{
				Provider: "openai",
				Type:     "usage",
				Data:     chunk.Usage,
			})
		}
		for _, choice := range chunk.Choices {
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				streamFinishReason = *choice.FinishReason
			}
			if choice.Delta.Content != "" {
				content.WriteString(choice.Delta.Content)
				emitStreamEvent(sink, StreamEvent{
					Provider: "openai",
					Type:     "text_delta",
					Data:     map[string]any{"text": choice.Delta.Content},
				})
			}
			for _, tc := range choice.Delta.ToolCalls {
				builder, ok := toolBuilders[tc.Index]
				if !ok {
					builder = &streamToolCallBuilder{}
					toolBuilders[tc.Index] = builder
					emitStreamEvent(sink, StreamEvent{
						Provider: "openai",
						Type:     "tool_call_start",
						Data: map[string]any{
							"tool_call_index": tc.Index,
							"tool_call_id":    tc.ID,
							"tool_name":       tc.Function.Name,
						},
					})
					emitStreamEvent(sink, StreamEvent{
						Provider: "openai",
						Type:     "tool_args_start",
						Data: map[string]any{
							"tool_call_index": tc.Index,
							"tool_call_id":    tc.ID,
							"tool_name":       tc.Function.Name,
						},
					})
				}
				if tc.ID != "" {
					builder.ID = tc.ID
				}
				if tc.Type != "" {
					builder.Type = tc.Type
				}
				if tc.Function.Name != "" {
					builder.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					builder.Args.WriteString(tc.Function.Arguments)
					emitStreamEvent(sink, StreamEvent{
						Provider: "openai",
						Type:     "tool_args_delta",
						Data: map[string]any{
							"tool_call_index": tc.Index,
							"tool_name":       builder.Name,
							"arguments_delta": tc.Function.Arguments,
						},
					})
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return core.Message{}, err
	}
	calls := make([]core.ToolCall, 0, len(toolBuilders))
	indexes := make([]int, 0, len(toolBuilders))
	for idx := range toolBuilders {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	for _, idx := range indexes {
		b := toolBuilders[idx]
		if strings.TrimSpace(b.Name) == "" {
			continue
		}
		callID := b.ID
		if callID == "" {
			callID = "stream-call-" + strconv.Itoa(idx)
		}
		callType := b.Type
		if callType == "" {
			callType = "function"
		}
		calls = append(calls, core.ToolCall{
			ID:   callID,
			Type: callType,
			Function: core.ToolFunction{
				Name:      b.Name,
				Arguments: b.Args.String(),
			},
		})
		emitStreamEvent(sink, StreamEvent{
			Provider: "openai",
			Type:     "tool_call_done",
			Data: map[string]any{
				"tool_call_index": idx,
				"tool_call_id":    callID,
				"tool_name":       b.Name,
				"arguments":       b.Args.String(),
			},
		})
		emitStreamEvent(sink, StreamEvent{
			Provider: "openai",
			Type:     "tool_args_done",
			Data: map[string]any{
				"tool_call_index": idx,
				"tool_call_id":    callID,
				"tool_name":       b.Name,
				"arguments":       b.Args.String(),
			},
		})
	}
	finishReason := "stop"
	if len(calls) > 0 {
		finishReason = "tool_calls"
	}
	if streamFinishReason != "" {
		finishReason = streamFinishReason
	}
	doneData := map[string]any{
		"text":            content.String(),
		"tool_call_count": len(calls),
		"finish_reason":   finishReason,
	}
	if finishReason == "length" {
		doneData["incomplete_reason"] = "length"
	}
	emitStreamEvent(sink, StreamEvent{
		Provider: "openai",
		Type:     "message_done",
		Data:     doneData,
	})
	return core.Message{
		Role:      "assistant",
		Content:   content.String(),
		ToolCalls: calls,
	}, nil
}

func emitStreamEvent(sink StreamEventSink, evt StreamEvent) {
	if sink == nil {
		return
	}
	sink(evt)
}
