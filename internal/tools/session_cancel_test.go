package tools

import "testing"

func TestBuildSessionCancelPayload(t *testing.T) {
	got := BuildSessionCancelPayload(" s1 ", true, " cancelled ")
	if got["session_id"] != "s1" {
		t.Fatalf("unexpected session_id: %#v", got["session_id"])
	}
	if got["cancelled"] != true {
		t.Fatalf("unexpected cancelled: %#v", got["cancelled"])
	}
	if got["reason"] != "cancelled" {
		t.Fatalf("unexpected reason: %#v", got["reason"])
	}

	got = BuildSessionCancelPayload("s2", false, "")
	if _, ok := got["reason"]; ok {
		t.Fatalf("reason should be omitted when empty: %#v", got["reason"])
	}
}
