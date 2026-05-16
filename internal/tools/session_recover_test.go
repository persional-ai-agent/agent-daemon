package tools

import "testing"

func TestBuildSessionRecoverPayload(t *testing.T) {
	got := BuildSessionRecoverPayload(" s-old ", " s-new ", true)
	if got["recovered"] != true || got["previous_session_id"] != "s-old" || got["session_id"] != "s-new" || got["replay"] != true {
		t.Fatalf("unexpected recover payload: %+v", got)
	}
}

