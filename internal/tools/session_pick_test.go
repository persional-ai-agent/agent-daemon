package tools

import "testing"

func TestBuildSessionPickPayload(t *testing.T) {
	got := BuildSessionPickPayload(" s-old ", " s-new ", 3)
	if got["previous_session_id"] != "s-old" || got["session_id"] != "s-new" || got["index"] != 3 {
		t.Fatalf("unexpected pick payload: %+v", got)
	}
}

