package agent

import (
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestCompressMessagesCompactsMiddleHistory(t *testing.T) {
	messages := []core.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: strings.Repeat("u1 ", 200)},
		{Role: "assistant", Content: strings.Repeat("a1 ", 200)},
		{Role: "user", Content: strings.Repeat("u2 ", 200)},
		{Role: "assistant", Content: strings.Repeat("a2 ", 200)},
		{Role: "user", Content: "latest"},
	}
	out, meta := compressMessages(messages, 800, 2)
	if meta == nil {
		t.Fatalf("expected compression metadata, got nil")
	}
	if len(out) >= len(messages) {
		t.Fatalf("expected fewer messages after compression, before=%d after=%d", len(messages), len(out))
	}
	if out[0].Role != "system" {
		t.Fatalf("expected system head to be preserved, got %+v", out[0])
	}
	if out[1].Role != "assistant" || !strings.Contains(out[1].Content, contextSummaryPrefix) {
		t.Fatalf("expected inserted summary message, got %+v", out[1])
	}
	if out[len(out)-1].Content != "latest" {
		t.Fatalf("expected latest tail message preserved, got %+v", out[len(out)-1])
	}
}
