package tools

import (
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestBuildUISessionBranchPayload(t *testing.T) {
	got := BuildUISessionBranchPayload("s1", "s2", 7)
	if got["source_session_id"] != "s1" || got["new_session_id"] != "s2" || got["copied_messages"] != 7 {
		t.Fatalf("unexpected branch payload: %+v", got)
	}
}

func TestBuildUISessionResumePayload(t *testing.T) {
	got := BuildUISessionResumePayload("s1", "t1", "http")
	if got["session_id"] != "s1" || got["turn_id"] != "t1" || got["resumed"] != true || got["transport"] != "http" {
		t.Fatalf("unexpected resume payload: %+v", got)
	}
}

func TestBuildUISessionCompressPayload(t *testing.T) {
	got := BuildUISessionCompressPayload("s1", 10, 4, 4)
	if got["dropped_messages"] != 6 || got["keep_last_n"] != 4 {
		t.Fatalf("unexpected compress payload: %+v", got)
	}
}

func TestBuildUISessionUndoPayload(t *testing.T) {
	got := BuildUISessionUndoPayload("s1", "s2", 3, 2, false, "no_user_message")
	if got["undone"] != false || got["reason"] != "no_user_message" {
		t.Fatalf("unexpected undo payload: %+v", got)
	}
}

func TestBuildUISessionReplayPayload(t *testing.T) {
	msgs := []core.Message{{Role: "user", Content: "x"}}
	got := BuildUISessionReplayPayload("s1", 0, 20, msgs)
	if got["count"] != 1 || got["replayed"] != true {
		t.Fatalf("unexpected replay payload: %+v", got)
	}
}

