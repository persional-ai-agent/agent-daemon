package tools

import (
	"testing"
	"time"
)

func TestApprovalStoreGrantStatusAndExpiry(t *testing.T) {
	s := NewApprovalStore(50 * time.Millisecond)
	if ok := s.IsApproved("s1"); ok {
		t.Fatal("expected unapproved session by default")
	}
	exp := s.Grant("s1", 0)
	if exp.IsZero() {
		t.Fatal("expected non-zero expiry")
	}
	if ok := s.IsApproved("s1"); !ok {
		t.Fatal("expected approved after grant")
	}
	time.Sleep(80 * time.Millisecond)
	if ok := s.IsApproved("s1"); ok {
		t.Fatal("expected approval to expire")
	}
}

func TestApprovalStoreRevoke(t *testing.T) {
	s := NewApprovalStore(time.Minute)
	s.Grant("s2", 0)
	if !s.Revoke("s2") {
		t.Fatal("expected revoke=true")
	}
	if s.IsApproved("s2") {
		t.Fatal("expected revoked session to be unapproved")
	}
}

func TestApprovalStorePatternGrant(t *testing.T) {
	s := NewApprovalStore(time.Minute)
	if ok := s.IsApprovedPattern("s1", "recursive_delete"); ok {
		t.Fatal("expected pattern not approved by default")
	}
	exp := s.GrantPattern("s1", "recursive_delete", 0)
	if exp.IsZero() {
		t.Fatal("expected non-zero expiry")
	}
	if ok := s.IsApprovedPattern("s1", "recursive_delete"); !ok {
		t.Fatal("expected pattern approved after grant")
	}
	if ok := s.IsApprovedPattern("s1", "world_writable"); ok {
		t.Fatal("expected different pattern not approved")
	}
}

func TestApprovalStoreSessionOverridesPattern(t *testing.T) {
	s := NewApprovalStore(time.Minute)
	s.Grant("s1", time.Minute)
	if ok := s.IsApprovedPattern("s1", "recursive_delete"); !ok {
		t.Fatal("session-level approval should override pattern check")
	}
}

func TestApprovalStorePatternExpiry(t *testing.T) {
	s := NewApprovalStore(50 * time.Millisecond)
	s.GrantPattern("s1", "recursive_delete", 0)
	time.Sleep(80 * time.Millisecond)
	if ok := s.IsApprovedPattern("s1", "recursive_delete"); ok {
		t.Fatal("expected pattern approval to expire")
	}
}

func TestApprovalStoreRevokePattern(t *testing.T) {
	s := NewApprovalStore(time.Minute)
	s.GrantPattern("s1", "recursive_delete", 0)
	s.GrantPattern("s1", "world_writable", 0)
	if !s.RevokePattern("s1", "recursive_delete") {
		t.Fatal("expected revoke pattern=true")
	}
	if ok := s.IsApprovedPattern("s1", "recursive_delete"); ok {
		t.Fatal("expected revoked pattern to be unapproved")
	}
	if ok := s.IsApprovedPattern("s1", "world_writable"); !ok {
		t.Fatal("expected other pattern to still be approved")
	}
}

func TestApprovalStoreListApprovals(t *testing.T) {
	s := NewApprovalStore(time.Minute)
	s.Grant("s1", time.Minute)
	s.GrantPattern("s1", "recursive_delete", time.Minute)
	approvals := s.ListApprovals("s1")
	if len(approvals) != 2 {
		t.Fatalf("expected 2 approvals, got %d", len(approvals))
	}
	found := map[string]bool{}
	for _, a := range approvals {
		scope, _ := a["scope"].(string)
		found[scope] = true
	}
	if !found["session"] || !found["pattern"] {
		t.Fatalf("expected both session and pattern scopes, got %v", found)
	}
}

func TestApprovalStoreRevokeClearsPatterns(t *testing.T) {
	s := NewApprovalStore(time.Minute)
	s.Grant("s1", time.Minute)
	s.GrantPattern("s1", "recursive_delete", time.Minute)
	s.Revoke("s1")
	if ok := s.IsApproved("s1"); ok {
		t.Fatal("expected session revoked")
	}
	if ok := s.IsApprovedPattern("s1", "recursive_delete"); ok {
		t.Fatal("expected patterns cleared on session revoke")
	}
}
