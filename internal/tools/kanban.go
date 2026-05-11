package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type kanbanBoard struct {
	UpdatedAt string                 `json:"updated_at"`
	Tasks     []kanbanTask           `json:"tasks"`
	Links     []map[string]any       `json:"links,omitempty"`
	Meta      map[string]any         `json:"meta,omitempty"`
}

type kanbanTask struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Status    string         `json:"status"` // open|blocked|completed
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	Notes     []kanbanNote   `json:"notes,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type kanbanNote struct {
	At      string `json:"at"`
	Message string `json:"message"`
}

func kanbanPath(workdir string) (string, error) {
	root, err := normalizedWorkdir(workdir)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".agent-daemon", "kanban.json"), nil
}

func loadKanban(workdir string) (kanbanBoard, string, error) {
	p, err := kanbanPath(workdir)
	if err != nil {
		return kanbanBoard{}, "", err
	}
	bs, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return kanbanBoard{UpdatedAt: time.Now().Format(time.RFC3339)}, p, nil
		}
		return kanbanBoard{}, "", err
	}
	var b kanbanBoard
	if err := json.Unmarshal(bs, &b); err != nil {
		return kanbanBoard{}, "", err
	}
	return b, p, nil
}

func saveKanban(p string, b kanbanBoard) error {
	b.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, bs, 0o644)
}

func (b *BuiltinTools) kanbanShow(_ context.Context, _ map[string]any, tc ToolContext) (map[string]any, error) {
	board, _, err := loadKanban(tc.Workdir)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "board": board, "count": len(board.Tasks)}, nil
}

func (b *BuiltinTools) kanbanCreate(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	title := strings.TrimSpace(strArg(args, "title"))
	if title == "" {
		return nil, errors.New("title required")
	}
	board, p, err := loadKanban(tc.Workdir)
	if err != nil {
		return nil, err
	}
	id := strArg(args, "id")
	if strings.TrimSpace(id) == "" {
		id = fmt.Sprintf("t-%d", time.Now().UnixNano())
	}
	now := time.Now().Format(time.RFC3339)
	task := kanbanTask{
		ID:        id,
		Title:     title,
		Status:    "open",
		CreatedAt: now,
		UpdatedAt: now,
		Fields:    map[string]any{},
	}
	if f, ok := args["fields"].(map[string]any); ok {
		task.Fields = f
	}
	board.Tasks = append(board.Tasks, task)
	if err := saveKanban(p, board); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "task": task, "written": p}, nil
}

func (b *BuiltinTools) kanbanComment(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	id := strings.TrimSpace(strArg(args, "id"))
	msg := strings.TrimSpace(strArg(args, "message"))
	if id == "" || msg == "" {
		return nil, errors.New("id and message required")
	}
	board, p, err := loadKanban(tc.Workdir)
	if err != nil {
		return nil, err
	}
	for i := range board.Tasks {
		if board.Tasks[i].ID == id {
			board.Tasks[i].Notes = append(board.Tasks[i].Notes, kanbanNote{At: time.Now().Format(time.RFC3339), Message: msg})
			board.Tasks[i].UpdatedAt = time.Now().Format(time.RFC3339)
			if err := saveKanban(p, board); err != nil {
				return nil, err
			}
			return map[string]any{"success": true, "task": board.Tasks[i], "written": p}, nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", id)
}

func (b *BuiltinTools) kanbanSetStatus(_ context.Context, args map[string]any, tc ToolContext, status string) (map[string]any, error) {
	id := strings.TrimSpace(strArg(args, "id"))
	if id == "" {
		return nil, errors.New("id required")
	}
	board, p, err := loadKanban(tc.Workdir)
	if err != nil {
		return nil, err
	}
	for i := range board.Tasks {
		if board.Tasks[i].ID == id {
			board.Tasks[i].Status = status
			board.Tasks[i].UpdatedAt = time.Now().Format(time.RFC3339)
			if status == "blocked" {
				reason := strings.TrimSpace(strArg(args, "reason"))
				if reason != "" {
					board.Tasks[i].Notes = append(board.Tasks[i].Notes, kanbanNote{At: time.Now().Format(time.RFC3339), Message: "Blocked: " + reason})
				}
			}
			if err := saveKanban(p, board); err != nil {
				return nil, err
			}
			return map[string]any{"success": true, "task": board.Tasks[i], "written": p}, nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", id)
}

func (b *BuiltinTools) kanbanComplete(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	return b.kanbanSetStatus(ctx, args, tc, "completed")
}

func (b *BuiltinTools) kanbanBlock(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	return b.kanbanSetStatus(ctx, args, tc, "blocked")
}

func (b *BuiltinTools) kanbanHeartbeat(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	worker := strings.TrimSpace(strArg(args, "worker"))
	if worker == "" {
		worker = tc.SessionID
	}
	board, p, err := loadKanban(tc.Workdir)
	if err != nil {
		return nil, err
	}
	if board.Meta == nil {
		board.Meta = map[string]any{}
	}
	hb := board.Meta["heartbeats"]
	m := map[string]any{}
	if mm, ok := hb.(map[string]any); ok {
		m = mm
	}
	m[worker] = time.Now().Format(time.RFC3339)
	board.Meta["heartbeats"] = m
	if err := saveKanban(p, board); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "worker": worker, "at": m[worker], "written": p}, nil
}

func (b *BuiltinTools) kanbanLink(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	from := strings.TrimSpace(strArg(args, "from"))
	to := strings.TrimSpace(strArg(args, "to"))
	if from == "" || to == "" {
		return nil, errors.New("from and to required")
	}
	kind := strings.TrimSpace(strArg(args, "kind"))
	if kind == "" {
		kind = "relates"
	}
	board, p, err := loadKanban(tc.Workdir)
	if err != nil {
		return nil, err
	}
	board.Links = append(board.Links, map[string]any{
		"from": from,
		"to":   to,
		"kind": kind,
		"at":   time.Now().Format(time.RFC3339),
	})
	if err := saveKanban(p, board); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "written": p, "link": board.Links[len(board.Links)-1]}, nil
}

func (b *BuiltinTools) kanbanShowParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// Param schemas are declared in builtin.go to keep all schema defs co-located.

var _ = errors.New // silence in case build tags strip usage

