package platforms

import (
	"context"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
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
