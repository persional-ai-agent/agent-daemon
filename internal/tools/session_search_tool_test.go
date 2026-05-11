package tools

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/store"
)

func TestSessionSearchIncludeCurrentSession(t *testing.T) {
	workdir := t.TempDir()
	ss, err := store.NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()

	b := &BuiltinTools{}
	sessionID := "s1"
	_ = ss.AppendMessage(sessionID, core.Message{Role: "user", Content: "hello world"})

	res, err := b.sessionSearch(context.Background(), map[string]any{
		"query":                   "hello",
		"include_current_session": true,
	}, ToolContext{Workdir: workdir, SessionID: sessionID, SessionStore: ss})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := res["success"].(bool); !v {
		t.Fatalf("expected success: %v", res)
	}
	results, _ := res["results"].([]map[string]any)
	if len(results) == 0 {
		t.Fatalf("expected results: %v", res)
	}
}

