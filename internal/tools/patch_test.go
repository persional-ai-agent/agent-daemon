package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPatchSingleReplacement(t *testing.T) {
	workdir := t.TempDir()
	p := filepath.Join(workdir, "a.txt")
	if err := os.WriteFile(p, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}
	_, err := b.patch(context.Background(), map[string]any{
		"path":       "a.txt",
		"old_string": "world",
		"new_string": "there",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	bs, _ := os.ReadFile(p)
	if strings.TrimSpace(string(bs)) != "hello there" {
		t.Fatalf("got %q", string(bs))
	}
}

func TestPatchWarnsWhenFileChangedSinceRead(t *testing.T) {
	workdir := t.TempDir()
	p := filepath.Join(workdir, "a.txt")
	if err := os.WriteFile(p, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir, SessionID: "s-stale"}
	if _, err := b.readFile(context.Background(), map[string]any{"path": "a.txt"}, tc); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(p, []byte("hello world!!!"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := b.patch(context.Background(), map[string]any{
		"path":       "a.txt",
		"old_string": "world",
		"new_string": "there",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res["_warning"]; !ok {
		t.Fatalf("expected _warning for stale file, got %+v", res)
	}
}
