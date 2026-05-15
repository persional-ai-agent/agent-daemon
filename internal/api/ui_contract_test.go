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
	reg.Register(apiTestTool{
		name: "skill_search",
		call: func(context.Context, map[string]any, tools.ToolContext) (map[string]any, error) {
			return map[string]any{
				"success": true,
				"count":   1,
				"skills":  []map[string]any{{"name": "skill-remote"}},
			}, nil
		},
	})
	reg.Register(apiTestTool{
		name: "skill_manage",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{
				"success": true,
				"action":  args["action"],
				"source":  args["source"],
				"name":    args["name"],
			}, nil
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
		VoiceStatusFn: func() (map[string]any, error) {
			return map[string]any{"enabled": true, "recording": false, "tts": true}, nil
		},
		VoiceToggleFn: func(action string) (map[string]any, error) {
			return map[string]any{"action": action, "enabled": true, "recording": false, "tts": true}, nil
		},
		VoiceRecordFn: func(action string) (map[string]any, error) {
			return map[string]any{"action": action, "enabled": true, "recording": action == "start", "tts": true}, nil
		},
		VoiceTTSFn: func(text string) (map[string]any, error) {
			return map[string]any{"spoken": true, "text": text, "length": len(text)}, nil
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
		{name: "session_branch", method: http.MethodPost, path: "/v1/ui/sessions/branch", body: `{"session_id":"s1","new_session_id":"s-branch","last_n":1}`},
		{name: "session_resume", method: http.MethodPost, path: "/v1/ui/sessions/resume", body: `{"session_id":"s1","turn_id":"t-1"}`},
		{name: "session_compress", method: http.MethodPost, path: "/v1/ui/sessions/compress", body: `{"session_id":"s1","keep_last_n":20}`},
		{name: "session_undo", method: http.MethodPost, path: "/v1/ui/sessions/undo", body: `{"session_id":"s1"}`},
		{name: "session_replay", method: http.MethodPost, path: "/v1/ui/sessions/replay", body: `{"session_id":"s1","offset":0,"limit":2}`},
		{name: "config", method: http.MethodGet, path: "/v1/ui/config"},
		{name: "gateway", method: http.MethodGet, path: "/v1/ui/gateway/status"},
		{name: "gateway_continuity_get", method: http.MethodGet, path: "/v1/ui/gateway/continuity"},
		{name: "gateway_continuity_set", method: http.MethodPost, path: "/v1/ui/gateway/continuity", body: `{"mode":"user_name"}`},
		{name: "gateway_identity_set", method: http.MethodPost, path: "/v1/ui/gateway/identity", body: `{"platform":"telegram","user_id":"u1","global_id":"gid-1"}`},
		{name: "gateway_identity_get", method: http.MethodGet, path: "/v1/ui/gateway/identity?platform=telegram&user_id=u1"},
		{name: "gateway_identity_delete", method: http.MethodDelete, path: "/v1/ui/gateway/identity", body: `{"platform":"telegram","user_id":"u1"}`},
		{name: "gateway_session_resolve", method: http.MethodGet, path: "/v1/ui/gateway/session/resolve?platform=telegram&chat_type=group&chat_id=1001&user_id=u1&user_name=Alice"},
		{name: "gateway_diagnostics", method: http.MethodGet, path: "/v1/ui/gateway/diagnostics"},
		{name: "targets", method: http.MethodGet, path: "/v1/ui/targets?platform=telegram"},
		{name: "targets_home", method: http.MethodPost, path: "/v1/ui/targets/home", body: `{"target":"telegram:1001"}`},
		{name: "skills", method: http.MethodGet, path: "/v1/ui/skills"},
		{name: "skills_reload", method: http.MethodPost, path: "/v1/ui/skills/reload"},
		{name: "skills_search", method: http.MethodPost, path: "/v1/ui/skills/search", body: `{"query":"code review","repo":"anthropics/skills"}`},
		{name: "skills_sync", method: http.MethodPost, path: "/v1/ui/skills/sync", body: `{"name":"skill-sync","source":"url","url":"https://example.com/SKILL.md"}`},
		{name: "voice_status", method: http.MethodGet, path: "/v1/ui/voice/status"},
		{name: "voice_toggle", method: http.MethodPost, path: "/v1/ui/voice/toggle", body: `{"action":"on"}`},
		{name: "voice_record", method: http.MethodPost, path: "/v1/ui/voice/record", body: `{"action":"start"}`},
		{name: "voice_tts", method: http.MethodPost, path: "/v1/ui/voice/tts", body: `{"text":"hello"}`},
		{name: "agents", method: http.MethodGet, path: "/v1/ui/agents"},
		{name: "agents_active", method: http.MethodGet, path: "/v1/ui/agents/active"},
		{name: "agents_detail", method: http.MethodGet, path: "/v1/ui/agents/detail?session_id=s1"},
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
