package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

func normalizeJSONMap(in map[string]any) map[string]any {
	bs, _ := json.Marshal(in)
	out := map[string]any{}
	_ = json.Unmarshal(bs, &out)
	return out
}

func loadContractFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	path := filepath.Join("testdata", "contracts", name+".json")
	bs, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("decode fixture %s: %v", path, err)
	}
	return out
}

func assertContractSnapshot(t *testing.T, name string, got map[string]any) {
	t.Helper()
	want := normalizeJSONMap(loadContractFixture(t, name))
	got = normalizeJSONMap(got)
	if !reflect.DeepEqual(want, got) {
		wb, _ := json.MarshalIndent(want, "", "  ")
		gb, _ := json.MarshalIndent(got, "", "  ")
		t.Fatalf("snapshot mismatch: %s\nwant=%s\ngot=%s", name, string(wb), string(gb))
	}
}

func TestContractSnapshotUIToolsSuccess(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(apiTestTool{
		name: "x",
		call: func(_ context.Context, _ map[string]any, _ tools.ToolContext) (map[string]any, error) {
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
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ui/tools", nil)
	srv.Handler().ServeHTTP(rec, req)
	body := decodeJSONMap(t, rec)
	assertContractSnapshot(t, "ui_tools_success", map[string]any{
		"status": rec.Code,
		"headers": map[string]any{
			"X-Agent-UI-API-Version": rec.Header().Get("X-Agent-UI-API-Version"),
			"X-Agent-UI-API-Compat":  rec.Header().Get("X-Agent-UI-API-Compat"),
		},
		"body": body,
	})
}

func TestContractSnapshotUIToolsMethodNotAllowed(t *testing.T) {
	srv := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/ui/tools", nil)
	srv.Handler().ServeHTTP(rec, req)
	body := decodeJSONMap(t, rec)
	assertContractSnapshot(t, "ui_tools_method_not_allowed", map[string]any{
		"status": rec.Code,
		"headers": map[string]any{
			"X-Agent-UI-API-Version": rec.Header().Get("X-Agent-UI-API-Version"),
			"X-Agent-UI-API-Compat":  rec.Header().Get("X-Agent-UI-API-Compat"),
		},
		"body": body,
	})
}

func TestContractSnapshotChatSuccess(t *testing.T) {
	srv := &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "ok"}},
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewBufferString(`{"session_id":"s1","message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.Handler().ServeHTTP(rec, req)
	out := decodeJSONMap(t, rec)
	result, _ := out["result"].(map[string]any)
	summary, _ := result["summary"].(map[string]any)
	assertContractSnapshot(t, "chat_success", map[string]any{
		"status": rec.Code,
		"headers": map[string]any{
			"X-Agent-UI-API-Version": rec.Header().Get("X-Agent-UI-API-Version"),
			"X-Agent-UI-API-Compat":  rec.Header().Get("X-Agent-UI-API-Compat"),
		},
		"body": map[string]any{
			"ok":          out["ok"],
			"api_version": out["api_version"],
			"compat":      out["compat"],
			"result": map[string]any{
				"session_id":         result["session_id"],
				"final_response":     result["final_response"],
				"turns_used":         result["turns_used"],
				"finished_naturally": result["finished_naturally"],
				"summary":            summary,
			},
			"session_id":         out["session_id"],
			"final_response":     out["final_response"],
			"turns_used":         out["turns_used"],
			"finished_naturally": out["finished_naturally"],
		},
	})
}

func TestContractSnapshotCancelSuccess(t *testing.T) {
	srv := &Server{}
	srv.mu.Lock()
	srv.active = map[string]activeRun{
		"s-cancel": {token: "t1", cancel: func() {}},
	}
	srv.mu.Unlock()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/cancel", bytes.NewBufferString(`{"session_id":"s-cancel"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.Handler().ServeHTTP(rec, req)
	out := decodeJSONMap(t, rec)
	result, _ := out["result"].(map[string]any)
	assertContractSnapshot(t, "chat_cancel_success", map[string]any{
		"status": rec.Code,
		"headers": map[string]any{
			"X-Agent-UI-API-Version": rec.Header().Get("X-Agent-UI-API-Version"),
			"X-Agent-UI-API-Compat":  rec.Header().Get("X-Agent-UI-API-Compat"),
		},
		"body": map[string]any{
			"ok":          out["ok"],
			"api_version": out["api_version"],
			"compat":      out["compat"],
			"result":      result,
			"session_id":  out["session_id"],
			"cancelled":   out["cancelled"],
		},
	})
}
