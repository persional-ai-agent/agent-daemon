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

func TestMemorySnapshot(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Manage("add", "memory", "prefers concise answers", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Manage("add", "user", "project uses Go", ""); err != nil {
		t.Fatal(err)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snapshot["memory"], "prefers concise answers") {
		t.Fatalf("missing memory snapshot: %+v", snapshot)
	}
	if !strings.Contains(snapshot["user"], "project uses Go") {
		t.Fatalf("missing user snapshot: %+v", snapshot)
	}
}

func TestMemoryExtractDeduplicatesFacts(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Manage("add", "memory", "We use Go for backend services", ""); err != nil {
		t.Fatal(err)
	}

	res, err := store.Manage("extract", "memory", "We use Go for backend services. I prefer concise answers. My password is secret.", "")
	if err != nil {
		t.Fatal(err)
	}
	added, _ := res["added"].([]string)
	skipped, _ := res["skipped"].([]string)
	if len(added) != 1 || added[0] != "I prefer concise answers" {
		t.Fatalf("unexpected added facts: %+v", res)
	}
	if len(skipped) != 1 || skipped[0] != "We use Go for backend services" {
		t.Fatalf("unexpected skipped facts: %+v", res)
	}

	b, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if strings.Count(content, "We use Go for backend services") != 1 {
		t.Fatalf("duplicate fact was written: %s", content)
	}
	if strings.Contains(content, "password") {
		t.Fatalf("sensitive text should not be persisted: %s", content)
	}
	if !strings.Contains(content, "I prefer concise answers") {
		t.Fatalf("missing extracted preference: %s", content)
	}
}
