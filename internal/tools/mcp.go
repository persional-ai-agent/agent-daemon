package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type MCPClient struct {
	Endpoint     string
	HTTPClient   *http.Client
	StdioCommand string
	OAuth        MCPOAuthConfig

	mu                sync.Mutex
	cachedAccessToken string
	cachedExpiry      time.Time
}

type MCPOAuthConfig struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       string
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
	trimmed := strings.TrimSpace(endpoint)
	if strings.HasPrefix(trimmed, "stdio:") {
		return &MCPClient{
			StdioCommand: strings.TrimSpace(strings.TrimPrefix(trimmed, "stdio:")),
			HTTPClient:   &http.Client{Timeout: timeout},
		}
	}
	return &MCPClient{
		Endpoint:   strings.TrimRight(trimmed, "/"),
		HTTPClient: &http.Client{Timeout: timeout},
	}
}

func NewMCPStdioClient(command string, timeout time.Duration) *MCPClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &MCPClient{
		StdioCommand: strings.TrimSpace(command),
		HTTPClient:   &http.Client{Timeout: timeout},
	}
}

func (c *MCPClient) ConfigureOAuthClientCredentials(cfg MCPOAuthConfig) {
	c.OAuth = MCPOAuthConfig{
		TokenURL:     strings.TrimSpace(cfg.TokenURL),
		ClientID:     strings.TrimSpace(cfg.ClientID),
		ClientSecret: cfg.ClientSecret,
		Scopes:       strings.TrimSpace(cfg.Scopes),
	}
}

func RegisterMCPTools(ctx context.Context, r *Registry, client *MCPClient) ([]string, error) {
	if r == nil || client == nil || (client.Endpoint == "" && strings.TrimSpace(client.StdioCommand) == "") {
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
	if strings.TrimSpace(c.StdioCommand) != "" {
		return c.discoverToolsStdio(ctx)
	}
	if c.Endpoint == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint+"/tools", nil)
	if err != nil {
		return nil, err
	}
	if err := c.injectAuthHeader(ctx, req); err != nil {
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
	if m.client == nil || (m.client.Endpoint == "" && strings.TrimSpace(m.client.StdioCommand) == "") {
		return nil, fmt.Errorf("mcp client unavailable")
	}
	if strings.TrimSpace(m.client.StdioCommand) != "" {
		return m.callStdio(ctx, args, tc)
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
	if err := m.client.injectAuthHeader(ctx, req); err != nil {
		return nil, err
	}
	resp, err := m.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		return parseMCPCallSSE(resp.Body)
	}
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

func (c *MCPClient) discoverToolsStdio(ctx context.Context) ([]Tool, error) {
	res, err := c.withStdioSession(ctx, func(sess *mcpStdioSession) (map[string]any, error) {
		return sess.request("tools/list", map[string]any{})
	})
	if err != nil {
		return nil, err
	}
	rawTools, _ := res["tools"].([]any)
	registered := make([]Tool, 0, len(rawTools))
	for _, item := range rawTools {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(strMap(m, "name"))
		if name == "" {
			continue
		}
		params := mapAnyMap(m["inputSchema"])
		if params == nil {
			params = mapAnyMap(m["parameters"])
		}
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		registered = append(registered, &mcpToolProxy{
			name:   name,
			desc:   strMap(m, "description"),
			params: params,
			client: c,
		})
	}
	return registered, nil
}

func (m *mcpToolProxy) callStdio(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	res, err := m.client.withStdioSession(ctx, func(sess *mcpStdioSession) (map[string]any, error) {
		return sess.request("tools/call", map[string]any{
			"name":      m.name,
			"arguments": args,
			"context": map[string]any{
				"session_id": tc.SessionID,
				"workdir":    tc.Workdir,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	if structured, ok := res["structuredContent"].(map[string]any); ok {
		return structured, nil
	}
	return res, nil
}

type mcpStdioSession struct {
	stdin  io.Writer
	stdout *bufio.Reader
	nextID int
}

func (c *MCPClient) withStdioSession(ctx context.Context, fn func(sess *mcpStdioSession) (map[string]any, error)) (map[string]any, error) {
	command := strings.TrimSpace(c.StdioCommand)
	if command == "" {
		return nil, fmt.Errorf("mcp stdio command required")
	}
	cmd := exec.CommandContext(ctx, "sh", "-lc", command)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	sess := &mcpStdioSession{
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		nextID: 1,
	}
	closeOnce := func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	}
	if err := sess.initialize(); err != nil {
		closeOnce()
		errText := strings.TrimSpace(stderr.String())
		if errText != "" {
			return nil, fmt.Errorf("mcp stdio initialize failed: %w (stderr: %s)", err, errText)
		}
		return nil, fmt.Errorf("mcp stdio initialize failed: %w", err)
	}
	out, callErr := fn(sess)
	closeOnce()
	if callErr != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText != "" {
			return nil, fmt.Errorf("mcp stdio call failed: %w (stderr: %s)", callErr, errText)
		}
		return nil, callErr
	}
	return out, nil
}

func (s *mcpStdioSession) initialize() error {
	_, err := s.request("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "agent-daemon",
			"version": "0.1.0",
		},
	})
	if err != nil {
		return err
	}
	if err := s.notify("notifications/initialized", map[string]any{}); err != nil {
		return err
	}
	return nil
}

func (s *mcpStdioSession) notify(method string, params map[string]any) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return writeMCPFrame(s.stdin, b)
}

func (s *mcpStdioSession) request(method string, params map[string]any) (map[string]any, error) {
	id := s.nextID
	s.nextID++
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if err := writeMCPFrame(s.stdin, b); err != nil {
		return nil, err
	}
	for {
		frame, err := readMCPFrame(s.stdout)
		if err != nil {
			return nil, err
		}
		var envelope map[string]any
		if err := json.Unmarshal(frame, &envelope); err != nil {
			continue
		}
		respID, hasID := envelope["id"]
		if !hasID {
			continue
		}
		if !mcpIDMatches(respID, id) {
			continue
		}
		if errObj, ok := envelope["error"].(map[string]any); ok {
			return nil, fmt.Errorf("mcp rpc error: %s", strMap(errObj, "message"))
		}
		result, _ := envelope["result"].(map[string]any)
		if result == nil {
			result = map[string]any{}
		}
		return result, nil
	}
}

func writeMCPFrame(w io.Writer, payload []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func readMCPFrame(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "content-length:") {
			raw := strings.TrimSpace(strings.TrimPrefix(lower, "content-length:"))
			n, err := strconv.Atoi(raw)
			if err != nil {
				return nil, err
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func mcpIDMatches(got any, expected int) bool {
	switch v := got.(type) {
	case float64:
		return int(v) == expected
	case int:
		return v == expected
	case string:
		return v == strconv.Itoa(expected)
	default:
		return false
	}
}

func mapAnyMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func parseMCPCallSSE(body io.Reader) (map[string]any, error) {
	scanner := bufio.NewScanner(body)
	// Allow larger event payloads.
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	var dataLines []string
	final := map[string]any{}
	textChunks := make([]string, 0)
	flushEvent := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		payload = strings.TrimSpace(payload)
		if payload == "" {
			return nil
		}
		if payload == "[DONE]" {
			return nil
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			textChunks = append(textChunks, payload)
			return nil
		}
		if errObj, ok := event["error"].(map[string]any); ok {
			return fmt.Errorf("mcp stream error: %s", strMap(errObj, "message"))
		}
		// Common MCP wrapper style.
		if result, ok := event["result"].(map[string]any); ok {
			final = result
			return nil
		}
		// Some servers emit direct result payloads as stream events.
		if structured, ok := event["structuredContent"].(map[string]any); ok {
			final = structured
			return nil
		}
		if content, ok := event["content"]; ok {
			final["content"] = content
		}
		// Keep latest event when no explicit result structure exists.
		final["last_event"] = event
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flushEvent(); err != nil {
				return nil, err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := flushEvent(); err != nil {
		return nil, err
	}
	if len(final) == 0 && len(textChunks) > 0 {
		return map[string]any{"content": strings.Join(textChunks, "\n")}, nil
	}
	if len(final) == 0 {
		return map[string]any{}, nil
	}
	return final, nil
}

func (c *MCPClient) injectAuthHeader(ctx context.Context, req *http.Request) error {
	token, err := c.oauthAccessToken(ctx)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func (c *MCPClient) oauthAccessToken(ctx context.Context) (string, error) {
	if strings.TrimSpace(c.OAuth.TokenURL) == "" {
		return "", nil
	}
	now := time.Now()
	c.mu.Lock()
	if c.cachedAccessToken != "" && now.Before(c.cachedExpiry.Add(-10*time.Second)) {
		token := c.cachedAccessToken
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	if strings.TrimSpace(c.OAuth.Scopes) != "" {
		form.Set("scope", c.OAuth.Scopes)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.OAuth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.OAuth.ClientID, c.OAuth.ClientSecret)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("mcp oauth token request failed (%d): %s", resp.StatusCode, string(body))
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", fmt.Errorf("mcp oauth token response missing access_token")
	}
	if tokenResp.ExpiresIn <= 0 {
		tokenResp.ExpiresIn = 3600
	}
	expiry := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	c.mu.Lock()
	c.cachedAccessToken = tokenResp.AccessToken
	c.cachedExpiry = expiry
	c.mu.Unlock()
	return tokenResp.AccessToken, nil
}
