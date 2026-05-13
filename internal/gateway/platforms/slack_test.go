package platforms

import (
	"context"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
	"github.com/slack-go/slack"
)

func TestSlackAdapterName(t *testing.T) {
	a := &SlackAdapter{}
	if a.Name() != "slack" {
		t.Errorf("Name() = %q, want %q", a.Name(), "slack")
	}
}

func TestSlackAdapterOnMessage(t *testing.T) {
	a := &SlackAdapter{}
	called := false
	a.OnMessage(context.Background(), func(_ context.Context, _ gateway.MessageEvent) {
		called = true
	})
	if a.handler == nil {
		t.Error("handler should not be nil after OnMessage")
	}
	_ = called
}

func TestTruncateSlack(t *testing.T) {
	if s := truncateSlack("hello"); s != "hello" {
		t.Errorf("short: %q", s)
	}
	long := make([]byte, 50000)
	for i := range long {
		long[i] = 'x'
	}
	s := truncateSlack(string(long))
	if len(s) > 40000 {
		t.Errorf("too long: %d", len(s))
	}
}

func TestRenderSlackSlashCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		text    string
		want    string
	}{
		{name: "empty command", command: "", text: "status", want: ""},
		{name: "built in with arg", command: "/approve", text: "abc", want: "/approve abc"},
		{name: "built in direct", command: "/status", text: "", want: "/status"},
		{name: "text already slash", command: "/agent", text: "/pending", want: "/pending"},
		{name: "generic entrypoint", command: "/agent", text: "status", want: "/status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderSlackSlashCommand(fakeSlashCommand(tt.command, tt.text))
			if got != tt.want {
				t.Fatalf("renderSlash command=%q text=%q got=%q want=%q", tt.command, tt.text, got, tt.want)
			}
		})
	}
}

func fakeSlashCommand(command, text string) slack.SlashCommand {
	return slack.SlashCommand{Command: command, Text: text}
}
