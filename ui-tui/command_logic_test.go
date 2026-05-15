package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
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
		{"/targets a b", "用法: /targets [platform]"},
		{"/sethome", "用法: /sethome <platform:chat_id>|<platform> <chat_id>"},
		{"/resume", "用法: /resume <session_id>"},
		{"/compress x", "用法: /compress [tail_messages]"},
		{"/recover", "用法: /recover context"},
		{"/whoami", "用法: /whoami <platform> <user_id>"},
		{"/continuity x y", "用法: /continuity [off|user_id|user_name]"},
		{"/setid a b", "用法: /setid <platform> <user_id> <global_user_id>"},
		{"/unsetid a", "用法: /unsetid <platform> <user_id>"},
		{"/resolve telegram", "用法: /resolve <platform> <chat_type> <chat_id> <user_id> [user_name]"},
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
	if err == nil || err.Error() != "用法: /gateway status|enable|disable|resolve <platform> <chat_type> <chat_id> <user_id> [user_name]" {
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
	_, err, _ = handleTUICommand(s, "/reconnect", nil, nil)
	if err == nil || err.Error() != "用法: /reconnect status|on|off|now|timeout ..." {
		t.Fatalf("unexpected /reconnect error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/panel auto maybe", nil, nil)
	if err == nil || err.Error() != "用法: /panel auto on|off" {
		t.Fatalf("unexpected /panel auto error: %v", err)
	}
	_, err, _ = handleTUICommand(s, "/panel interval 0", nil, nil)
	if err == nil || err.Error() != "panel 刷新间隔必须在 1..300 秒之间" {
		t.Fatalf("unexpected /panel interval error: %v", err)
	}
	if _, err, _ = handleTUICommand(s, "/panel interval   10", nil, nil); err != nil {
		t.Fatalf("unexpected spaced /panel interval error: %v", err)
	}
	if s.panelRefreshSec != 10 {
		t.Fatalf("panelRefreshSec=%d", s.panelRefreshSec)
	}
	if _, err, _ = handleTUICommand(s, "/panel interval +11", nil, nil); err != nil {
		t.Fatalf("unexpected plus /panel interval error: %v", err)
	}
	if s.panelRefreshSec != 11 {
		t.Fatalf("panelRefreshSec=%d", s.panelRefreshSec)
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

func TestGatewayResolveCommand(t *testing.T) {
	lastPath := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.String()
		switch r.URL.Path {
		case "/v1/ui/gateway/session/resolve":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"result": map[string]any{
					"route_session":    "telegram:group:1001",
					"resolved_session": "global:user:uid:u1",
				},
			})
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
	if _, err, _ := handleTUICommand(s, "/gateway resolve telegram group 1001 u1 Alice", nil, nil); err != nil {
		t.Fatalf("gateway resolve failed: %v", err)
	}
	if !strings.HasPrefix(lastPath, "/v1/ui/gateway/session/resolve?platform=telegram") {
		t.Fatalf("unexpected resolve path: %s", lastPath)
	}
}

func TestResolveCommand(t *testing.T) {
	lastPath := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.String()
		switch r.URL.Path {
		case "/v1/ui/gateway/session/resolve":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"result": map[string]any{
					"route_session":    "telegram:group:1001",
					"resolved_session": "global:user:uid:u1",
				},
			})
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
	if _, err, _ := handleTUICommand(s, "/resolve telegram group 1001 u1 Alice", nil, nil); err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if !strings.HasPrefix(lastPath, "/v1/ui/gateway/session/resolve?platform=telegram") {
		t.Fatalf("unexpected resolve path: %s", lastPath)
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

func TestPrettyAndFullscreenCaseInsensitiveArgs(t *testing.T) {
	s := &appState{
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
	}
	if _, err, _ := handleTUICommand(s, "/pretty ON", nil, nil); err != nil {
		t.Fatalf("pretty ON failed: %v", err)
	}
	if !s.pretty {
		t.Fatal("expected pretty=true")
	}
	if _, err, _ := handleTUICommand(s, "/fullscreen OFF", nil, nil); err != nil {
		t.Fatalf("fullscreen OFF failed: %v", err)
	}
	if s.fullscreen {
		t.Fatal("expected fullscreen=false")
	}
}

func TestReconnectTimeoutCommandCaseInsensitive(t *testing.T) {
	s := &appState{
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		timeoutAction:    "wait",
	}
	if _, err, _ := handleTUICommand(s, "/reconnect timeout CANCEL", nil, nil); err != nil {
		t.Fatalf("reconnect timeout CANCEL failed: %v", err)
	}
	if s.timeoutAction != "cancel" {
		t.Fatalf("timeoutAction=%q", s.timeoutAction)
	}
	if _, err, _ := handleTUICommand(s, "/reconnect TIMEOUT WAIT", nil, nil); err != nil {
		t.Fatalf("reconnect TIMEOUT WAIT failed: %v", err)
	}
	if s.timeoutAction != "wait" {
		t.Fatalf("timeoutAction=%q", s.timeoutAction)
	}
}

func TestReconnectCommandCaseInsensitiveSubcommands(t *testing.T) {
	s := &appState{
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		wsBase:           "ws://127.0.0.1:1/v1/chat/ws",
	}
	if _, err, _ := handleTUICommand(s, "/reconnect OFF", nil, nil); err != nil {
		t.Fatalf("reconnect OFF failed: %v", err)
	}
	if s.reconnectEnabled {
		t.Fatal("expected reconnectEnabled=false")
	}
	if _, err, _ := handleTUICommand(s, "/reconnect ON", nil, nil); err != nil {
		t.Fatalf("reconnect ON failed: %v", err)
	}
	if !s.reconnectEnabled {
		t.Fatal("expected reconnectEnabled=true")
	}
	if _, err, _ := handleTUICommand(s, "/reconnect STATUS", nil, nil); err != nil {
		t.Fatalf("reconnect STATUS failed: %v", err)
	}
}

func TestDiagExportCaseInsensitiveSubcommand(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "diag.json")
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
	if _, err, _ := handleTUICommand(s, "/diag EXPORT "+outFile, nil, nil); err != nil {
		t.Fatalf("diag EXPORT failed: %v", err)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("expected diagnostics file: %v", err)
	}
}

func TestParseEventSaveArgsCaseInsensitive(t *testing.T) {
	path, format, _, _, err := parseEventSaveArgs("/events SAVE /tmp/events.ndjson NDJSON")
	if err != nil {
		t.Fatalf("parseEventSaveArgs failed: %v", err)
	}
	if path != "/tmp/events.ndjson" || format != "ndjson" {
		t.Fatalf("unexpected parse result: path=%q format=%q", path, format)
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
	if _, err, _ := handleTUICommand(s, "/config set agent.note hello world", nil, nil); err != nil {
		t.Fatalf("config set with spaced value failed: %v", err)
	}
	if setKey != "agent.note" || setValue != "hello world" {
		t.Fatalf("unexpected spaced payload: key=%q value=%q", setKey, setValue)
	}
}

func TestTargetsAndSetHomeCommands(t *testing.T) {
	lastMethod := ""
	lastPath := ""
	lastBody := map[string]any{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		lastPath = r.URL.String()
		switch r.URL.Path {
		case "/v1/ui/targets":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"targets": []map[string]any{{"target": "telegram:1001"}},
			})
		case "/v1/ui/targets/home":
			_ = json.NewDecoder(r.Body).Decode(&lastBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":     true,
				"result": map[string]any{"target": "telegram:1001"},
			})
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

	if _, err, _ := handleTUICommand(s, "/targets telegram", nil, nil); err != nil {
		t.Fatalf("targets command failed: %v", err)
	}
	if lastMethod != http.MethodGet || lastPath != "/v1/ui/targets?platform=telegram" {
		t.Fatalf("unexpected targets request: %s %s", lastMethod, lastPath)
	}

	if _, err, _ := handleTUICommand(s, "/sethome telegram:1001", nil, nil); err != nil {
		t.Fatalf("sethome target command failed: %v", err)
	}
	if lastMethod != http.MethodPost || lastPath != "/v1/ui/targets/home" {
		t.Fatalf("unexpected sethome request: %s %s", lastMethod, lastPath)
	}
	if got, _ := lastBody["target"].(string); got != "telegram:1001" {
		t.Fatalf("unexpected sethome target body: %+v", lastBody)
	}

	lastBody = map[string]any{}
	if _, err, _ := handleTUICommand(s, "/sethome telegram 2002", nil, nil); err != nil {
		t.Fatalf("sethome platform chat command failed: %v", err)
	}
	if got, _ := lastBody["platform"].(string); got != "telegram" {
		t.Fatalf("unexpected platform in body: %+v", lastBody)
	}
	if got, _ := lastBody["chat_id"].(string); got != "2002" {
		t.Fatalf("unexpected chat_id in body: %+v", lastBody)
	}
}

func TestContinuityAndIdentityCommands(t *testing.T) {
	lastMethod := ""
	lastPath := ""
	lastBody := map[string]any{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		lastPath = r.URL.String()
		switch r.URL.Path {
		case "/v1/ui/gateway/continuity":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{"continuity_mode": "off"}})
				return
			}
			_ = json.NewDecoder(r.Body).Decode(&lastBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{"continuity_mode": "user_name"}})
		case "/v1/ui/gateway/identity":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{"platform": "telegram", "user_id": "u1", "global_id": "gid-1"}})
				return
			}
			_ = json.NewDecoder(r.Body).Decode(&lastBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{"updated": true}})
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

	if _, err, _ := handleTUICommand(s, "/continuity", nil, nil); err != nil {
		t.Fatalf("/continuity get failed: %v", err)
	}
	if lastMethod != http.MethodGet || lastPath != "/v1/ui/gateway/continuity" {
		t.Fatalf("unexpected continuity get request: %s %s", lastMethod, lastPath)
	}
	if _, err, _ := handleTUICommand(s, "/continuity user_name", nil, nil); err != nil {
		t.Fatalf("/continuity set failed: %v", err)
	}
	if got, _ := lastBody["mode"].(string); got != "user_name" {
		t.Fatalf("unexpected continuity body: %+v", lastBody)
	}
	if _, err, _ := handleTUICommand(s, "/whoami telegram u1", nil, nil); err != nil {
		t.Fatalf("/whoami failed: %v", err)
	}
	if lastMethod != http.MethodGet || lastPath != "/v1/ui/gateway/identity?platform=telegram&user_id=u1" {
		t.Fatalf("unexpected whoami request: %s %s", lastMethod, lastPath)
	}
	if _, err, _ := handleTUICommand(s, "/setid telegram u1 gid-2", nil, nil); err != nil {
		t.Fatalf("/setid failed: %v", err)
	}
	if got, _ := lastBody["global_id"].(string); got != "gid-2" {
		t.Fatalf("unexpected setid body: %+v", lastBody)
	}
	if _, err, _ := handleTUICommand(s, "/unsetid telegram u1", nil, nil); err != nil {
		t.Fatalf("/unsetid failed: %v", err)
	}
	if lastMethod != http.MethodDelete || lastPath != "/v1/ui/gateway/identity" {
		t.Fatalf("unexpected unsetid request: %s %s", lastMethod, lastPath)
	}
}

func TestNewResetUsageUndoRetrySkillsModelPersonalityCommands(t *testing.T) {
	lastMethod := ""
	lastPath := ""
	lastBody := map[string]any{}
	undoCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		lastPath = r.URL.String()
		switch r.URL.Path {
		case "/v1/ui/sessions/s1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"messages": []map[string]any{{"role": "user", "content": "hello"}},
				"stats":    map[string]any{"total_messages": 1},
				"count":    1,
			})
		case "/v1/ui/sessions/s3":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"messages": []map[string]any{{"role": "assistant", "content": "world"}},
				"stats":    map[string]any{"total_messages": 1},
				"count":    1,
			})
		case "/v1/ui/sessions/s2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"messages": []map[string]any{{"role": "assistant", "content": "after-undo"}},
				"stats":    map[string]any{"total_messages": 1},
				"count":    1,
			})
		case "/v1/ui/sessions/undo":
			undoCalls++
			_ = json.NewDecoder(r.Body).Decode(&lastBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":     true,
				"result": map[string]any{"new_session_id": "s2", "undone": true, "removed_messages": 1},
			})
		case "/v1/ui/sessions/compress":
			_ = json.NewDecoder(r.Body).Decode(&lastBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"result": map[string]any{
					"session_id":       "s1",
					"compressed":       true,
					"before_messages":  100,
					"after_messages":   20,
					"dropped_messages": 80,
					"keep_last_n":      20,
				},
			})
		case "/v1/ui/skills":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skills": []map[string]any{{"name": "skill-a"}}})
		case "/v1/ui/model":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "model": map[string]any{"provider": "openai", "model": "gpt-5"}})
		case "/v1/ui/model/set":
			_ = json.NewDecoder(r.Body).Decode(&lastBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{"provider": "openai", "model": "gpt-5-mini"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	wsUpgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()
		var req map[string]any
		if err := conn.ReadJSON(&req); err != nil {
			t.Fatalf("read req: %v", err)
		}
		_ = conn.WriteJSON(map[string]any{"type": "session", "session_id": req["session_id"]})
		_ = conn.WriteJSON(map[string]any{"type": "result", "final_response": "ok"})
	}))
	defer wsServer.Close()
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")

	s := &appState{
		httpBase:         ts.URL,
		wsBase:           wsURL,
		historyPath:      filepath.Join(t.TempDir(), "history.log"),
		historyMaxLines:  100,
		eventMaxItems:    100,
		panelData:        map[string]any{},
		fullscreenPanel:  "overview",
		panelRefreshSec:  8,
		reconnectEnabled: true,
		session:          "s1",
		systemPrompt:     "base-prompt",
	}

	if _, err, _ := handleTUICommand(s, "/new s-new", nil, nil); err != nil {
		t.Fatalf("/new failed: %v", err)
	}
	if s.session != "s-new" {
		t.Fatalf("session=%q", s.session)
	}
	if _, err, _ := handleTUICommand(s, "/reset s-reset", nil, nil); err != nil {
		t.Fatalf("/reset failed: %v", err)
	}
	if s.session != "s-reset" {
		t.Fatalf("session=%q", s.session)
	}
	s.session = "s1"

	if _, err, _ := handleTUICommand(s, "/usage s1", nil, nil); err != nil {
		t.Fatalf("/usage failed: %v", err)
	}
	if lastPath != "/v1/ui/sessions/s1?offset=0&limit=1" {
		t.Fatalf("unexpected usage request path: %s", lastPath)
	}
	if _, err, _ := handleTUICommand(s, "/resume s3", nil, nil); err != nil {
		t.Fatalf("/resume failed: %v", err)
	}
	if s.session != "s3" {
		t.Fatalf("session after resume=%q", s.session)
	}
	if _, err, _ := handleTUICommand(s, "/compress", nil, nil); err != nil {
		t.Fatalf("/compress failed: %v", err)
	}
	if got, _ := lastBody["session_id"].(string); got != "s3" {
		t.Fatalf("unexpected compress body: %+v", lastBody)
	}

	if _, err, _ := handleTUICommand(s, "/undo", nil, nil); err != nil {
		t.Fatalf("/undo failed: %v", err)
	}
	if undoCalls != 1 {
		t.Fatalf("undoCalls=%d", undoCalls)
	}
	if s.session != "s2" {
		t.Fatalf("session after undo=%q", s.session)
	}

	s.session = "s1"
	if _, err, _ := handleTUICommand(s, "/retry", nil, nil); err != nil {
		t.Fatalf("/retry failed: %v", err)
	}
	if undoCalls != 2 {
		t.Fatalf("retry should trigger undo, undoCalls=%d", undoCalls)
	}

	if _, err, _ := handleTUICommand(s, "/reload", nil, nil); err != nil {
		t.Fatalf("/reload failed: %v", err)
	}
	if lastPath != "/v1/ui/sessions/s2?offset=0&limit=500" {
		t.Fatalf("unexpected reload request path: %s", lastPath)
	}

	if _, err, _ := handleTUICommand(s, "/skills", nil, nil); err != nil {
		t.Fatalf("/skills failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/model", nil, nil); err != nil {
		t.Fatalf("/model failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/model openai:gpt-5-mini", nil, nil); err != nil {
		t.Fatalf("/model set failed: %v", err)
	}
	if lastMethod != http.MethodPost || lastPath != "/v1/ui/model/set" {
		t.Fatalf("unexpected model set request: %s %s", lastMethod, lastPath)
	}
	if got, _ := lastBody["provider"].(string); got != "openai" {
		t.Fatalf("unexpected provider in body: %+v", lastBody)
	}
	if got, _ := lastBody["model"].(string); got != "gpt-5-mini" {
		t.Fatalf("unexpected model in body: %+v", lastBody)
	}

	if _, err, _ := handleTUICommand(s, "/personality show", nil, nil); err != nil {
		t.Fatalf("/personality show failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/personality custom", nil, nil); err != nil {
		t.Fatalf("/personality set failed: %v", err)
	}
	if s.systemPrompt != "custom" {
		t.Fatalf("systemPrompt=%q", s.systemPrompt)
	}
	if _, err, _ := handleTUICommand(s, "/personality reset", nil, nil); err != nil {
		t.Fatalf("/personality reset failed: %v", err)
	}
	if strings.TrimSpace(s.systemPrompt) == "" {
		t.Fatal("expected non-empty default system prompt after reset")
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

func TestWorkflowCommandCaseInsensitiveSubcommands(t *testing.T) {
	dir := t.TempDir()
	s := &appState{
		workflowPath:     filepath.Join(dir, "workflows.json"),
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
	if _, err, _ := handleTUICommand(s, "/workflow SAVE demo /tools;/panel next", nil, nil); err != nil {
		t.Fatalf("workflow SAVE failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/workflow LIST", nil, nil); err != nil {
		t.Fatalf("workflow LIST failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/workflow RUN demo dry", nil, nil); err != nil {
		t.Fatalf("workflow RUN failed: %v", err)
	}
	if _, err, _ := handleTUICommand(s, "/workflow DELETE demo", nil, nil); err != nil {
		t.Fatalf("workflow DELETE failed: %v", err)
	}
}

func TestCanonicalInputAliasesCaseInsensitive(t *testing.T) {
	if got := canonicalInput("/Q"); got != "/quit" {
		t.Fatalf("/Q alias mismatch: %q", got)
	}
	if got := canonicalInput("/H"); got != "/help" {
		t.Fatalf("/H alias mismatch: %q", got)
	}
	if got := canonicalInput("/GW status"); got != "/gateway status" {
		t.Fatalf("/GW alias mismatch: %q", got)
	}
	if got := canonicalInput("/CFG get"); got != "/config get" {
		t.Fatalf("/CFG alias mismatch: %q", got)
	}
	if got := canonicalInput("/SESS 10"); got != "/sessions 10" {
		t.Fatalf("/SESS alias mismatch: %q", got)
	}
	if got := canonicalInput("/WB save demo"); got != "/workbench save demo" {
		t.Fatalf("/WB alias mismatch: %q", got)
	}
	if got := canonicalInput("/WF list"); got != "/workflow list" {
		t.Fatalf("/WF alias mismatch: %q", got)
	}
	if got := canonicalInput("/BM add demo"); got != "/bookmark add demo" {
		t.Fatalf("/BM alias mismatch: %q", got)
	}
	if got := canonicalInput("/GATEWAY status"); got != "/gateway status" {
		t.Fatalf("/GATEWAY canonical mismatch: %q", got)
	}
	if got := canonicalInput("/CONFIG SET agent.note Hello World"); got != "/config SET agent.note Hello World" {
		t.Fatalf("/CONFIG canonical mismatch: %q", got)
	}
	if got := canonicalInput("SHOW sid-1"); got != "/show sid-1" {
		t.Fatalf("SHOW alias mismatch: %q", got)
	}
	if got := canonicalInput("SESSIONS 5"); got != "/sessions 5" {
		t.Fatalf("SESSIONS alias mismatch: %q", got)
	}
	if got := canonicalInput("TOOL apply_patch"); got != "/tool apply_patch" {
		t.Fatalf("TOOL alias mismatch: %q", got)
	}
	if got := canonicalInput("GW status"); got != "/gateway status" {
		t.Fatalf("GW alias mismatch: %q", got)
	}
	if got := canonicalInput("gw status"); got != "/gateway status" {
		t.Fatalf("gw alias mismatch: %q", got)
	}
	if got := canonicalInput("CFG get"); got != "/config get" {
		t.Fatalf("CFG alias mismatch: %q", got)
	}
	if got := canonicalInput("cfg get"); got != "/config get" {
		t.Fatalf("cfg alias mismatch: %q", got)
	}
	if got := canonicalInput("stop"); got != "/cancel" {
		t.Fatalf("stop alias mismatch: %q", got)
	}
	if got := canonicalInput("/STOP"); got != "/cancel" {
		t.Fatalf("/STOP alias mismatch: %q", got)
	}
}

func TestIsContextLimitError(t *testing.T) {
	cases := []struct {
		errText string
		want    bool
	}{
		{"openai api error (400): {\"error\":{\"type\":\"exceed_context_size_error\"}}", true},
		{"request (32985 tokens) exceeds the available context size (32768 tokens)", true},
		{"network timeout", false},
	}
	for _, tc := range cases {
		got := isContextLimitError(fmt.Errorf("%s", tc.errText))
		if got != tc.want {
			t.Fatalf("err=%q got=%v want=%v", tc.errText, got, tc.want)
		}
	}
}

func TestAutoRecoverContextSession(t *testing.T) {
	s := &appState{
		session:         "old-session",
		lastShowSession: "old-session",
		statePath:       filepath.Join(t.TempDir(), "runtime.json"),
	}
	prev, next := autoRecoverContextSession(s)
	if prev != "old-session" {
		t.Fatalf("unexpected prev session: %q", prev)
	}
	if next == "" || next == prev {
		t.Fatalf("expected new session id, got %q", next)
	}
	if s.session != next || s.lastShowSession != next {
		t.Fatalf("session state not updated: session=%q last_show=%q next=%q", s.session, s.lastShowSession, next)
	}
}

func TestCompressSessionForRetry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/ui/sessions/compress" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"result": map[string]any{
				"dropped_messages": 7,
			},
		})
	}))
	defer ts.Close()
	s := &appState{httpBase: ts.URL, session: "s1"}
	dropped, err := compressSessionForRetry(s, 20)
	if err != nil {
		t.Fatalf("compressSessionForRetry error: %v", err)
	}
	if dropped != 7 {
		t.Fatalf("unexpected dropped count: %d", dropped)
	}
}
