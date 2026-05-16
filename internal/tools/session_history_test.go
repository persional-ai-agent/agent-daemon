package tools

import "testing"

func TestBuildSessionHistoryPayload(t *testing.T) {
	got := BuildSessionHistoryPayload(" s1 ", 8, 10)
	if got["session_id"] != "s1" || got["limit"] != 10 || got["total_messages"] != 8 || got["count"] != 8 {
		t.Fatalf("unexpected history payload: %+v", got)
	}
}

