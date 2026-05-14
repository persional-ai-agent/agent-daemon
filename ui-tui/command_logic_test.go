package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	t.Run("action_without_index", func(t *testing.T) {
		_, _, _, err := parsePendingArgs("/pending approve")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestHandleTUICommandRerunEmptyHistory(t *testing.T) {
	s := &appState{
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
	}
	_, err, _ := handleTUICommand(s, "/rerun 1", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "no history available" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestHandleTUICommandRerunSkipsTrailingSelfEntries(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "history.log")
	content := strings.Join([]string{
		"2026-01-01T00:00:00Z\t/help",
		"2026-01-01T00:00:01Z\t/rerun 1",
		"2026-01-01T00:00:02Z\t/rerun 1",
	}, "\n") + "\n"
	if err := os.WriteFile(historyPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &appState{
		historyPath:      historyPath,
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
	}
	lines, err, _ := handleTUICommand(s, "/rerun 1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	found := false
	for _, line := range lines {
		if strings.HasPrefix(line, "commands:") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected /help output, got lines=%v", lines)
	}
}

func TestParseSessionsArgs(t *testing.T) {
	cases := []struct {
		in      string
		limit   int
		pick    int
		wantErr bool
	}{
		{"/sessions", 20, 0, false},
		{"/sessions 50", 50, 0, false},
		{"/sessions pick 2", 20, 2, false},
		{"/sessions 30 pick 4", 30, 4, false},
		{"/sessions pick", 0, 0, true},
		{"/sessions x", 0, 0, true},
		{"/sessions 0", 0, 0, true},
		{"/sessions 10 pick x", 0, 0, true},
	}
	for _, tc := range cases {
		limit, pick, err := parseSessionsArgs(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q unexpected err: %v", tc.in, err)
		}
		if limit != tc.limit || pick != tc.pick {
			t.Fatalf("%q got limit=%d pick=%d", tc.in, limit, pick)
		}
	}
}

func TestParseShowArgs(t *testing.T) {
	cases := []struct {
		in      string
		sid     string
		offset  int
		limit   int
		pick    int
		wantErr bool
	}{
		{"/show", "s1", 0, 20, 0, false},
		{"/show s2", "s2", 0, 20, 0, false},
		{"/show s2 10", "s2", 10, 20, 0, false},
		{"/show s2 10 30", "s2", 10, 30, 0, false},
		{"/show s2 10 30 pick 2", "s2", 10, 30, 2, false},
		{"/show s2 x", "", 0, 0, 0, true},
		{"/show s2 -1", "", 0, 0, 0, true},
		{"/show s2 1 0", "", 0, 0, 0, true},
		{"/show s2 1 2 pick", "", 0, 0, 0, true},
		{"/show s2 1 2 pick x", "", 0, 0, 0, true},
	}
	for _, tc := range cases {
		sid, offset, limit, pick, err := parseShowArgs(tc.in, "s1")
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q unexpected err: %v", tc.in, err)
		}
		if sid != tc.sid || offset != tc.offset || limit != tc.limit || pick != tc.pick {
			t.Fatalf("%q got sid=%s offset=%d limit=%d pick=%d", tc.in, sid, offset, limit, pick)
		}
	}
}

func TestHandleTUICommandNextPrevRequireShow(t *testing.T) {
	s := &appState{
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
	}
	_, err, _ := handleTUICommand(s, "/next", nil, nil)
	if err == nil || err.Error() != "run /show first before /next" {
		t.Fatalf("unexpected /next err: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/prev", nil, nil)
	if err == nil || err.Error() != "run /show first before /prev" {
		t.Fatalf("unexpected /prev err: %v", err)
	}
}

func TestParseStatsArgs(t *testing.T) {
	sid, err := parseStatsArgs("/stats", "s1")
	if err != nil || sid != "s1" {
		t.Fatalf("unexpected /stats parse: sid=%q err=%v", sid, err)
	}
	sid, err = parseStatsArgs("/stats s2", "s1")
	if err != nil || sid != "s2" {
		t.Fatalf("unexpected /stats s2 parse: sid=%q err=%v", sid, err)
	}
	_, err = parseStatsArgs("/stats s2 extra", "s1")
	if err == nil {
		t.Fatal("expected error for extra args")
	}
}

func TestHandleTUICommandArgumentValidationErrors(t *testing.T) {
	s := &appState{
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		session:          "s1",
	}
	cases := []struct {
		cmd  string
		want string
	}{
		{"/sessions pick", "用法: /sessions [limit] [pick <index>]"},
		{"/show s1 bad", "用法: /show [session] [offset>=0] [limit>0] [pick <index>]"},
		{"/stats s1 extra", "用法: /stats [session]"},
	}
	for _, tc := range cases {
		_, err, _ := handleTUICommand(s, tc.cmd, nil, nil)
		if err == nil {
			t.Fatalf("%q expected error", tc.cmd)
		}
		if err.Error() != tc.want {
			t.Fatalf("%q got err=%q want=%q", tc.cmd, err.Error(), tc.want)
		}
	}
}
