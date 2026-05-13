package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func TestNewProviderClientAndChat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell plugin test not supported on windows")
	}
	root := t.TempDir()
	script := filepath.Join(root, "provider.sh")
	body := "#!/usr/bin/env bash\ncat <<'EOF'\n{\"message\":{\"role\":\"assistant\",\"content\":\"plugin-ok\"}}\nEOF\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	m := Manifest{
		Name: "demo_provider",
		Type: "provider",
		File: filepath.Join(root, "demo_provider.json"),
		Provider: &ProviderManifest{
			Command: "./provider.sh",
		},
	}
	client, ok, err := NewProviderClient("demo_provider", "x-model", []Manifest{m})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || client == nil {
		t.Fatalf("expected provider client, ok=%v client=%T", ok, client)
	}
	msg, err := client.ChatCompletion(context.Background(), []core.Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "plugin-ok" {
		t.Fatalf("content=%q", msg.Content)
	}
}

func TestProviderNames(t *testing.T) {
	enabled := true
	names := ProviderNames([]Manifest{
		{Name: "x", Type: "provider", Enabled: &enabled, Provider: &ProviderManifest{Command: "echo"}},
		{Name: "y", Type: "tool", Enabled: &enabled, Tool: &ToolManifest{Command: "echo", Schema: map[string]any{"type": "object"}}},
	})
	if len(names) != 1 || names[0] != "x" {
		t.Fatalf("provider names=%#v", names)
	}
}
