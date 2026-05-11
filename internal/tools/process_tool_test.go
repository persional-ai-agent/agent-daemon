package tools

import (
	"context"
	"testing"
)

func TestProcessToolStatusMissingID(t *testing.T) {
	b := &BuiltinTools{proc: NewProcessRegistry(t.TempDir())}
	_, err := b.process(context.Background(), map[string]any{"action": "status"}, ToolContext{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProcessToolListEmpty(t *testing.T) {
	b := &BuiltinTools{proc: NewProcessRegistry(t.TempDir())}
	res, err := b.process(context.Background(), map[string]any{"action": "list"}, ToolContext{})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := res["success"].(bool); !v {
		t.Fatalf("expected success: %v", res)
	}
}
