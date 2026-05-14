package main

import (
	"strings"
	"testing"
	"time"
)

func TestRuntimeThinkingBlockCollapsedAndExpanded(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("test")

	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "hello <thi"},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "nking>secret content</thinking> world"},
		},
	})
	rt.consumePendingEvents()
	rt.endTurn()

	collapsed, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	if !strings.Contains(collapsed, "hello ") || !strings.Contains(collapsed, " world") {
		t.Fatalf("expected assistant text rendered, got: %q", collapsed)
	}
	if !strings.Contains(collapsed, "Thinking") {
		t.Fatalf("expected thinking summary, got: %q", collapsed)
	}
	if strings.Contains(collapsed, "secret content") {
		t.Fatalf("expected thinking content hidden by default, got: %q", collapsed)
	}

	rt.toggleThinkingExpanded()
	expanded, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render after expand")
	}
	if !strings.Contains(expanded, "secret content") {
		t.Fatalf("expected thinking content visible when expanded, got: %q", expanded)
	}
}

func TestRuntimeToolLineUpdatesWithDuration(t *testing.T) {
	rt := newTerminalRuntime(80)
	startEvt := map[string]any{
		"type":      "tool_started",
		"tool_name": "web_search",
		"data": map[string]any{
			"tool_call_id": "call-1",
			"tool_name":    "web_search",
			"status":       "running",
		},
	}
	doneEvt := map[string]any{
		"type":      "tool_finished",
		"tool_name": "web_search",
		"data": map[string]any{
			"tool_call_id": "call-1",
			"tool_name":    "web_search",
			"status":       "completed",
		},
	}

	rt.publishTurnEvent(startEvt)
	rt.consumePendingEvents()
	time.Sleep(8 * time.Millisecond)
	rt.publishTurnEvent(doneEvt)
	rt.consumePendingEvents()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	if !strings.Contains(out, "✔ Done: web_search") {
		t.Fatalf("expected completed tool line, got: %q", out)
	}
	if strings.Count(out, "web_search") != 1 {
		t.Fatalf("expected single tool line update, got: %q", out)
	}
}

func TestRuntimeToolFailedLine(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.publishTurnEvent(map[string]any{
		"type":      "tool_finished",
		"tool_name": "exec_command",
		"data": map[string]any{
			"tool_call_id": "call-2",
			"tool_name":    "exec_command",
			"status":       "failed",
		},
	})
	rt.consumePendingEvents()

	out, _ := rt.render(true)
	if !strings.Contains(out, "✖ Failed: exec_command") {
		t.Fatalf("expected failed tool style line, got: %q", out)
	}
}
