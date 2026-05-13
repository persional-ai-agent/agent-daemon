package tools

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
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

func TestWriteFileWarnsWhenFileChangedSinceRead(t *testing.T) {
	workdir := t.TempDir()
	p := filepath.Join(workdir, "a.txt")
	if err := os.WriteFile(p, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir, SessionID: "s-stale"}

	if _, err := b.readFile(context.Background(), map[string]any{"path": "a.txt"}, tc); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(p, []byte("external"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := b.writeFile(context.Background(), map[string]any{
		"path":    "a.txt",
		"content": "new",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res["_warning"]; !ok {
		t.Fatalf("expected _warning for stale file, got %+v", res)
	}
}

func TestWriteFileRejectsInternalReadFileDedupStatusText(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}

	_, err := b.writeFile(context.Background(), map[string]any{
		"path":    "a.txt",
		"content": readFileDedupStatusMessage,
	}, ToolContext{Workdir: workdir})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "internal read_file status text") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteFileRejectsSymlinkPathEscape(t *testing.T) {
	workdir := t.TempDir()
	outside := t.TempDir()
	outsideTarget := filepath.Join(outside, "x.txt")
	if err := os.WriteFile(outsideTarget, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}

	linkDir := filepath.Join(workdir, "out")
	if err := os.Symlink(outside, linkDir); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	b := &BuiltinTools{}
	_, err := b.writeFile(context.Background(), map[string]any{
		"path":    "out/created.txt",
		"content": "hello",
	}, ToolContext{Workdir: workdir})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(outside, "created.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no write outside workdir; stat err=%v", statErr)
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
	res, err := b.terminal(context.Background(), map[string]any{
		"command": "rm -rf tmp-dir",
	}, ToolContext{Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	status, _ := res["status"].(string)
	if status != "pending_approval" {
		t.Fatalf("expected pending_approval status, got %q", status)
	}
	if id, _ := res["approval_id"].(string); id == "" {
		t.Fatal("expected approval_id in pending result")
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

func TestProcessSchemaActionEnumIncludesExtendedActions(t *testing.T) {
	registry := NewRegistry()
	RegisterBuiltins(registry, NewProcessRegistry(t.TempDir()))
	for _, schema := range registry.Schemas() {
		if schema.Function.Name != "process" {
			continue
		}
		props, _ := schema.Function.Parameters["properties"].(map[string]any)
		action, _ := props["action"].(map[string]any)
		enum, _ := action["enum"].([]string)
		want := []string{"list", "status", "poll", "log", "wait", "stop", "kill", "write"}
		if !reflect.DeepEqual(enum, want) {
			t.Fatalf("process action enum=%v, want=%v", enum, want)
		}
		return
	}
	t.Fatal("process schema not found")
}

func TestApprovalSchemaDocumentsDefaultAction(t *testing.T) {
	params := approvalParams()
	props, _ := params["properties"].(map[string]any)
	action, _ := props["action"].(map[string]any)
	desc, _ := action["description"].(string)
	if !strings.Contains(desc, "default: status") {
		t.Fatalf("approval action description=%q, want default hint", desc)
	}
}

func TestApprovalToolPatternGrantAndStatus(t *testing.T) {
	b := &BuiltinTools{}
	store := NewApprovalStore(time.Minute)
	tc := ToolContext{SessionID: "s-pattern", ApprovalStore: store, Workdir: t.TempDir()}

	grantRes, err := b.approval(context.Background(), map[string]any{
		"action":  "grant",
		"scope":   "pattern",
		"pattern": "recursive_delete",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if grantRes["scope"] != "pattern" || grantRes["pattern"] != "recursive_delete" {
		t.Fatalf("expected pattern grant, got %+v", grantRes)
	}

	statusRes, err := b.approval(context.Background(), map[string]any{"action": "status"}, tc)
	if err != nil {
		t.Fatal(err)
	}
	approvals, _ := statusRes["approvals"].([]map[string]any)
	found := false
	for _, a := range approvals {
		if a["scope"] == "pattern" && a["pattern"] == "recursive_delete" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected pattern approval in status, got %+v", statusRes)
	}
}

func TestApprovalToolPatternRevoke(t *testing.T) {
	b := &BuiltinTools{}
	store := NewApprovalStore(time.Minute)
	tc := ToolContext{SessionID: "s-revoke-pattern", ApprovalStore: store, Workdir: t.TempDir()}

	b.approval(context.Background(), map[string]any{
		"action":  "grant",
		"scope":   "pattern",
		"pattern": "recursive_delete",
	}, tc)

	revokedRes, err := b.approval(context.Background(), map[string]any{
		"action":  "revoke",
		"scope":   "pattern",
		"pattern": "recursive_delete",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	if revokedRes["revoked"] != true {
		t.Fatalf("expected pattern revoked=true, got %+v", revokedRes)
	}
}

func TestTerminalPatternApprovalAllowsMatchingCategory(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()
	store := NewApprovalStore(time.Minute)
	target := filepath.Join(workdir, "tmp-dir")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	store.GrantPattern("s-pattern", "recursive_delete", time.Minute)

	_, err := b.terminal(context.Background(), map[string]any{
		"command": "rm -rf tmp-dir",
	}, ToolContext{SessionID: "s-pattern", ApprovalStore: store, Workdir: workdir})
	if err != nil {
		t.Fatalf("expected pattern-approved command to run, got %v", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected target directory removed, stat err=%v", statErr)
	}
}

func TestTerminalPatternApprovalBlocksDifferentCategory(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()
	store := NewApprovalStore(time.Minute)

	store.GrantPattern("s-pattern", "recursive_delete", time.Minute)

	res, err := b.terminal(context.Background(), map[string]any{
		"command": "chmod 777 somefile",
	}, ToolContext{SessionID: "s-pattern", ApprovalStore: store, Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	status, _ := res["status"].(string)
	if status != "pending_approval" {
		t.Fatalf("expected pending_approval status for different category, got %q", status)
	}
}

func TestApprovalConfirmApproveAndExecute(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()
	target := filepath.Join(workdir, "tmp-dir")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := b.terminal(context.Background(), map[string]any{
		"command": "rm -rf tmp-dir",
	}, ToolContext{SessionID: "s-confirm", Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	approvalID, _ := res["approval_id"].(string)
	if approvalID == "" {
		t.Fatal("expected approval_id")
	}

	store := NewApprovalStore(time.Minute)
	confirmRes, err := b.approval(context.Background(), map[string]any{
		"action":      "confirm",
		"approval_id": approvalID,
		"approve":     true,
	}, ToolContext{SessionID: "s-confirm", ApprovalStore: store, Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := confirmRes["approved"].(bool); !v {
		t.Fatal("expected approved=true")
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected target directory removed after confirm, stat err=%v", statErr)
	}
}

func TestApprovalConfirmDeny(t *testing.T) {
	b := &BuiltinTools{}
	workdir := t.TempDir()

	res, err := b.terminal(context.Background(), map[string]any{
		"command": "rm -rf tmp-dir",
	}, ToolContext{SessionID: "s-deny", Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	approvalID, _ := res["approval_id"].(string)

	store := NewApprovalStore(time.Minute)
	confirmRes, err := b.approval(context.Background(), map[string]any{
		"action":      "confirm",
		"approval_id": approvalID,
		"approve":     false,
	}, ToolContext{SessionID: "s-deny", ApprovalStore: store, Workdir: workdir})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := confirmRes["approved"].(bool); v {
		t.Fatal("expected approved=false")
	}
}

func TestApprovalConfirmMissingID(t *testing.T) {
	b := &BuiltinTools{}
	_, err := b.approval(context.Background(), map[string]any{
		"action": "confirm",
	}, ToolContext{SessionID: "x", ApprovalStore: NewApprovalStore(time.Minute), Workdir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "approval_id required") {
		t.Fatalf("expected approval_id required error, got %v", err)
	}
}

func TestApprovalConfirmUnknownID(t *testing.T) {
	b := &BuiltinTools{}
	_, err := b.approval(context.Background(), map[string]any{
		"action":      "confirm",
		"approval_id": "nonexistent",
		"approve":     true,
	}, ToolContext{SessionID: "x", ApprovalStore: NewApprovalStore(time.Minute), Workdir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "pending approval not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestApprovalToolPatternGrantRequiresPattern(t *testing.T) {
	b := &BuiltinTools{}
	store := NewApprovalStore(time.Minute)
	tc := ToolContext{SessionID: "s-no-pattern", ApprovalStore: store, Workdir: t.TempDir()}

	_, err := b.approval(context.Background(), map[string]any{
		"action": "grant",
		"scope":  "pattern",
	}, tc)
	if err == nil || !strings.Contains(err.Error(), "pattern is required") {
		t.Fatalf("expected error for missing pattern, got %v", err)
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

func TestSkillManageWriteAndRemoveSupportingFile(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}

	if _, err := b.skillManage(context.Background(), map[string]any{
		"action":  "create",
		"name":    "ops-playbook",
		"content": "# Ops Playbook\n",
	}, tc); err != nil {
		t.Fatal(err)
	}

	writeRes, err := b.skillManage(context.Background(), map[string]any{
		"action":       "write_file",
		"name":         "ops-playbook",
		"file_path":    "references/restart.md",
		"file_content": "restart procedure",
	}, tc)
	if err != nil {
		t.Fatal(err)
	}
	writtenPath, _ := writeRes["path"].(string)
	bs, err := os.ReadFile(writtenPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(bs) != "restart procedure" {
		t.Fatalf("unexpected file content: %q", string(bs))
	}

	if _, err := b.skillManage(context.Background(), map[string]any{
		"action":    "remove_file",
		"name":      "ops-playbook",
		"file_path": "references/restart.md",
	}, tc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(writtenPath); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err=%v", err)
	}
}

func TestSkillManageRejectsInvalidSupportingFilePath(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}

	if _, err := b.skillManage(context.Background(), map[string]any{
		"action":  "create",
		"name":    "ops-playbook",
		"content": "# Ops Playbook\n",
	}, tc); err != nil {
		t.Fatal(err)
	}

	_, err := b.skillManage(context.Background(), map[string]any{
		"action":       "write_file",
		"name":         "ops-playbook",
		"file_path":    "../../escape.txt",
		"file_content": "x",
	}, tc)
	if err == nil || !strings.Contains(err.Error(), "escapes skill directory") {
		t.Fatalf("expected traversal rejection, got %v", err)
	}

	_, err = b.skillManage(context.Background(), map[string]any{
		"action":       "write_file",
		"name":         "ops-playbook",
		"file_path":    "notes.md",
		"file_content": "x",
	}, tc)
	if err == nil || !strings.Contains(err.Error(), "allowed subdirectories") {
		t.Fatalf("expected allowed-subdir rejection, got %v", err)
	}
}

func TestSkillManageSyncURL(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("# My Synced Skill\nThis is a synced skill."))
	}))
	defer srv.Close()

	workdir := t.TempDir()
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}

	res, err := b.skillManage(context.Background(), map[string]any{
		"action": "sync",
		"source": "url",
		"url":    srv.URL,
		"name":   "synced-skill",
	}, tc)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if v, _ := res["success"].(bool); !v {
		t.Fatal("expected success=true")
	}
	content, err := os.ReadFile(filepath.Join(workdir, "skills", "synced-skill", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "My Synced Skill") {
		t.Fatalf("unexpected content: %s", content)
	}
}

func TestSkillManageSyncGitHub(t *testing.T) {
	t.Skip("github sync requires real GitHub API; tested manually")
}

func TestSkillManageSyncMissingSource(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}

	_, err := b.skillManage(context.Background(), map[string]any{
		"action": "sync",
		"name":   "some-skill",
	}, tc)
	if err == nil || !strings.Contains(err.Error(), "source required") {
		t.Fatalf("expected source required error, got %v", err)
	}
}

func TestSkillManageSyncUnsupportedSource(t *testing.T) {
	workdir := t.TempDir()
	b := &BuiltinTools{}
	tc := ToolContext{Workdir: workdir}

	_, err := b.skillManage(context.Background(), map[string]any{
		"action": "sync",
		"source": "unknown",
		"name":   "some-skill",
	}, tc)
	if err == nil || !strings.Contains(err.Error(), "unsupported sync source") {
		t.Fatalf("expected unsupported source error, got %v", err)
	}
}
