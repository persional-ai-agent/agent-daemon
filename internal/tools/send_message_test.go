package tools

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/platform"
)

type fakeAdapter struct {
	name string
}

func (f fakeAdapter) Name() string                     { return f.name }
func (f fakeAdapter) Connect(context.Context) error    { return nil }
func (f fakeAdapter) Disconnect(context.Context) error { return nil }
func (f fakeAdapter) Send(_ context.Context, _ string, _ string, _ string) (platform.SendResult, error) {
	return platform.SendResult{Success: true, MessageID: "m1"}, nil
}
func (f fakeAdapter) EditMessage(context.Context, string, string, string) error { return nil }
func (f fakeAdapter) SendTyping(context.Context, string) error                  { return nil }
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
	if _, ok := res["targets"]; !ok {
		t.Fatalf("missing targets in list response: %v", res)
	}

	res, err = tool.Call(context.Background(), map[string]any{
		"action":  "send",
		"target":  "telegram:1",
		"message": "hi",
	}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := res["success"].(bool); !v {
		t.Fatalf("expected success: %v", res)
	}
}

func TestSendMessageSupportsBarePlatformTargetWithHomeChannel(t *testing.T) {
	platform.Register(fakeAdapter{name: "telegram"})
	t.Cleanup(func() { platform.Unregister("telegram") })
	workdir := t.TempDir()
	if err := SetHomeTarget(workdir, "telegram", "10001"); err != nil {
		t.Fatal(err)
	}

	tool := NewSendMessageTool()
	res, err := tool.Call(context.Background(), map[string]any{
		"action":  "send",
		"target":  "telegram",
		"message": "hi",
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := res["success"].(bool); !v {
		t.Fatalf("expected success: %v", res)
	}
	if got, _ := res["chat_id"].(string); got != "10001" {
		t.Fatalf("chat_id=%q want=10001", got)
	}
}

func TestHomeTargetEnvVarIncludesYuanbao(t *testing.T) {
	if got := homeTargetEnvVar("yuanbao"); got != "YUANBAO_HOME_CHANNEL" {
		t.Fatalf("homeTargetEnvVar(yuanbao)=%q, want %q", got, "YUANBAO_HOME_CHANNEL")
	}
}

func TestSendMessageListIncludesHomeTarget(t *testing.T) {
	platform.Register(fakeAdapter{name: "telegram"})
	t.Cleanup(func() { platform.Unregister("telegram") })
	t.Setenv("TELEGRAM_HOME_CHANNEL", "999")

	res, err := NewSendMessageTool().Call(context.Background(), map[string]any{"action": "list"}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	targets, _ := res["targets"].([]map[string]any)
	if len(targets) == 0 {
		t.Fatalf("targets should not be empty: %v", res)
	}
	found := false
	for _, it := range targets {
		if it["platform"] == "telegram" {
			found = true
			if got, _ := it["home_target"].(string); got != "999" {
				t.Fatalf("home_target=%q want=999", got)
			}
		}
	}
	if !found {
		t.Fatalf("telegram target not found in %v", targets)
	}
}

func TestSendMessageListIncludesDirectoryTargets(t *testing.T) {
	workdir := t.TempDir()
	if err := UpsertChannelDirectory(workdir, ChannelDirectoryEntry{
		Platform:   "discord",
		ChatID:     "chan-1",
		ChatType:   "group",
		UserID:     "u-1",
		UserName:   "bob",
		GlobalID:   "g-bob",
		HomeTarget: "chan-1",
	}); err != nil {
		t.Fatal(err)
	}

	res, err := NewSendMessageTool().Call(context.Background(), map[string]any{"action": "list"}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	targets, _ := res["targets"].([]map[string]any)
	found := false
	for _, it := range targets {
		if it["platform"] == "discord" && it["chat_id"] == "chan-1" {
			found = true
			if v, _ := it["connected"].(bool); v {
				t.Fatalf("directory-only target should be disconnected: %+v", it)
			}
			if g, _ := it["global_id"].(string); g != "g-bob" {
				t.Fatalf("global_id=%q want=g-bob row=%+v", g, it)
			}
			break
		}
	}
	if !found {
		t.Fatalf("directory target not found: %+v", targets)
	}
}

func TestSendMessageSchemaActionIsOptional(t *testing.T) {
	schema := NewSendMessageTool().Schema()
	required, _ := schema.Function.Parameters["required"].([]string)
	if len(required) != 0 && !reflect.DeepEqual(required, []string{}) {
		t.Fatalf("required=%v, want empty", required)
	}
}

func TestSendMessageSchemaDocumentsDefaultAction(t *testing.T) {
	schema := NewSendMessageTool().Schema()
	props, _ := schema.Function.Parameters["properties"].(map[string]any)
	action, _ := props["action"].(map[string]any)
	desc, _ := action["description"].(string)
	if !strings.Contains(desc, "default: send") {
		t.Fatalf("send_message action description=%q, want default hint", desc)
	}
}
