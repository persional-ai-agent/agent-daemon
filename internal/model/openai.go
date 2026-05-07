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

type Client interface {
	ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error)
}

type OpenAIClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type chatRequest struct {
	Model       string            `json:"model"`
	Messages    []core.Message    `json:"messages"`
	Tools       []core.ToolSchema `json:"tools,omitempty"`
	Temperature float64           `json:"temperature"`
	N           int               `json:"n"`
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
