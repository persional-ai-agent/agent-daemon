package tools

import (
	"context"
	"strings"
	"testing"
)

func TestExecuteCodeRunsPython(t *testing.T) {
	tool := NewExecuteCodeTool()
	tc := ToolContext{Workdir: t.TempDir()}
	res, err := tool.Call(context.Background(), map[string]any{
		"code":            "print('hi')",
		"timeout_seconds": 5,
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := res["success"].(bool); !v {
		t.Fatalf("expected success: %v", res)
	}
	stdout, _ := res["stdout"].(string)
	if !strings.Contains(stdout, "hi") {
		t.Fatalf("stdout=%q", stdout)
	}
}

