package tools

import (
	"sync"
)

type TodoItem struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type TodoStore struct {
	mu    sync.Mutex
	items map[string][]TodoItem
}

func NewTodoStore() *TodoStore {
	return &TodoStore{items: map[string][]TodoItem{}}
}

func (t *TodoStore) Update(sessionID string, todos []TodoItem, merge bool) []TodoItem {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !merge {
		t.items[sessionID] = todos
		return t.items[sessionID]
	}
	current := t.items[sessionID]
	idx := map[string]int{}
	for i, it := range current {
		idx[it.ID] = i
	}
	for _, it := range todos {
		if i, ok := idx[it.ID]; ok {
			current[i] = it
		} else {
			current = append(current, it)
		}
	}
	t.items[sessionID] = current
	return current
}

func (t *TodoStore) List(sessionID string) []TodoItem {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]TodoItem(nil), t.items[sessionID]...)
}
