package tools

import "testing"

func TestBuildSessionListPayload(t *testing.T) {
	rows := []map[string]any{{"session_id": "s1"}}
	got := BuildSessionListPayload(10, rows)
	if got["count"] != 1 || got["limit"] != 10 {
		t.Fatalf("unexpected list payload: %+v", got)
	}
	out, ok := got["sessions"].([]map[string]any)
	if !ok || len(out) != 1 || out[0]["session_id"] != "s1" {
		t.Fatalf("unexpected sessions payload: %+v", got)
	}
}

