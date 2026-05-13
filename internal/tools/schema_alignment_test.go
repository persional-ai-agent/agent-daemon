package tools

import (
	"reflect"
	"strings"
	"testing"
)

func TestMixtureOfAgentsSchemaDocumentsDefaults(t *testing.T) {
	props, _ := mixtureOfAgentsParams()["properties"].(map[string]any)
	maxIterations, _ := props["max_iterations"].(map[string]any)
	timeoutSeconds, _ := props["timeout_seconds"].(map[string]any)
	maxDesc, _ := maxIterations["description"].(string)
	timeoutDesc, _ := timeoutSeconds["description"].(string)
	if !strings.Contains(maxDesc, "default 12") {
		t.Fatalf("mixture_of_agents.max_iterations description=%q, want default hint", maxDesc)
	}
	if !strings.Contains(timeoutDesc, "default 180") {
		t.Fatalf("mixture_of_agents.timeout_seconds description=%q, want default hint", timeoutDesc)
	}
}

func TestVideoAnalyzeSchemaDocumentsTimeoutDefault(t *testing.T) {
	props, _ := videoAnalyzeParams()["properties"].(map[string]any)
	timeout, _ := props["timeout"].(map[string]any)
	desc, _ := timeout["description"].(string)
	if !strings.Contains(desc, "default 30") {
		t.Fatalf("video_analyze.timeout description=%q, want default hint", desc)
	}
}

func TestFeishuDriveSchemasDocumentPagingDefaults(t *testing.T) {
	check := func(name string, params map[string]any) {
		t.Helper()
		props, _ := params["properties"].(map[string]any)
		fileType, _ := props["file_type"].(map[string]any)
		pageSize, _ := props["page_size"].(map[string]any)
		ftDesc, _ := fileType["description"].(string)
		psDesc, _ := pageSize["description"].(string)
		if !strings.Contains(ftDesc, "default docx") {
			t.Fatalf("%s.file_type description=%q, want default docx", name, ftDesc)
		}
		if !strings.Contains(psDesc, "default 100") || !strings.Contains(psDesc, "max 100") {
			t.Fatalf("%s.page_size description=%q, want default/max hint", name, psDesc)
		}
	}
	check("feishu_drive_list_comments", feishuDriveListCommentsParams())
	check("feishu_drive_list_comment_replies", feishuDriveListCommentRepliesParams())
}

func TestBuiltinSchemasDocumentDefaultsAndBounds(t *testing.T) {
	terminalProps, _ := terminalParams()["properties"].(map[string]any)
	timeout, _ := terminalProps["timeout"].(map[string]any)
	if desc, _ := timeout["description"].(string); !strings.Contains(desc, "default 120") {
		t.Fatalf("terminal.timeout description=%q", desc)
	}

	processProps, _ := processParams()["properties"].(map[string]any)
	includeDone, _ := processProps["include_done"].(map[string]any)
	limit, _ := processProps["limit"].(map[string]any)
	waitTimeout, _ := processProps["timeout_seconds"].(map[string]any)
	if desc, _ := includeDone["description"].(string); !strings.Contains(desc, "default false") {
		t.Fatalf("process.include_done description=%q", desc)
	}
	if desc, _ := limit["description"].(string); !strings.Contains(desc, "default 50") {
		t.Fatalf("process.limit description=%q", desc)
	}
	if desc, _ := waitTimeout["description"].(string); !strings.Contains(desc, "default 60") {
		t.Fatalf("process.timeout_seconds description=%q", desc)
	}

	webSearchProps, _ := webSearchParams()["properties"].(map[string]any)
	webLimit, _ := webSearchProps["limit"].(map[string]any)
	if desc, _ := webLimit["description"].(string); !strings.Contains(desc, "default 5") || !strings.Contains(desc, "max 20") {
		t.Fatalf("web_search.limit description=%q", desc)
	}

	sessionProps, _ := sessionSearchParams()["properties"].(map[string]any)
	sessionLimit, _ := sessionProps["limit"].(map[string]any)
	if desc, _ := sessionLimit["description"].(string); !strings.Contains(desc, "default 5") {
		t.Fatalf("session_search.limit description=%q", desc)
	}
}

func TestMemorySchemaActionTargetEnums(t *testing.T) {
	props, _ := memoryParams()["properties"].(map[string]any)
	action, _ := props["action"].(map[string]any)
	target, _ := props["target"].(map[string]any)
	actionEnum, _ := action["enum"].([]string)
	targetEnum, _ := target["enum"].([]string)
	if !reflect.DeepEqual(actionEnum, []string{"add", "replace", "update", "delete", "remove"}) {
		t.Fatalf("memory.action enum=%v", actionEnum)
	}
	if !reflect.DeepEqual(targetEnum, []string{"memory", "memory.md", "user", "user.md"}) {
		t.Fatalf("memory.target enum=%v", targetEnum)
	}
}

func TestDelegateTaskSchemaDocumentsConditionalRequiredAndDefaults(t *testing.T) {
	props, _ := delegateTaskParams()["properties"].(map[string]any)
	goal, _ := props["goal"].(map[string]any)
	tasks, _ := props["tasks"].(map[string]any)
	failFast, _ := props["fail_fast"].(map[string]any)
	if desc, _ := goal["description"].(string); !strings.Contains(desc, "Required when tasks is empty") {
		t.Fatalf("delegate_task.goal description=%q", desc)
	}
	if desc, _ := tasks["description"].(string); !strings.Contains(desc, "goal becomes optional") {
		t.Fatalf("delegate_task.tasks description=%q", desc)
	}
	if desc, _ := failFast["description"].(string); !strings.Contains(desc, "default false") {
		t.Fatalf("delegate_task.fail_fast description=%q", desc)
	}
}
