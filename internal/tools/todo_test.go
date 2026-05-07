package tools

import "testing"

func TestTodoStoreMerge(t *testing.T) {
	store := NewTodoStore()
	store.Update("s1", []TodoItem{{ID: "1", Content: "a", Status: "pending", Priority: "high"}}, false)
	items := store.Update("s1", []TodoItem{{ID: "1", Content: "b", Status: "completed", Priority: "high"}, {ID: "2", Content: "c", Status: "pending", Priority: "low"}}, true)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Content != "b" || items[0].Status != "completed" {
		t.Fatalf("merge did not update existing item: %+v", items[0])
	}
}
