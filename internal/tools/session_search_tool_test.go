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

func TestSessionSearchBlankQueryReturnsRecentSummaries(t *testing.T) {
	workdir := t.TempDir()
	ss, err := store.NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()

	b := &BuiltinTools{}
	_ = ss.AppendMessage("s1", core.Message{Role: "user", Content: "first session"})
	_ = ss.AppendMessage("s2", core.Message{Role: "user", Content: "second session"})

	res, err := b.sessionSearch(context.Background(), map[string]any{
		"limit":                   5,
		"include_current_session": true,
	}, ToolContext{Workdir: workdir, SessionID: "s2", SessionStore: ss})
	if err != nil {
		t.Fatal(err)
	}
	if mode, _ := res["mode"].(string); mode != "recent" {
		t.Fatalf("expected recent mode: %v", res)
	}
	results, _ := res["results"].([]map[string]any)
	if len(results) != 2 || results[0]["session_id"] != "s2" {
		t.Fatalf("unexpected recent results: %v", res)
	}
	if summary, _ := results[0]["summary"].(string); summary == "" {
		t.Fatalf("missing summary: %v", res)
	}
}
