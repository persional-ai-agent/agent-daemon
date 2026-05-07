package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type SessionSearchStore interface {
	Search(query string, limit int, sessionID string) ([]map[string]any, error)
}

type MemoryStore interface {
	Manage(action, target, content, oldText string) (map[string]any, error)
}

type DelegateRunner interface {
	RunSubtask(ctx context.Context, parentSessionID, goal, taskContext string, maxIterations int) (map[string]any, error)
}

type ToolContext struct {
	SessionID      string
	SessionStore   SessionSearchStore
	MemoryStore    MemoryStore
	TodoStore      *TodoStore
	DelegateRunner DelegateRunner
	Workdir        string
}

type Tool interface {
	Name() string
	Schema() core.ToolSchema
	Call(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error)
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Schemas() []core.ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]core.ToolSchema, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Schema())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Function.Name < out[j].Function.Name })
	return out
}

func (r *Registry) Dispatch(ctx context.Context, name string, args map[string]any, tc ToolContext) string {
	r.mu.RLock()
	tool, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		b, _ := json.Marshal(map[string]any{"error": fmt.Sprintf("unknown tool: %s", name)})
		return string(b)
	}
	res, err := tool.Call(ctx, args, tc)
	if err != nil {
		b, _ := json.Marshal(map[string]any{"error": err.Error()})
		return string(b)
	}
	if res == nil {
		res = map[string]any{"success": true}
	}
	b, _ := json.Marshal(res)
	return string(b)
}
