package tools

import "testing"

func TestBuildSessionUndoPayload(t *testing.T) {
	got := BuildSessionUndoPayload(" s-new ", 3, 12)
	if got["session_id"] != "s-new" || got["removed_messages"] != 3 || got["messages_in_context"] != 12 {
		t.Fatalf("unexpected undo payload: %+v", got)
	}
}

