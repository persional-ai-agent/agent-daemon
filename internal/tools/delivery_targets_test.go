package tools

import (
	"github.com/dingjingmaster/agent-daemon/internal/platform"
	"testing"
)

func TestBuildDeliveryTargets(t *testing.T) {
	platform.Register(fakeAdapter{name: "telegram"})
	platform.Register(fakeAdapter{name: "slack"})
	t.Cleanup(func() {
		platform.Unregister("telegram")
		platform.Unregister("slack")
	})

	workdir := t.TempDir()
	if err := SetHomeTarget(workdir, "telegram", "100"); err != nil {
		t.Fatal(err)
	}
	if err := UpsertChannelDirectory(workdir, ChannelDirectoryEntry{
		Platform: "discord",
		ChatID:   "c1",
		UserID:   "u1",
	}); err != nil {
		t.Fatal(err)
	}

	platforms, targets, err := BuildDeliveryTargets(workdir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(platforms) < 2 || len(targets) < 3 {
		t.Fatalf("unexpected targets build result: platforms=%v targets=%v", platforms, targets)
	}
}

func TestBuildDeliveryTargetsFilter(t *testing.T) {
	platform.Register(fakeAdapter{name: "telegram"})
	t.Cleanup(func() { platform.Unregister("telegram") })

	workdir := t.TempDir()
	platforms, targets, err := BuildDeliveryTargets(workdir, "telegram")
	if err != nil {
		t.Fatal(err)
	}
	if len(platforms) != 1 || platforms[0] != "telegram" {
		t.Fatalf("platform filter mismatch: %v", platforms)
	}
	for _, it := range targets {
		if it["platform"] != "telegram" {
			t.Fatalf("unexpected platform in filtered targets: %+v", it)
		}
	}
}

func TestBuildTargetsPayload(t *testing.T) {
	payload := BuildTargetsPayload("Telegram", []string{"telegram"}, []map[string]any{{"platform": "telegram"}})
	if payload["platform"] != "telegram" || payload["count"] != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
