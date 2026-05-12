package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Minimal Chrome DevTools Protocol (CDP) client.
// Used as an optional upgrade path for browser_* tools when BROWSER_CDP_URL is configured.

type cdpClient struct {
	ws  *websocket.Conn
	mu  sync.Mutex
	nextID int64
	pending map[int64]chan cdpResponse
	closed  bool
	onEvent func(method string, params json.RawMessage)
}

type cdpResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func dialCDP(ctx context.Context, endpoint string) (*cdpClient, error) {
	wsURL, err := resolveCDPWebSocket(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}
	c := &cdpClient{
		ws:      conn,
		nextID:  1,
		pending: make(map[int64]chan cdpResponse),
	}
	go c.recvLoop()
	return c, nil
}

func (c *cdpClient) close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	ws := c.ws
	c.ws = nil
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.mu.Unlock()
	if ws != nil {
		return ws.Close()
	}
	return nil
}

func (c *cdpClient) recvLoop() {
	for {
		c.mu.Lock()
		ws := c.ws
		closed := c.closed
		c.mu.Unlock()
		if ws == nil || closed {
			return
		}
		_, data, err := ws.ReadMessage()
		if err != nil {
			_ = c.close()
			return
		}
		// Response or event.
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}
		if rawID, ok := envelope["id"]; ok && len(rawID) > 0 {
			var resp cdpResponse
			if err := json.Unmarshal(data, &resp); err != nil || resp.ID == 0 {
				continue
			}
			c.mu.Lock()
			ch := c.pending[resp.ID]
			c.mu.Unlock()
			if ch != nil {
				select {
				case ch <- resp:
				default:
				}
			}
			continue
		}
		if rawMethod, ok := envelope["method"]; ok && len(rawMethod) > 0 {
			var method string
			_ = json.Unmarshal(rawMethod, &method)
			params := envelope["params"]
			c.mu.Lock()
			cb := c.onEvent
			c.mu.Unlock()
			if cb != nil && strings.TrimSpace(method) != "" {
				cb(method, params)
			}
			continue
		}
	}
}

func (c *cdpClient) call(ctx context.Context, method string, params any, out any) error {
	c.mu.Lock()
	if c.closed || c.ws == nil {
		c.mu.Unlock()
		return errors.New("cdp closed")
	}
	id := c.nextID
	c.nextID++
	ch := make(chan cdpResponse, 1)
	c.pending[id] = ch
	ws := c.ws
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	payload := map[string]any{"id": id, "method": method}
	if params != nil {
		payload["params"] = params
	}
	bs, _ := json.Marshal(payload)
	if err := ws.WriteMessage(websocket.TextMessage, bs); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp, ok := <-ch:
		if !ok {
			return errors.New("cdp connection closed")
		}
		if resp.Error != nil {
			return fmt.Errorf("cdp %s error: %s", method, resp.Error.Message)
		}
		if out == nil {
			return nil
		}
		if len(resp.Result) == 0 {
			return nil
		}
		return json.Unmarshal(resp.Result, out)
	case <-time.After(20 * time.Second):
		return errors.New("cdp timeout")
	}
}

func resolveCDPWebSocket(ctx context.Context, endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errors.New("BROWSER_CDP_URL required")
	}
	// If it's already a websocket URL, use as-is.
	if strings.HasPrefix(endpoint, "ws://") || strings.HasPrefix(endpoint, "wss://") {
		return endpoint, nil
	}

	// Accept a Chrome remote-debugging HTTP base, e.g. http://127.0.0.1:9222
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" {
		// Try default to http.
		u, err = url.Parse("http://" + endpoint)
		if err != nil {
			return "", err
		}
	}

	base := strings.TrimRight(u.String(), "/")

	// Prefer /json/version for browser-level ws endpoint.
	verURL := base + "/json/version"
	wsURL, err := fetchWSURL(ctx, verURL, "webSocketDebuggerUrl")
	if err == nil && strings.TrimSpace(wsURL) != "" {
		return wsURL, nil
	}

	// Fall back: create a new target and use its ws URL.
	newURL := base + "/json/new"
	wsURL, err = fetchWSURL(ctx, newURL, "webSocketDebuggerUrl")
	if err == nil && strings.TrimSpace(wsURL) != "" {
		return wsURL, nil
	}

	return "", errors.New("could not resolve CDP websocket URL from endpoint; provide a ws://... target URL")
}

func fetchWSURL(ctx context.Context, u string, key string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return "", err
	}
	if v, ok := m[key].(string); ok {
		return v, nil
	}
	return "", errors.New("missing ws url field")
}

func cdpEnabled() bool {
	if strings.TrimSpace(os.Getenv("BROWSER_CDP_URL")) == "" {
		return false
	}
	// Keep it easy to disable without editing config.
	if strings.EqualFold(strings.TrimSpace(os.Getenv("BROWSER_CDP_ENABLED")), "false") {
		return false
	}
	return true
}
