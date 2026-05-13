package cli

import (
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

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
