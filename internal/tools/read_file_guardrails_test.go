package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestReadFileMaxCharsTruncates(t *testing.T) {
	workdir := t.TempDir()
	p := filepath.Join(workdir, "big.txt")
	if err := os.WriteFile(p, []byte("aaaaabbbbbcccccddddd"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BuiltinTools{}
	res, err := b.readFile(context.Background(), map[string]any{
		"path":      "big.txt",
		"max_chars": 5,
		// Compatibility: allow truncated content instead of Hermes-style error.
		"reject_on_truncate": false,
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if tr, _ := res["truncated"].(bool); !tr {
		t.Fatalf("expected truncated: %v", res)
	}
}

func TestReadFileMaxCharsRejectsByDefault(t *testing.T) {
	workdir := t.TempDir()
	p := filepath.Join(workdir, "big.txt")
	if err := os.WriteFile(p, []byte("aaaaabbbbbcccccddddd"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BuiltinTools{}
	res, err := b.readFile(context.Background(), map[string]any{
		"path":      "big.txt",
		"max_chars": 5,
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := res["success"].(bool); ok {
		t.Fatalf("expected success=false, got %+v", res)
	}
	if _, ok := res["error"]; !ok {
		t.Fatalf("expected error field, got %+v", res)
	}
}

func TestReadFileRejectsFIFO(t *testing.T) {
	workdir := t.TempDir()
	fifoPath := filepath.Join(workdir, "pipe")
	if err := syscall.Mkfifo(fifoPath, 0o644); err != nil {
		// Some sandboxed filesystems may block FIFO creation.
		t.Skipf("mkfifo not available: %v", err)
	}

	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}

	done := make(chan error, 1)
	go func() {
		_, err := b.readFile(context.Background(), map[string]any{
			"path": "pipe",
		}, tc)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "non-regular file") {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("read_file blocked on FIFO; expected immediate rejection")
	}
}

func TestReadFileRejectsSymlinkEscape(t *testing.T) {
	workdir := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(workdir, "link.txt")
	if err := os.Symlink(outsideFile, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	b := &BuiltinTools{}
	_, err := b.readFile(context.Background(), map[string]any{
		"path": "link.txt",
	}, ToolContext{Workdir: workdir})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadFileDedupReturnsStubWhenUnchanged(t *testing.T) {
	workdir := t.TempDir()
	p := filepath.Join(workdir, "a.txt")
	if err := os.WriteFile(p, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir, SessionID: "s-dedup"}

	r1, err := b.readFile(context.Background(), map[string]any{
		"path": "a.txt",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := r1["dedup"].(bool); d {
		t.Fatalf("expected dedup=false on first read: %+v", r1)
	}
	if _, ok := r1["content"]; !ok {
		t.Fatalf("expected content on first read: %+v", r1)
	}

	r2, err := b.readFile(context.Background(), map[string]any{
		"path": "a.txt",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := r2["dedup"].(bool); !d {
		t.Fatalf("expected dedup=true on second read: %+v", r2)
	}
	if _, ok := r2["content"]; ok {
		t.Fatalf("expected no content on dedup stub: %+v", r2)
	}
	if cr, _ := r2["content_returned"].(bool); cr {
		t.Fatalf("expected content_returned=false: %+v", r2)
	}

	// Modify file to invalidate dedup.
	if err := os.WriteFile(p, []byte("hello2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r3, err := b.readFile(context.Background(), map[string]any{
		"path": "a.txt",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := r3["dedup"].(bool); d {
		t.Fatalf("expected dedup=false after modification: %+v", r3)
	}
	if c, _ := r3["content"].(string); !strings.Contains(c, "hello2") {
		t.Fatalf("expected updated content, got: %+v", r3)
	}
}

func TestReadFileBlocksAfterFourIdenticalReads(t *testing.T) {
	workdir := t.TempDir()
	p := filepath.Join(workdir, "a.txt")
	if err := os.WriteFile(p, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir, SessionID: "s-loop"}

	// First read returns content.
	if _, err := b.readFile(context.Background(), map[string]any{"path": "a.txt"}, tc); err != nil {
		t.Fatal(err)
	}

	// Next two reads return dedup stubs, third includes warning.
	r2, err := b.readFile(context.Background(), map[string]any{"path": "a.txt"}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := r2["dedup"].(bool); !d {
		t.Fatalf("expected dedup stub: %+v", r2)
	}
	r3, err := b.readFile(context.Background(), map[string]any{"path": "a.txt"}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r3["_warning"]; !ok {
		t.Fatalf("expected warning on third identical read: %+v", r3)
	}

	// Fourth identical read blocks.
	r4, err := b.readFile(context.Background(), map[string]any{"path": "a.txt"}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := r4["success"].(bool); ok {
		t.Fatalf("expected success=false on block: %+v", r4)
	}
	if !strings.Contains(fmt.Sprint(r4["error"]), "BLOCKED") {
		t.Fatalf("expected BLOCKED error: %+v", r4)
	}
}
