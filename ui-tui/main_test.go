package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestParseEventSaveArgsAndFilter(t *testing.T) {
	since := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	until := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	path, format, st, ut, err := parseEventSaveArgs("/events save /tmp/e.ndjson ndjson since=" + since + " until=" + until)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}
	if path != "/tmp/e.ndjson" || format != "ndjson" {
		t.Fatalf("unexpected parse result: path=%s format=%s", path, format)
	}
	if st.IsZero() || ut.IsZero() {
		t.Fatal("expected since/until parsed")
	}
	events := []map[string]any{
		{"_captured_at": time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339), "type": "a"},
		{"_captured_at": time.Now().Add(-90 * time.Minute).UTC().Format(time.RFC3339), "type": "b"},
	}
	filtered := filterEventsByTime(events, st, ut)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered event, got %d", len(filtered))
	}
}

func TestFindLatestPendingApproval(t *testing.T) {
	msgs := []any{
		map[string]any{"role": "tool", "content": `{"status":"ok"}`},
		map[string]any{"role": "tool", "content": `{"status":"pending_approval","approval_id":"ap-123","tool_name":"terminal"}`},
	}
	id, payload := findLatestPendingApproval(msgs)
	if id != "ap-123" {
		t.Fatalf("unexpected approval id: %s", id)
	}
	if payload["status"] != "pending_approval" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestFindPendingApprovals(t *testing.T) {
	msgs := []any{
		map[string]any{"role": "tool", "content": `{"status":"pending_approval","approval_id":"ap-1","tool_name":"terminal","command":"rm -rf x"}`},
		map[string]any{"role": "tool", "content": `{"status":"pending_approval","approval_id":"ap-2","tool_name":"terminal","command":"chmod 777 y"}`},
	}
	out := findPendingApprovals(msgs, 2)
	if len(out) != 2 {
		t.Fatalf("expected 2 items, got %d", len(out))
	}
	if out[0]["approval_id"] != "ap-2" || out[1]["approval_id"] != "ap-1" {
		t.Fatalf("unexpected order: %+v", out)
	}
}

func TestParseStartupFlags(t *testing.T) {
	old := os.Getenv("AGENT_UI_TUI_FULLSCREEN")
	defer func() { _ = os.Setenv("AGENT_UI_TUI_FULLSCREEN", old) }()

	_ = os.Unsetenv("AGENT_UI_TUI_FULLSCREEN")
	noDoctor, fullscreen, fullscreenSet := parseStartupFlags([]string{"--no-doctor", "--fullscreen"})
	if !noDoctor || !fullscreen || !fullscreenSet {
		t.Fatalf("unexpected flags: noDoctor=%v fullscreen=%v fullscreenSet=%v", noDoctor, fullscreen, fullscreenSet)
	}

	_ = os.Setenv("AGENT_UI_TUI_FULLSCREEN", "1")
	noDoctor, fullscreen, fullscreenSet = parseStartupFlags(nil)
	if noDoctor || !fullscreen || !fullscreenSet {
		t.Fatalf("unexpected env parse: noDoctor=%v fullscreen=%v fullscreenSet=%v", noDoctor, fullscreen, fullscreenSet)
	}
}

func TestAddChatLineTruncateAndCap(t *testing.T) {
	s := newState()
	s.chatMaxLines = 2
	s.addChatLine("a")
	s.addChatLine(strings.Repeat("x", 500))
	s.addChatLine("b")
	if len(s.chatLog) != 2 {
		t.Fatalf("chatLog len=%d", len(s.chatLog))
	}
	if s.chatLog[0] == "a" {
		t.Fatalf("expected oldest line dropped: %+v", s.chatLog)
	}
	if len(s.chatLog[0]) > 303 {
		t.Fatalf("expected line truncated, got len=%d", len(s.chatLog[0]))
	}
}

func TestActionCommandByIndex(t *testing.T) {
	s := newState()
	s.fullscreen = false
	cmd, ok := actionCommandByIndex(s, 1)
	if !ok || cmd != "/tools" {
		t.Fatalf("idx1 cmd=%q ok=%v", cmd, ok)
	}
	cmd, ok = actionCommandByIndex(s, 10)
	if !ok || cmd != "/panel next" {
		t.Fatalf("idx10 cmd=%q ok=%v", cmd, ok)
	}
	cmd, ok = actionCommandByIndex(s, 11)
	if !ok || cmd != "/panel status" {
		t.Fatalf("idx11 cmd=%q ok=%v", cmd, ok)
	}
	cmd, ok = actionCommandByIndex(s, 12)
	if !ok || cmd != "/workbench list" {
		t.Fatalf("idx12 cmd=%q ok=%v", cmd, ok)
	}
	cmd, ok = actionCommandByIndex(s, 13)
	if !ok || cmd != "/workflow list" {
		t.Fatalf("idx13 cmd=%q ok=%v", cmd, ok)
	}
	cmd, ok = actionCommandByIndex(s, 14)
	if !ok || cmd != "/fullscreen on" {
		t.Fatalf("idx14 cmd=%q ok=%v", cmd, ok)
	}
	s.fullscreen = true
	cmd, ok = actionCommandByIndex(s, 14)
	if !ok || cmd != "/fullscreen off" {
		t.Fatalf("idx14 fullscreen cmd=%q ok=%v", cmd, ok)
	}
	_, ok = actionCommandByIndex(s, 99)
	if ok {
		t.Fatal("expected invalid index")
	}
}

func TestTimelineSlice(t *testing.T) {
	s := newState()
	s.addChatLine("u1")
	s.addChatLine("a1")
	s.addChatLine("u2")
	got := s.timelineSlice(2)
	if len(got) != 2 {
		t.Fatalf("timelineSlice len=%d", len(got))
	}
	if got[0] != "a1" || got[1] != "u2" {
		t.Fatalf("timelineSlice=%v", got)
	}
}

func TestPanelCycle(t *testing.T) {
	if nextPanel("overview") != "dashboard" {
		t.Fatalf("next panel mismatch")
	}
	if prevPanel("overview") != "diag" {
		t.Fatalf("prev panel mismatch")
	}
	names := panelNames()
	foundApprovals := false
	for _, n := range names {
		if n == "approvals" {
			foundApprovals = true
			break
		}
	}
	if !foundApprovals {
		t.Fatalf("panelNames missing approvals: %v", names)
	}
}

func TestPanelSelectionHelpers(t *testing.T) {
	sessions := []any{map[string]any{"session_id": "s1"}}
	sid, ok := selectSessionIDFromPanelData(sessions, 1)
	if !ok || sid != "s1" {
		t.Fatalf("selectSessionID failed sid=%q ok=%v", sid, ok)
	}
	tools := []any{"terminal"}
	tool, ok := selectToolNameFromPanelData(tools, 1)
	if !ok || tool != "terminal" {
		t.Fatalf("selectToolName failed tool=%q ok=%v", tool, ok)
	}
	approvals := []any{map[string]any{"approval_id": "ap-1"}}
	aid, ok := selectApprovalIDFromPanelData(approvals, 1)
	if !ok || aid != "ap-1" {
		t.Fatalf("selectApprovalID failed id=%q ok=%v", aid, ok)
	}
}

func TestWorkbenchProfileSaveLoadDelete(t *testing.T) {
	dir := t.TempDir()
	s := newState()
	s.workbenchPath = filepath.Join(dir, "ui-tui-workbenches.json")
	s.session = "s-1"
	s.wsBase = "ws://x/ws"
	s.httpBase = "http://x"
	s.fullscreen = true
	s.fullscreenPanel = "dashboard"
	s.panelAutoRefresh = true
	s.panelRefreshSec = 9
	s.viewMode = "json"
	if err := s.saveWorkbench("daily"); err != nil {
		t.Fatal(err)
	}
	list, err := s.loadWorkbenchProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "daily" {
		t.Fatalf("unexpected workbench list: %+v", list)
	}
	s2 := newState()
	s2.workbenchPath = s.workbenchPath
	if err := s2.loadWorkbench("daily"); err != nil {
		t.Fatal(err)
	}
	if s2.session != "s-1" || s2.fullscreenPanel != "dashboard" || s2.panelRefreshSec != 9 || s2.viewMode != "json" {
		t.Fatalf("workbench load mismatch: %+v", s2)
	}
	if err := s2.deleteWorkbench("daily"); err != nil {
		t.Fatal(err)
	}
	list, err = s2.loadWorkbenchProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list after delete: %+v", list)
	}
}

func TestParseWorkflowCommands(t *testing.T) {
	got := parseWorkflowCommands("tools; panel next ;/refresh;;")
	if len(got) != 3 {
		t.Fatalf("unexpected len=%d cmds=%v", len(got), got)
	}
	if got[0] != "/tools" || got[1] != "/panel next" || got[2] != "/refresh" {
		t.Fatalf("unexpected cmds=%v", got)
	}
}

func TestWorkflowSaveRunDelete(t *testing.T) {
	dir := t.TempDir()
	s := newState()
	s.workflowPath = filepath.Join(dir, "ui-tui-workflows.json")
	cmds := []string{"/tools", "/panel next"}
	if err := s.saveWorkflow("daily", cmds); err != nil {
		t.Fatal(err)
	}
	wf, ok, err := s.getWorkflow("daily")
	if err != nil || !ok {
		t.Fatalf("getWorkflow err=%v ok=%v", err, ok)
	}
	if len(wf.Commands) != 2 || wf.Commands[0] != "/tools" {
		t.Fatalf("unexpected workflow %+v", wf)
	}
	if err := s.deleteWorkflow("daily"); err != nil {
		t.Fatal(err)
	}
	_, ok, err = s.getWorkflow("daily")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected workflow deleted")
	}
}

func TestRuntimeStatePersistsFullscreenPanel(t *testing.T) {
	dir := t.TempDir()
	s := newState()
	s.statePath = filepath.Join(dir, "ui-tui-state.json")
	s.session = "sid-x"
	s.wsBase = "ws://127.0.0.1:8080/v1/chat/ws"
	s.httpBase = "http://127.0.0.1:8080"
	s.fullscreen = true
	s.fullscreenPanel = "dashboard"
	s.panelAutoRefresh = true
	s.panelRefreshSec = 12
	if err := s.saveRuntimeState(); err != nil {
		t.Fatal(err)
	}
	s2 := newState()
	s2.statePath = s.statePath
	s2.fullscreen = false
	s2.fullscreenPanel = "overview"
	s2.panelAutoRefresh = false
	s2.panelRefreshSec = 3
	s2.loadRuntimeState()
	if !s2.fullscreen || s2.fullscreenPanel != "dashboard" || !s2.panelAutoRefresh || s2.panelRefreshSec != 12 {
		t.Fatalf("runtime restore failed: fullscreen=%v panel=%s auto=%v sec=%d", s2.fullscreen, s2.fullscreenPanel, s2.panelAutoRefresh, s2.panelRefreshSec)
	}
}

func TestLoadRuntimeStateCorruptBackup(t *testing.T) {
	dir := t.TempDir()
	s := newState()
	s.statePath = filepath.Join(dir, "ui-tui-state.json")
	if err := os.WriteFile(s.statePath, []byte("{broken-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.loadRuntimeState()
	if _, err := os.Stat(s.statePath); err != nil {
		t.Fatalf("state file should be recreated: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "ui-tui-state.json.corrupt.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("expected corrupt backup file")
	}
}

func TestSendTurnReconnect(t *testing.T) {
	var tries int32
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()
		var req map[string]any
		if err := conn.ReadJSON(&req); err != nil {
			t.Fatalf("read req: %v", err)
		}
		try := atomic.AddInt32(&tries, 1)
		_ = conn.WriteJSON(map[string]any{"type": "assistant_message", "content": "partial"})
		if try == 1 {
			return
		}
		if resume, _ := req["resume"].(bool); !resume {
			t.Fatalf("expected resume on reconnect, got req=%+v", req)
		}
		_ = conn.WriteJSON(map[string]any{"type": "result", "final_response": "ok"})
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	s := newState()
	s.wsBase = wsURL
	s.wsReadTimeout = 2 * time.Second
	s.wsTurnTimeout = 10 * time.Second
	s.wsMaxReconnect = 2
	assistantPartial := 0
	resultSeen := 0
	if err := s.sendTurn("ping", func(evt map[string]any) {
		typ, _ := evt["type"].(string)
		if typ == "assistant_message" && evt["content"] == "partial" {
			assistantPartial++
		}
		if typ == "result" {
			resultSeen++
		}
	}); err != nil {
		t.Fatalf("sendTurn failed: %v", err)
	}
	if assistantPartial != 1 {
		t.Fatalf("expected deduped assistant partial=1, got %d", assistantPartial)
	}
	if resultSeen != 1 {
		t.Fatalf("expected resultSeen=1, got %d", resultSeen)
	}
	if s.reconnectCount < 1 {
		t.Fatalf("expected reconnectCount>=1, got %d", s.reconnectCount)
	}
	if strings.TrimSpace(s.fallbackHint) == "" {
		t.Fatalf("expected fallbackHint recorded on reconnect")
	}
	if s.lastTurnID == "" {
		t.Fatal("expected lastTurnID to be set")
	}
}

func TestExportDiagnostics(t *testing.T) {
	s := newState()
	s.session = "diag-session"
	s.lastTurnID = "turn-1"
	s.reconnectState = "degraded"
	s.reconnectCount = 2
	s.fallbackHint = "ws reconnect attempt=1 reason=WS_CLOSED"
	s.lastErrorCode = "network"
	s.lastErrorText = "connection reset"
	s.eventLog = []map[string]any{{"type": "assistant_message", "content": "hi"}}
	path := filepath.Join(t.TempDir(), "diag.json")
	if err := s.exportDiagnostics(path); err != nil {
		t.Fatalf("export diagnostics: %v", err)
	}
	bs, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read diag file: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(bs, &payload); err != nil {
		t.Fatalf("unmarshal diag file: %v", err)
	}
	if payload["schema_version"] != "diag.v1" {
		t.Fatalf("unexpected schema_version: %v", payload["schema_version"])
	}
	if payload["source"] != "ui-tui" {
		t.Fatalf("unexpected source: %v", payload["source"])
	}
	if payload["session_id"] != "diag-session" {
		t.Fatalf("unexpected session_id: %v", payload["session_id"])
	}
	if payload["reconnect_count"] != float64(2) {
		t.Fatalf("unexpected reconnect_count: %v", payload["reconnect_count"])
	}
}

func TestRunDoctor(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/v1/ui/sessions/", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}, "stats": map[string]any{}})
	})
	mux.HandleFunc("/v1/ui/approval/confirm", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "pending approval not found", http.StatusBadRequest)
	})
	mux.HandleFunc("/v1/chat/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		_ = conn.Close()
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	s := newState()
	s.httpBase = ts.URL
	s.wsBase = "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/chat/ws"
	items, ok := s.runDoctor()
	if !ok {
		t.Fatalf("expected doctor ok, items=%+v", items)
	}
	if len(items) < 5 {
		t.Fatalf("expected >=5 checks, got %d", len(items))
	}
}

func TestRunDoctorMissingApprovalEndpoint(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/v1/ui/sessions/", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}, "stats": map[string]any{}})
	})
	mux.HandleFunc("/v1/chat/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		_ = conn.Close()
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	s := newState()
	s.httpBase = ts.URL
	s.wsBase = "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/chat/ws"
	items, ok := s.runDoctor()
	if ok {
		t.Fatalf("expected doctor to fail when approval endpoint missing, items=%+v", items)
	}
	found := false
	for _, it := range items {
		if it.Name == "approval_confirm" && it.Status == "fail" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected approval_confirm fail, items=%+v", items)
	}
}

func TestHTTPJSONParsesUIErrorEnvelope(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":    "invalid_argument",
				"message": "bad input",
			},
			"api_version": "v1",
			"compat":      "2026-05-13",
		})
	}))
	defer ts.Close()
	_, err := httpJSON(http.MethodGet, ts.URL, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid_argument") {
		t.Fatalf("expected parsed ui error, got %v", err)
	}
}

func TestUIPayloadFallsBackToResultEnvelope(t *testing.T) {
	out := map[string]any{
		"ok": true,
		"result": map[string]any{
			"snapshot": "from-result",
		},
		"status": "legacy",
	}
	got := uiPayload(out, "snapshot")
	res, _ := got.(map[string]any)
	if res["snapshot"] != "from-result" {
		t.Fatalf("unexpected payload: %v", got)
	}
}

func TestUIPayloadFallbackLegacyField(t *testing.T) {
	out := map[string]any{
		"ok":     true,
		"status": "legacy",
	}
	got := uiPayload(out, "status")
	if got != "legacy" {
		t.Fatalf("unexpected payload: %v", got)
	}
}
