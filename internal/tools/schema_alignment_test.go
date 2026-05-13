package tools

import (
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
