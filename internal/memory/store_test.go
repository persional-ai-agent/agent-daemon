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

func TestMemoryStatusOffOnAndReset(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ManageWithContext("add", "memory", "prefers concise answers", "", map[string]any{"session_id": "s1", "turn_id": "t1", "confidence": 0.9}); err != nil {
		t.Fatal(err)
	}
	status, err := store.ManageWithContext("status", "memory", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if status["external_enabled"] != true {
		t.Fatalf("unexpected status: %+v", status)
	}
	if _, err := store.ManageWithContext("off", "memory", "", "", nil); err != nil {
		t.Fatal(err)
	}
	status, _ = store.ManageWithContext("status", "memory", "", "", nil)
	if status["external_enabled"] != false {
		t.Fatalf("expected external_enabled=false, got %+v", status)
	}
	if _, err := store.ManageWithContext("on", "memory", "", "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ManageWithContext("reset", "memory", "", "", nil); err != nil {
		t.Fatal(err)
	}
	list, err := store.ManageWithContext("list", "memory", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if list["count"] != 0 {
		t.Fatalf("expected reset memory to be empty: %+v", list)
	}
}

func TestMemoryRevokeEntry(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ManageWithContext("add", "memory", "project uses Go", "", map[string]any{"session_id": "s2", "turn_id": "t2", "confidence": 0.8}); err != nil {
		t.Fatal(err)
	}
	list, err := store.ManageWithContext("list", "memory", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	entries, _ := list["entries"].([]map[string]any)
	if len(entries) != 1 {
		t.Fatalf("unexpected entries: %+v", list)
	}
	id, _ := entries[0]["id"].(string)
	if strings.TrimSpace(id) == "" {
		t.Fatalf("expected entry id: %+v", entries[0])
	}
	res, err := store.ManageWithContext("revoke", "memory", id, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res["revoked"] != true {
		t.Fatalf("expected revoked=true: %+v", res)
	}
	list, _ = store.ManageWithContext("list", "memory", "", "", nil)
	if list["count"] != 0 {
		t.Fatalf("expected no active entries after revoke: %+v", list)
	}
}
