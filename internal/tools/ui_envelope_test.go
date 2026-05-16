package tools

import "testing"

func TestBuildUIResultEnvelope(t *testing.T) {
	got := BuildUIResultEnvelope(map[string]any{"x": 1})
	if got["ok"] != true {
		t.Fatalf("expected ok=true: %+v", got)
	}
	res, ok := got["result"].(map[string]any)
	if !ok || res["x"] != 1 {
		t.Fatalf("unexpected result payload: %+v", got)
	}
}

