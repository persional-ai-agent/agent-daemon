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

func TestResolveToolsetDetailedUnavailable(t *testing.T) {
	res, err := ResolveToolsetDetailed([]string{"discord"}, ToolsetResolveOptions{
		Env: map[string]string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ResolvedTools) != 0 {
		t.Fatalf("expected no resolved tools, got: %v", res.ResolvedTools)
	}
	if len(res.UnavailableToolset) == 0 {
		t.Fatal("expected unavailable toolset reasons")
	}
}

func TestResolveToolsetDetailedWithEnv(t *testing.T) {
	res, err := ResolveToolsetDetailed([]string{"discord"}, ToolsetResolveOptions{
		Env: map[string]string{"DISCORD_BOT_TOKEN": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ResolvedTools) == 0 {
		t.Fatal("expected resolved tools")
	}
}

func TestResolveToolsetDetailedConflict(t *testing.T) {
	res, err := ResolveToolsetDetailed([]string{"safe", "core"}, ToolsetResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) == 0 {
		t.Fatal("expected conflicts for safe+core")
	}
}
