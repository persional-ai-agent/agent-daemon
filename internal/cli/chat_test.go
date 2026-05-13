package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return strings.TrimSpace(buf.String())
}

type testSessionStore struct {
	msgs []core.Message
}

func (s *testSessionStore) AppendMessage(_ string, msg core.Message) error {
	s.msgs = append(s.msgs, msg)
	return nil
}

func (s *testSessionStore) LoadMessages(_ string, _ int) ([]core.Message, error) {
	out := make([]core.Message, len(s.msgs))
	copy(out, s.msgs)
	return out, nil
}

func (s *testSessionStore) ListRecentSessions(limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	return []map[string]any{
		{"session_id": "s1", "last_seen": "2026-05-13T00:00:00Z"},
	}, nil
}

func (s *testSessionStore) LoadMessagesPage(_ string, offset, limit int) ([]core.Message, error) {
	if limit <= 0 {
		limit = 20
	}
	start := offset
	if start > len(s.msgs) {
		start = len(s.msgs)
	}
	end := start + limit
	if end > len(s.msgs) {
		end = len(s.msgs)
	}
	out := make([]core.Message, end-start)
	copy(out, s.msgs[start:end])
	return out, nil
}

func (s *testSessionStore) SessionStats(sessionID string) (map[string]any, error) {
	return map[string]any{"session_id": sessionID, "message_count": len(s.msgs)}, nil
}

func makeEngineForSlashTests(msgs []core.Message) *agent.Engine {
	reg := tools.NewRegistry()
	reg.Register(tools.NewSendMessageTool())
	store := &testSessionStore{msgs: msgs}
	return &agent.Engine{
		Registry:    reg,
		SessionStore: store,
	}
}

func TestHandleSlashCommandClear(t *testing.T) {
	eng := makeEngineForSlashTests([]core.Message{{Role: "user", Content: "hello"}})
	history := []core.Message{{Role: "assistant", Content: "hi"}}
	next, prompt, handled, err := handleSlashCommand("/clear", "s1", "sp", history, eng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
	if next != nil {
		t.Fatalf("expected nil history after clear, got %v", next)
	}
	if prompt != "sp" {
		t.Fatalf("prompt changed unexpectedly: %q", prompt)
	}
}

func TestHandleSlashCommandReload(t *testing.T) {
	eng := makeEngineForSlashTests([]core.Message{{Role: "user", Content: "a"}, {Role: "assistant", Content: "b"}})
	next, _, handled, err := handleSlashCommand("/reload", "s1", "sp", nil, eng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
	if len(next) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(next))
	}
}

func TestHandleSlashCommandStats(t *testing.T) {
	eng := makeEngineForSlashTests([]core.Message{{Role: "user", Content: "a"}})
	_, _, handled, err := handleSlashCommand("/stats", "s1", "sp", nil, eng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
}

func TestPrintCLIEnvelopeSuccess(t *testing.T) {
	line := captureStdout(t, func() {
		printCLIEnvelope(true, map[string]any{"session_id": "s1"}, "", "")
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(line), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["ok"] != true || out["api_version"] != "v1" || out["compat"] == "" {
		t.Fatalf("unexpected envelope: %+v", out)
	}
	if out["session_id"] != "s1" {
		t.Fatalf("missing payload: %+v", out)
	}
}

func TestPrintCLIEnvelopeError(t *testing.T) {
	line := captureStdout(t, func() {
		printCLIEnvelope(false, nil, "invalid_argument", "bad")
	})
	var out map[string]any
	if err := json.Unmarshal([]byte(line), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["ok"] != false || out["api_version"] != "v1" || out["compat"] == "" {
		t.Fatalf("unexpected envelope: %+v", out)
	}
	errObj, _ := out["error"].(map[string]any)
	if errObj["code"] != "invalid_argument" || errObj["message"] != "bad" {
		t.Fatalf("unexpected error payload: %+v", out)
	}
}
