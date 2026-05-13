package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

func TestChatContractErrorEnvelope(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/v1/chat", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorEnvelope(t, rec, "engine_unavailable")
}

func TestCancelContractMethodNotAllowedEnvelope(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/cancel", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorEnvelope(t, rec, "method_not_allowed")
}

func TestChatContractSuccessEnvelopeAndCompatFields(t *testing.T) {
	srv := &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "ok"}},
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewBufferString(`{"session_id":"s1","message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertUIContractHeaders(t, rec)

	out := decodeJSONMap(t, rec)
	if out["ok"] != true {
		t.Fatalf("ok=%v body=%s", out["ok"], rec.Body.String())
	}
	result, _ := out["result"].(map[string]any)
	if result["session_id"] != "s1" || result["final_response"] != "ok" {
		t.Fatalf("unexpected result=%+v", result)
	}
	// backward-compat top-level fields
	if out["session_id"] != "s1" || out["final_response"] != "ok" {
		t.Fatalf("compat fields missing body=%s", rec.Body.String())
	}
}

func TestCancelContractSuccessEnvelopeAndCompatFields(t *testing.T) {
	srv := &Server{}
	srv.mu.Lock()
	srv.active = map[string]activeRun{
		"s-cancel": {token: "t1", cancel: func() {}},
	}
	srv.mu.Unlock()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/cancel", bytes.NewBufferString(`{"session_id":"s-cancel"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertUIContractHeaders(t, rec)

	out := decodeJSONMap(t, rec)
	if out["ok"] != true {
		t.Fatalf("ok=%v body=%s", out["ok"], rec.Body.String())
	}
	result, _ := out["result"].(map[string]any)
	if result["session_id"] != "s-cancel" || result["cancelled"] != true {
		t.Fatalf("unexpected result=%+v", result)
	}
	// backward-compat top-level fields
	if out["session_id"] != "s-cancel" || out["cancelled"] != true {
		t.Fatalf("compat fields missing body=%s", rec.Body.String())
	}
}
