package store

import (
	"path/filepath"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestSessionStoreAppendLoadAndSearch(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.AppendMessage("s1", core.Message{Role: "user", Content: "hello hermes"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendMessage("s2", core.Message{Role: "assistant", Content: "hello world"}); err != nil {
		t.Fatal(err)
	}
	msgs, err := s.LoadMessages("s1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Content != "hello hermes" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
	rows, err := s.Search("hello", 10, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["session_id"] != "s2" {
		t.Fatalf("unexpected search rows: %+v", rows)
	}
}
