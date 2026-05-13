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
	sessionID, _ := processProps["session_id"].(map[string]any)
	includeDone, _ := processProps["include_done"].(map[string]any)
	limit, _ := processProps["limit"].(map[string]any)
	offset, _ := processProps["offset"].(map[string]any)
	waitTimeout, _ := processProps["timeout_seconds"].(map[string]any)
	input, _ := processProps["input"].(map[string]any)
	if desc, _ := sessionID["description"].(string); !strings.Contains(desc, "Required for action=status/poll/log/wait/stop/kill/write") {
		t.Fatalf("process.session_id description=%q", desc)
	}
	if desc, _ := includeDone["description"].(string); !strings.Contains(desc, "default false") {
		t.Fatalf("process.include_done description=%q", desc)
	}
	if desc, _ := limit["description"].(string); !strings.Contains(desc, "default 50") {
		t.Fatalf("process.limit description=%q", desc)
	}
	if desc, _ := offset["description"].(string); !strings.Contains(desc, "default 0") {
		t.Fatalf("process.offset description=%q", desc)
	}
	if desc, _ := waitTimeout["description"].(string); !strings.Contains(desc, "default 60") {
		t.Fatalf("process.timeout_seconds description=%q", desc)
	}
	if desc, _ := input["description"].(string); !strings.Contains(desc, "required") {
		t.Fatalf("process.input description=%q", desc)
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
	items, _ := tasks["items"].(map[string]any)
	if typ, _ := items["type"].(string); typ != "object" {
		t.Fatalf("delegate_task.tasks.items.type=%q, want object", typ)
	}
	required, _ := items["required"].([]string)
	if !reflect.DeepEqual(required, []string{"goal"}) {
		t.Fatalf("delegate_task.tasks.items.required=%v, want [goal]", required)
	}
	anyOf, _ := delegateTaskParams()["anyOf"].([]any)
	if len(anyOf) != 2 {
		t.Fatalf("delegate_task.anyOf len=%d, want 2", len(anyOf))
	}
}

func TestFeishuDriveReplyAndAddSchemasDocumentFileTypeDefault(t *testing.T) {
	check := func(name string, params map[string]any) {
		t.Helper()
		props, _ := params["properties"].(map[string]any)
		fileType, _ := props["file_type"].(map[string]any)
		desc, _ := fileType["description"].(string)
		if !strings.Contains(desc, "default docx") {
			t.Fatalf("%s.file_type description=%q, want default docx", name, desc)
		}
	}
	check("feishu_drive_reply_comment", feishuDriveReplyCommentParams())
	check("feishu_drive_add_comment", feishuDriveAddCommentParams())
}

func TestSchemaMachineReadableBounds(t *testing.T) {
	assertBound := func(name string, field map[string]any, key string, want int) {
		t.Helper()
		got, _ := field[key].(int)
		if got != want {
			t.Fatalf("%s %s=%d, want %d", name, key, got, want)
		}
	}

	tp, _ := terminalParams()["properties"].(map[string]any)
	timeout, _ := tp["timeout"].(map[string]any)
	ttl, _ := tp["approval_ttl_seconds"].(map[string]any)
	assertBound("terminal.timeout", timeout, "minimum", 0)
	assertBound("terminal.approval_ttl_seconds", ttl, "minimum", 0)

	pp, _ := processParams()["properties"].(map[string]any)
	limit, _ := pp["limit"].(map[string]any)
	waitTimeout, _ := pp["timeout_seconds"].(map[string]any)
	assertBound("process.limit", limit, "minimum", 1)
	assertBound("process.timeout_seconds", waitTimeout, "minimum", 1)

	sp, _ := sessionSearchParams()["properties"].(map[string]any)
	sLimit, _ := sp["limit"].(map[string]any)
	assertBound("session_search.limit", sLimit, "minimum", 1)

	wp, _ := webSearchParams()["properties"].(map[string]any)
	wLimit, _ := wp["limit"].(map[string]any)
	assertBound("web_search.limit", wLimit, "minimum", 1)
	assertBound("web_search.limit", wLimit, "maximum", 20)

	ap, _ := approvalParams()["properties"].(map[string]any)
	aTTL, _ := ap["ttl_seconds"].(map[string]any)
	assertBound("approval.ttl_seconds", aTTL, "minimum", 0)

	rp, _ := readFileParams()["properties"].(map[string]any)
	rOffset, _ := rp["offset"].(map[string]any)
	rMaxChars, _ := rp["max_chars"].(map[string]any)
	assertBound("read_file.offset", rOffset, "minimum", 1)
	assertBound("read_file.max_chars", rMaxChars, "minimum", 1)
	assertBound("read_file.max_chars", rMaxChars, "maximum", 200000)

	bcp, _ := browserConsoleParams()["properties"].(map[string]any)
	bcLimit, _ := bcp["limit"].(map[string]any)
	assertBound("browser_console.limit", bcLimit, "minimum", 1)
	assertBound("browser_console.limit", bcLimit, "maximum", 1000)
}

func TestExternalToolSchemasMachineReadableBounds(t *testing.T) {
	assertBound := func(name string, field map[string]any, key string, want int) {
		t.Helper()
		got, _ := field[key].(int)
		if got != want {
			t.Fatalf("%s %s=%d, want %d", name, key, got, want)
		}
	}

	discordProps, _ := discordToolParams()["properties"].(map[string]any)
	discordLimit, _ := discordProps["limit"].(map[string]any)
	assertBound("discord.limit", discordLimit, "minimum", 1)
	assertBound("discord.limit", discordLimit, "maximum", 100)

	spotifySearchProps, _ := spotifySearchParams()["properties"].(map[string]any)
	spotifySearchLimit, _ := spotifySearchProps["limit"].(map[string]any)
	assertBound("spotify_search.limit", spotifySearchLimit, "minimum", 1)
	assertBound("spotify_search.limit", spotifySearchLimit, "maximum", 50)
	for _, tc := range []struct {
		name   string
		params map[string]any
	}{
		{"spotify_playlists", spotifyPlaylistsParams()},
		{"spotify_albums", spotifyAlbumsParams()},
		{"spotify_library", spotifyLibraryParams()},
	} {
		props, _ := tc.params["properties"].(map[string]any)
		limit, _ := props["limit"].(map[string]any)
		offset, _ := props["offset"].(map[string]any)
		assertBound(tc.name+".limit", limit, "minimum", 1)
		assertBound(tc.name+".limit", limit, "maximum", 50)
		assertBound(tc.name+".offset", offset, "minimum", 0)
	}

	cronProps, _ := NewCronJobTool(nil).Schema().Function.Parameters["properties"].(map[string]any)
	cronRepeat, _ := cronProps["repeat"].(map[string]any)
	cronLimit, _ := cronProps["limit"].(map[string]any)
	assertBound("cronjob.repeat", cronRepeat, "minimum", 0)
	assertBound("cronjob.limit", cronLimit, "minimum", 1)
	assertBound("cronjob.limit", cronLimit, "maximum", 200)

	for _, tc := range []struct {
		name   string
		params map[string]any
	}{
		{"feishu_drive_list_comments", feishuDriveListCommentsParams()},
		{"feishu_drive_list_comment_replies", feishuDriveListCommentRepliesParams()},
	} {
		props, _ := tc.params["properties"].(map[string]any)
		pageSize, _ := props["page_size"].(map[string]any)
		assertBound(tc.name+".page_size", pageSize, "minimum", 1)
		assertBound(tc.name+".page_size", pageSize, "maximum", 100)
	}
}

func TestProcessSchemaOneOfConditionalRequirements(t *testing.T) {
	oneOf, _ := processParams()["oneOf"].([]any)
	if len(oneOf) < 9 {
		t.Fatalf("process.oneOf len=%d, want at least 9", len(oneOf))
	}
}

func TestRegistrySchemasLint(t *testing.T) {
	registry := NewRegistry()
	RegisterBuiltins(registry, NewProcessRegistry(t.TempDir()))
	for _, schema := range registry.Schemas() {
		params := schema.Function.Parameters
		props, _ := params["properties"].(map[string]any)
		if props == nil {
			t.Fatalf("%s missing properties", schema.Function.Name)
		}
		if required, ok := params["required"].([]string); ok {
			for _, k := range required {
				if _, exists := props[k]; !exists {
					t.Fatalf("%s required field %q missing from properties", schema.Function.Name, k)
				}
			}
		}
		for key, raw := range props {
			field, _ := raw.(map[string]any)
			if field == nil {
				continue
			}
			if enum, ok := field["enum"]; ok {
				switch vv := enum.(type) {
				case []string:
					if len(vv) == 0 {
						t.Fatalf("%s.%s enum empty", schema.Function.Name, key)
					}
				case []any:
					if len(vv) == 0 {
						t.Fatalf("%s.%s enum empty", schema.Function.Name, key)
					}
				default:
					t.Fatalf("%s.%s enum has unsupported type %T", schema.Function.Name, key, enum)
				}
			}
			minV, minOK := field["minimum"].(int)
			maxV, maxOK := field["maximum"].(int)
			if minOK && maxOK && minV > maxV {
				t.Fatalf("%s.%s minimum=%d > maximum=%d", schema.Function.Name, key, minV, maxV)
			}
		}
	}
}
