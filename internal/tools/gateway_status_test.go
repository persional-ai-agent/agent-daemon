package tools

import (
	"strings"
	"testing"
)

func TestGatewayStatusHelpers(t *testing.T) {
	s := GatewayStatusSnapshot{
		Platform:       "telegram",
		RouteSession:   "route-1",
		ActiveSession:  "active-1",
		QueueLen:       2,
		Paired:         true,
		ContinuityMode: "user_id",
		MappedSession:  "mapped-1",
		MessageCount:   12,
		Running:        true,
		LastApprovalID: "ap-1",
	}
	payload := BuildGatewayStatusPayload(s)
	if payload["platform"] != "telegram" || payload["running"] != true || payload["message_count"] != 12 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	text := RenderGatewayStatusText(s)
	if !strings.Contains(text, "platform: telegram") || !strings.Contains(text, "last_approval_id: ap-1") {
		t.Fatalf("unexpected status text: %q", text)
	}
}

func TestGatewayDiagnosticsFallback(t *testing.T) {
	diag := BuildGatewayDiagnosticsFallback([]string{"s2", "s1"}, -1, true, false)
	if diag["uptime_sec"] != int64(0) {
		t.Fatalf("unexpected uptime: %+v", diag)
	}
	if diag["active_run_count"] != 2 {
		t.Fatalf("unexpected count: %+v", diag)
	}
	ids, _ := diag["active_session_ids"].([]string)
	if len(ids) != 2 || ids[0] != "s1" || ids[1] != "s2" {
		t.Fatalf("unexpected ids: %+v", diag["active_session_ids"])
	}
}

func TestExtractGatewayStatusSnapshot(t *testing.T) {
	src := map[string]any{
		"platform":         "telegram",
		"route_session":    "r1",
		"active_session":   "a1",
		"queue":            float64(3),
		"paired":           true,
		"continuity_mode":  "user_id",
		"mapped_session":   "m1",
		"message_count":    9,
		"running":          true,
		"last_approval_id": "ap-1",
	}
	got := ExtractGatewayStatusSnapshot(src)
	if got.Platform != "telegram" || got.QueueLen != 3 || got.ContinuityMode != "user_id" || got.LastApprovalID != "ap-1" {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

func TestNormalizeGatewayStatusMap(t *testing.T) {
	raw := map[string]any{"enabled": true, "running": false}
	got := NormalizeGatewayStatusMap(raw)
	if got["enabled"] != true || got["running"] != false {
		t.Fatalf("unexpected normalized map: %+v", got)
	}
	if _, ok := got["platform"]; ok {
		t.Fatalf("should not inject empty snapshot fields: %+v", got)
	}

	raw = map[string]any{"platform": "telegram", "route_session": "r1"}
	got = NormalizeGatewayStatusMap(raw)
	if got["platform"] != "telegram" || got["route_session"] != "r1" {
		t.Fatalf("unexpected normalized map with snapshot: %+v", got)
	}
	if _, ok := got["active_session"]; !ok {
		t.Fatalf("expected normalized snapshot fields merged: %+v", got)
	}
}

func TestNormalizeGatewayDiagnosticsMap(t *testing.T) {
	raw := map[string]any{
		"uptime_sec": 10,
		"status": map[string]any{
			"platform": "telegram",
		},
	}
	got := NormalizeGatewayDiagnosticsMap(raw)
	status, _ := got["status"].(map[string]any)
	if status["platform"] != "telegram" {
		t.Fatalf("unexpected diagnostics status: %+v", got)
	}
	if _, ok := status["active_session"]; !ok {
		t.Fatalf("expected normalized status fields in diagnostics: %+v", status)
	}
}
