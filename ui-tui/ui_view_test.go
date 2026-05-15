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

func TestPrintEventFormatsErrorFromContent(t *testing.T) {
	out := printEvent(map[string]any{"type": "error", "content": "from content"}, false)
	if out != "error: from content" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestPrintEventFormatsErrorFromData(t *testing.T) {
	out := printEvent(map[string]any{"type": "error", "data": map[string]any{"error": "from data"}}, false)
	if out != "error: from data" {
		t.Fatalf("unexpected output: %q", out)
	}
}
