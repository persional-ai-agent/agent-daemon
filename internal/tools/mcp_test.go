package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
