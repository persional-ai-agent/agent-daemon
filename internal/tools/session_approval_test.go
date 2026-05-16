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

