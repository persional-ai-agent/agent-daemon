package tools

import (
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestBuildSessionReloadPayload(t *testing.T) {
	msgs := []core.Message{{Role: "user", Content: "hello"}}
	got := BuildSessionReloadPayload(" s1 ", msgs)
	if got["session_id"] != "s1" || got["count"] != 1 {
		t.Fatalf("unexpected reload payload: %+v", got)
	}
	out, ok := got["messages"].([]core.Message)
	if !ok || len(out) != 1 || out[0].Content != "hello" {
		t.Fatalf("unexpected reload messages: %+v", got)
	}
}

