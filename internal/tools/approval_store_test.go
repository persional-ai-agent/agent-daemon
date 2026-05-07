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
