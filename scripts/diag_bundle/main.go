package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type diagReport struct {
	File         string   `json:"file"`
	Schema       string   `json:"schema"`
	Valid        bool     `json:"valid"`
	Errors       []string `json:"errors"`
	EventCount   int      `json:"event_count"`
	TerminalSeen bool     `json:"terminal_seen"`
	Replay       []string `json:"replay"`
}

func main() {
	file := flag.String("file", "", "diagnostics bundle path")
	report := flag.String("report", "", "optional report output path")
	flag.Parse()
	if strings.TrimSpace(*file) == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/diag_bundle -file <diag.json> [-report <out.json>]")
		os.Exit(2)
	}
	out, err := validateAndReplay(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "diag validate failed: %v\n", err)
		os.Exit(1)
	}
	bs, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(bs))
	if strings.TrimSpace(*report) != "" {
		if err := os.WriteFile(*report, bs, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write report failed: %v\n", err)
			os.Exit(1)
		}
	}
}

func validateAndReplay(path string) (diagReport, error) {
	rep := diagReport{File: path, Schema: "diag.v1", Errors: make([]string, 0), Replay: make([]string, 0)}
	bs, err := os.ReadFile(path)
	if err != nil {
		return rep, err
	}
	var payload map[string]any
	if err := json.Unmarshal(bs, &payload); err != nil {
		return rep, err
	}
	validateBundle(payload, &rep)
	replayBundle(payload, &rep)
	rep.Valid = len(rep.Errors) == 0
	if !rep.Valid {
		return rep, fmt.Errorf("bundle invalid: %s", strings.Join(rep.Errors, "; "))
	}
	return rep, nil
}

func validateBundle(payload map[string]any, rep *diagReport) {
	requireString(payload, rep, "schema_version")
	requireString(payload, rep, "source")
	requireString(payload, rep, "session_id")
	requireString(payload, rep, "turn_id")
	requireString(payload, rep, "configured_transport")
	requireString(payload, rep, "active_transport")
	requireString(payload, rep, "reconnect_status")
	requireString(payload, rep, "timeout_action")
	requireString(payload, rep, "last_error_code")
	requireString(payload, rep, "error_text")
	requireString(payload, rep, "fallback_hint")
	requireBool(payload, rep, "stream_mode")
	requireNonNegativeInt(payload, rep, "reconnect_count")
	requirePositiveInt(payload, rep, "read_timeout_sec")
	requirePositiveInt(payload, rep, "turn_timeout_sec")
	ts := requireString(payload, rep, "exported_at")
	if ts != "" {
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			rep.Errors = append(rep.Errors, "exported_at must be RFC3339")
		}
	}
	if v, ok := payload["schema_version"].(string); ok && v != "diag.v1" {
		rep.Errors = append(rep.Errors, "schema_version must be diag.v1")
	}
	if v, ok := payload["source"].(string); ok && v != "web" && v != "ui-tui" {
		rep.Errors = append(rep.Errors, "source must be web|ui-tui")
	}
	if v, ok := payload["configured_transport"].(string); ok && v != "ws" && v != "sse" {
		rep.Errors = append(rep.Errors, "configured_transport must be ws|sse")
	}
	if v, ok := payload["active_transport"].(string); ok && v != "ws" && v != "sse" {
		rep.Errors = append(rep.Errors, "active_transport must be ws|sse")
	}
	if arr, ok := payload["events"].([]any); ok {
		rep.EventCount = len(arr)
	} else {
		rep.Errors = append(rep.Errors, "events must be array")
	}
}

func replayBundle(payload map[string]any, rep *diagReport) {
	events, _ := payload["events"].([]any)
	for _, row := range events {
		evt, ok := row.(map[string]any)
		if !ok {
			continue
		}
		name := ""
		if v, ok := evt["event"].(string); ok {
			name = v
		} else if v, ok := evt["type"].(string); ok {
			name = v
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		rep.Replay = append(rep.Replay, name)
		if name == "result" || name == "error" || name == "cancelled" {
			rep.TerminalSeen = true
		}
	}
}

func requireString(payload map[string]any, rep *diagReport, key string) string {
	v, ok := payload[key].(string)
	if !ok {
		rep.Errors = append(rep.Errors, key+" must be string")
		return ""
	}
	return v
}

func requireBool(payload map[string]any, rep *diagReport, key string) {
	if _, ok := payload[key].(bool); !ok {
		rep.Errors = append(rep.Errors, key+" must be boolean")
	}
}

func requirePositiveInt(payload map[string]any, rep *diagReport, key string) {
	v, ok := payload[key].(float64)
	if !ok || int(v) < 1 {
		rep.Errors = append(rep.Errors, key+" must be integer >= 1")
	}
}

func requireNonNegativeInt(payload map[string]any, rep *diagReport, key string) {
	v, ok := payload[key].(float64)
	if !ok || int(v) < 0 {
		rep.Errors = append(rep.Errors, key+" must be integer >= 0")
	}
}
