package tools

import "testing"

func TestBuildSessionSwitchPayload(t *testing.T) {
	got := BuildSessionSwitchPayload(" s-old ", " s-new ", true, 0)
	if got["previous_session_id"] != "s-old" || got["session_id"] != "s-new" || got["reset"] != true || got["loaded_messages"] != 0 {
		t.Fatalf("unexpected switch payload: %+v", got)
	}
}

func TestBuildSessionSwitchPayloadWithoutLoaded(t *testing.T) {
	got := BuildSessionSwitchPayload("s1", "s2", false, -1)
	if _, ok := got["loaded_messages"]; ok {
		t.Fatalf("unexpected loaded_messages: %+v", got)
	}
}

