package main

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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

func TestRuntimeMultiThinkingBlocks(t *testing.T) {
	rt := newTerminalRuntime(100)
	evt := func(eventType string, data map[string]any) map[string]any {
		return map[string]any{
			"type": "model_stream_event",
			"data": map[string]any{
				"event_type": eventType,
				"event_data": data,
			},
		}
	}

	rt.publishTurnEvent(evt("response.reasoning_text.delta", map[string]any{
		"item_id":       "rs_a",
		"output_index":  0,
		"content_index": 0,
		"text":          "alpha",
	}))
	rt.publishTurnEvent(evt("response.reasoning_text.delta", map[string]any{
		"item_id":       "rs_b",
		"output_index":  0,
		"content_index": 0,
		"text":          "beta",
	}))
	rt.publishTurnEvent(evt("response.reasoning_text.done", map[string]any{
		"item_id":       "rs_a",
		"output_index":  0,
		"content_index": 0,
	}))
	rt.publishTurnEvent(evt("response.reasoning_text.done", map[string]any{
		"item_id":       "rs_b",
		"output_index":  0,
		"content_index": 0,
	}))
	rt.consumePendingEvents()

	collapsed, changed := rt.render(true)
	if !changed {
		t.Fatal("expected render changed")
	}
	if count := strings.Count(collapsed, "Thinking"); count < 2 {
		t.Fatalf("expected >=2 thinking lines, got %d: %q", count, collapsed)
	}
	if strings.Contains(collapsed, "alpha") || strings.Contains(collapsed, "beta") {
		t.Fatalf("expected thinking content hidden in collapsed mode, got: %q", collapsed)
	}

	rt.toggleThinkingExpanded()
	expanded, changed := rt.render(true)
	if !changed {
		t.Fatal("expected render changed after expand")
	}
	if !strings.Contains(expanded, "alpha") || !strings.Contains(expanded, "beta") {
		t.Fatalf("expected all thinking content visible when expanded, got: %q", expanded)
	}
}

func TestRuntimeDiffRenderNoChange(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("hello")
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "pong"},
		},
	})
	rt.consumePendingEvents()
	rt.endTurn()

	if _, changed := rt.render(true); !changed {
		t.Fatal("expected first force render changed")
	}
	if _, changed := rt.render(false); changed {
		t.Fatal("expected no diff patch on unchanged tree")
	}
}

func TestRuntimeStreamingMarkdownVisibleBeforeDone(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("markdown")
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "**bold** text"},
		},
	})
	rt.consumePendingEvents()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	if !strings.Contains(out, "bold") {
		t.Fatalf("expected rendered output contains bold text, got: %q", out)
	}
	if !strings.Contains(out, "**bold**") {
		t.Fatalf("expected raw markdown visible during streaming, got: %q", out)
	}
}

func TestRuntimeNoDuplicateOnCompletedAfterStream(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("dup-check")
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "hello world"},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type":    "assistant_message",
		"content": "hello world",
	})
	rt.publishTurnEvent(map[string]any{
		"type":    "completed",
		"content": "hello world",
	})
	rt.consumePendingEvents()
	rt.endTurn()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	cleaned := ansiEscapePattern.ReplaceAllString(out, "")
	if strings.Count(cleaned, "hello") != 1 {
		t.Fatalf("expected no duplicate final output, got: %q", out)
	}
}

func TestRuntimeAssistantMessageWithoutCompletedContent(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("no-result-content")
	rt.publishTurnEvent(map[string]any{
		"type":    "assistant_message",
		"content": "hello from assistant message",
	})
	rt.publishTurnEvent(map[string]any{
		"type": "completed",
	})
	rt.consumePendingEvents()
	rt.endTurn()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	cleaned := ansiEscapePattern.ReplaceAllString(out, "")
	if !strings.Contains(cleaned, "hello from assistant") || !strings.Contains(cleaned, "message") {
		t.Fatalf("expected assistant content rendered, got: %q", out)
	}
}

func TestRuntimePublishLineAssistantFallback(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("fallback")
	rt.publishLine("assistant: hello fallback")
	rt.consumePendingEvents()
	rt.endTurn()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	cleaned := ansiEscapePattern.ReplaceAllString(out, "")
	if !strings.Contains(cleaned, "hello fallback") {
		t.Fatalf("expected fallback assistant text rendered, got: %q", out)
	}
}

func TestRuntimeStreamEventVariantDeltaAndDone(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("variant")
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "response.output_text.delta",
			"event_data": map[string]any{"delta": "hello "},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "response.output_text.done",
			"event_data": map[string]any{"output_text": "hello world"},
		},
	})
	rt.consumePendingEvents()
	rt.endTurn()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	cleaned := ansiEscapePattern.ReplaceAllString(out, "")
	if !strings.Contains(cleaned, "hello world") {
		t.Fatalf("expected variant stream text rendered, got: %q", out)
	}
}

func TestRuntimeFreezeStableAssistantPrefixDuringStream(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("freeze")

	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "# Headline\n\nBody tail"},
		},
	})
	rt.consumePendingEvents()

	doneCount := 0
	streamCount := 0
	for _, node := range rt.stateTree.nodes {
		if node == nil || node.Type != nodeAssistant {
			continue
		}
		if node.Status == "done" {
			doneCount++
		}
		if node.Status == "streaming" {
			streamCount++
		}
	}
	if doneCount == 0 || streamCount == 0 {
		t.Fatalf("expected done+streaming assistant nodes, got done=%d streaming=%d", doneCount, streamCount)
	}

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	plain := ansiEscapePattern.ReplaceAllString(out, "")
	if strings.Contains(plain, "# Headline") {
		t.Fatalf("expected finalized prefix rendered as markdown heading, got raw output: %q", out)
	}
	if !strings.Contains(plain, "Body tail") {
		t.Fatalf("expected streaming tail still visible, got: %q", out)
	}
}

func TestRuntimeFinalContentOverridesDivergedStream(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("final-override")

	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "temporary text"},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type":    "completed",
		"content": "# Final Title\n\nfinal body",
	})
	rt.consumePendingEvents()
	rt.endTurn()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	plain := ansiEscapePattern.ReplaceAllString(out, "")
	if strings.Contains(plain, "temporary text") {
		t.Fatalf("expected diverged streamed content replaced by final, got: %q", out)
	}
	if strings.Contains(plain, "# Final Title") {
		t.Fatalf("expected final markdown rendered, got raw heading: %q", out)
	}
	if !strings.Contains(plain, "Final Title") || !strings.Contains(plain, "final body") {
		t.Fatalf("expected final content visible, got: %q", out)
	}
}

func TestRuntimeCollapseAssistantChunksOnFinal(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("collapse-final")
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "# T\n\ntail"},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type":    "completed",
		"content": "# T\n\ntail",
	})
	rt.consumePendingEvents()
	rt.endTurn()

	assistantCount := 0
	for _, node := range rt.stateTree.nodes {
		if node != nil && node.Type == nodeAssistant {
			assistantCount++
			if node.Status != "done" {
				t.Fatalf("expected final assistant node done, got: %s", node.Status)
			}
		}
	}
	if assistantCount != 1 {
		t.Fatalf("expected single assistant node after final collapse, got: %d", assistantCount)
	}

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	plain := ansiEscapePattern.ReplaceAllString(out, "")
	if strings.Contains(plain, "# T") {
		t.Fatalf("expected markdown formatted output, got raw heading: %q", out)
	}
	if !strings.Contains(plain, "tail") {
		t.Fatalf("expected final content visible, got: %q", out)
	}
}

func TestRuntimeStreamLiteralEscapedNewlineGetsFormatted(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("escaped-nl")
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "line1\\n\\n- item1\\n- item2"},
		},
	})
	rt.consumePendingEvents()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	plain := ansiEscapePattern.ReplaceAllString(out, "")
	if strings.Contains(plain, "\\n") {
		t.Fatalf("expected escaped newline converted, got: %q", out)
	}
	if !strings.Contains(plain, "line1") || !strings.Contains(plain, "item1") || !strings.Contains(plain, "item2") {
		t.Fatalf("expected formatted streamed content visible, got: %q", out)
	}
}

func TestRuntimeStreamingKeepsWhitespaceOnlyDeltaTokens(t *testing.T) {
	rt := newTerminalRuntime(80)
	rt.startTurn("ws-delta")

	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "line1"},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "\n"},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "line2"},
		},
	})
	rt.consumePendingEvents()

	full := rt.stateTree.fullAssistantText()
	if full != "line1\nline2" {
		t.Fatalf("expected newline delta preserved, got: %q", full)
	}
}

func TestRuntimeMultiTurnKeepsAssistantBoundaries(t *testing.T) {
	rt := newTerminalRuntime(80)

	rt.startTurn("turn-one")
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "hello from one"},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type":    "completed",
		"content": "hello from one",
	})
	rt.consumePendingEvents()
	rt.endTurn()

	rt.startTurn("turn-two")
	rt.publishTurnEvent(map[string]any{
		"type": "model_stream_event",
		"data": map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"text": "# Title Two\n\nbody two"},
		},
	})
	rt.publishTurnEvent(map[string]any{
		"type":    "completed",
		"content": "# Title Two\n\nbody two",
	})
	rt.consumePendingEvents()
	rt.endTurn()

	out, changed := rt.render(true)
	if !changed {
		t.Fatal("expected changed render")
	}
	plain := ansiEscapePattern.ReplaceAllString(out, "")
	idxU1 := strings.Index(plain, "turn-one")
	idxA1 := strings.Index(plain, "hello from one")
	idxU2 := strings.Index(plain, "turn-two")
	idxA2 := strings.Index(plain, "body two")
	if idxU1 < 0 || idxA1 < 0 || idxU2 < 0 || idxA2 < 0 {
		t.Fatalf("expected all turn markers present, got: %q", out)
	}
	if !(idxU1 < idxA1 && idxA1 < idxU2 && idxU2 < idxA2) {
		t.Fatalf("expected chronological turn ordering, got: %q", out)
	}
}
