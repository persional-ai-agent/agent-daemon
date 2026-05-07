package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type AnthropicClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
	MaxTokens  int
}

type anthropicRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []anthropicTurn `json:"messages"`
	Tools     []anthropicTool `json:"tools,omitempty"`
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
