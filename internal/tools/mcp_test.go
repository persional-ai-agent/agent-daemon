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
