package tools

import "testing"

func TestBuildSessionCompressPayload(t *testing.T) {
	got := BuildSessionCompressPayload(" s1 ", 30, 12, 20, 18, true, "")
	if got["session_id"] != "s1" || got["compacted"] != true || got["before"] != 30 || got["after"] != 12 || got["dropped"] != 18 {
		t.Fatalf("unexpected base payload: %+v", got)
	}
	if got["tail_messages"] != 20 || got["summarized_messages"] != 18 {
		t.Fatalf("unexpected extra payload fields: %+v", got)
	}
}

func TestBuildSessionCompressPayloadWithReason(t *testing.T) {
	got := BuildSessionCompressPayload("s2", 5, 5, 20, 0, false, "history shorter than tail")
	if got["reason"] != "history shorter than tail" || got["compacted"] != false || got["dropped"] != 0 {
		t.Fatalf("unexpected reason payload: %+v", got)
	}
	if _, ok := got["summarized_messages"]; ok {
		t.Fatalf("unexpected summarized_messages: %+v", got)
	}
}

