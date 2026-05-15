package platforms

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

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

func TestRenderTelegramInboundTextCommandWithMention(t *testing.T) {
	msg := &tgbotapi.Message{
		Text: "/Approval@agent_bot AP-7",
		Entities: []tgbotapi.MessageEntity{
			{Type: "bot_command", Offset: 0, Length: len("/Approval@agent_bot")},
		},
	}
	got := normalizeInboundSlashText(renderTelegramInboundText(msg))
	if got != "/approvals AP-7" {
		t.Fatalf("normalized telegram command mismatch: got=%q want=%q", got, "/approvals AP-7")
	}
}
