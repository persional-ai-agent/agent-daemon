package gateway

import (
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type StreamCollector struct {
	buf        strings.Builder
	lastEdit   time.Time
	editThrottle time.Duration
}

func NewStreamCollector() *StreamCollector {
	return &StreamCollector{editThrottle: 500 * time.Millisecond}
}

func (s *StreamCollector) Ingest(event core.AgentEvent) {
	if event.Type != "model_stream_event" {
		return
	}
	innerType, _ := event.Data["event_type"].(string)
	if innerType != "text_delta" {
		return
	}
	innerData, _ := event.Data["event_data"].(map[string]any)
	if innerData == nil {
		return
	}
	delta, _ := innerData["delta"].(string)
	s.buf.WriteString(delta)
}

func (s *StreamCollector) Content() string {
	return s.buf.String()
}

func (s *StreamCollector) ShouldEdit() bool {
	now := time.Now()
	if now.Sub(s.lastEdit) < s.editThrottle {
		return false
	}
	s.lastEdit = now
	return true
}

func (s *StreamCollector) Reset() {
	s.buf.Reset()
	s.lastEdit = time.Time{}
}
