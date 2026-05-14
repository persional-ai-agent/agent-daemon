package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCommandAndDashboardInfos(t *testing.T) {
	items := []Manifest{{
		Name: "demo",
		Commands: []CommandManifest{{
			Name:        "hello",
			Description: "say hello",
			Command:     "./hello.sh",
		}},
		Dashboard: &DashboardManifest{Label: "Demo", Entry: "dist/index.js", CSS: "dist/style.css"},
		File:      filepath.Join(t.TempDir(), "plugin.yaml"),
	}}
	cmds := CommandInfos(items)
	if len(cmds) != 1 || cmds[0].Name != "hello" || cmds[0].Plugin != "demo" {
		t.Fatalf("commands=%#v", cmds)
	}
	dashboards := DashboardInfos(items)
	if len(dashboards) != 1 || dashboards[0].Name != "demo" || dashboards[0].Label != "Demo" {
		t.Fatalf("dashboards=%#v", dashboards)
	}
	if dashboards[0].Entry == "dist/index.js" {
		t.Fatalf("dashboard entry should be resolved relative to manifest: %#v", dashboards[0])
	}
}

func TestRunCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell plugin command test not supported on windows")
	}
	root := t.TempDir()
	script := filepath.Join(root, "hello.sh")
	body := "#!/usr/bin/env bash\ncat <<'EOF'\n{\"success\":true,\"message\":\"hello\"}\nEOF\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	items := []Manifest{{
		Name: "demo",
		Commands: []CommandManifest{{
			Name:    "hello",
			Command: "./hello.sh",
		}},
		File: filepath.Join(root, "plugin.yaml"),
	}}
	res, err := RunCommand(context.Background(), items, "hello", []string{"world"}, root)
	if err != nil {
		t.Fatal(err)
	}
	if res["message"] != "hello" {
		t.Fatalf("result=%#v", res)
	}
}

func TestRunCommandUsesSandboxedEnvironmentByDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell plugin command test not supported on windows")
	}
	t.Setenv("SECRET_SHOULD_NOT_LEAK", "yes")
	root := t.TempDir()
	script := filepath.Join(root, "env.sh")
	body := "#!/usr/bin/env bash\nif [ -n \"$SECRET_SHOULD_NOT_LEAK\" ]; then echo '{\"leaked\":true}'; else echo '{\"leaked\":false}'; fi\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	items := []Manifest{{
		Name: "demo",
		Commands: []CommandManifest{{
			Name:    "env",
			Command: "./env.sh",
		}},
		File: filepath.Join(root, "plugin.yaml"),
	}}
	res, err := RunCommand(context.Background(), items, "env", nil, root)
	if err != nil {
		t.Fatal(err)
	}
	if leaked, _ := res["leaked"].(bool); leaked {
		t.Fatalf("host env leaked into plugin sandbox: %#v", res)
	}
}
