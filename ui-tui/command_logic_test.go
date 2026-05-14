package main

import "testing"

func TestParseOptionalPositiveIntArg(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got, err := parseOptionalPositiveIntArg("/history", "/history", 20)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != 20 {
			t.Fatalf("got %d want 20", got)
		}
	})

	t.Run("explicit", func(t *testing.T) {
		got, err := parseOptionalPositiveIntArg("/history 5", "/history", 20)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != 5 {
			t.Fatalf("got %d want 5", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseOptionalPositiveIntArg("/history abc", "/history", 20)
		if err == nil {
			t.Fatal("expected error for invalid arg")
		}
	})

	t.Run("non_positive", func(t *testing.T) {
		_, err := parseOptionalPositiveIntArg("/history 0", "/history", 20)
		if err == nil {
			t.Fatal("expected error for non-positive arg")
		}
	})

	t.Run("too_many", func(t *testing.T) {
		_, err := parseOptionalPositiveIntArg("/history 1 2", "/history", 20)
		if err == nil {
			t.Fatal("expected error for extra args")
		}
	})
}
