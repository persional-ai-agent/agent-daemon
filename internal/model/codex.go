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

type CodexClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type codexRequest struct {
	Model string      `json:"model"`
	Input []any       `json:"input"`
	Tools []codexTool `json:"tools,omitempty"`
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
