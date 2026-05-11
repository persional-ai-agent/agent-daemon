package tools

import (
	"context"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/platform"
)

type fakeAdapter struct {
	name string
}

func (f fakeAdapter) Name() string { return f.name }
func (f fakeAdapter) Connect(context.Context) error { return nil }
func (f fakeAdapter) Disconnect(context.Context) error { return nil }
func (f fakeAdapter) Send(_ context.Context, _ string, _ string, _ string) (platform.SendResult, error) {
	return platform.SendResult{Success: true, MessageID: "m1"}, nil
}
func (f fakeAdapter) EditMessage(context.Context, string, string, string) error { return nil }
func (f fakeAdapter) SendTyping(context.Context, string) error { return nil }
func (f fakeAdapter) OnMessage(context.Context, platform.MessageHandler)        {}

func TestSendMessageListAndSend(t *testing.T) {
	platform.Register(fakeAdapter{name: "telegram"})
	t.Cleanup(func() { platform.Unregister("telegram") })

	tool := NewSendMessageTool()

	res, err := tool.Call(context.Background(), map[string]any{"action": "list"}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res["platforms"]; !ok {
		t.Fatalf("missing platforms: %v", res)
	}

	res, err = tool.Call(context.Background(), map[string]any{
		"action":   "send",
		"target":   "telegram:1",
		"message":  "hi",
	}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := res["success"].(bool); !v {
		t.Fatalf("expected success: %v", res)
	}
}
