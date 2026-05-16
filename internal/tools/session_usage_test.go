package tools

import "testing"

func TestBuildSessionUsagePayload(t *testing.T) {
	stats := map[string]any{"prompt_tokens": 12}
	got := BuildSessionUsagePayload(" s1 ", stats)
	if got["session_id"] != "s1" {
		t.Fatalf("unexpected session_id: %+v", got)
	}
	if usage, ok := got["usage"].(map[string]any); !ok || usage["prompt_tokens"] != 12 {
		t.Fatalf("unexpected usage payload: %+v", got)
	}
}

func TestBuildSessionStatsPayload(t *testing.T) {
	stats := map[string]any{"messages": 5}
	got := BuildSessionStatsPayload("s2", stats)
	if got["session_id"] != "s2" {
		t.Fatalf("unexpected session_id: %+v", got)
	}
	if out, ok := got["stats"].(map[string]any); !ok || out["messages"] != 5 {
		t.Fatalf("unexpected stats payload: %+v", got)
	}
}

