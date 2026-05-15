package main

import "testing"

func TestPrintEventSkipsNilError(t *testing.T) {
	out := printEvent(map[string]any{"type": "error", "error": nil}, false)
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
}

func TestPrintEventFormatsNonNilError(t *testing.T) {
	out := printEvent(map[string]any{"type": "error", "error": "boom"}, false)
	if out != "error: boom" {
		t.Fatalf("unexpected output: %q", out)
	}
}
