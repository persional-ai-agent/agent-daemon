package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if tr, _ := res["truncated"].(bool); !tr {
		t.Fatalf("expected truncated: %v", res)
	}
}

