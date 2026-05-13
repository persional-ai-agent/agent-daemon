package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

func TestRegisterToolPluginsAndCall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell plugin test not supported on windows")
	}
	root := t.TempDir()
	script := filepath.Join(root, "echo-plugin.sh")
	body := "#!/usr/bin/env bash\ncat <<'EOF'\n{\"success\":true,\"echo\":\"ok\"}\nEOF\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	m := Manifest{
		Name: "demo_plugin_tool",
		Type: "tool",
		File: filepath.Join(root, "demo.json"),
		Tool: &ToolManifest{
			Command: "./echo-plugin.sh",
			Schema:  map[string]any{"type": "object"},
		},
	}
	registry := tools.NewRegistry()
	n, err := RegisterToolPlugins(registry, []Manifest{m})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("registered=%d want=1", n)
	}
	got := tools.ParseJSONArgs(registry.Dispatch(context.Background(), "demo_plugin_tool", map[string]any{"x": 1}, tools.ToolContext{Workdir: root}))
	if ok, _ := got["success"].(bool); !ok {
		t.Fatalf("unexpected result: %#v", got)
	}
}
