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

func TestParsePendingArgs(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		limit, action, idx, err := parsePendingArgs("/pending")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if limit != 1 || action != "" || idx != 0 {
			t.Fatalf("got limit=%d action=%q idx=%d", limit, action, idx)
		}
	})

	t.Run("limit_only", func(t *testing.T) {
		limit, action, idx, err := parsePendingArgs("/pending 3")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if limit != 3 || action != "" || idx != 0 {
			t.Fatalf("got limit=%d action=%q idx=%d", limit, action, idx)
		}
	})

	t.Run("action_only", func(t *testing.T) {
		limit, action, idx, err := parsePendingArgs("/pending approve 2")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if limit != 1 || action != "approve" || idx != 2 {
			t.Fatalf("got limit=%d action=%q idx=%d", limit, action, idx)
		}
	})

	t.Run("limit_and_action", func(t *testing.T) {
		limit, action, idx, err := parsePendingArgs("/pending 5 d 1")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if limit != 5 || action != "d" || idx != 1 {
			t.Fatalf("got limit=%d action=%q idx=%d", limit, action, idx)
		}
	})

	t.Run("invalid_limit", func(t *testing.T) {
		_, _, _, err := parsePendingArgs("/pending 0")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid_action", func(t *testing.T) {
		_, _, _, err := parsePendingArgs("/pending nope")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid_index", func(t *testing.T) {
		_, _, _, err := parsePendingArgs("/pending approve xx")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
