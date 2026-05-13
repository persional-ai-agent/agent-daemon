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
