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
	targets, _ := res["targets"].([]map[string]any)
	if len(targets) == 0 || targets[0]["target"] == "" {
		t.Fatalf("target canonical field missing in list response: %+v", targets)
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

func TestParseDeliveryTarget(t *testing.T) {
	p, c, err := ParseDeliveryTarget("telegram")
	if err != nil || p != "telegram" || c != "" {
		t.Fatalf("telegram parse failed: p=%q c=%q err=%v", p, c, err)
	}
	p, c, err = ParseDeliveryTarget("telegram:123")
	if err != nil || p != "telegram" || c != "123" {
		t.Fatalf("telegram:123 parse failed: p=%q c=%q err=%v", p, c, err)
	}
	p, c, err = ParseDeliveryTarget("yuanbao:group:123")
	if err != nil || p != "yuanbao" || c != "group:123" {
		t.Fatalf("yuanbao grouped parse failed: p=%q c=%q err=%v", p, c, err)
	}
	if _, _, err = ParseDeliveryTarget(":123"); err == nil {
		t.Fatal("expected error for missing platform")
	}
	if _, _, err = ParseDeliveryTarget("telegram:"); err == nil {
		t.Fatal("expected error for missing chat id")
	}
}

func TestSendMessageSupportsMultiSegmentTarget(t *testing.T) {
	platform.Register(fakeAdapter{name: "yuanbao"})
	t.Cleanup(func() { platform.Unregister("yuanbao") })
	res, err := NewSendMessageTool().Call(context.Background(), map[string]any{
		"action":  "send",
		"target":  "yuanbao:group:123",
		"message": "hi",
	}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := res["platform"].(string); got != "yuanbao" {
		t.Fatalf("platform=%q want=yuanbao", got)
	}
	if got, _ := res["chat_id"].(string); got != "group:123" {
		t.Fatalf("chat_id=%q want=group:123", got)
	}
}

func TestSendMessageListFilterByPlatform(t *testing.T) {
	platform.Register(fakeAdapter{name: "telegram"})
	platform.Register(fakeAdapter{name: "slack"})
	t.Cleanup(func() {
		platform.Unregister("telegram")
		platform.Unregister("slack")
	})
	res, err := NewSendMessageTool().Call(context.Background(), map[string]any{
		"action":   "list",
		"platform": "slack",
	}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	platforms, _ := res["platforms"].([]string)
	if len(platforms) != 1 || platforms[0] != "slack" {
		t.Fatalf("platform filter mismatch: %+v", platforms)
	}
	targets, _ := res["targets"].([]map[string]any)
	for _, it := range targets {
		if it["platform"] != "slack" {
			t.Fatalf("unexpected target platform in filtered list: %+v", it)
		}
	}
}
