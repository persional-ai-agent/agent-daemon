package tools

import "testing"

func TestBuildSessionClearPayload(t *testing.T) {
	got := BuildSessionClearPayload(" s-old ", " s-new ", true)
	if got["previous_session_id"] != "s-old" || got["session_id"] != "s-new" || got["cleared"] != true {
		t.Fatalf("unexpected clear payload: %+v", got)
	}
}

