package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAndReplayOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diag.json")
	payload := map[string]any{
		"schema_version":       "diag.v1",
		"source":               "web",
		"exported_at":          "2026-05-13T10:00:00Z",
		"session_id":           "s1",
		"turn_id":              "t1",
		"stream_mode":          true,
		"configured_transport": "ws",
		"active_transport":     "sse",
		"reconnect_status":     "degraded",
		"reconnect_count":      1,
		"timeout_action":       "reconnect",
		"read_timeout_sec":     15,
		"turn_timeout_sec":     120,
		"fallback_hint":        "ws->sse",
		"last_error_code":      "timeout",
		"error_text":           "read timeout",
		"events": []map[string]any{
			{"event": "assistant_message"},
			{"event": "result"},
		},
	}
	bs, _ := json.Marshal(payload)
	if err := os.WriteFile(path, bs, 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := validateAndReplay(path)
	if err != nil {
		t.Fatalf("validateAndReplay failed: %v", err)
	}
	if !rep.Valid {
		t.Fatalf("expected valid report: %+v", rep)
	}
	if !rep.TerminalSeen {
		t.Fatalf("expected terminal event seen: %+v", rep)
	}
}

func TestValidateAndReplayInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diag.bad.json")
	payload := map[string]any{
		"schema_version": "diag.v0",
		"source":         "unknown",
		"events":         "not-array",
	}
	bs, _ := json.Marshal(payload)
	if err := os.WriteFile(path, bs, 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := validateAndReplay(path)
	if err == nil {
		t.Fatalf("expected validation error, rep=%+v", rep)
	}
	if rep.Valid {
		t.Fatalf("expected invalid report: %+v", rep)
	}
	if len(rep.Errors) == 0 {
		t.Fatalf("expected errors: %+v", rep)
	}
}
