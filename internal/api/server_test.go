package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type fakeModelClient struct {
	response core.Message
}

func (f fakeModelClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	return f.response, nil
}

type scriptedModelClient struct {
	mu        sync.Mutex
	responses []core.Message
}

func (c *scriptedModelClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.responses) == 0 {
		return core.Message{}, nil
	}
	msg := c.responses[0]
	c.responses = c.responses[1:]
	return msg, nil
}

type apiTestTool struct {
	name string
	call func(context.Context, map[string]any, tools.ToolContext) (map[string]any, error)
}

func (t apiTestTool) Name() string { return t.name }

func (t apiTestTool) Schema() core.ToolSchema {
	return core.ToolSchema{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:        t.name,
			Description: "api test tool",
			Parameters:  map[string]any{"type": "object"},
		},
	}
}

func (t apiTestTool) Call(ctx context.Context, args map[string]any, tc tools.ToolContext) (map[string]any, error) {
	return t.call(ctx, args, tc)
}

type blockingModelClient struct {
	started chan struct{}
	once    sync.Once
}

func (b *blockingModelClient) ChatCompletion(ctx context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	return core.Message{}, ctx.Err()
}

type waitOnContextModelClient struct{}

func (waitOnContextModelClient) ChatCompletion(ctx context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	<-ctx.Done()
	return core.Message{}, ctx.Err()
}

type stubSessionStore struct{}

func (s *stubSessionStore) AppendMessage(string, core.Message) error {
	return nil
}

func (s *stubSessionStore) LoadMessages(string, int) ([]core.Message, error) {
	return nil, nil
}

func (s *stubSessionStore) ListRecentSessions(limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	return []map[string]any{
		{"session_id": "s1", "last_seen": "2026-05-13T00:00:00Z"},
		{"session_id": "s2", "last_seen": "2026-05-12T00:00:00Z"},
	}, nil
}

func (s *stubSessionStore) LoadMessagesPage(sessionID string, offset, limit int) ([]core.Message, error) {
	_ = offset
	_ = limit
	return []core.Message{
		{Role: "user", Content: "hello " + sessionID},
		{Role: "assistant", Content: "world"},
	}, nil
}

func (s *stubSessionStore) SessionStats(sessionID string) (map[string]any, error) {
	return map[string]any{
		"session_id":    sessionID,
		"message_count": 2,
	}, nil
}

func TestHandleChatStream(t *testing.T) {
	srv := &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "hello from stream"}},
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", got)
	}

	body := rec.Body.String()
	parts := []string{
		"event: session",
		"event: user_message",
		"event: turn_started",
		"event: assistant_message",
		"event: completed",
		"event: result",
		`"final_response":"hello from stream"`,
	}
	last := -1
	for _, part := range parts {
		idx := strings.Index(body, part)
		if idx < 0 {
			t.Fatalf("expected response to contain %q, body=%s", part, body)
		}
		if idx < last {
			t.Fatalf("expected %q to appear after previous event, body=%s", part, body)
		}
		last = idx
	}
	if !strings.Contains(body, `"session_id":"`) {
		t.Fatalf("expected response to contain session id, body=%s", body)
	}
}

func TestUIEndpoints(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(apiTestTool{
		name: "test_tool",
		call: func(context.Context, map[string]any, tools.ToolContext) (map[string]any, error) {
			return map[string]any{"success": true}, nil
		},
	})
	reg.Register(apiTestTool{
		name: "approval",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{
				"success":     true,
				"action":      args["action"],
				"approval_id": args["approval_id"],
				"approved":    args["approve"],
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
		ConfigSnapshotFn: func() map[string]any {
			return map[string]any{"model_provider": "openai"}
		},
		ConfigUpdateFn: func(key, value string) (map[string]any, error) {
			return map[string]any{"success": true, "key": key, "value": value}, nil
		},
		GatewayStatusFn: func() map[string]any {
			return map[string]any{"enabled": true, "running": false}
		},
		GatewayActionFn: func(action string) (map[string]any, error) {
			return map[string]any{"success": true, "action": action}, nil
		},
		SkillListFn: func() ([]map[string]any, error) {
			return []map[string]any{
				{"name": "skill-a", "path": "skills/skill-a/SKILL.md"},
				{"name": "skill-b", "path": "skills/skill-b/SKILL.md"},
			}, nil
		},
		SkillsReloadFn: func() (map[string]any, error) {
			return map[string]any{"success": true, "count": 2}, nil
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

	t.Run("tools", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/tools", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"test_tool"`) {
			t.Fatalf("expected tool in response: %s", rec.Body.String())
		}
	})

	t.Run("tool_schema", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/tools/test_tool/schema", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"name":"test_tool"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("sessions", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/sessions?limit=2", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"s1"`) {
			t.Fatalf("expected session in response: %s", rec.Body.String())
		}
	})

	t.Run("session_detail", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/sessions/s1?offset=0&limit=2", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"session_id":"s1"`) || !strings.Contains(rec.Body.String(), `"messages"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("config", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/config", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"model_provider":"openai"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("config_set", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/config/set", bytes.NewBufferString(`{"key":"api.type","value":"openai"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"success":true`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("gateway", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/gateway/status", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"enabled":true`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("gateway_action", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/gateway/action", bytes.NewBufferString(`{"action":"enable"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"action":"enable"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("skills", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/skills", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"skill-a"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("skills_reload", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/skills/reload", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"count":2`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("skills_detail_and_manage", func(t *testing.T) {
		workdir := t.TempDir()
		skillDir := filepath.Join(workdir, "skills", "skill-x")
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# old"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		localSrv := &Server{
			Engine: &agent.Engine{
				Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "ok"}},
				Registry:     reg,
				SessionStore: &stubSessionStore{},
				SystemPrompt: agent.DefaultSystemPrompt(),
				Workdir:      workdir,
			},
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/skills/detail?name=skill-x", nil)
		localSrv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "# old") {
			t.Fatalf("detail status=%d body=%s", rec.Code, rec.Body.String())
		}
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/v1/ui/skills/manage", bytes.NewBufferString(`{"action":"patch","name":"skill-x","old_string":"old","new_string":"new"}`))
		req.Header.Set("Content-Type", "application/json")
		localSrv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("manage status=%d body=%s", rec.Code, rec.Body.String())
		}
		bs, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
		if err != nil || !strings.Contains(string(bs), "new") {
			t.Fatalf("patch failed err=%v content=%q", err, string(bs))
		}
	})

	t.Run("voice_status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/voice/status", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"enabled":true`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("voice_toggle", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/voice/toggle", bytes.NewBufferString(`{"action":"on"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"action":"on"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("voice_record", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/voice/record", bytes.NewBufferString(`{"action":"start"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"recording":true`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("voice_tts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/voice/tts", bytes.NewBufferString(`{"text":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"spoken":true`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("agents", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/agents?limit=2", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"agents"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("agents_detail", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/agents/detail?session_id=s1", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"session"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("agents_history", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/ui/agents/history?limit=2", nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"history"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("agents_interrupt_not_found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/agents/interrupt", bytes.NewBufferString(`{"session_id":"missing"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("agents_interrupt_success", func(t *testing.T) {
		srv.mu.Lock()
		srv.active = map[string]activeRun{
			"s-int": {token: "tok-1", cancel: func() {}},
		}
		srv.mu.Unlock()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/agents/interrupt", bytes.NewBufferString(`{"session_id":"s-int"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"interrupted":true`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("complete_slash", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/complete/slash", bytes.NewBufferString(`{"text":"/to"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"/tools"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("complete_path", func(t *testing.T) {
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "alpha.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/complete/path", bytes.NewBufferString(`{"path":"`+tmp+`/a"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "alpha.txt") {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("acp_sessions", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/acp/sessions", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"session_id":"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("acp_message", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/acp/message", bytes.NewBufferString(`{"session_id":"acp-1","input":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"ok":true`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("acp_stream", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/acp/message/stream", bytes.NewBufferString(`{"session_id":"acp-2","input":"ping"}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
			t.Fatalf("expected stream content type, got %q", got)
		}
	})

	t.Run("approval_confirm", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/ui/approval/confirm", bytes.NewBufferString(`{"session_id":"s-1","approval_id":"ap-1","approve":true}`))
		req.Header.Set("Content-Type", "application/json")
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"approval_id":"ap-1"`) || !strings.Contains(rec.Body.String(), `"approved":true`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})
}

func TestHandleChatIncludesSummary(t *testing.T) {
	srv := &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "hello summary"}},
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewBufferString(`{"session_id":"s-summary","message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["final_response"] != "hello summary" {
		t.Fatalf("expected final response to remain top-level, got %+v", body)
	}
	summary, ok := body["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got %+v", body)
	}
	if summary["status"] != "completed" || summary["tool_call_count"] != float64(0) || summary["assistant_message_count"] != float64(1) {
		t.Fatalf("unexpected summary payload: %+v", summary)
	}
}

func TestHandleChatStreamCancel(t *testing.T) {
	client := &blockingModelClient{started: make(chan struct{})}
	srv := &Server{
		Engine: &agent.Engine{
			Client:       client,
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-cancel","message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")

	done := make(chan struct{})
	go func() {
		srv.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-client.started:
	case <-time.After(2 * time.Second):
		t.Fatal("streaming request did not start model call in time")
	}

	cancelReq := httptest.NewRequest(http.MethodPost, "/v1/chat/cancel", bytes.NewBufferString(`{"session_id":"s-cancel"}`))
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(cancelRec, cancelReq)

	if cancelRec.Code != http.StatusOK {
		t.Fatalf("unexpected cancel status: %d, body=%s", cancelRec.Code, cancelRec.Body.String())
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("streaming request did not finish after cancellation")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: cancelled") {
		t.Fatalf("expected cancelled event, body=%s", body)
	}
	if strings.Contains(body, "event: result") {
		t.Fatalf("did not expect result event after cancellation, body=%s", body)
	}
	if !strings.Contains(cancelRec.Body.String(), `"cancelled":true`) {
		t.Fatalf("expected cancel response payload, body=%s", cancelRec.Body.String())
	}
}

func TestHandleChatStreamDelegateEvents(t *testing.T) {
	args := `{"goal":"child-task","context":"child-context","max_iterations":2}`
	client := &scriptedModelClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "delegate_task",
						Arguments: args,
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "child completed",
			},
			{
				Role:    "assistant",
				Content: "parent completed",
			},
		},
	}
	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, nil)
	srv := &Server{
		Engine: &agent.Engine{
			Client:       client,
			Registry:     registry,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-delegate","message":"delegate now"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	parts := []string{
		"event: delegate_started",
		`"goal":"child-task"`,
		`"status":"running"`,
		"event: delegate_finished",
		`"status":"completed"`,
		`"success":true`,
		`"final_response":"child completed"`,
		`"final_response":"parent completed"`,
	}
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected stream body to contain %q, body=%s", part, body)
		}
	}
}

func TestHandleChatIncludesToolSummary(t *testing.T) {
	client := &scriptedModelClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "echo_tool",
						Arguments: `{"value":"ping"}`,
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "tool finished",
			},
		},
	}
	registry := tools.NewRegistry()
	registry.Register(apiTestTool{
		name: "echo_tool",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"echo": args["value"]}, nil
		},
	})
	srv := &Server{
		Engine: &agent.Engine{
			Client:       client,
			Registry:     registry,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewBufferString(`{"session_id":"s-tool-summary","message":"run tool"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["final_response"] != "tool finished" {
		t.Fatalf("expected top-level final response, got %+v", body)
	}
	summary, ok := body["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got %+v", body)
	}
	if summary["tool_call_count"] != float64(1) || summary["delegate_count"] != float64(0) {
		t.Fatalf("unexpected tool summary counts: %+v", summary)
	}
	toolNames, ok := summary["tool_names"].([]any)
	if !ok || len(toolNames) != 1 || toolNames[0] != "echo_tool" {
		t.Fatalf("unexpected tool_names summary: %+v", summary)
	}
}

func TestHandleChatStreamToolFinishedStructuredData(t *testing.T) {
	client := &scriptedModelClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "echo_tool",
						Arguments: `{"value":"ping"}`,
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "tool done",
			},
		},
	}
	registry := tools.NewRegistry()
	registry.Register(apiTestTool{
		name: "echo_tool",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"echo": args["value"]}, nil
		},
	})
	srv := &Server{
		Engine: &agent.Engine{
			Client:       client,
			Registry:     registry,
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-tool","message":"run tool"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	parts := []string{
		"event: tool_started",
		`"tool_call_id":"call-1"`,
		`"tool_name":"echo_tool"`,
		`"status":"running"`,
		`"arguments":{"value":"ping"}`,
		"event: tool_finished",
		`"tool_call_id":"call-1"`,
		`"status":"completed"`,
		`"success":true`,
		`"echo":"ping"`,
	}
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected stream body to contain %q, body=%s", part, body)
		}
	}
}

func TestHandleChatStreamAssistantAndCompletedStructuredData(t *testing.T) {
	srv := &Server{
		Engine: &agent.Engine{
			Client:       fakeModelClient{response: core.Message{Role: "assistant", Content: "hello metadata"}},
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-assistant-meta","message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	parts := []string{
		"event: assistant_message",
		`"message_role":"assistant"`,
		`"content_length":14`,
		`"tool_call_count":0`,
		`"has_tool_calls":false`,
		"event: completed",
		`"finished_naturally":true`,
	}
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected stream body to contain %q, body=%s", part, body)
		}
	}
}

func TestHandleChatStreamErrorStructuredData(t *testing.T) {
	srv := &Server{
		Engine: &agent.Engine{
			Client:       waitOnContextModelClient{},
			Registry:     tools.NewRegistry(),
			SessionStore: &stubSessionStore{},
			SystemPrompt: agent.DefaultSystemPrompt(),
		},
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-error-meta","message":"ping"}`)).WithContext(reqCtx)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	parts := []string{
		"event: error",
		`"status":"error"`,
		`"turn":1`,
		`"error":"context deadline exceeded"`,
	}
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected stream body to contain %q, body=%s", part, body)
		}
	}
}

func TestHandleChatStreamMaxIterationsStructuredData(t *testing.T) {
	client := &scriptedModelClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "echo_tool",
						Arguments: `{"value":"ping"}`,
					},
				}},
			},
		},
	}
	registry := tools.NewRegistry()
	registry.Register(apiTestTool{
		name: "echo_tool",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"echo": args["value"]}, nil
		},
	})
	srv := &Server{
		Engine: &agent.Engine{
			Client:        client,
			Registry:      registry,
			SessionStore:  &stubSessionStore{},
			SystemPrompt:  agent.DefaultSystemPrompt(),
			MaxIterations: 1,
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", bytes.NewBufferString(`{"session_id":"s-max-meta","message":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	parts := []string{
		"event: max_iterations_reached",
		`"status":"max_iterations_reached"`,
		`"max_iterations":1`,
		`"finished":false`,
	}
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected stream body to contain %q, body=%s", part, body)
		}
	}
}
