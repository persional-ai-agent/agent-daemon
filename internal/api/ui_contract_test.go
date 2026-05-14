package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

func decodeJSONMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	return out
}

func assertUIContractHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Header().Get("X-Agent-UI-API-Version") != "v1" {
		t.Fatalf("missing UI version header: %v", rec.Header())
	}
	if rec.Header().Get("X-Agent-UI-API-Compat") == "" {
		t.Fatalf("missing UI compat header: %v", rec.Header())
	}
}

func assertErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder, expectedCode string) {
	t.Helper()
	assertUIContractHeaders(t, rec)
	out := decodeJSONMap(t, rec)
	if out["ok"] != false {
		t.Fatalf("expected ok=false, got %+v", out)
	}
	errObj, _ := out["error"].(map[string]any)
	if errObj["code"] != expectedCode {
		t.Fatalf("error.code=%v want %s body=%s", errObj["code"], expectedCode, rec.Body.String())
	}
	if errObj["message"] == "" {
		t.Fatalf("missing error message: %+v", out)
	}
}

func TestUIContractSuccessEnvelopeAndHeaders(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(apiTestTool{
		name: "x",
		call: func(context.Context, map[string]any, tools.ToolContext) (map[string]any, error) {
			return map[string]any{"success": true}, nil
		},
	})
	srv := &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "ok"}},
			Registry:     reg,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
		ConfigSnapshotFn: func() map[string]any { return map[string]any{"k": "v"} },
		GatewayStatusFn:  func() map[string]any { return map[string]any{"enabled": true} },
		SkillListFn: func() ([]map[string]any, error) {
			return []map[string]any{{"name": "skill-a", "path": "skills/skill-a/SKILL.md"}}, nil
		},
		SkillsReloadFn: func() (map[string]any, error) {
			return map[string]any{"success": true, "count": 1}, nil
		},
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "tools", method: http.MethodGet, path: "/v1/ui/tools"},
		{name: "tool_schema", method: http.MethodGet, path: "/v1/ui/tools/x/schema"},
		{name: "sessions", method: http.MethodGet, path: "/v1/ui/sessions?limit=1"},
		{name: "session_detail", method: http.MethodGet, path: "/v1/ui/sessions/s1?offset=0&limit=1"},
		{name: "config", method: http.MethodGet, path: "/v1/ui/config"},
		{name: "gateway", method: http.MethodGet, path: "/v1/ui/gateway/status"},
		{name: "skills", method: http.MethodGet, path: "/v1/ui/skills"},
		{name: "skills_reload", method: http.MethodPost, path: "/v1/ui/skills/reload"},
		{name: "agents", method: http.MethodGet, path: "/v1/ui/agents"},
		{name: "agents_history", method: http.MethodGet, path: "/v1/ui/agents/history?limit=2"},
		{name: "complete_slash", method: http.MethodPost, path: "/v1/ui/complete/slash", body: `{"text":"/to"}`},
		{name: "complete_path", method: http.MethodPost, path: "/v1/ui/complete/path", body: `{"path":"./"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			assertUIContractHeaders(t, rec)
			out := decodeJSONMap(t, rec)
			if out["api_version"] != "v1" {
				t.Fatalf("api_version=%v", out["api_version"])
			}
			if out["compat"] == "" {
				t.Fatalf("compat missing: %+v", out)
			}
			if out["ok"] != true {
				t.Fatalf("ok missing/false: %+v", out)
			}
		})
	}
}

func TestUIContractErrorEnvelope(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/v1/ui/tools", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorEnvelope(t, rec, "engine_unavailable")
}

func TestUIAgentsInterruptErrorEnvelope(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/v1/ui/agents/interrupt", bytes.NewBufferString(`{"session_id":"s1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertUIContractHeaders(t, rec)
	out := decodeJSONMap(t, rec)
	if out["ok"] != false {
		t.Fatalf("expected ok=false, got %+v", out)
	}
}
