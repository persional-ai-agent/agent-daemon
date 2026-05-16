package tools

import "testing"

func TestBuildApprovalCommandPayload(t *testing.T) {
	got := BuildApprovalCommandPayload(" /approve ", " ap-1 ")
	if got["slash"] != "/approve" || got["approval_id"] != "ap-1" {
		t.Fatalf("unexpected approval payload: %+v", got)
	}
}

func TestBuildApprovalCommandPayloadWithoutApprovalID(t *testing.T) {
	got := BuildApprovalCommandPayload("/grant", "")
	if got["slash"] != "/grant" {
		t.Fatalf("unexpected slash value: %+v", got)
	}
	if _, ok := got["approval_id"]; ok {
		t.Fatalf("unexpected approval_id: %+v", got)
	}
}

func TestApprovalTextHelpers(t *testing.T) {
	if got := ApprovalDeniedEN(""); got != "Denied." {
		t.Fatalf("unexpected denied empty: %s", got)
	}
	if got := ApprovalDeniedEN("cmd"); got != "Denied: cmd" {
		t.Fatalf("unexpected denied with cmd: %s", got)
	}
	if got := ApprovalApprovedEN(""); got != "Approved." {
		t.Fatalf("unexpected approved empty: %s", got)
	}
	if got := ApprovalApprovedEN("cmd"); got != "Approved: cmd" {
		t.Fatalf("unexpected approved with cmd: %s", got)
	}
	if got := GrantedPatternApprovalEN("p", ""); got != "Granted pattern approval: p" {
		t.Fatalf("unexpected grant pattern: %s", got)
	}
	if got := GrantedSessionApprovalEN(""); got != "Granted session approval." {
		t.Fatalf("unexpected grant session: %s", got)
	}
	if got := RevokedSessionApprovalEN(); got != "Revoked session approval." {
		t.Fatalf("unexpected revoke session: %s", got)
	}
	if got := NoPendingApprovalEN(); got != "No pending approval." {
		t.Fatalf("unexpected pending none: %s", got)
	}
}
