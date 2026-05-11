package tools

import "testing"

func TestResolveToolsetCoreIncludes(t *testing.T) {
	allowed, err := ResolveToolset([]string{"core"})
	if err != nil {
		t.Fatal(err)
	}
	for _, must := range []string{"terminal", "read_file", "memory", "delegate_task"} {
		if _, ok := allowed[must]; !ok {
			t.Fatalf("expected %q in resolved toolset", must)
		}
	}
}

func TestResolveToolsetUnknown(t *testing.T) {
	_, err := ResolveToolset([]string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error")
	}
}

