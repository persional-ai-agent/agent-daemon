package platforms

import (
	"context"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

func TestDiscordAdapterName(t *testing.T) {
	a := &DiscordAdapter{}
	if a.Name() != "discord" {
		t.Errorf("Name() = %q, want %q", a.Name(), "discord")
	}
}

func TestDiscordAdapterOnMessage(t *testing.T) {
	a := &DiscordAdapter{}
	called := false
	a.OnMessage(context.Background(), func(_ context.Context, _ gateway.MessageEvent) {
		called = true
	})
	if a.handler == nil {
		t.Error("handler should not be nil after OnMessage")
	}
	_ = called
}

func TestTruncateDiscord(t *testing.T) {
	if s := truncateDiscord("hello", 2000); s != "hello" {
		t.Errorf("short string: %q", s)
	}
	long := make([]byte, 3000)
	for i := range long {
		long[i] = 'x'
	}
	s := truncateDiscord(string(long), 2000)
	if s[:3] != "xxx" {
		t.Errorf("should start with xxx: %q", s[:3])
	}
	if s[len(s)-3:] != "..." {
		t.Errorf("should end with ...: %q", s[len(s)-3:])
	}
}
