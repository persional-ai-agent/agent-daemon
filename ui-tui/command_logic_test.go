package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestParseRequiredPositiveIntArg(t *testing.T) {
	v, err := parseRequiredPositiveIntArg("/pick 3", "/pick")
	if err != nil || v != 3 {
		t.Fatalf("unexpected parse result: v=%d err=%v", v, err)
	}
	_, err = parseRequiredPositiveIntArg("/pick", "/pick")
	if err == nil {
		t.Fatal("expected error for missing argument")
	}
	_, err = parseRequiredPositiveIntArg("/pick 0", "/pick")
	if err == nil {
		t.Fatal("expected error for non-positive integer")
	}
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
	if err.Error() != "没有可重放的历史记录" {
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

func TestParseOpenArgs(t *testing.T) {
	idx, action, err := parseOpenArgs("/open 2")
	if err != nil || idx != 2 || action != "" {
		t.Fatalf("unexpected parse: idx=%d action=%q err=%v", idx, action, err)
	}
	idx, action, err = parseOpenArgs("/open 3 approve")
	if err != nil || idx != 3 || action != "approve" {
		t.Fatalf("unexpected parse with action: idx=%d action=%q err=%v", idx, action, err)
	}
	_, _, err = parseOpenArgs("/open")
	if err == nil {
		t.Fatal("expected error for missing index")
	}
	_, _, err = parseOpenArgs("/open x")
	if err == nil {
		t.Fatal("expected error for invalid index")
	}
	_, _, err = parseOpenArgs("/open 1 maybe")
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if err.Error() != "用法: /open <index> [a|d|approve|deny]" {
		t.Fatalf("unexpected usage error: %v", err)
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
		{"/pending approve", "用法: /pending [limit] [approve|deny|a|d <index>]"},
		{"/api http://bad", "API 地址必须以 ws:// 或 wss:// 开头"},
		{"/http ws://bad", "HTTP API 地址必须以 http:// 或 https:// 开头"},
		{"/pretty maybe", "用法: /pretty on|off"},
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

	_, err, _ := handleTUICommand(s, "/tool", nil, nil)
	if err == nil || err.Error() != "用法: /tool <name>" {
		t.Fatalf("unexpected /tool error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/save", nil, nil)
	if err == nil || err.Error() != "用法: /save <file>" {
		t.Fatalf("unexpected /save error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/pretty", nil, nil)
	if err == nil || err.Error() != "用法: /pretty on|off" {
		t.Fatalf("unexpected /pretty error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/rerun", nil, nil)
	if err == nil || err.Error() != "用法: /rerun <index>" {
		t.Fatalf("unexpected /rerun error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/pick", nil, nil)
	if err == nil || err.Error() != "用法: /pick <index>" {
		t.Fatalf("unexpected /pick error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/gateway", nil, nil)
	if err == nil || err.Error() != "用法: /gateway status|enable|disable" {
		t.Fatalf("unexpected /gateway error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/config", nil, nil)
	if err == nil || err.Error() != "用法: /config get|set <section.key> <value>|tui" {
		t.Fatalf("unexpected /config error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/bookmark", nil, nil)
	if err == nil || err.Error() != "用法: /bookmark add <name> | /bookmark list | /bookmark use <name>" {
		t.Fatalf("unexpected /bookmark error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/view bad", nil, nil)
	if err == nil || err.Error() != "用法: /view human|json" {
		t.Fatalf("unexpected /view error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/diag export", nil, nil)
	if err == nil || err.Error() != "用法: /diag export <file>" {
		t.Fatalf("unexpected /diag export error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/reconnect timeout bad", nil, nil)
	if err == nil || err.Error() != "用法: /reconnect timeout wait|reconnect|cancel" {
		t.Fatalf("unexpected /reconnect timeout error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/panel auto maybe", nil, nil)
	if err == nil || err.Error() != "用法: /panel auto on|off" {
		t.Fatalf("unexpected /panel auto error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/panel interval 0", nil, nil)
	if err == nil || err.Error() != "panel 刷新间隔必须在 1..300 秒之间" {
		t.Fatalf("unexpected /panel interval error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/workbench", nil, nil)
	if err == nil || err.Error() != "用法: /workbench save|list|load|delete ..." {
		t.Fatalf("unexpected /workbench error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/workflow", nil, nil)
	if err == nil || err.Error() != "用法: /workflow save|list|run|delete ..." {
		t.Fatalf("unexpected /workflow error: %v", err)
	}
}

func TestGatewayCommandCaseInsensitive(t *testing.T) {
	lastAction := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/ui/gateway/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": map[string]any{"enabled": true}})
		case "/v1/ui/gateway/action":
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			if v, ok := in["action"].(string); ok {
				lastAction = v
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"ok": true}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	s := &appState{
		httpBase:         ts.URL,
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		session:          "s1",
	}
	if _, err, _ := handleTUICommand(s, "/gateway STATUS", nil, nil); err != nil {
		t.Fatalf("status command failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/gateway ENABLE", nil, nil); err != nil {
		t.Fatalf("enable command failed: %v", err)
	}
	if lastAction != "enable" {
		t.Fatalf("lastAction=%q", lastAction)
	}
}

func TestPanelCommandCaseInsensitiveSubcommands(t *testing.T) {
	s := &appState{
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		panelAutoRefresh: false,
	}
	if _, err, _ := handleTUICommand(s, "/panel AUTO ON", nil, nil); err != nil {
		t.Fatalf("panel auto on failed: %v", err)
	}
	if !s.panelAutoRefresh {
		t.Fatal("expected panelAutoRefresh=true")
	}
	if _, err, _ := handleTUICommand(s, "/panel NEXT", nil, nil); err != nil {
		t.Fatalf("panel next failed: %v", err)
	}
}

func TestConfigCommandCaseInsensitiveSubcommands(t *testing.T) {
	var setKey, setValue string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/ui/config":
			_ = json.NewEncoder(w).Encode(map[string]any{"snapshot": map[string]any{"ok": true}})
		case "/v1/ui/config/set":
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			setKey, _ = in["key"].(string)
			setValue, _ = in["value"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	s := &appState{
		httpBase:         ts.URL,
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		session:          "s1",
	}
	if _, err, _ := handleTUICommand(s, "/config GET", nil, nil); err != nil {
		t.Fatalf("config GET failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/config SET agent.workdir /tmp/demo", nil, nil); err != nil {
		t.Fatalf("config SET failed: %v", err)
	}
	if setKey != "agent.workdir" || setValue != "/tmp/demo" {
		t.Fatalf("unexpected set payload: key=%q value=%q", setKey, setValue)
	}
}

func TestBookmarkCommandCaseInsensitiveSubcommands(t *testing.T) {
	dir := t.TempDir()
	s := &appState{
		bookmarkPath:     filepath.Join(dir, "bookmarks.json"),
		historyPath:      filepath.Join(dir, "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		session:          "s1",
		wsBase:           "ws://127.0.0.1:8200/v1/chat/ws",
		httpBase:         "http://127.0.0.1:8200",
	}
	if _, err, _ := handleTUICommand(s, "/bookmark ADD demo", nil, nil); err != nil {
		t.Fatalf("bookmark ADD failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/bookmark LIST", nil, nil); err != nil {
		t.Fatalf("bookmark LIST failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/bookmark USE demo", nil, nil); err != nil {
		t.Fatalf("bookmark USE failed: %v", err)
	}
}

func TestWorkbenchCommandCaseInsensitiveSubcommands(t *testing.T) {
	dir := t.TempDir()
	s := &appState{
		workbenchPath:    filepath.Join(dir, "workbenches.json"),
		historyPath:      filepath.Join(dir, "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		session:          "s1",
		wsBase:           "ws://127.0.0.1:8200/v1/chat/ws",
		httpBase:         "http://127.0.0.1:8200",
	}
	if _, err, _ := handleTUICommand(s, "/workbench SAVE demo", nil, nil); err != nil {
		t.Fatalf("workbench SAVE failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/workbench LIST", nil, nil); err != nil {
		t.Fatalf("workbench LIST failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/workbench LOAD demo", nil, nil); err != nil {
		t.Fatalf("workbench LOAD failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/workbench DELETE demo", nil, nil); err != nil {
		t.Fatalf("workbench DELETE failed: %v", err)
	}
}
