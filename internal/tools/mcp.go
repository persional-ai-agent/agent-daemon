package tools

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

type MCPClient struct {
	Endpoint   string
	HTTPClient *http.Client
}

type mcpDiscoveryResponse struct {
	Tools []struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"tools"`
}

type mcpToolProxy struct {
	name   string
	desc   string
	params map[string]any
	client *MCPClient
}

func NewMCPClient(endpoint string, timeout time.Duration) *MCPClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &MCPClient{
		Endpoint:   strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		HTTPClient: &http.Client{Timeout: timeout},
	}
}

func RegisterMCPTools(ctx context.Context, r *Registry, client *MCPClient) ([]string, error) {
	if r == nil || client == nil || client.Endpoint == "" {
		return nil, nil
	}
	tools, err := client.DiscoverTools(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if strings.TrimSpace(t.Name()) == "" {
			continue
		}
		r.Register(t)
		names = append(names, t.Name())
	}
	return names, nil
}

func (c *MCPClient) DiscoverTools(ctx context.Context) ([]Tool, error) {
	if c.Endpoint == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint+"/tools", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mcp tools discovery failed (%d): %s", resp.StatusCode, string(data))
	}
	var out mcpDiscoveryResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	registered := make([]Tool, 0, len(out.Tools))
	for _, tool := range out.Tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		params := tool.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		registered = append(registered, &mcpToolProxy{
			name:   name,
			desc:   tool.Description,
			params: params,
			client: c,
		})
	}
	return registered, nil
}

func (m *mcpToolProxy) Name() string { return m.name }

func (m *mcpToolProxy) Schema() core.ToolSchema {
	return core.ToolSchema{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:        m.name,
			Description: m.desc,
			Parameters:  m.params,
		},
	}
}

func (m *mcpToolProxy) Call(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if m.client == nil || m.client.Endpoint == "" {
		return nil, fmt.Errorf("mcp client unavailable")
	}
	payload := map[string]any{
		"name":      m.name,
		"arguments": args,
		"context": map[string]any{
			"session_id": tc.SessionID,
			"workdir":    tc.Workdir,
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.client.Endpoint+"/call", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mcp tool call failed (%d): %s", resp.StatusCode, string(data))
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if result, ok := body["result"].(map[string]any); ok {
		return result, nil
	}
	return body, nil
}
