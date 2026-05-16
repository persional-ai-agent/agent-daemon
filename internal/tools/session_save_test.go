package tools

import "testing"

func TestBuildSessionSavePayload(t *testing.T) {
	got := BuildSessionSavePayload(" s1 ", " /tmp/s1.json ", 7)
	if got["session_id"] != "s1" || got["path"] != "/tmp/s1.json" || got["messages"] != 7 {
		t.Fatalf("unexpected save payload: %+v", got)
	}
}

