package tools

import (
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestBuildSessionShowPayload(t *testing.T) {
	msgs := []core.Message{{Role: "user", Content: "hello"}}
	got := BuildSessionShowPayload(" s1 ", 5, 20, msgs)
	if got["session_id"] != "s1" || got["offset"] != 5 || got["limit"] != 20 || got["count"] != 1 {
		t.Fatalf("unexpected show payload header: %+v", got)
	}
	out, ok := got["messages"].([]core.Message)
	if !ok || len(out) != 1 || out[0].Content != "hello" {
		t.Fatalf("unexpected show messages: %+v", got)
	}
}

