package gateway

import (
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestStreamCollectorIngestTextDelta(t *testing.T) {
	sc := NewStreamCollector()

	sc.Ingest(core.AgentEvent{
		Type: "model_stream_event",
		Data: map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{
				"delta": "Hello ",
			},
		},
	})
	sc.Ingest(core.AgentEvent{
		Type: "model_stream_event",
		Data: map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{
				"delta": "World",
			},
		},
	})

	if got := sc.Content(); got != "Hello World" {
		t.Errorf("Content() = %q, want %q", got, "Hello World")
	}
}

func TestStreamCollectorIgnoresNonTextDelta(t *testing.T) {
	sc := NewStreamCollector()

	sc.Ingest(core.AgentEvent{
		Type: "model_stream_event",
		Data: map[string]any{
			"event_type": "tool_call_start",
			"event_data": map[string]any{},
		},
	})
	sc.Ingest(core.AgentEvent{
		Type: "user_message",
		Data: map[string]any{},
	})

	if got := sc.Content(); got != "" {
		t.Errorf("Content() = %q, want empty", got)
	}
}

func TestStreamCollectorReset(t *testing.T) {
	sc := NewStreamCollector()

	sc.Ingest(core.AgentEvent{
		Type: "model_stream_event",
		Data: map[string]any{
			"event_type": "text_delta",
			"event_data": map[string]any{"delta": "test"},
		},
	})
	sc.Reset()

	if got := sc.Content(); got != "" {
		t.Errorf("Content() after reset = %q, want empty", got)
	}
}
