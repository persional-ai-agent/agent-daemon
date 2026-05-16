package tools

import "testing"

func TestBuildSessionOverviewPayload(t *testing.T) {
	got := BuildSessionOverviewPayload(" s1 ", " route-1 ", 3, 9)
	if got["session_id"] != "s1" || got["route_session"] != "route-1" || got["messages_in_context"] != 3 || got["tools"] != 9 {
		t.Fatalf("unexpected overview payload: %+v", got)
	}
}

func TestBuildSessionOverviewPayloadOptional(t *testing.T) {
	got := BuildSessionOverviewPayload("s2", "", -1, -1)
	if _, ok := got["route_session"]; ok {
		t.Fatalf("unexpected route_session: %+v", got)
	}
	if _, ok := got["messages_in_context"]; ok {
		t.Fatalf("unexpected messages_in_context: %+v", got)
	}
	if _, ok := got["tools"]; ok {
		t.Fatalf("unexpected tools: %+v", got)
	}
}

