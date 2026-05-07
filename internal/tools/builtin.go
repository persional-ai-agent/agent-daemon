package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type BuiltinTools struct {
	proc *ProcessRegistry
}

func RegisterBuiltins(r *Registry, proc *ProcessRegistry) {
	b := &BuiltinTools{proc: proc}
	r.Register(toolDef{name: "terminal", desc: "Execute shell commands on Linux", params: terminalParams(), call: b.terminal})
	r.Register(toolDef{name: "process_status", desc: "Poll background process status by session_id", params: processStatusParams(), call: b.processStatus})
	r.Register(toolDef{name: "stop_process", desc: "Stop a background process", params: stopProcessParams(), call: b.stopProcess})
	r.Register(toolDef{name: "read_file", desc: "Read file from filesystem", params: readFileParams(), call: b.readFile})
	r.Register(toolDef{name: "write_file", desc: "Write content to file", params: writeFileParams(), call: b.writeFile})
	r.Register(toolDef{name: "search_files", desc: "Search text in files", params: searchFilesParams(), call: b.searchFiles})
	r.Register(toolDef{name: "todo", desc: "Maintain per-session todo list", params: todoParams(), call: b.todo})
	r.Register(toolDef{name: "memory", desc: "Manage persistent MEMORY.md/USER.md", params: memoryParams(), call: b.memory})
	r.Register(toolDef{name: "session_search", desc: "Search previous session messages", params: sessionSearchParams(), call: b.sessionSearch})
	r.Register(toolDef{name: "web_fetch", desc: "Fetch URL content over HTTP", params: webFetchParams(), call: b.webFetch})
}

type toolFn func(context.Context, map[string]any, ToolContext) (map[string]any, error)

type toolDef struct {
	name   string
	desc   string
	params map[string]any
	call   toolFn
}

func (t toolDef) Name() string { return t.name }
func (t toolDef) Schema() core.ToolSchema {
	return core.ToolSchema{Type: "function", Function: core.ToolSchemaDetail{Name: t.name, Description: t.desc, Parameters: t.params}}
}
func (t toolDef) Call(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	return t.call(ctx, args, tc)
}

func (b *BuiltinTools) terminal(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	command := strArg(args, "command")
	if strings.TrimSpace(command) == "" {
		return nil, errors.New("command is required")
	}
	background := boolArg(args, "background", false)
	timeout := intArg(args, "timeout", 120)
	cwd := strArg(args, "workdir")
	if cwd == "" {
		cwd = tc.Workdir
	}
	if background {
		s, err := b.proc.StartBackground(ctx, command, cwd)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"output":      "background process started",
			"session_id":  s.ID,
			"output_file": s.OutputFile,
			"status":      "running",
			"exit_code":   0,
		}, nil
	}
	out, code, err := RunForeground(ctx, command, cwd, timeout)
	res := map[string]any{"output": out, "exit_code": code, "error": nil}
	if err != nil {
		res["error"] = err.Error()
	}
	return res, nil
}

func (b *BuiltinTools) processStatus(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	s, ok := b.proc.Poll(id)
	if !ok {
		return nil, fmt.Errorf("process not found: %s", id)
	}
	return map[string]any{"session_id": id, "status": statusFromDone(s.Done), "exit_code": s.ExitCode, "error": s.Err, "output_file": s.OutputFile}, nil
}

func (b *BuiltinTools) stopProcess(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	if err := b.proc.Stop(id); err != nil {
		return nil, err
	}
	return map[string]any{"session_id": id, "stopped": true}, nil
}

func (b *BuiltinTools) readFile(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	path := strArg(args, "path")
	if path == "" {
		return nil, errors.New("path required")
	}
	offset := intArg(args, "offset", 1)
	limit := intArg(args, "limit", 0)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	lineNo := 0
	out := make([]string, 0)
	for s.Scan() {
		lineNo++
		if lineNo < offset {
			continue
		}
		out = append(out, fmt.Sprintf("%d→%s", lineNo, s.Text()))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"path": path, "content": strings.Join(out, "\n")}, nil
}

func (b *BuiltinTools) writeFile(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	path := strArg(args, "path")
	content := strArg(args, "content")
	if path == "" {
		return nil, errors.New("path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"path": path, "bytes": len(content), "written": true}, nil
}

func (b *BuiltinTools) searchFiles(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	root := strArg(args, "path")
	if root == "" {
		root = "."
	}
	pattern := strArg(args, "pattern")
	glob := strArg(args, "glob")
	if pattern == "" {
		return nil, errors.New("pattern required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	matches := make([]map[string]any, 0)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if glob != "" {
			ok, _ := filepath.Match(glob, filepath.Base(path))
			if !ok {
				return nil
			}
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		ln := 0
		for s.Scan() {
			ln++
			line := s.Text()
			if re.MatchString(line) {
				matches = append(matches, map[string]any{"path": path, "line": ln, "content": line})
				if len(matches) >= 200 {
					return io.EOF
				}
			}
		}
		return nil
	})
	return map[string]any{"count": len(matches), "matches": matches}, nil
}

func (b *BuiltinTools) todo(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.TodoStore == nil {
		return nil, errors.New("todo store unavailable")
	}
	merge := boolArg(args, "merge", false)
	val, ok := args["todos"].([]any)
	if !ok {
		return map[string]any{"todos": tc.TodoStore.List(tc.SessionID)}, nil
	}
	items := make([]TodoItem, 0, len(val))
	for _, x := range val {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, TodoItem{
			ID:       strMap(m, "id"),
			Content:  strMap(m, "content"),
			Status:   strMap(m, "status"),
			Priority: strMap(m, "priority"),
		})
	}
	return map[string]any{"todos": tc.TodoStore.Update(tc.SessionID, items, merge)}, nil
}

func (b *BuiltinTools) memory(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.MemoryStore == nil {
		return nil, errors.New("memory store unavailable")
	}
	return tc.MemoryStore.Manage(strArg(args, "action"), strArg(args, "target"), strArg(args, "content"), strArg(args, "old_text"))
}

func (b *BuiltinTools) sessionSearch(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.SessionStore == nil {
		return nil, errors.New("session store unavailable")
	}
	query := strArg(args, "query")
	limit := intArg(args, "limit", 5)
	rows, err := tc.SessionStore.Search(query, limit, tc.SessionID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"query": query, "results": rows}, nil
}

func (b *BuiltinTools) webFetch(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	url := strArg(args, "url")
	if url == "" {
		return nil, errors.New("url required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 200_000))
	return map[string]any{"status": resp.StatusCode, "url": url, "content": string(body)}, nil
}

func statusFromDone(done bool) string {
	if done {
		return "done"
	}
	return "running"
}

func terminalParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}, "background": map[string]any{"type": "boolean"}, "timeout": map[string]any{"type": "integer"}, "workdir": map[string]any{"type": "string"}}, "required": []string{"command"}}
}
func processStatusParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"session_id": map[string]any{"type": "string"}}, "required": []string{"session_id"}}
}
func stopProcessParams() map[string]any {
	return processStatusParams()
}
func readFileParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "offset": map[string]any{"type": "integer"}, "limit": map[string]any{"type": "integer"}}, "required": []string{"path"}}
}
func writeFileParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}}, "required": []string{"path", "content"}}
}
func searchFilesParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "pattern": map[string]any{"type": "string"}, "glob": map[string]any{"type": "string"}}, "required": []string{"pattern"}}
}
func todoParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"todos": map[string]any{"type": "array"}, "merge": map[string]any{"type": "boolean"}}}
}
func memoryParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"action": map[string]any{"type": "string"}, "target": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}, "old_text": map[string]any{"type": "string"}}, "required": []string{"action", "target"}}
}
func sessionSearchParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}}, "required": []string{"query"}}
}
func webFetchParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"url": map[string]any{"type": "string"}}, "required": []string{"url"}}
}

func strArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		s, _ := v.(string)
		return s
	}
	return ""
}
func strMap(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}
func intArg(args map[string]any, key string, d int) int {
	v, ok := args[key]
	if !ok {
		return d
	}
	switch vv := v.(type) {
	case float64:
		return int(vv)
	case int:
		return vv
	case string:
		i, err := strconv.Atoi(vv)
		if err == nil {
			return i
		}
	}
	return d
}
func boolArg(args map[string]any, key string, d bool) bool {
	v, ok := args[key]
	if !ok {
		return d
	}
	switch vv := v.(type) {
	case bool:
		return vv
	case string:
		if strings.EqualFold(vv, "true") || vv == "1" {
			return true
		}
		if strings.EqualFold(vv, "false") || vv == "0" {
			return false
		}
	}
	return d
}

func ParseJSONArgs(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}
