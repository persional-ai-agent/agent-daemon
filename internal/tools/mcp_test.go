package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegisterMCPToolsAndDispatch(t *testing.T) {
	var callPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tools":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tools": []map[string]any{
					{
						"name":        "mcp_echo",
						"description": "echo through mcp",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"text": map[string]any{"type": "string"},
							},
						},
					},
				},
			})
		case "/call":
			if err := json.NewDecoder(r.Body).Decode(&callPayload); err != nil {
				t.Fatalf("decode call payload failed: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					"echo": "ok",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	registry := NewRegistry()
	client := NewMCPClient(srv.URL, 5*time.Second)
	names, err := RegisterMCPTools(context.Background(), registry, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "mcp_echo" {
		t.Fatalf("unexpected mcp names: %+v", names)
	}

	raw := registry.Dispatch(context.Background(), "mcp_echo", map[string]any{"text": "hello"}, ToolContext{
		SessionID: "s1",
		Workdir:   "/tmp/work",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	if out["echo"] != "ok" {
		t.Fatalf("unexpected mcp result: %+v", out)
	}
	if callPayload["name"] != "mcp_echo" {
		t.Fatalf("unexpected call payload: %+v", callPayload)
	}
	ctxMap, _ := callPayload["context"].(map[string]any)
	if ctxMap["session_id"] != "s1" {
		t.Fatalf("missing session context in payload: %+v", callPayload)
	}
}

func TestRegisterMCPToolsHandlesDiscoveryFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tools" {
			http.Error(w, "unavailable", http.StatusBadGateway)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := RegisterMCPTools(context.Background(), NewRegistry(), NewMCPClient(srv.URL, 5*time.Second))
	if err == nil {
		t.Fatal("expected discovery error")
	}
}

func TestRegisterMCPToolsAndDispatchWithOAuthClientCredentials(t *testing.T) {
	var tokenRequests int32
	var toolsAuthHeader string
	var callAuthHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			atomic.AddInt32(&tokenRequests, 1)
			id, secret, ok := r.BasicAuth()
			if !ok || id != "mcp-client" || secret != "mcp-secret" {
				http.Error(w, "invalid client", http.StatusUnauthorized)
				return
			}
			_ = r.ParseForm()
			if r.FormValue("grant_type") != "client_credentials" {
				http.Error(w, "invalid grant_type", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "oauth-test-token",
				"expires_in":   3600,
			})
		case "/tools":
			toolsAuthHeader = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tools": []map[string]any{
					{
						"name":        "mcp_oauth_echo",
						"description": "echo through oauth mcp",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"text": map[string]any{"type": "string"},
							},
						},
					},
				},
			})
		case "/call":
			callAuthHeader = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					"echo": "oauth-ok",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	registry := NewRegistry()
	client := NewMCPClient(srv.URL, 5*time.Second)
	client.ConfigureOAuthClientCredentials(MCPOAuthConfig{
		TokenURL:     srv.URL + "/oauth/token",
		ClientID:     "mcp-client",
		ClientSecret: "mcp-secret",
		Scopes:       "mcp.read mcp.call",
	})

	names, err := RegisterMCPTools(context.Background(), registry, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "mcp_oauth_echo" {
		t.Fatalf("unexpected mcp names: %+v", names)
	}
	if toolsAuthHeader != "Bearer oauth-test-token" {
		t.Fatalf("unexpected tools auth header: %q", toolsAuthHeader)
	}

	raw := registry.Dispatch(context.Background(), "mcp_oauth_echo", map[string]any{"text": "hello"}, ToolContext{
		SessionID: "s-oauth",
		Workdir:   "/tmp/work",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	if out["echo"] != "oauth-ok" {
		t.Fatalf("unexpected mcp result: %+v", out)
	}
	if callAuthHeader != "Bearer oauth-test-token" {
		t.Fatalf("unexpected call auth header: %q", callAuthHeader)
	}
	if atomic.LoadInt32(&tokenRequests) != 1 {
		t.Fatalf("expected cached oauth token reused, token_requests=%d", tokenRequests)
	}
}

func TestRegisterMCPToolsAndDispatchSSECall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tools":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tools": []map[string]any{
					{
						"name":        "mcp_stream_echo",
						"description": "stream echo through mcp",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"text": map[string]any{"type": "string"},
							},
						},
					},
				},
			})
		case "/call":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"result\":{\"echo\":\"sse-ok\"}}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	registry := NewRegistry()
	client := NewMCPClient(srv.URL, 5*time.Second)
	names, err := RegisterMCPTools(context.Background(), registry, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "mcp_stream_echo" {
		t.Fatalf("unexpected mcp names: %+v", names)
	}
	raw := registry.Dispatch(context.Background(), "mcp_stream_echo", map[string]any{"text": "hello"}, ToolContext{
		SessionID: "s-sse",
		Workdir:   "/tmp/work",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	if out["echo"] != "sse-ok" {
		t.Fatalf("unexpected mcp sse result: %+v", out)
	}
}

func TestRegisterMCPToolsAndDispatchStdio(t *testing.T) {
	if strings.TrimSpace(os.Getenv("GO_WANT_HELPER_PROCESS")) == "1" {
		t.Skip("running as helper process")
	}
	cmd := fmt.Sprintf("GO_WANT_HELPER_PROCESS=1 %s -test.run TestMCPStdioHelperProcess --", os.Args[0])
	registry := NewRegistry()
	client := NewMCPStdioClient(cmd, 5*time.Second)
	names, err := RegisterMCPTools(context.Background(), registry, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "mcp_echo" {
		t.Fatalf("unexpected mcp names: %+v", names)
	}

	raw := registry.Dispatch(context.Background(), "mcp_echo", map[string]any{"text": "hello"}, ToolContext{
		SessionID: "s-stdio",
		Workdir:   "/tmp/work",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	if out["echo"] != "stdio-ok" {
		t.Fatalf("unexpected mcp stdio result: %+v", out)
	}
}

func TestMCPStdioHelperProcess(t *testing.T) {
	if strings.TrimSpace(os.Getenv("GO_WANT_HELPER_PROCESS")) != "1" {
		t.Skip("helper process only")
	}
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout
	for {
		frame, err := readMCPTestFrame(reader)
		if err != nil {
			if err == io.EOF {
				os.Exit(0)
			}
			_ = writeMCPTestFrame(writer, map[string]any{
				"jsonrpc": "2.0",
				"id":      0,
				"error":   map[string]any{"message": err.Error()},
			})
			os.Exit(1)
		}
		var req map[string]any
		if err := json.Unmarshal(frame, &req); err != nil {
			continue
		}
		method, _ := req["method"].(string)
		id, hasID := req["id"]
		if !hasID {
			continue
		}
		switch method {
		case "initialize":
			_ = writeMCPTestFrame(writer, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{},
					"serverInfo":      map[string]any{"name": "helper", "version": "0.0.1"},
				},
			})
		case "tools/list":
			_ = writeMCPTestFrame(writer, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "mcp_echo",
							"description": "echo from stdio helper",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"text": map[string]any{"type": "string"},
								},
							},
						},
					},
				},
			})
		case "tools/call":
			_ = writeMCPTestFrame(writer, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"structuredContent": map[string]any{
						"echo": "stdio-ok",
					},
				},
			})
		default:
			_ = writeMCPTestFrame(writer, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error":   map[string]any{"message": "unsupported method"},
			})
		}
	}
}

func writeMCPTestFrame(w io.Writer, payload map[string]any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(b), string(b))
	return err
}

func readMCPTestFrame(r *bufio.Reader) ([]byte, error) {
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
			var n int
			if _, err := fmt.Sscanf(lower, "content-length: %d", &n); err != nil {
				return nil, err
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing content-length")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func TestRegisterMCPToolsAndDispatchSSECallWithCallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tools":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tools": []map[string]any{
					{
						"name":        "mcp_progress_echo",
						"description": "progress echo through mcp",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"text": map[string]any{"type": "string"},
							},
						},
					},
				},
			})
		case "/call":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"type\":\"progress\",\"percent\":50}\n\n")
			_, _ = fmt.Fprint(w, "data: {\"type\":\"progress\",\"percent\":100}\n\n")
			_, _ = fmt.Fprint(w, "data: {\"result\":{\"echo\":\"stream-callback-ok\"}}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	registry := NewRegistry()
	client := NewMCPClient(srv.URL, 5*time.Second)
	names, err := RegisterMCPTools(context.Background(), registry, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "mcp_progress_echo" {
		t.Fatalf("unexpected mcp names: %+v", names)
	}

	var events []struct {
		eventType string
		data      map[string]any
	}
	raw := registry.Dispatch(context.Background(), "mcp_progress_echo", map[string]any{"text": "hello"}, ToolContext{
		SessionID: "s-sse-cb",
		Workdir:   "/tmp/work",
		ToolEventSink: func(eventType string, data map[string]any) {
			events = append(events, struct {
				eventType string
				data      map[string]any
			}{eventType, data})
		},
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	if out["echo"] != "stream-callback-ok" {
		t.Fatalf("unexpected mcp sse callback result: %+v", out)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 intermediate events, got %d", len(events))
	}
	if events[0].eventType != "progress" {
		t.Fatalf("expected first event type=progress, got %s", events[0].eventType)
	}
	if events[0].data["percent"] != float64(50) {
		t.Fatalf("expected first event percent=50, got %v", events[0].data["percent"])
	}
	if events[1].eventType != "progress" {
		t.Fatalf("expected second event type=progress, got %s", events[1].eventType)
	}
	if events[2].eventType != "mcp_event" {
		t.Fatalf("expected third event type=mcp_event (no type field), got %s", events[2].eventType)
	}
}

func TestRegisterMCPToolsSSECallbackNoSinkFallsBackToAggregated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tools":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tools": []map[string]any{
					{
						"name":        "mcp_no_sink",
						"description": "no sink echo",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"text": map[string]any{"type": "string"},
							},
						},
					},
				},
			})
		case "/call":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"type\":\"progress\",\"percent\":50}\n\n")
			_, _ = fmt.Fprint(w, "data: {\"result\":{\"echo\":\"aggregated-ok\"}}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	registry := NewRegistry()
	client := NewMCPClient(srv.URL, 5*time.Second)
	RegisterMCPTools(context.Background(), registry, client)

	raw := registry.Dispatch(context.Background(), "mcp_no_sink", map[string]any{"text": "hello"}, ToolContext{
		SessionID: "s-no-sink",
		Workdir:   "/tmp/work",
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	if out["echo"] != "aggregated-ok" {
		t.Fatalf("unexpected aggregated result: %+v", out)
	}
}

func TestMCPClientOAuthRefreshToken(t *testing.T) {
	var tokenRequests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			atomic.AddInt32(&tokenRequests, 1)
			_ = r.ParseForm()
			grantType := r.FormValue("grant_type")
			switch grantType {
			case "client_credentials":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token":  "initial-token",
					"refresh_token": "refresh-1",
					"expires_in":    1,
				})
			case "refresh_token":
				rt := r.FormValue("refresh_token")
				if rt != "refresh-1" {
					http.Error(w, "invalid refresh_token", http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token":  "refreshed-token",
					"refresh_token": "refresh-2",
					"expires_in":    3600,
				})
			default:
				http.Error(w, "unsupported grant_type", http.StatusBadRequest)
			}
		case "/tools":
			_ = json.NewEncoder(w).Encode(map[string]any{"tools": []map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewMCPClient(srv.URL, 5*time.Second)
	client.ConfigureOAuthClientCredentials(MCPOAuthConfig{
		TokenURL:     srv.URL + "/oauth/token",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})

	token1, err := client.oauthAccessToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token1 != "initial-token" {
		t.Fatalf("expected initial-token, got %s", token1)
	}

	time.Sleep(2 * time.Second)

	token2, err := client.oauthAccessToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token2 != "refreshed-token" {
		t.Fatalf("expected refreshed-token, got %s", token2)
	}
	if atomic.LoadInt32(&tokenRequests) != 2 {
		t.Fatalf("expected 2 token requests (initial + refresh), got %d", tokenRequests)
	}
}

func TestMCPClientOAuthAuthCodeExchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			_ = r.ParseForm()
			grantType := r.FormValue("grant_type")
			if grantType == "authorization_code" {
				code := r.FormValue("code")
				if code != "test-auth-code" {
					http.Error(w, "invalid code", http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token":  "auth-code-token",
					"refresh_token": "auth-refresh-1",
					"expires_in":    3600,
				})
			} else if grantType == "refresh_token" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token":  "refreshed-from-auth",
					"refresh_token": "auth-refresh-2",
					"expires_in":    3600,
				})
			} else {
				http.Error(w, "unsupported grant_type", http.StatusBadRequest)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewMCPClient(srv.URL, 5*time.Second)
	client.ConfigureOAuthAuthCode(MCPOAuthConfig{
		TokenURL:    srv.URL + "/oauth/token",
		AuthURL:     "https://example.com/oauth/authorize",
		RedirectURL: "http://localhost:9876/callback",
		ClientID:    "test-client",
		Scopes:      "read write",
	})

	token, err := client.ExchangeAuthCode(context.Background(), "test-auth-code")
	if err != nil {
		t.Fatal(err)
	}
	if token != "auth-code-token" {
		t.Fatalf("expected auth-code-token, got %s", token)
	}
	if client.cachedRefreshToken != "auth-refresh-1" {
		t.Fatalf("expected cached refresh token auth-refresh-1, got %s", client.cachedRefreshToken)
	}
}

func TestMCPClientBuildAuthURL(t *testing.T) {
	client := NewMCPClient("https://mcp.example.com", 5*time.Second)
	client.ConfigureOAuthAuthCode(MCPOAuthConfig{
		AuthURL:     "https://auth.example.com/authorize",
		RedirectURL: "http://localhost:9876/callback",
		ClientID:    "my-client",
		Scopes:      "read write",
	})
	authURL := client.BuildAuthURL("test-state")
	if !strings.Contains(authURL, "client_id=my-client") {
		t.Fatalf("expected client_id in auth URL, got %s", authURL)
	}
	if !strings.Contains(authURL, "response_type=code") {
		t.Fatalf("expected response_type=code in auth URL, got %s", authURL)
	}
	if !strings.Contains(authURL, "state=test-state") {
		t.Fatalf("expected state in auth URL, got %s", authURL)
	}
	if !strings.Contains(authURL, "scope=read+write") {
		t.Fatalf("expected scope in auth URL, got %s", authURL)
	}
}

type mockTokenStore struct {
	tokens map[string]struct {
		accessToken  string
		refreshToken string
		expiresAt    time.Time
	}
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]struct {
		accessToken  string
		refreshToken string
		expiresAt    time.Time
	})}
}

func (m *mockTokenStore) SaveOAuthToken(provider, accessToken, refreshToken string, expiresAt time.Time) error {
	m.tokens[provider] = struct {
		accessToken  string
		refreshToken string
		expiresAt    time.Time
	}{accessToken, refreshToken, expiresAt}
	return nil
}

func (m *mockTokenStore) LoadOAuthToken(provider string) (string, string, time.Time, error) {
	t, ok := m.tokens[provider]
	if !ok {
		return "", "", time.Time{}, nil
	}
	return t.accessToken, t.refreshToken, t.expiresAt, nil
}

func (m *mockTokenStore) DeleteOAuthToken(provider string) error {
	delete(m.tokens, provider)
	return nil
}

func TestMCPClientOAuthTokenPersistence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "persisted-token",
				"refresh_token": "persisted-refresh",
				"expires_in":    3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	store := newMockTokenStore()
	client := NewMCPClient(srv.URL, 5*time.Second)
	client.TokenStore = store
	client.ConfigureOAuthClientCredentials(MCPOAuthConfig{
		TokenURL:     srv.URL + "/oauth/token",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})

	token, err := client.oauthAccessToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token != "persisted-token" {
		t.Fatalf("expected persisted-token, got %s", token)
	}

	stored, ok := store.tokens[srv.URL]
	if !ok {
		t.Fatal("expected token to be persisted in store")
	}
	if stored.accessToken != "persisted-token" {
		t.Fatalf("expected persisted-token in store, got %s", stored.accessToken)
	}
	if stored.refreshToken != "persisted-refresh" {
		t.Fatalf("expected persisted-refresh in store, got %s", stored.refreshToken)
	}
}
