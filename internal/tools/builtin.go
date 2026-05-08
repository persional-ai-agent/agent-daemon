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
	"sync"
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
	r.Register(toolDef{name: "delegate_task", desc: "Run a child agent on a subtask or a batch of subtasks", params: delegateTaskParams(), call: b.delegateTask})
	r.Register(toolDef{name: "approval", desc: "Manage session-level dangerous command approvals", params: approvalParams(), call: b.approval})
	r.Register(toolDef{name: "skill_list", desc: "List available local skills", params: skillListParams(), call: b.skillList})
	r.Register(toolDef{name: "skill_view", desc: "Read a local skill by name", params: skillViewParams(), call: b.skillView})
	r.Register(toolDef{name: "skill_manage", desc: "Manage local skills (create/edit/patch/delete)", params: skillManageParams(), call: b.skillManage})
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
	if reason, blocked := detectHardlineCommand(command); blocked {
		return nil, fmt.Errorf("blocked dangerous command: %s", reason)
	}
	requiresApproval := boolArg(args, "requires_approval", false)
	if reason, dangerous := detectDangerousCommand(command); dangerous {
		approved := tc.ApprovalStore != nil && tc.ApprovalStore.IsApproved(tc.SessionID)
		if !requiresApproval && !approved {
			return nil, fmt.Errorf("dangerous command requires approval: %s (set requires_approval=true or grant session approval)", reason)
		}
		if requiresApproval && tc.ApprovalStore != nil {
			ttlSeconds := intArg(args, "approval_ttl_seconds", 0)
			tc.ApprovalStore.Grant(tc.SessionID, time.Duration(ttlSeconds)*time.Second)
		}
	}
	background := boolArg(args, "background", false)
	timeout := intArg(args, "timeout", 120)
	cwd := tc.Workdir
	if v := strArg(args, "workdir"); strings.TrimSpace(v) != "" {
		var err error
		cwd, err = resolvePathWithinWorkdir(tc.Workdir, v)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		cwd, err = normalizedWorkdir(tc.Workdir)
		if err != nil {
			return nil, err
		}
	}
	if background {
		s, err := b.proc.StartBackground(ctx, command, cwd)
		if err != nil {
			return nil, err
		}
		return map[string]any{"output": "background process started", "session_id": s.ID, "output_file": s.OutputFile, "status": "running", "exit_code": 0, "requires_approval": requiresApproval}, nil
	}
	out, code, err := RunForeground(ctx, command, cwd, timeout)
	res := map[string]any{"output": out, "exit_code": code, "error": nil, "requires_approval": requiresApproval}
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

func (b *BuiltinTools) readFile(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
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

func (b *BuiltinTools) writeFile(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	content := strArg(args, "content")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"path": path, "bytes": len(content), "written": true}, nil
}

func (b *BuiltinTools) searchFiles(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	root := strArg(args, "path")
	if strings.TrimSpace(root) == "" {
		root = tc.Workdir
	}
	root, err := resolvePathWithinWorkdir(tc.Workdir, root)
	if err != nil {
		return nil, err
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
		items = append(items, TodoItem{ID: strMap(m, "id"), Content: strMap(m, "content"), Status: strMap(m, "status"), Priority: strMap(m, "priority")})
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

func (b *BuiltinTools) delegateTask(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.DelegateRunner == nil {
		return nil, errors.New("delegate runner unavailable")
	}
	maxIterations := intArg(args, "max_iterations", 0)
	timeoutSeconds := intArg(args, "timeout_seconds", 0)
	failFast := boolArg(args, "fail_fast", false)
	if tasks, ok := args["tasks"].([]any); ok && len(tasks) > 0 {
		validTasks := make([]map[string]any, 0, len(tasks))
		for _, item := range tasks {
			taskMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			validTasks = append(validTasks, taskMap)
		}
		results := make([]map[string]any, len(validTasks))
		maxConcurrency := intArg(args, "max_concurrency", len(validTasks))
		if maxConcurrency <= 0 || maxConcurrency > len(validTasks) {
			maxConcurrency = len(validTasks)
		}
		batchCtx, batchCancel := context.WithCancel(ctx)
		defer batchCancel()
		sem := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup
		wg.Add(len(validTasks))
		for i, taskMap := range validTasks {
			go func(index int, task map[string]any) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
				case <-batchCtx.Done():
					results[index] = delegateTaskErrorResult(strMap(task, "goal"), batchCtx.Err())
					return
				}
				defer func() { <-sem }()
				goal := strMap(task, "goal")
				res, err := runDelegateSubtask(batchCtx, tc.DelegateRunner, tc.SessionID, goal, strMap(task, "context"), maxIterations, timeoutSeconds)
				if err != nil {
					results[index] = delegateTaskErrorResult(goal, err)
					if failFast {
						batchCancel()
					}
					return
				}
				results[index] = delegateTaskSuccessResult(goal, res)
			}(i, taskMap)
		}
		wg.Wait()
		summary := summarizeDelegateResults(results)
		summary["results"] = results
		summary["count"] = len(results)
		return summary, nil
	}
	goal := strArg(args, "goal")
	if strings.TrimSpace(goal) == "" {
		return nil, errors.New("goal or tasks is required")
	}
	res, err := runDelegateSubtask(ctx, tc.DelegateRunner, tc.SessionID, goal, strArg(args, "context"), maxIterations, timeoutSeconds)
	if err != nil {
		return delegateTaskErrorResult(goal, err), nil
	}
	return delegateTaskSuccessResult(goal, res), nil
}

func (b *BuiltinTools) approval(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.ApprovalStore == nil {
		return nil, errors.New("approval store unavailable")
	}
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	switch action {
	case "", "status":
		approved, expiresAt := tc.ApprovalStore.Status(tc.SessionID)
		return map[string]any{
			"session_id": tc.SessionID,
			"approved":   approved,
			"expires_at": expiresAt.Format(time.RFC3339),
		}, nil
	case "grant":
		ttlSeconds := intArg(args, "ttl_seconds", 0)
		expiresAt := tc.ApprovalStore.Grant(tc.SessionID, time.Duration(ttlSeconds)*time.Second)
		return map[string]any{
			"session_id": tc.SessionID,
			"approved":   true,
			"expires_at": expiresAt.Format(time.RFC3339),
		}, nil
	case "revoke":
		revoked := tc.ApprovalStore.Revoke(tc.SessionID)
		return map[string]any{
			"session_id": tc.SessionID,
			"approved":   false,
			"revoked":    revoked,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported approval action: %s", action)
	}
}

func (b *BuiltinTools) skillList(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	root, err := resolveSkillsRoot(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"path": root, "skills": []map[string]any{}}, nil
		}
		return nil, err
	}
	skills := make([]map[string]any, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		skillPath := filepath.Join(root, name, "SKILL.md")
		desc := readSkillDescription(skillPath)
		skills = append(skills, map[string]any{
			"name":        name,
			"description": desc,
			"path":        skillPath,
		})
	}
	return map[string]any{"path": root, "skills": skills, "count": len(skills)}, nil
}

func (b *BuiltinTools) skillView(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	name := strings.TrimSpace(strArg(args, "name"))
	if name == "" {
		return nil, errors.New("name required")
	}
	root, err := resolveSkillsRoot(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	path, err := resolveSkillPath(root, name)
	if err != nil {
		return nil, err
	}
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"name":    name,
		"path":    path,
		"content": string(bs),
	}, nil
}

func (b *BuiltinTools) skillManage(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	name := strings.TrimSpace(strArg(args, "name"))
	if name == "" {
		return nil, errors.New("name required")
	}
	if !skillNameRE.MatchString(name) {
		return nil, fmt.Errorf("invalid skill name: %s", name)
	}
	root, err := resolveSkillsRoot(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	skillDir := filepath.Join(root, name)
	skillMD := filepath.Join(skillDir, "SKILL.md")
	switch action {
	case "create":
		content := strArg(args, "content")
		if strings.TrimSpace(content) == "" {
			return nil, errors.New("content required for create")
		}
		if _, err := os.Stat(skillMD); err == nil {
			return nil, fmt.Errorf("skill already exists: %s", name)
		}
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(skillMD, []byte(content), 0o644); err != nil {
			return nil, err
		}
		return map[string]any{"action": action, "name": name, "path": skillMD, "success": true}, nil
	case "edit":
		content := strArg(args, "content")
		if strings.TrimSpace(content) == "" {
			return nil, errors.New("content required for edit")
		}
		if _, err := os.Stat(skillMD); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("skill not found: %s", name)
			}
			return nil, err
		}
		if err := os.WriteFile(skillMD, []byte(content), 0o644); err != nil {
			return nil, err
		}
		return map[string]any{"action": action, "name": name, "path": skillMD, "success": true}, nil
	case "patch":
		oldString := strArg(args, "old_string")
		if oldString == "" {
			return nil, errors.New("old_string required for patch")
		}
		newString, hasNew := args["new_string"]
		if !hasNew {
			return nil, errors.New("new_string required for patch")
		}
		newText, ok := newString.(string)
		if !ok {
			return nil, errors.New("new_string must be a string")
		}
		bs, err := os.ReadFile(skillMD)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("skill not found: %s", name)
			}
			return nil, err
		}
		content := string(bs)
		replaceAll := boolArg(args, "replace_all", false)
		matchCount := strings.Count(content, oldString)
		if matchCount == 0 {
			return nil, fmt.Errorf("old_string not found in %s", name)
		}
		if !replaceAll && matchCount != 1 {
			return nil, fmt.Errorf("old_string matched %d times; set replace_all=true for bulk replacement", matchCount)
		}
		var updated string
		replacements := 1
		if replaceAll {
			updated = strings.ReplaceAll(content, oldString, newText)
			replacements = matchCount
		} else {
			updated = strings.Replace(content, oldString, newText, 1)
		}
		if err := os.WriteFile(skillMD, []byte(updated), 0o644); err != nil {
			return nil, err
		}
		return map[string]any{"action": action, "name": name, "path": skillMD, "success": true, "replacements": replacements}, nil
	case "delete":
		if _, err := os.Stat(skillDir); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("skill not found: %s", name)
			}
			return nil, err
		}
		if err := os.RemoveAll(skillDir); err != nil {
			return nil, err
		}
		return map[string]any{"action": action, "name": name, "path": skillDir, "success": true}, nil
	case "write_file":
		filePath := strArg(args, "file_path")
		if strings.TrimSpace(filePath) == "" {
			return nil, errors.New("file_path required for write_file")
		}
		relativePath, err := validateSkillFilePath(filePath)
		if err != nil {
			return nil, err
		}
		fileContent, ok := args["file_content"]
		if !ok {
			return nil, errors.New("file_content required for write_file")
		}
		content, ok := fileContent.(string)
		if !ok {
			return nil, errors.New("file_content must be a string")
		}
		if _, err := os.Stat(skillDir); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("skill not found: %s", name)
			}
			return nil, err
		}
		targetPath, err := resolvePathWithinWorkdir(skillDir, relativePath)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return nil, err
		}
		return map[string]any{"action": action, "name": name, "path": targetPath, "success": true}, nil
	case "remove_file":
		filePath := strArg(args, "file_path")
		if strings.TrimSpace(filePath) == "" {
			return nil, errors.New("file_path required for remove_file")
		}
		relativePath, err := validateSkillFilePath(filePath)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(skillDir); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("skill not found: %s", name)
			}
			return nil, err
		}
		targetPath, err := resolvePathWithinWorkdir(skillDir, relativePath)
		if err != nil {
			return nil, err
		}
		if err := os.Remove(targetPath); err != nil {
			return nil, err
		}
		return map[string]any{"action": action, "name": name, "path": targetPath, "success": true}, nil
	default:
		return nil, fmt.Errorf("unsupported skill_manage action: %s", action)
	}
}

func statusFromDone(done bool) string {
	if done {
		return "done"
	}
	return "running"
}

func terminalParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}, "background": map[string]any{"type": "boolean"}, "timeout": map[string]any{"type": "integer"}, "workdir": map[string]any{"type": "string"}, "requires_approval": map[string]any{"type": "boolean"}, "approval_ttl_seconds": map[string]any{"type": "integer"}}, "required": []string{"command"}}
}
func processStatusParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"session_id": map[string]any{"type": "string"}}, "required": []string{"session_id"}}
}
func stopProcessParams() map[string]any { return processStatusParams() }
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
func delegateTaskParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"goal": map[string]any{"type": "string"}, "context": map[string]any{"type": "string"}, "max_iterations": map[string]any{"type": "integer"}, "max_concurrency": map[string]any{"type": "integer"}, "timeout_seconds": map[string]any{"type": "integer"}, "fail_fast": map[string]any{"type": "boolean"}, "tasks": map[string]any{"type": "array"}}}
}
func approvalParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"action": map[string]any{"type": "string", "enum": []string{"status", "grant", "revoke"}}, "ttl_seconds": map[string]any{"type": "integer"}}}
}
func skillListParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}}
}
func skillViewParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}, "path": map[string]any{"type": "string"}}, "required": []string{"name"}}
}
func skillManageParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":       map[string]any{"type": "string", "enum": []string{"create", "edit", "patch", "delete", "write_file", "remove_file"}},
			"name":         map[string]any{"type": "string"},
			"content":      map[string]any{"type": "string"},
			"old_string":   map[string]any{"type": "string"},
			"new_string":   map[string]any{"type": "string"},
			"replace_all":  map[string]any{"type": "boolean"},
			"file_path":    map[string]any{"type": "string"},
			"file_content": map[string]any{"type": "string"},
			"path":         map[string]any{"type": "string"},
		},
		"required": []string{"action", "name"},
	}
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

func runDelegateSubtask(ctx context.Context, runner DelegateRunner, sessionID, goal, taskContext string, maxIterations, timeoutSeconds int) (map[string]any, error) {
	if timeoutSeconds <= 0 {
		return runner.RunSubtask(ctx, sessionID, goal, taskContext, maxIterations)
	}
	taskCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	return runner.RunSubtask(taskCtx, sessionID, goal, taskContext, maxIterations)
}

func delegateTaskSuccessResult(goal string, res map[string]any) map[string]any {
	out := make(map[string]any, len(res)+3)
	for k, v := range res {
		out[k] = v
	}
	if _, ok := out["goal"]; !ok {
		out["goal"] = goal
	}
	out["status"] = "completed"
	out["success"] = true
	return out
}

func delegateTaskErrorResult(goal string, err error) map[string]any {
	return map[string]any{
		"goal":    goal,
		"status":  delegateTaskStatusFromError(err),
		"success": false,
		"error":   err.Error(),
	}
}

func delegateTaskStatusFromError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	default:
		return "failed"
	}
}

func summarizeDelegateResults(results []map[string]any) map[string]any {
	completed := 0
	failed := 0
	cancelled := 0
	timeout := 0
	for _, result := range results {
		switch strMap(result, "status") {
		case "completed":
			completed++
		case "cancelled":
			cancelled++
		case "timeout":
			timeout++
		default:
			failed++
		}
	}
	status := "completed"
	if failed > 0 {
		status = "failed"
	} else if timeout > 0 {
		status = "timeout"
	} else if cancelled > 0 {
		status = "cancelled"
	}
	return map[string]any{
		"status":          status,
		"success":         failed == 0 && timeout == 0 && cancelled == 0,
		"completed_count": completed,
		"failed_count":    failed,
		"cancelled_count": cancelled,
		"timeout_count":   timeout,
	}
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

var skillNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var skillManageAllowedSubdirs = map[string]struct{}{
	"references": {},
	"templates":  {},
	"scripts":    {},
	"assets":     {},
}

func validateSkillFilePath(filePath string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(filePath))
	if clean == "." || clean == "" {
		return "", errors.New("file_path required")
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("file_path must be relative: %s", filePath)
	}
	if strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		return "", fmt.Errorf("file_path escapes skill directory: %s", filePath)
	}
	parts := strings.Split(clean, string(os.PathSeparator))
	if len(parts) < 2 {
		return "", fmt.Errorf("file_path must be under allowed subdirectories (%s): %s", strings.Join(skillManageAllowedSubdirNames(), ", "), filePath)
	}
	if _, ok := skillManageAllowedSubdirs[parts[0]]; !ok {
		return "", fmt.Errorf("file_path must be under allowed subdirectories (%s): %s", strings.Join(skillManageAllowedSubdirNames(), ", "), filePath)
	}
	return clean, nil
}

func skillManageAllowedSubdirNames() []string {
	return []string{"references", "templates", "scripts", "assets"}
}

func resolveSkillsRoot(workdir, customPath string) (string, error) {
	if strings.TrimSpace(customPath) != "" {
		return resolvePathWithinWorkdir(workdir, customPath)
	}
	return resolvePathWithinWorkdir(workdir, "skills")
}

func resolveSkillPath(root, name string) (string, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "", errors.New("name required")
	}
	if strings.Contains(clean, "..") || strings.ContainsAny(clean, `/\`) {
		return "", fmt.Errorf("invalid skill name: %s", name)
	}
	path := filepath.Join(root, clean, "SKILL.md")
	return path, nil
}

func readSkillDescription(path string) string {
	bs, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(bs), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if line != "" {
			return line
		}
	}
	return ""
}
