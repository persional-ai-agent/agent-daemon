package tools

import "testing"

func TestSessionMessageTextHelpers(t *testing.T) {
	if got := SessionSwitchedEN("a", "b"); got != "_Session switched: a -> b_" {
		t.Fatalf("unexpected switched text: %s", got)
	}
	if got := SessionCompressedEN(10, 4); got != "_Compressed: before=10, after=4, dropped=6_" {
		t.Fatalf("unexpected compressed text: %s", got)
	}
}
