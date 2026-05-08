package platforms

import (
	"context"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestParseChatID(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		err   bool
	}{
		{"123", 123, false},
		{"-456", -456, false},
		{" 789 ", 789, false},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseChatID(tt.input)
			if tt.err && err == nil {
				t.Errorf("parseChatID(%q) expected error, got %d", tt.input, got)
			}
			if !tt.err && err != nil {
				t.Errorf("parseChatID(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseChatID(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestTelegramAdapterName(t *testing.T) {
	a := &TelegramAdapter{}
	if a.Name() != "telegram" {
		t.Errorf("Name() = %q, want %q", a.Name(), "telegram")
	}
}

func TestTelegramAdapterOnMessage(t *testing.T) {
	a := &TelegramAdapter{}
	called := false
	a.OnMessage(context.Background(), func(_ context.Context, _ gateway.MessageEvent) {
		called = true
	})
	if a.handler == nil {
		t.Error("handler should not be nil after OnMessage")
	}
	_ = called
}
