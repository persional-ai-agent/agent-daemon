package tools

import "testing"

func TestBuildCollectionPayload(t *testing.T) {
	got := BuildCollectionPayload("tools", 2, []string{"a", "b"})
	if got["count"] != 2 {
		t.Fatalf("unexpected count: %+v", got)
	}
	items, ok := got["tools"].([]string)
	if !ok || len(items) != 2 {
		t.Fatalf("unexpected tools payload: %+v", got)
	}
}

func TestBuildMemoryPayloads(t *testing.T) {
	got := BuildMemoryContentPayload("USER", "abc")
	if got["target"] != "user" || got["content"] != "abc" {
		t.Fatalf("unexpected memory content payload: %+v", got)
	}
	snap := BuildMemorySnapshotPayload(map[string]any{"x": 1})
	mem, ok := snap["memory"].(map[string]any)
	if !ok || mem["x"] != 1 {
		t.Fatalf("unexpected memory snapshot payload: %+v", snap)
	}
}

func TestBuildPersonalityPayload(t *testing.T) {
	show := BuildPersonalityPayload("show", "p")
	if show["system_prompt"] != "p" {
		t.Fatalf("unexpected show payload: %+v", show)
	}
	reset := BuildPersonalityPayload("reset", "p")
	if reset["reset"] != true {
		t.Fatalf("unexpected reset payload: %+v", reset)
	}
	set := BuildPersonalityPayload("set", "p")
	if set["updated"] != true {
		t.Fatalf("unexpected set payload: %+v", set)
	}
}

func TestBuildObjectPayload(t *testing.T) {
	got := BuildObjectPayload("schema", 123)
	if got["schema"] != 123 {
		t.Fatalf("unexpected object payload: %+v", got)
	}
}
