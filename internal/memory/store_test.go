package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryManageAddAndReplace(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Manage("add", "memory", "prefers go", ""); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "prefers go") {
		t.Fatalf("missing added content: %s", string(b))
	}
	if _, err := store.Manage("replace", "memory", "prefers rust", "prefers go"); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if !strings.Contains(string(b), "prefers rust") {
		t.Fatalf("replace failed: %s", string(b))
	}
}
