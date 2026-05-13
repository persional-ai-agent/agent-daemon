package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDirs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	okJSON := `{"name":"demo-tool","type":"tool","version":"1.0.0","entry":"./tool.so","enabled":true}`
	if err := os.WriteFile(filepath.Join(dir, "tool.json"), []byte(okJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invalid.json"), []byte(`{"type":"tool"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := LoadFromDirs([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(items))
	}
	if items[0].Name != "demo-tool" || items[0].Type != "tool" {
		t.Fatalf("unexpected manifest: %+v", items[0])
	}
}
