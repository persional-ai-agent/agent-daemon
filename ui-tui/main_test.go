package main

import (
	"net/http"
	"net/http/httptest"
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
	seen := 0
	if err := s.sendTurn("ping", func(_ map[string]any) { seen++ }); err != nil {
		t.Fatalf("sendTurn failed: %v", err)
	}
	if seen < 2 {
		t.Fatalf("expected >=2 events, got %d", seen)
	}
}
