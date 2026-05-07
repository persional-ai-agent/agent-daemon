package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileResolvesRelativePathWithinWorkdir(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}

	res, err := b.writeFile(context.Background(), map[string]any{
		"path":    "notes/todo.txt",
		"content": "hello",
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	path, _ := res["path"].(string)
	if path != filepath.Join(workdir, "notes", "todo.txt") {
		t.Fatalf("unexpected resolved path: %+v", res)
	}
}

func TestReadFileRejectsPathOutsideWorkdir(t *testing.T) {
	workdir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BuiltinTools{}

	_, err := b.readFile(context.Background(), map[string]any{
		"path": outside,
	}, ToolContext{Workdir: workdir})
	if err == nil || !strings.Contains(err.Error(), "path escapes workdir") {
		t.Fatalf("expected workdir escape error, got %v", err)
	}
}

func TestSearchFilesRejectsEscapingRoot(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}

	_, err := b.searchFiles(context.Background(), map[string]any{
		"path":    "..",
		"pattern": "hello",
	}, ToolContext{Workdir: workdir})
	if err == nil || !strings.Contains(err.Error(), "path escapes workdir") {
		t.Fatalf("expected workdir escape error, got %v", err)
	}
}

func TestTerminalBlocksHardlineCommand(t *testing.T) {
	b := &BuiltinTools{}

	_, err := b.terminal(context.Background(), map[string]any{
		"command": "rm -rf /",
	}, ToolContext{Workdir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "blocked dangerous command") {
		t.Fatalf("expected blocked dangerous command error, got %v", err)
	}
}

func TestTerminalRequiresApprovalForDangerousCommand(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()
	target := filepath.Join(workdir, "tmp-dir")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := b.terminal(context.Background(), map[string]any{
		"command": "rm -rf tmp-dir",
	}, ToolContext{Workdir: workdir})
	if err == nil || !strings.Contains(err.Error(), "requires approval") {
		t.Fatalf("expected approval required error, got %v", err)
	}
}

func TestTerminalAllowsDangerousCommandWithApproval(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()
	target := filepath.Join(workdir, "tmp-dir")
	file := filepath.Join(target, "a.txt")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := b.terminal(context.Background(), map[string]any{
		"command":           "rm -rf tmp-dir",
		"requires_approval": true,
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if res["requires_approval"] != true {
		t.Fatalf("expected requires_approval metadata, got %+v", res)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected target directory removed, stat err=%v", statErr)
	}
}

func TestTerminalHardlineStillBlockedWithApproval(t *testing.T) {
	b := &BuiltinTools{}
	_, err := b.terminal(context.Background(), map[string]any{
		"command":           "rm -rf /",
		"requires_approval": true,
	}, ToolContext{Workdir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "blocked dangerous command") {
		t.Fatalf("expected hardline block, got %v", err)
	}
}
