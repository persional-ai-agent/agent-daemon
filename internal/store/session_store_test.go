package store

import (
	"path/filepath"
	"testing"
	"time"

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

func TestSessionStoreApprovalGrantAndCheck(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "approvals.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	expiresAt := time.Now().Add(time.Minute)
	if err := s.GrantApproval("s1", "session", "", expiresAt); err != nil {
		t.Fatal(err)
	}
	approved, err := s.IsApproved("s1", "session", "")
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Fatal("expected session approved")
	}
}

func TestSessionStoreApprovalPatternGrant(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "approvals-pattern.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	expiresAt := time.Now().Add(time.Minute)
	if err := s.GrantApproval("s1", "pattern", "recursive_delete", expiresAt); err != nil {
		t.Fatal(err)
	}
	approved, err := s.IsApproved("s1", "pattern", "recursive_delete")
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Fatal("expected pattern approved")
	}
	approved, err = s.IsApproved("s1", "pattern", "world_writable")
	if err != nil {
		t.Fatal(err)
	}
	if approved {
		t.Fatal("expected different pattern not approved")
	}
}

func TestSessionStoreApprovalRevoke(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "approvals-revoke.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	expiresAt := time.Now().Add(time.Minute)
	s.GrantApproval("s1", "session", "", expiresAt)
	revoked, err := s.RevokeApproval("s1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Fatal("expected revoked=true")
	}
	approved, _ := s.IsApproved("s1", "session", "")
	if approved {
		t.Fatal("expected not approved after revoke")
	}
}

func TestSessionStoreApprovalList(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "approvals-list.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	expiresAt := time.Now().Add(time.Minute)
	s.GrantApproval("s1", "session", "", expiresAt)
	s.GrantApproval("s1", "pattern", "recursive_delete", expiresAt)
	records, err := s.ListApprovals("s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 approval records, got %d", len(records))
	}
}

func TestSessionStoreApprovalExpiredNotVisible(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "approvals-expired.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	expiresAt := time.Now().Add(-time.Second)
	s.GrantApproval("s1", "session", "", expiresAt)
	approved, _ := s.IsApproved("s1", "session", "")
	if approved {
		t.Fatal("expected expired approval not visible")
	}
	records, _ := s.ListApprovals("s1")
	if len(records) != 0 {
		t.Fatalf("expected 0 records for expired approval, got %d", len(records))
	}
}

func TestSessionStoreApprovalCleanup(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "approvals-cleanup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	s.GrantApproval("s1", "session", "", time.Now().Add(-time.Second))
	s.GrantApproval("s2", "session", "", time.Now().Add(time.Minute))
	n, err := s.CleanupExpiredApprovals()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 expired approval cleaned up, got %d", n)
	}
}

func TestSessionStoreOAuthTokenSaveLoadAndDelete(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	provider := "https://mcp.example.com"
	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second)
	if err := s.SaveOAuthToken(provider, "access-1", "refresh-1", expiresAt); err != nil {
		t.Fatal(err)
	}
	accessToken, refreshToken, exp, err := s.LoadOAuthToken(provider)
	if err != nil {
		t.Fatal(err)
	}
	if accessToken != "access-1" {
		t.Fatalf("expected access-1, got %s", accessToken)
	}
	if refreshToken != "refresh-1" {
		t.Fatalf("expected refresh-1, got %s", refreshToken)
	}
	if !exp.Equal(expiresAt) {
		t.Fatalf("expected %v, got %v", expiresAt, exp)
	}

	if err := s.SaveOAuthToken(provider, "access-2", "refresh-2", expiresAt); err != nil {
		t.Fatal(err)
	}
	accessToken, _, _, err = s.LoadOAuthToken(provider)
	if err != nil {
		t.Fatal(err)
	}
	if accessToken != "access-2" {
		t.Fatalf("expected access-2 after upsert, got %s", accessToken)
	}

	if err := s.DeleteOAuthToken(provider); err != nil {
		t.Fatal(err)
	}
	accessToken, _, _, err = s.LoadOAuthToken(provider)
	if err != nil {
		t.Fatal(err)
	}
	if accessToken != "" {
		t.Fatalf("expected empty after delete, got %s", accessToken)
	}
}

func TestSessionStoreOAuthTokenLoadMissing(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	accessToken, refreshToken, _, err := s.LoadOAuthToken("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if accessToken != "" || refreshToken != "" {
		t.Fatalf("expected empty for missing provider, got access=%s refresh=%s", accessToken, refreshToken)
	}
}

func TestSessionStoreListRecentSessions(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.AppendMessage("s1", core.Message{Role: "user", Content: "one"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendMessage("s2", core.Message{Role: "user", Content: "two"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendMessage("s1", core.Message{Role: "assistant", Content: "three"}); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ListRecentSessions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 sessions, got %d: %#v", len(rows), rows)
	}
	if rows[0]["session_id"] != "s1" {
		t.Fatalf("expected most recent session s1, got %#v", rows[0])
	}
}

func TestSessionStoreLoadMessagesPage(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	for i := 0; i < 5; i++ {
		if err := s.AppendMessage("s1", core.Message{Role: "user", Content: "m" + string(rune('0'+i))}); err != nil {
			t.Fatal(err)
		}
	}

	msgs, err := s.LoadMessagesPage("s1", 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d: %#v", len(msgs), msgs)
	}
	if msgs[0].Content != "m2" || msgs[1].Content != "m3" {
		t.Fatalf("unexpected messages: %#v", msgs)
	}
}

func TestSessionStoreSessionStats(t *testing.T) {
	s, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.AppendMessage("s1", core.Message{Role: "user", Content: "hi"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendMessage("s1", core.Message{Role: "assistant", Content: "ok"}); err != nil {
		t.Fatal(err)
	}

	stats, err := s.SessionStats("s1")
	if err != nil {
		t.Fatal(err)
	}
	if stats["session_id"] != "s1" {
		t.Fatalf("expected session_id s1, got %#v", stats["session_id"])
	}
	if stats["message_count"] != int64(2) {
		t.Fatalf("expected message_count=2, got %#v", stats["message_count"])
	}
}
