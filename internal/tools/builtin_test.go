package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteFileResolvesRelativePathWithinWorkdir(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}

	res, err := b.writeFile(context.Background(), map[string]any{
		"path":    "notes/todo.txt",
		"content": "hello",
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	path, _ := res["path"].(string)
	if path != filepath.Join(workdir, "notes", "todo.txt") {
		t.Fatalf("unexpected resolved path: %+v", res)
	}
}

func TestReadFileRejectsPathOutsideWorkdir(t *testing.T) {
	workdir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &BuiltinTools{}

	_, err := b.readFile(context.Background(), map[string]any{
		"path": outside,
	}, ToolContext{Workdir: workdir})
	if err == nil || !strings.Contains(err.Error(), "path escapes workdir") {
		t.Fatalf("expected workdir escape error, got %v", err)
	}
}

func TestSearchFilesRejectsEscapingRoot(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}

	_, err := b.searchFiles(context.Background(), map[string]any{
		"path":    "..",
		"pattern": "hello",
	}, ToolContext{Workdir: workdir})
	if err == nil || !strings.Contains(err.Error(), "path escapes workdir") {
		t.Fatalf("expected workdir escape error, got %v", err)
	}
}

func TestTerminalBlocksHardlineCommand(t *testing.T) {
	b := &BuiltinTools{}

	_, err := b.terminal(context.Background(), map[string]any{
		"command": "rm -rf /",
	}, ToolContext{Workdir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "blocked dangerous command") {
		t.Fatalf("expected blocked dangerous command error, got %v", err)
	}
}

func TestTerminalRequiresApprovalForDangerousCommand(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()
	target := filepath.Join(workdir, "tmp-dir")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := b.terminal(context.Background(), map[string]any{
		"command": "rm -rf tmp-dir",
	}, ToolContext{Workdir: workdir})
	if err == nil || !strings.Contains(err.Error(), "requires approval") {
		t.Fatalf("expected approval required error, got %v", err)
	}
}

func TestTerminalAllowsDangerousCommandWithApproval(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()
	target := filepath.Join(workdir, "tmp-dir")
	file := filepath.Join(target, "a.txt")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := b.terminal(context.Background(), map[string]any{
		"command":           "rm -rf tmp-dir",
		"requires_approval": true,
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if res["requires_approval"] != true {
		t.Fatalf("expected requires_approval metadata, got %+v", res)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected target directory removed, stat err=%v", statErr)
	}
}

func TestTerminalHardlineStillBlockedWithApproval(t *testing.T) {
	b := &BuiltinTools{}
	_, err := b.terminal(context.Background(), map[string]any{
		"command":           "rm -rf /",
		"requires_approval": true,
	}, ToolContext{Workdir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "blocked dangerous command") {
		t.Fatalf("expected hardline block, got %v", err)
	}
}

func TestTerminalAllowsDangerousCommandWithSessionApprovalGrant(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()
	store := NewApprovalStore(time.Minute)
	target := filepath.Join(workdir, "tmp-dir")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := b.approval(context.Background(), map[string]any{
		"action":      "grant",
		"ttl_seconds": 60.0,
	}, ToolContext{SessionID: "s-approval", ApprovalStore: store, Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.terminal(context.Background(), map[string]any{
		"command": "rm -rf tmp-dir",
	}, ToolContext{SessionID: "s-approval", ApprovalStore: store, Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected target directory removed, stat err=%v", statErr)
	}
}

func TestApprovalToolStatusAndRevoke(t *testing.T) {
	b := &BuiltinTools{}
	store := NewApprovalStore(time.Minute)
	tc := ToolContext{SessionID: "s-status", ApprovalStore: store, Workdir: t.TempDir()}

	status, err := b.approval(context.Background(), map[string]any{"action": "status"}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if status["approved"] != false {
		t.Fatalf("expected unapproved status, got %+v", status)
	}

	if _, err := b.approval(context.Background(), map[string]any{"action": "grant", "ttl_seconds": 60.0}, tc); err != nil {
		t.Fatal(err)
	}
	status, err = b.approval(context.Background(), map[string]any{"action": "status"}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if status["approved"] != true {
		t.Fatalf("expected approved status, got %+v", status)
	}

	revoked, err := b.approval(context.Background(), map[string]any{"action": "revoke"}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if revoked["revoked"] != true {
		t.Fatalf("expected revoked=true, got %+v", revoked)
	}
}

func TestSkillListAndView(t *testing.T) {
	workdir := t.TempDir()
	skillDir := filepath.Join(workdir, "skills", "code-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "# Code Review\nCheck risk first."
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	b := &BuiltinTools{}
	listRes, err := b.skillList(context.Background(), map[string]any{}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	skills, _ := listRes["skills"].([]map[string]any)
	if len(skills) != 1 || skills[0]["name"] != "code-review" {
		t.Fatalf("unexpected skill list result: %+v", listRes)
	}

	viewRes, err := b.skillView(context.Background(), map[string]any{"name": "code-review"}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(viewRes["content"].(string), "Check risk first.") {
		t.Fatalf("unexpected skill view result: %+v", viewRes)
	}
}

func TestSkillViewRejectsInvalidName(t *testing.T) {
	b := &BuiltinTools{}
	_, err := b.skillView(context.Background(), map[string]any{"name": "../escape"}, ToolContext{Workdir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "invalid skill name") {
		t.Fatalf("expected invalid skill name error, got %v", err)
	}
}

func TestSkillManageCreateEditPatchDelete(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}
	name := "deploy-checklist"
	initial := "# Deploy Checklist\nStep A\n"
	edited := "# Deploy Checklist\nStep B\n"

	if _, err := b.skillManage(context.Background(), map[string]any{
		"action":  "create",
		"name":    name,
		"content": initial,
	}, tc); err != nil {
		t.Fatal(err)
	}

	if _, err := b.skillManage(context.Background(), map[string]any{
		"action":  "edit",
		"name":    name,
		"content": edited,
	}, tc); err != nil {
		t.Fatal(err)
	}

	patchRes, err := b.skillManage(context.Background(), map[string]any{
		"action":     "patch",
		"name":       name,
		"old_string": "Step B",
		"new_string": "Step C",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if patchRes["replacements"] != 1 {
		t.Fatalf("expected one replacement, got %+v", patchRes)
	}

	viewRes, err := b.skillView(context.Background(), map[string]any{"name": name}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(viewRes["content"].(string), "Step C") {
		t.Fatalf("expected patched content, got %+v", viewRes)
	}

	if _, err := b.skillManage(context.Background(), map[string]any{
		"action": "delete",
		"name":   name,
	}, tc); err != nil {
		t.Fatal(err)
	}

	_, err = b.skillView(context.Background(), map[string]any{"name": name}, tc)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected missing skill after delete, got %v", err)
	}
}

func TestSkillManageRejectsInvalidNameAndAmbiguousPatch(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}

	_, err := b.skillManage(context.Background(), map[string]any{
		"action":  "create",
		"name":    "../escape",
		"content": "x",
	}, tc)
	if err == nil || !strings.Contains(err.Error(), "invalid skill name") {
		t.Fatalf("expected invalid skill name error, got %v", err)
	}

	if _, err := b.skillManage(context.Background(), map[string]any{
		"action":  "create",
		"name":    "repeat-lines",
		"content": "foo\nfoo\n",
	}, tc); err != nil {
		t.Fatal(err)
	}

	_, err = b.skillManage(context.Background(), map[string]any{
		"action":     "patch",
		"name":       "repeat-lines",
		"old_string": "foo",
		"new_string": "bar",
	}, tc)
	if err == nil || !strings.Contains(err.Error(), "replace_all=true") {
		t.Fatalf("expected ambiguous patch error, got %v", err)
	}
}
