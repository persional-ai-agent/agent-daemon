package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/google/uuid"
)

type BuiltinTools struct {
	proc             *ProcessRegistry
	pendingApprovals map[string]pendingApproval
	pendingMu        sync.Mutex

	readDedupMu sync.Mutex
	readDedup   map[string]readDedupEntry

	readStampMu sync.Mutex
	readStamp   map[string]int64 // key: sessionID|absPath -> modUnixNano

	readLoopMu sync.Mutex
	readLoop   map[string]readLoopState // key: sessionID

	procPollMu sync.Mutex
	procPoll   map[string]int64 // key: sessionID|procID -> last byte offset
}

type pendingApproval struct {
	ID         string
	Command    string
	CWD        string
	Category   string
	Reason     string
	Background bool
	Timeout    int
	Args       map[string]any
	ToolCtx    ToolContext
}

func (b *BuiltinTools) storePending(pa pendingApproval) {
	b.pendingMu.Lock()
	defer b.pendingMu.Unlock()
	if b.pendingApprovals == nil {
		b.pendingApprovals = make(map[string]pendingApproval)
	}
	b.pendingApprovals[pa.ID] = pa
}

func (b *BuiltinTools) retrievePending(id string) (pendingApproval, bool) {
	b.pendingMu.Lock()
	defer b.pendingMu.Unlock()
	pa, ok := b.pendingApprovals[id]
	if ok {
		delete(b.pendingApprovals, id)
	}
	return pa, ok
}

func RegisterBuiltins(r *Registry, proc *ProcessRegistry) {
	b := &BuiltinTools{proc: proc}
	r.Register(toolDef{name: "terminal", desc: "Execute shell commands on Linux", params: terminalParams(), call: b.terminal})
	r.Register(toolDef{name: "process", desc: "Process management wrapper (list/status/poll/log/wait/stop/kill/write)", params: processParams(), call: b.process})
	r.Register(toolDef{name: "process_status", desc: "Poll background process status by session_id", params: processStatusParams(), call: b.processStatus})
	r.Register(toolDef{name: "stop_process", desc: "Stop a background process", params: stopProcessParams(), call: b.stopProcess})
	r.Register(toolDef{name: "read_file", desc: "Read file from filesystem", params: readFileParams(), call: b.readFile})
	r.Register(toolDef{name: "write_file", desc: "Write content to file", params: writeFileParams(), call: b.writeFile})
	r.Register(toolDef{name: "patch", desc: "Patch file by replacing old_string with new_string", params: patchParams(), call: b.patch})
	r.Register(toolDef{name: "search_files", desc: "Search text in files", params: searchFilesParams(), call: b.searchFiles})
	// Optional filesystem admin tools (Hermes parity beyond core)
	r.Register(toolDef{name: "mkdir", desc: "Create a directory (within workdir)", params: mkdirParams(), call: b.mkdir})
	r.Register(toolDef{name: "list_dir", desc: "List directory entries (within workdir)", params: listDirParams(), call: b.listDir})
	r.Register(toolDef{name: "delete_file", desc: "Delete a file (within workdir)", params: deleteFileParams(), call: b.deleteFile})
	r.Register(toolDef{name: "move_file", desc: "Move/rename a file (within workdir)", params: moveFileParams(), call: b.moveFile})
	// Home Assistant (Hermes core tool parity; gated by HASS_URL/HASS_TOKEN)
	r.Register(toolDef{name: "ha_list_entities", desc: "List Home Assistant entities (requires HASS_URL/HASS_TOKEN)", params: haListEntitiesParams(), call: b.haListEntities})
	r.Register(toolDef{name: "ha_get_state", desc: "Get Home Assistant entity state (requires HASS_URL/HASS_TOKEN)", params: haGetStateParams(), call: b.haGetState})
	r.Register(toolDef{name: "ha_list_services", desc: "List Home Assistant services (requires HASS_URL/HASS_TOKEN)", params: haListServicesParams(), call: b.haListServices})
	r.Register(toolDef{name: "ha_call_service", desc: "Call Home Assistant service (requires HASS_URL/HASS_TOKEN)", params: haCallServiceParams(), call: b.haCallService})
	// Kanban coordination (Hermes core tool parity; local persisted board)
	r.Register(toolDef{name: "kanban_show", desc: "Show local kanban board", params: kanbanShowParams(), call: b.kanbanShow})
	r.Register(toolDef{name: "kanban_create", desc: "Create a kanban task", params: kanbanCreateParams(), call: b.kanbanCreate})
	r.Register(toolDef{name: "kanban_comment", desc: "Comment on a kanban task", params: kanbanCommentParams(), call: b.kanbanComment})
	r.Register(toolDef{name: "kanban_complete", desc: "Mark a kanban task completed", params: kanbanIDParams(), call: b.kanbanComplete})
	r.Register(toolDef{name: "kanban_block", desc: "Mark a kanban task blocked", params: kanbanBlockParams(), call: b.kanbanBlock})
	r.Register(toolDef{name: "kanban_heartbeat", desc: "Send kanban worker heartbeat", params: kanbanHeartbeatParams(), call: b.kanbanHeartbeat})
	r.Register(toolDef{name: "kanban_link", desc: "Link two kanban tasks", params: kanbanLinkParams(), call: b.kanbanLink})
	// Video analysis (optional toolset in Hermes; implemented when ffprobe is available)
	r.Register(toolDef{name: "video_analyze", desc: "Analyze video file via ffprobe (if available)", params: videoAnalyzeParams(), call: b.videoAnalyze})
	// Integration tools (credential-gated)
	r.Register(toolDef{name: "discord_admin", desc: "Discord admin tools (requires DISCORD_BOT_TOKEN)", params: discordAdminParams(), call: b.discordAdmin})
	r.Register(toolDef{name: "feishu_doc_read", desc: "Feishu/Lark doc read (requires FEISHU_APP_ID/FEISHU_APP_SECRET)", params: feishuDocReadParams(), call: b.feishuDocRead})
	r.Register(toolDef{name: "feishu_drive_list_comments", desc: "Feishu/Lark drive list comments", params: feishuDriveListCommentsParams(), call: b.feishuDriveListComments})
	r.Register(toolDef{name: "feishu_drive_list_comment_replies", desc: "Feishu/Lark drive list comment replies", params: feishuDriveListCommentRepliesParams(), call: b.feishuDriveListCommentReplies})
	r.Register(toolDef{name: "feishu_drive_add_comment", desc: "Feishu/Lark drive add comment", params: feishuDriveAddCommentParams(), call: b.feishuDriveAddComment})
	r.Register(toolDef{name: "feishu_drive_reply_comment", desc: "Feishu/Lark drive reply comment", params: feishuDriveReplyCommentParams(), call: b.feishuDriveReplyComment})
	// RL training tools (minimal local runner)
	r.Register(toolDef{name: "rl_list_environments", desc: "RL: list environments (from RL_ENVIRONMENTS)", params: rlListEnvironmentsParams(), call: b.rlListEnvironments})
	r.Register(toolDef{name: "rl_select_environment", desc: "RL: select environment (persisted in workdir)", params: rlSelectEnvParams(), call: b.rlSelectEnvironment})
	r.Register(toolDef{name: "rl_get_current_config", desc: "RL: get current config (persisted in workdir)", params: rlGetCurrentConfigParams(), call: b.rlGetCurrentConfig})
	r.Register(toolDef{name: "rl_edit_config", desc: "RL: edit current config key/value (persisted in workdir)", params: rlEditConfigParams(), call: b.rlEditConfig})
	r.Register(toolDef{name: "rl_start_training", desc: "RL: start training via RL_TRAIN_COMMAND (background)", params: rlStartTrainingParams(), call: b.rlStartTraining})
	r.Register(toolDef{name: "rl_stop_training", desc: "RL: stop training background process", params: rlStopTrainingParams(), call: b.rlStopTraining})
	r.Register(toolDef{name: "rl_check_status", desc: "RL: check training status", params: rlCheckStatusParams(), call: b.rlCheckStatus})
	r.Register(toolDef{name: "rl_get_results", desc: "RL: get latest training metadata", params: rlGetResultsParams(), call: b.rlGetResults})
	r.Register(toolDef{name: "rl_list_runs", desc: "RL: list runs (minimal)", params: rlListRunsParams(), call: b.rlListRuns})
	r.Register(toolDef{name: "rl_test_inference", desc: "RL: test inference via RL_INFER_COMMAND", params: rlTestInferenceParams(), call: b.rlTestInference})
	// Spotify tools (requires SPOTIFY_ACCESS_TOKEN)
	r.Register(toolDef{name: "spotify_search", desc: "Spotify search", params: spotifySearchParams(), call: b.spotifySearch})
	r.Register(toolDef{name: "spotify_devices", desc: "Spotify devices", params: spotifyDevicesParams(), call: b.spotifyDevices})
	r.Register(toolDef{name: "spotify_playback", desc: "Spotify playback control/status", params: spotifyPlaybackParams(), call: b.spotifyPlayback})
	r.Register(toolDef{name: "spotify_queue", desc: "Spotify queue get/add", params: spotifyQueueParams(), call: b.spotifyQueue})
	r.Register(toolDef{name: "spotify_playlists", desc: "Spotify playlists", params: spotifyPlaylistsParams(), call: b.spotifyPlaylists})
	r.Register(toolDef{name: "spotify_albums", desc: "Spotify saved albums", params: spotifyAlbumsParams(), call: b.spotifyAlbums})
	r.Register(toolDef{name: "spotify_library", desc: "Spotify saved tracks", params: spotifyLibraryParams(), call: b.spotifyLibrary})
	// Yuanbao tools (requires gateway adapter + YUANBAO_*)
	r.Register(toolDef{name: "yb_search_sticker", desc: "Yuanbao search sticker (built-in catalogue)", params: ybSearchStickerParams(), call: b.ybSearchSticker})
	r.Register(toolDef{name: "yb_send_dm", desc: "Yuanbao send DM (requires gateway adapter + YUANBAO_*)", params: ybSendParams(), call: b.ybSendDM})
	r.Register(toolDef{name: "yb_send_sticker", desc: "Yuanbao send sticker (TIMFaceElem; requires gateway adapter + YUANBAO_*)", params: ybSendParams(), call: b.ybSendSticker})
	r.Register(toolDef{name: "yb_query_group_info", desc: "Yuanbao query group info (requires gateway adapter + YUANBAO_*)", params: ybQueryGroupInfoParams(), call: b.ybQueryGroupInfo})
	r.Register(toolDef{name: "yb_query_group_members", desc: "Yuanbao query group members (requires gateway adapter + YUANBAO_*)", params: ybQueryGroupMembersParams(), call: b.ybQueryGroupMembers})
	// Hermes-compatible tools (minimal local implementations)
	r.Register(toolDef{name: "vision_analyze", desc: "Vision analysis (minimal: image metadata)", params: visionAnalyzeParams(), call: b.visionAnalyze})
	r.Register(toolDef{name: "image_generate", desc: "Image generation (minimal implementation with backend fallback)", params: imageGenerateParams(), call: b.imageGenerate})
	r.Register(toolDef{name: "text_to_speech", desc: "Text-to-speech (minimal implementation with backend fallback)", params: textToSpeechParams(), call: b.textToSpeech})
	r.Register(toolDef{name: "mixture_of_agents", desc: "Mixture-of-agents: run multiple subagents and synthesize results", params: mixtureOfAgentsParams(), call: b.mixtureOfAgents})
	// Browser automation (lightweight HTTP fetch + snapshot)
	r.Register(toolDef{name: "browser_navigate", desc: "Browser navigate (lightweight HTTP fetch; no JS)", params: browserNavigateParams(), call: b.browserNavigate})
	r.Register(toolDef{name: "browser_snapshot", desc: "Browser snapshot (lightweight HTML->text)", params: browserSnapshotParams(), call: b.browserSnapshot})
	r.Register(toolDef{name: "browser_back", desc: "Browser back (lightweight history stack)", params: browserBackParams(), call: b.browserBack})
	r.Register(toolDef{name: "browser_click", desc: "Browser click (lightweight: follow <a href> by text match)", params: browserClickParams(), call: b.browserClick})
	r.Register(toolDef{name: "browser_type", desc: "Browser type (lightweight: stored for best-effort form submit)", params: browserTypeParams(), call: b.browserType})
	r.Register(toolDef{name: "browser_scroll", desc: "Browser scroll (lightweight: no-op)", params: browserScrollParams(), call: b.browserScroll})
	r.Register(toolDef{name: "browser_press", desc: "Browser press (lightweight: Enter submits first GET form; otherwise no-op)", params: browserPressParams(), call: b.browserPress})
	r.Register(toolDef{name: "browser_get_images", desc: "Browser get images (lightweight: parse <img src>, supports limit)", params: browserGetImagesParams(), call: b.browserGetImages})
	r.Register(toolDef{name: "browser_vision", desc: "Browser vision (lightweight: fetch <img src> metadata)", params: browserVisionParams(), call: b.browserVision})
	r.Register(toolDef{name: "browser_console", desc: "Browser console output and JS errors (CDP-backed when configured)", params: browserConsoleParams(), call: b.browserConsole})
	r.Register(toolDef{name: "browser_cdp", desc: "Browser CDP (lightweight: page metadata)", params: browserCDPParams(), call: b.browserCDP})
	r.Register(toolDef{name: "browser_dialog", desc: "Respond to a native JS dialog (CDP-backed when configured)", params: browserDialogParams(), call: b.browserDialog})

	r.Register(toolDef{name: "todo", desc: "Maintain per-session todo list", params: todoParams(), call: b.todo})
	r.Register(toolDef{name: "memory", desc: "Manage persistent MEMORY.md/USER.md", params: memoryParams(), call: b.memory})
	r.Register(toolDef{name: "session_search", desc: "Search previous session messages", params: sessionSearchParams(), call: b.sessionSearch})
	r.Register(toolDef{name: "web_fetch", desc: "Fetch URL content over HTTP", params: webFetchParams(), call: b.webFetch})
	r.Register(toolDef{name: "web_search", desc: "Search the web (DuckDuckGo HTML scrape)", params: webSearchParams(), call: b.webSearch})
	r.Register(toolDef{name: "web_extract", desc: "Extract readable text from a URL", params: webExtractParams(), call: b.webExtract})
	r.Register(toolDef{name: "clarify", desc: "Ask the user a clarifying question with optional choices", params: clarifyParams(), call: b.clarify})
	r.Register(toolDef{name: "delegate_task", desc: "Run a child agent on a subtask or a batch of subtasks", params: delegateTaskParams(), call: b.delegateTask})
	r.Register(toolDef{name: "approval", desc: "Manage session-level dangerous command approvals", params: approvalParams(), call: b.approval})
	r.Register(toolDef{name: "skill_list", desc: "List available local skills", params: skillListParams(), call: b.skillList})
	r.Register(toolDef{name: "skills_list", desc: "Alias of skill_list (Hermes-compatible)", params: skillListParams(), call: b.skillList})
	r.Register(toolDef{name: "skill_search", desc: "Search for skills from GitHub repositories (e.g. anthropics/skills)", params: skillSearchParams(), call: b.skillSearch})
	r.Register(toolDef{name: "skill_view", desc: "Read a local skill by name", params: skillViewParams(), call: b.skillView})
	r.Register(toolDef{name: "skill_manage", desc: "Manage local skills (create/edit/patch/delete)", params: skillManageParams(), call: b.skillManage})

	// Hermes `discord` tool (minimal parity; separate from discord_admin).
	r.Register(toolDef{name: "discord", desc: "Discord server introspection (requires DISCORD_BOT_TOKEN)", params: discordToolParams(), call: b.discordTool})
}

func stubCall(name string) func(context.Context, map[string]any, ToolContext) (map[string]any, error) {
	return func(_ context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
		return map[string]any{
			"success":   false,
			"tool":      name,
			"error":     "not implemented in agent-daemon",
			"hint":      "This tool exists as an interface-alignment stub. Use other tools or extend agent-daemon to implement it.",
			"available": false,
		}, nil
	}
}

func (b *BuiltinTools) patch(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	mode := strings.ToLower(strings.TrimSpace(strArg(args, "mode")))
	if mode == "" {
		mode = "replace"
	}
	if mode == "patch" {
		patchText := strArg(args, "patch")
		if strings.TrimSpace(patchText) == "" {
			return nil, errors.New("patch required when mode=patch")
		}
		ops, err := ParseV4APatch(patchText)
		if err != nil {
			return nil, err
		}
		applyRes, err := ApplyV4AOperations(ops, tc.Workdir)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"success":    true,
			"mode":       "patch",
			"operations": len(ops),
			"result":     applyRes,
		}, nil
	}
	if mode != "replace" {
		return nil, fmt.Errorf("unsupported patch mode: %s", mode)
	}
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	if err := rejectNonRegularFile(path); err != nil {
		return nil, err
	}
	staleWarning, warnErr := b.checkStaleSinceRead(tc.SessionID, path)
	if warnErr != nil {
		return nil, warnErr
	}
	oldString := strArg(args, "old_string")
	if oldString == "" {
		return nil, errors.New("old_string required")
	}
	newString, hasNew := args["new_string"]
	if !hasNew {
		return nil, errors.New("new_string required")
	}
	newText, ok := newString.(string)
	if !ok {
		return nil, errors.New("new_string must be a string")
	}
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(bs)
	replaceAll := boolArg(args, "replace_all", false)
	matchCount := strings.Count(content, oldString)
	if matchCount == 0 {
		return nil, fmt.Errorf("old_string not found in %s", path)
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
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return nil, err
	}
	res := map[string]any{"success": true, "path": path, "replacements": replacements}
	if staleWarning != "" {
		res["_warning"] = staleWarning
	}
	if info, err := os.Stat(path); err == nil {
		b.recordReadStamp(tc.SessionID, path, info.ModTime().UnixNano())
	}
	return res, nil
}

const readFileDedupStatusMessage = "File unchanged since last read. Refer to the earlier read_file result in this conversation instead of re-reading."

type readDedupEntry struct {
	size        int64
	modUnixNano int64
}

type readLoopState struct {
	lastKey     string
	consecutive int
}

func readDedupKey(sessionID, path string, offset, limit int, withLineNumbers bool, maxChars int) string {
	return fmt.Sprintf("%s|%s|%d|%d|%t|%d", sessionID, path, offset, limit, withLineNumbers, maxChars)
}

func readLoopKey(path string, offset, limit int) string {
	return fmt.Sprintf("%s|%d|%d", path, offset, limit)
}

func (b *BuiltinTools) isUnchangedRead(key string, entry readDedupEntry) bool {
	b.readDedupMu.Lock()
	defer b.readDedupMu.Unlock()
	if b.readDedup == nil {
		return false
	}
	prev, ok := b.readDedup[key]
	if !ok {
		return false
	}
	return prev.size == entry.size && prev.modUnixNano == entry.modUnixNano
}

func (b *BuiltinTools) rememberRead(key string, entry readDedupEntry) {
	b.readDedupMu.Lock()
	defer b.readDedupMu.Unlock()
	if b.readDedup == nil {
		b.readDedup = make(map[string]readDedupEntry)
	}
	if len(b.readDedup) > 1000 {
		// Best-effort bound: drop everything (simple and predictable).
		b.readDedup = make(map[string]readDedupEntry)
	}
	b.readDedup[key] = entry
}

func (b *BuiltinTools) bumpReadLoop(sessionID, key string) int {
	if sessionID == "" || key == "" {
		return 1
	}
	b.readLoopMu.Lock()
	defer b.readLoopMu.Unlock()
	if b.readLoop == nil {
		b.readLoop = make(map[string]readLoopState)
	}
	st := b.readLoop[sessionID]
	if st.lastKey == key {
		st.consecutive++
	} else {
		st.lastKey = key
		st.consecutive = 1
	}
	b.readLoop[sessionID] = st
	return st.consecutive
}

func (b *BuiltinTools) resetReadLoop(sessionID, key string) {
	if sessionID == "" {
		return
	}
	b.readLoopMu.Lock()
	defer b.readLoopMu.Unlock()
	if b.readLoop == nil {
		return
	}
	st := b.readLoop[sessionID]
	if st.lastKey == key {
		st.consecutive = 0
		b.readLoop[sessionID] = st
	}
}

func readStampKey(sessionID, path string) string {
	return sessionID + "|" + path
}

func (b *BuiltinTools) recordReadStamp(sessionID, path string, modUnixNano int64) {
	if sessionID == "" || path == "" || modUnixNano == 0 {
		return
	}
	key := readStampKey(sessionID, path)
	b.readStampMu.Lock()
	defer b.readStampMu.Unlock()
	if b.readStamp == nil {
		b.readStamp = make(map[string]int64)
	}
	if len(b.readStamp) > 1000 {
		b.readStamp = make(map[string]int64)
	}
	b.readStamp[key] = modUnixNano
}

func (b *BuiltinTools) checkStaleSinceRead(sessionID, path string) (string, error) {
	if sessionID == "" || path == "" {
		return "", nil
	}
	key := readStampKey(sessionID, path)
	b.readStampMu.Lock()
	readStamp := int64(0)
	if b.readStamp != nil {
		readStamp = b.readStamp[key]
	}
	b.readStampMu.Unlock()
	if readStamp == 0 {
		return "", nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	now := info.ModTime().UnixNano()
	if now != readStamp {
		return fmt.Sprintf("Warning: %s was modified since you last read it (external edit or concurrent agent). The content you read may be stale. Consider re-reading before writing.", path), nil
	}
	return "", nil
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

	requiresApproval := boolArg(args, "requires_approval", false)

	// Optional Tirith pre-exec scanning (Hermes parity). This is treated as an
	// approvable warning/block unless already approved.
	if tr := checkCommandSecurityWithTirith(ctx, command); tr.Action == "warn" || tr.Action == "block" {
		category := "tirith_" + tr.Action
		sessionApproved := tc.ApprovalStore != nil && tc.ApprovalStore.IsApproved(tc.SessionID)
		patternApproved := tc.ApprovalStore != nil && tc.ApprovalStore.IsApprovedPattern(tc.SessionID, category)
		if !requiresApproval && !sessionApproved && !patternApproved {
			approvalID := uuid.NewString()
			reason := strings.TrimSpace(tr.Summary)
			if reason == "" {
				reason = "tirith security " + tr.Action
			}
			b.storePending(pendingApproval{
				ID:         approvalID,
				Command:    command,
				CWD:        cwd,
				Category:   category,
				Reason:     reason,
				Background: background,
				Timeout:    timeout,
				Args:       args,
				ToolCtx:    tc,
			})
			return map[string]any{
				"success":     false,
				"action":      "terminal",
				"status":      "pending_approval",
				"approval_id": approvalID,
				"approved":    false,
				"command":     command,
				"category":    category,
				"reason":      reason,
				"tirith": map[string]any{
					"action":   tr.Action,
					"summary":  tr.Summary,
					"findings": tr.Findings,
				},
				"instruction": "Use approval action=confirm with this approval_id to approve or deny",
			}, nil
		}
	}

	if category, reason, dangerous := detectDangerousCommand(command); dangerous {
		sessionApproved := tc.ApprovalStore != nil && tc.ApprovalStore.IsApproved(tc.SessionID)
		patternApproved := tc.ApprovalStore != nil && tc.ApprovalStore.IsApprovedPattern(tc.SessionID, category)
		if !requiresApproval && !sessionApproved && !patternApproved {
			approvalID := uuid.NewString()
			b.storePending(pendingApproval{
				ID:         approvalID,
				Command:    command,
				CWD:        cwd,
				Category:   category,
				Reason:     reason,
				Background: background,
				Timeout:    timeout,
				Args:       args,
				ToolCtx:    tc,
			})
			return map[string]any{
				"success":     false,
				"action":      "terminal",
				"status":      "pending_approval",
				"approval_id": approvalID,
				"approved":    false,
				"command":     command,
				"category":    category,
				"reason":      reason,
				"instruction": "Use approval action=confirm with this approval_id to approve or deny",
			}, nil
		}
		if requiresApproval && tc.ApprovalStore != nil {
			ttlSeconds := intArg(args, "approval_ttl_seconds", 0)
			tc.ApprovalStore.Grant(tc.SessionID, time.Duration(ttlSeconds)*time.Second)
		}
	}
	if background {
		s, err := b.proc.StartBackground(ctx, command, cwd)
		if err != nil {
			return nil, err
		}
		if boolArg(args, "notify_on_complete", false) && tc.ToolEventSink != nil {
			sessionID := tc.SessionID
			procID := s.ID
			go func() {
				deadline := time.Now().Add(2 * time.Hour)
				for {
					if time.Now().After(deadline) {
						return
					}
					ps, ok := b.proc.Poll(procID)
					if ok && ps.Done {
						tc.ToolEventSink("process_complete", map[string]any{
							"session_id":  sessionID,
							"process_id":  procID,
							"exit_code":   ps.ExitCode,
							"error":       ps.Err,
							"output_file": ps.OutputFile,
						})
						return
					}
					time.Sleep(1 * time.Second)
				}
			}()
		}
		return map[string]any{"success": true, "output": "background process started", "session_id": s.ID, "output_file": s.OutputFile, "status": "running", "exit_code": 0, "requires_approval": requiresApproval}, nil
	}
	out, code, err := RunForeground(ctx, command, cwd, timeout)
	res := map[string]any{"success": err == nil && code == 0, "output": out, "exit_code": code, "error": nil, "requires_approval": requiresApproval}
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
	return map[string]any{"success": true, "session_id": id, "status": statusFromDone(s.Done), "exit_code": s.ExitCode, "error": s.Err, "output_file": s.OutputFile}, nil
}

func (b *BuiltinTools) stopProcess(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	if err := b.proc.Stop(id); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "session_id": id, "stopped": true}, nil
}

func (b *BuiltinTools) process(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	if action == "" {
		action = "status"
	}
	switch action {
	case "list":
		includeDone := boolArg(args, "include_done", false)
		limit := intArg(args, "limit", 50)
		if b.proc == nil {
			return nil, errors.New("process registry unavailable")
		}
		items := b.proc.List(includeDone, limit)
		out := make([]map[string]any, 0, len(items))
		for _, s := range items {
			out = append(out, map[string]any{
				"session_id":  s.ID,
				"command":     s.Command,
				"started_at":  s.StartedAt.Format(time.RFC3339),
				"status":      statusFromDone(s.Done),
				"exit_code":   s.ExitCode,
				"output_file": s.OutputFile,
				"error":       s.Err,
			})
		}
		return map[string]any{"success": true, "count": len(out), "processes": out}, nil
	case "status":
		return b.processStatus(ctx, args, tc)
	case "poll":
		return b.processPoll(ctx, args, tc)
	case "log":
		return b.processLog(ctx, args, tc)
	case "wait":
		return b.processWait(ctx, args, tc)
	case "stop":
		return b.processTerminate(ctx, args, tc)
	case "kill":
		return b.processKill(ctx, args, tc)
	case "write":
		return b.processWrite(ctx, args, tc)
	default:
		return nil, fmt.Errorf("unsupported process action: %s", action)
	}
}

func (b *BuiltinTools) processTerminate(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	if b.proc == nil {
		return nil, errors.New("process registry unavailable")
	}
	if err := b.proc.Terminate(id); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "session_id": id, "stopped": true, "signal": "TERM"}, nil
}

func (b *BuiltinTools) processKill(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	if b.proc == nil {
		return nil, errors.New("process registry unavailable")
	}
	if err := b.proc.Kill(id); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "session_id": id, "killed": true, "signal": "KILL"}, nil
}

func (b *BuiltinTools) processWrite(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	input := strArg(args, "input")
	if input == "" {
		return nil, errors.New("input required")
	}
	if b.proc == nil {
		return nil, errors.New("process registry unavailable")
	}
	if err := b.proc.Write(id, input); err != nil {
		return map[string]any{"success": false, "session_id": id, "error": err.Error()}, nil
	}
	return map[string]any{"success": true, "session_id": id, "written": len(input)}, nil
}

func procPollKey(sessionID, procID string) string { return sessionID + "|" + procID }

func (b *BuiltinTools) getProcOffset(sessionID, procID string) int64 {
	if sessionID == "" || procID == "" {
		return 0
	}
	key := procPollKey(sessionID, procID)
	b.procPollMu.Lock()
	defer b.procPollMu.Unlock()
	if b.procPoll == nil {
		b.procPoll = make(map[string]int64)
	}
	return b.procPoll[key]
}

func (b *BuiltinTools) setProcOffset(sessionID, procID string, off int64) {
	if sessionID == "" || procID == "" {
		return
	}
	key := procPollKey(sessionID, procID)
	b.procPollMu.Lock()
	defer b.procPollMu.Unlock()
	if b.procPoll == nil {
		b.procPoll = make(map[string]int64)
	}
	if len(b.procPoll) > 2000 {
		b.procPoll = make(map[string]int64)
	}
	b.procPoll[key] = off
}

func (b *BuiltinTools) processLog(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	if b.proc == nil {
		return nil, errors.New("process registry unavailable")
	}
	s, ok := b.proc.Poll(id)
	if !ok {
		return nil, fmt.Errorf("process not found: %s", id)
	}
	offset := int64(intArg(args, "offset", 0))
	maxChars := intArg(args, "max_chars", 50_000)
	if maxChars <= 0 {
		maxChars = 50_000
	}
	if maxChars > 200_000 {
		maxChars = 200_000
	}
	content, nextOffset, truncated, err := readFileFromOffset(s.OutputFile, offset, int64(maxChars))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"success":     true,
		"session_id":  id,
		"status":      statusFromDone(s.Done),
		"exit_code":   s.ExitCode,
		"error":       s.Err,
		"output_file": s.OutputFile,
		"offset":      offset,
		"next_offset": nextOffset,
		"truncated":   truncated,
		"content":     content,
	}, nil
}

func (b *BuiltinTools) processPoll(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	if b.proc == nil {
		return nil, errors.New("process registry unavailable")
	}
	s, ok := b.proc.Poll(id)
	if !ok {
		return nil, fmt.Errorf("process not found: %s", id)
	}
	maxChars := intArg(args, "max_chars", 20_000)
	if maxChars <= 0 {
		maxChars = 20_000
	}
	if maxChars > 200_000 {
		maxChars = 200_000
	}
	offset := b.getProcOffset(tc.SessionID, id)
	content, nextOffset, truncated, err := readFileFromOffset(s.OutputFile, offset, int64(maxChars))
	if err != nil {
		return nil, err
	}
	b.setProcOffset(tc.SessionID, id, nextOffset)
	return map[string]any{
		"success":     true,
		"session_id":  id,
		"status":      statusFromDone(s.Done),
		"done":        s.Done,
		"exit_code":   s.ExitCode,
		"error":       s.Err,
		"output_file": s.OutputFile,
		"offset":      offset,
		"next_offset": nextOffset,
		"truncated":   truncated,
		"new_output":  content,
	}, nil
}

func (b *BuiltinTools) processWait(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	id := strArg(args, "session_id")
	if id == "" {
		return nil, errors.New("session_id required")
	}
	timeoutSec := intArg(args, "timeout_seconds", 60)
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for {
		if b.proc == nil {
			return nil, errors.New("process registry unavailable")
		}
		s, ok := b.proc.Poll(id)
		if !ok {
			return nil, fmt.Errorf("process not found: %s", id)
		}
		if s.Done {
			// Return a final poll payload (including any unread output).
			args2 := map[string]any{
				"session_id": id,
				"max_chars":  intArg(args, "max_chars", 50_000),
			}
			out, err := b.processPoll(ctx, args2, tc)
			if err != nil {
				return nil, err
			}
			out["waited"] = true
			return out, nil
		}
		if time.Now().After(deadline) {
			out, err := b.processPoll(ctx, map[string]any{"session_id": id, "max_chars": intArg(args, "max_chars", 20_000)}, tc)
			if err != nil {
				return nil, err
			}
			out["waited"] = false
			out["timeout_seconds"] = timeoutSec
			return out, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func readFileFromOffset(path string, offset int64, maxBytes int64) (content string, nextOffset int64, truncated bool, err error) {
	if maxBytes <= 0 {
		maxBytes = 20_000
	}
	f, err := os.Open(path)
	if err != nil {
		return "", 0, false, err
	}
	defer f.Close()
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, 0); err != nil {
		return "", 0, false, err
	}
	buf := make([]byte, maxBytes)
	n, rerr := f.Read(buf)
	if rerr != nil && !errors.Is(rerr, io.EOF) {
		return "", 0, false, rerr
	}
	nextOffset = offset + int64(n)
	content = string(buf[:n])
	// Determine truncation by checking if more data exists.
	st, statErr := f.Stat()
	if statErr == nil && nextOffset < st.Size() {
		truncated = true
	}
	return content, nextOffset, truncated, nil
}

func (b *BuiltinTools) readFile(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	if err := rejectNonRegularFile(path); err != nil {
		return nil, err
	}
	offset := intArg(args, "offset", 1)
	limit := intArg(args, "limit", 0)
	withLineNumbers := boolArg(args, "with_line_numbers", false)
	maxChars := intArg(args, "max_chars", 100_000)
	dedup := boolArg(args, "dedup", true)
	rejectOnTruncate := boolArg(args, "reject_on_truncate", true)
	loopKey := ""
	loopCount := 1
	if tc.SessionID != "" {
		loopKey = readLoopKey(path, offset, limit)
		loopCount = b.bumpReadLoop(tc.SessionID, loopKey)
	}
	if maxChars <= 0 {
		maxChars = 100_000
	}
	if maxChars > 200_000 {
		maxChars = 200_000
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if tc.SessionID != "" {
		if stat, statErr := os.Stat(path); statErr == nil {
			entry := readDedupEntry{size: stat.Size(), modUnixNano: stat.ModTime().UnixNano()}
			b.recordReadStamp(tc.SessionID, path, entry.modUnixNano)
			if dedup {
				key := readDedupKey(tc.SessionID, path, offset, limit, withLineNumbers, maxChars)
				if b.isUnchangedRead(key, entry) {
					if loopCount >= 4 {
						return map[string]any{
							"success":      false,
							"error":        fmt.Sprintf("BLOCKED: You have read this exact file region %d times in a row and the content has NOT changed. Stop re-reading and proceed.", loopCount),
							"path":         path,
							"already_read": loopCount,
						}, nil
					}
					res := map[string]any{
						"success":          true,
						"path":             path,
						"dedup":            true,
						"status":           "unchanged",
						"message":          readFileDedupStatusMessage,
						"content_returned": false,
					}
					if loopCount >= 3 {
						res["_warning"] = fmt.Sprintf("You have read this exact file region %d times consecutively and it has not changed. Use the information you already have.", loopCount)
					}
					return res, nil
				}
				b.rememberRead(key, entry)
			}
		}
	}

	s := bufio.NewScanner(f)
	// Allow longer lines than bufio.Scanner's default 64K token limit.
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	out := make([]string, 0)
	totalChars := 0
	truncated := false
	for s.Scan() {
		lineNo++
		if lineNo < offset {
			continue
		}
		// Enforce size guard (approximate by rune count in line + newline).
		line := s.Text()
		remaining := maxChars - totalChars
		if remaining <= 0 {
			truncated = true
			break
		}
		if withLineNumbers {
			prefixed := fmt.Sprintf("%d→%s", lineNo, line)
			if len(prefixed) > remaining {
				prefixed = prefixed[:remaining]
				truncated = true
			}
			out = append(out, prefixed)
			totalChars += len(prefixed) + 1
		} else {
			if len(line) > remaining {
				line = line[:remaining]
				truncated = true
			}
			out = append(out, line)
			totalChars += len(line) + 1
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		if truncated {
			break
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	startLine := offset
	endLine := offset + len(out) - 1
	if len(out) == 0 {
		endLine = offset - 1
	}
	if truncated && rejectOnTruncate {
		fileSize := int64(0)
		if info, err := os.Stat(path); err == nil {
			fileSize = info.Size()
		}
		totalLines := 0
		if f2, err := os.Open(path); err == nil {
			defer f2.Close()
			s2 := bufio.NewScanner(f2)
			s2.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for s2.Scan() {
				totalLines++
			}
		}
		return map[string]any{
			"success":     false,
			"error":       fmt.Sprintf("Read produced more than %d characters (max_chars safety limit). Use offset/limit or increase max_chars (<=200000) to read a smaller range.", maxChars),
			"path":        path,
			"file_size":   fileSize,
			"total_lines": totalLines,
			"offset":      offset,
			"limit":       limit,
		}, nil
	}
	return map[string]any{
		"success":           true,
		"path":              path,
		"content":           strings.Join(out, "\n"),
		"start_line":        startLine,
		"end_line":          endLine,
		"with_line_numbers": withLineNumbers,
		"max_chars":         maxChars,
		"truncated":         truncated,
		"dedup":             false,
	}, nil
}

func (b *BuiltinTools) writeFile(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	content := strArg(args, "content")
	if err != nil {
		return nil, err
	}
	if isInternalReadFileStatusText(content) {
		return nil, errors.New("refusing to write internal read_file status text as file content; re-read the file or reconstruct the intended contents before writing")
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	if err := rejectNonRegularFile(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	staleWarning := ""
	if _, statErr := os.Stat(path); statErr == nil {
		w, warnErr := b.checkStaleSinceRead(tc.SessionID, path)
		if warnErr != nil {
			return nil, warnErr
		}
		staleWarning = w
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	res := map[string]any{"success": true, "path": path, "bytes": len(content), "written": true}
	if staleWarning != "" {
		res["_warning"] = staleWarning
	}
	if info, err := os.Stat(path); err == nil {
		b.recordReadStamp(tc.SessionID, path, info.ModTime().UnixNano())
	}
	return res, nil
}

func isInternalReadFileStatusText(content string) bool {
	stripped := strings.TrimSpace(content)
	if stripped == readFileDedupStatusMessage {
		return true
	}
	if strings.Contains(stripped, readFileDedupStatusMessage) && len(stripped) <= 2*len(readFileDedupStatusMessage) {
		return true
	}
	return false
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
	return map[string]any{"success": true, "count": len(matches), "matches": matches}, nil
}

func (b *BuiltinTools) todo(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.TodoStore == nil {
		return nil, errors.New("todo store unavailable")
	}
	merge := boolArg(args, "merge", false)
	val, ok := args["todos"].([]any)
	if !ok {
		return map[string]any{"success": true, "todos": tc.TodoStore.List(tc.SessionID)}, nil
	}
	items := make([]TodoItem, 0, len(val))
	for _, x := range val {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, TodoItem{ID: strMap(m, "id"), Content: strMap(m, "content"), Status: strMap(m, "status"), Priority: strMap(m, "priority")})
	}
	return map[string]any{"success": true, "todos": tc.TodoStore.Update(tc.SessionID, items, merge)}, nil
}

func (b *BuiltinTools) memory(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.MemoryStore == nil {
		return nil, errors.New("memory store unavailable")
	}
	res, err := tc.MemoryStore.Manage(strArg(args, "action"), strArg(args, "target"), strArg(args, "content"), strArg(args, "old_text"))
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = map[string]any{}
	}
	res["success"] = true
	return res, nil
}

func (b *BuiltinTools) sessionSearch(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.SessionStore == nil {
		return nil, errors.New("session store unavailable")
	}
	query := strArg(args, "query")
	limit := intArg(args, "limit", 5)
	exclude := strings.TrimSpace(strArg(args, "exclude_session_id"))
	includeCurrent := boolArg(args, "include_current_session", false)
	if includeCurrent {
		exclude = ""
	}
	if exclude == "" && !includeCurrent {
		exclude = tc.SessionID
	}
	rows, err := tc.SessionStore.Search(query, limit, exclude)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "query": query, "results": rows, "exclude_session_id": exclude}, nil
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
	return map[string]any{"success": true, "status": resp.StatusCode, "url": url, "content": string(body)}, nil
}

func (b *BuiltinTools) webSearch(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	query := strings.TrimSpace(strArg(args, "query"))
	if query == "" {
		return nil, errors.New("query required")
	}
	limit := intArg(args, "limit", 5)
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}
	baseURL := strings.TrimSpace(strArg(args, "base_url"))
	if baseURL == "" {
		baseURL = "https://duckduckgo.com/html/"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()
	bs, err := fetchHTTPBytes(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("fetch search: %w", err)
	}
	results := parseDuckDuckGoHTMLResults(string(bs), limit)
	return map[string]any{"success": true, "query": query, "count": len(results), "results": results}, nil
}

func (b *BuiltinTools) webExtract(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	rawURL := strings.TrimSpace(strArg(args, "url"))
	if rawURL == "" {
		return nil, errors.New("url required")
	}
	maxChars := intArg(args, "max_chars", 8000)
	if maxChars <= 0 {
		maxChars = 8000
	}
	bs, err := fetchHTTPBytes(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	text := htmlToText(string(bs))
	if len(text) > maxChars {
		text = text[:maxChars]
	}
	return map[string]any{"success": true, "url": rawURL, "content": text, "truncated": len(text) == maxChars}, nil
}

func (b *BuiltinTools) clarify(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	question := strings.TrimSpace(strArg(args, "question"))
	if question == "" {
		return nil, errors.New("question required")
	}
	allowFreeform := boolArg(args, "allow_freeform", true)

	options := make([]map[string]any, 0)
	if raw, ok := args["options"]; ok {
		if arr, ok := raw.([]any); ok {
			for _, item := range arr {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				label, _ := m["label"].(string)
				label = strings.TrimSpace(label)
				if label == "" {
					continue
				}
				desc, _ := m["description"].(string)
				options = append(options, map[string]any{
					"label":       label,
					"description": strings.TrimSpace(desc),
				})
			}
		}
	}
	return map[string]any{
		"success":        true,
		"question":       question,
		"options":        options,
		"allow_freeform": allowFreeform,
		"instruction":    "Ask the user to answer this question. If options are provided, ask them to pick one label.",
	}, nil
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

func (b *BuiltinTools) approval(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if tc.ApprovalStore == nil {
		return nil, errors.New("approval store unavailable")
	}
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	switch action {
	case "", "status":
		approvals := tc.ApprovalStore.ListApprovals(tc.SessionID)
		sessionApproved := false
		var sessionExpiresAt string
		for _, a := range approvals {
			if a["scope"] == "session" {
				sessionApproved = true
				sessionExpiresAt, _ = a["expires_at"].(string)
			}
		}
		return map[string]any{
			"success":    true,
			"session_id": tc.SessionID,
			"approved":   sessionApproved,
			"expires_at": sessionExpiresAt,
			"approvals":  approvals,
		}, nil
	case "grant":
		scope := strings.ToLower(strings.TrimSpace(strArg(args, "scope")))
		if scope == "" {
			scope = "session"
		}
		pattern := strings.ToLower(strings.TrimSpace(strArg(args, "pattern")))
		ttlSeconds := intArg(args, "ttl_seconds", 0)
		if scope == "pattern" {
			if pattern == "" {
				return nil, errors.New("pattern is required when scope=pattern")
			}
			expiresAt := tc.ApprovalStore.GrantPattern(tc.SessionID, pattern, time.Duration(ttlSeconds)*time.Second)
			return map[string]any{
				"success":    true,
				"session_id": tc.SessionID,
				"scope":      "pattern",
				"pattern":    pattern,
				"approved":   true,
				"expires_at": expiresAt.Format(time.RFC3339),
			}, nil
		}
		expiresAt := tc.ApprovalStore.Grant(tc.SessionID, time.Duration(ttlSeconds)*time.Second)
		return map[string]any{
			"success":    true,
			"session_id": tc.SessionID,
			"scope":      "session",
			"approved":   true,
			"expires_at": expiresAt.Format(time.RFC3339),
		}, nil
	case "revoke":
		scope := strings.ToLower(strings.TrimSpace(strArg(args, "scope")))
		pattern := strings.ToLower(strings.TrimSpace(strArg(args, "pattern")))
		if scope == "pattern" && pattern != "" {
			revoked := tc.ApprovalStore.RevokePattern(tc.SessionID, pattern)
			return map[string]any{
				"success":    true,
				"session_id": tc.SessionID,
				"scope":      "pattern",
				"pattern":    pattern,
				"approved":   false,
				"revoked":    revoked,
			}, nil
		}
		revoked := tc.ApprovalStore.Revoke(tc.SessionID)
		return map[string]any{
			"success":    true,
			"session_id": tc.SessionID,
			"approved":   false,
			"revoked":    revoked,
		}, nil
	case "confirm":
		approvalID := strings.TrimSpace(strArg(args, "approval_id"))
		if approvalID == "" {
			return nil, errors.New("approval_id required for confirm")
		}
		approved := boolArg(args, "approve", false)
		pa, ok := b.retrievePending(approvalID)
		if !ok {
			return nil, fmt.Errorf("pending approval not found: %s", approvalID)
		}
		if !approved {
			return map[string]any{
				"success":     true,
				"action":      "confirm",
				"approval_id": approvalID,
				"approved":    false,
				"command":     pa.Command,
			}, nil
		}
		if tc.ApprovalStore != nil {
			ttl := intArg(pa.Args, "approval_ttl_seconds", 0)
			tc.ApprovalStore.Grant(tc.SessionID, time.Duration(ttl)*time.Second)
		}
		if pa.Background {
			s, err := b.proc.StartBackground(ctx, pa.Command, pa.CWD)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"success":     true,
				"action":      "confirm",
				"approval_id": approvalID,
				"approved":    true,
				"command":     pa.Command,
				"output":      "background process started",
				"session_id":  s.ID,
				"output_file": s.OutputFile,
				"status":      "running",
				"exit_code":   0,
			}, nil
		}
		out, code, err := RunForeground(ctx, pa.Command, pa.CWD, pa.Timeout)
		res := map[string]any{
			"success":     err == nil && code == 0,
			"action":      "confirm",
			"approval_id": approvalID,
			"approved":    true,
			"command":     pa.Command,
			"output":      out,
			"exit_code":   code,
			"error":       nil,
		}
		if err != nil {
			res["error"] = err.Error()
		}
		return res, nil
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
			return map[string]any{"success": true, "path": root, "skills": []map[string]any{}, "count": 0}, nil
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
	return map[string]any{"success": true, "path": root, "skills": skills, "count": len(skills)}, nil
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
		"success": true,
		"name":    name,
		"path":    path,
		"content": string(bs),
	}, nil
}

func (b *BuiltinTools) skillSearch(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	query := strings.TrimSpace(strArg(args, "query"))
	repo := strings.TrimSpace(strArg(args, "repo"))
	if repo == "" {
		repo = "anthropics/skills"
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("repo must be owner/name format")
	}
	searchURL := fmt.Sprintf("https://api.github.com/search/code?q=%s+in:file+language:markdown+repo:%s/%s", query, parts[0], parts[1])
	bs, err := fetchHTTPBytes(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("github search: %w", err)
	}
	var result struct {
		Items []struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
		} `json:"items"`
	}
	if err := json.Unmarshal(bs, &result); err != nil {
		return nil, fmt.Errorf("parse search results: %w", err)
	}
	skills := make([]map[string]any, 0)
	for _, item := range result.Items {
		if !strings.HasSuffix(item.Path, "SKILL.md") && !strings.HasSuffix(item.Path, "skill.md") {
			continue
		}
		name := filepath.Base(filepath.Dir(item.Path))
		descURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/%s", item.Repo.FullName, item.Path)
		desc := fetchSkillDescription(ctx, descURL)

		skills = append(skills, map[string]any{
			"name":        name,
			"description": desc,
			"repo":        item.Repo.FullName,
			"path":        item.Path,
		})
		if len(skills) >= 20 {
			break
		}
	}
	return map[string]any{"success": true, "query": query, "repo": repo, "skills": skills, "count": len(skills)}, nil
}

func fetchSkillDescription(ctx context.Context, rawURL string) string {
	bs, err := fetchHTTPBytes(ctx, rawURL)
	if err != nil {
		return "(fetch failed)"
	}
	lines := strings.Split(string(bs), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if line != "" && !strings.HasPrefix(line, "---") {
			if len(line) > 120 {
				line = line[:120] + "..."
			}
			return line
		}
	}
	return "(no description)"
}

func (b *BuiltinTools) skillManage(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
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
	case "sync":
		source := strings.ToLower(strings.TrimSpace(strArg(args, "source")))
		if source == "" {
			return nil, errors.New("source required for sync (github or url)")
		}
		switch source {
		case "url":
			skillURL := strings.TrimSpace(strArg(args, "url"))
			if skillURL == "" {
				return nil, errors.New("url required for source=url sync")
			}
			bs, err := fetchHTTPBytes(ctx, skillURL)
			if err != nil {
				return nil, fmt.Errorf("fetch skill: %w", err)
			}
			if err := os.MkdirAll(skillDir, 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(skillMD, bs, 0o644); err != nil {
				return nil, err
			}
			return map[string]any{"action": "sync", "source": "url", "name": name, "path": skillMD, "success": true}, nil
		case "github":
			repo := strings.TrimSpace(strArg(args, "repo"))
			if repo == "" {
				return nil, errors.New("repo required for source=github sync (owner/name)")
			}
			subPath := strings.TrimSpace(strArg(args, "path"))
			synced, err := syncGitHubSkill(ctx, root, repo, subPath)
			if err != nil {
				return nil, fmt.Errorf("sync github: %w", err)
			}
			return map[string]any{"action": "sync", "source": "github", "repo": repo, "path": subPath, "synced_skills": synced, "success": true}, nil
		default:
			return nil, fmt.Errorf("unsupported sync source: %s (use github or url)", source)
		}
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
	return map[string]any{"type": "object", "properties": map[string]any{
		"command":              map[string]any{"type": "string"},
		"background":           map[string]any{"type": "boolean"},
		"timeout":              map[string]any{"type": "integer"},
		"workdir":              map[string]any{"type": "string"},
		"requires_approval":    map[string]any{"type": "boolean"},
		"approval_ttl_seconds": map[string]any{"type": "integer"},
		"notify_on_complete":   map[string]any{"type": "boolean", "description": "When background=true, emit a stream event when the process finishes (best-effort)."},
	}, "required": []string{"command"}}
}
func processParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"action":          map[string]any{"type": "string", "enum": []string{"list", "status", "poll", "log", "wait", "stop", "kill", "write"}, "description": "Action to perform (default: status)"},
		"session_id":      map[string]any{"type": "string"},
		"include_done":    map[string]any{"type": "boolean"},
		"limit":           map[string]any{"type": "integer"},
		"offset":          map[string]any{"type": "integer", "description": "Byte offset for action=log"},
		"max_chars":       map[string]any{"type": "integer", "description": "Max output chars to return (action=log default 50000, action=poll/wait default 20000, hard cap 200000)."},
		"timeout_seconds": map[string]any{"type": "integer", "description": "For action=wait"},
		"input":           map[string]any{"type": "string", "description": "For action=write"},
	}, "required": []string{}}
}
func processStatusParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"session_id": map[string]any{"type": "string"}}, "required": []string{"session_id"}}
}
func stopProcessParams() map[string]any { return processStatusParams() }
func readFileParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "offset": map[string]any{"type": "integer"}, "limit": map[string]any{"type": "integer"}, "with_line_numbers": map[string]any{"type": "boolean"}, "max_chars": map[string]any{"type": "integer"}, "dedup": map[string]any{"type": "boolean", "description": "Return a lightweight stub when the file is unchanged since the previous read in the same session (default true)"}, "reject_on_truncate": map[string]any{"type": "boolean", "description": "When true (default), return an error if the read would exceed max_chars instead of returning truncated content"}}, "required": []string{"path"}}
}
func writeFileParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}}, "required": []string{"path", "content"}}
}
func patchParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"mode":        map[string]any{"type": "string", "description": "replace (default) or patch (V4A patch format)"},
			"path":        map[string]any{"type": "string"},
			"old_string":  map[string]any{"type": "string"},
			"new_string":  map[string]any{"type": "string"},
			"replace_all": map[string]any{"type": "boolean"},
			"patch":       map[string]any{"type": "string", "description": "V4A patch text when mode=patch"},
		},
	}
}
func haListEntitiesParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func haGetStateParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"entity_id": map[string]any{"type": "string"}}, "required": []string{"entity_id"}}
}
func haListServicesParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func haCallServiceParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"domain": map[string]any{"type": "string"}, "service": map[string]any{"type": "string"}, "entity_id": map[string]any{"type": "string"}, "service_data": map[string]any{"type": "object"}}, "required": []string{"domain", "service"}}
}
func kanbanShowParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func kanbanCreateParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}, "title": map[string]any{"type": "string"}, "fields": map[string]any{"type": "object"}}, "required": []string{"title"}}
}
func kanbanCommentParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}, "message": map[string]any{"type": "string"}}, "required": []string{"id", "message"}}
}
func kanbanIDParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}}
}
func kanbanBlockParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}, "reason": map[string]any{"type": "string"}}, "required": []string{"id"}}
}
func kanbanHeartbeatParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"worker": map[string]any{"type": "string"}}}
}
func kanbanLinkParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"from": map[string]any{"type": "string"}, "to": map[string]any{"type": "string"}, "kind": map[string]any{"type": "string"}}, "required": []string{"from", "to"}}
}
func searchFilesParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "pattern": map[string]any{"type": "string"}, "glob": map[string]any{"type": "string"}}, "required": []string{"pattern"}}
}
func mkdirParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}, "required": []string{"path"}}
}
func listDirParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}}
}
func deleteFileParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "recursive": map[string]any{"type": "boolean"}}, "required": []string{"path"}}
}
func moveFileParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"src": map[string]any{"type": "string"}, "dst": map[string]any{"type": "string"}}, "required": []string{"src", "dst"}}
}
func todoParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"todos": map[string]any{"type": "array"}, "merge": map[string]any{"type": "boolean"}}}
}
func memoryParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"action": map[string]any{"type": "string"}, "target": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}, "old_text": map[string]any{"type": "string"}}, "required": []string{"action", "target"}}
}
func sessionSearchParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}, "exclude_session_id": map[string]any{"type": "string"}, "include_current_session": map[string]any{"type": "boolean"}}, "required": []string{"query"}}
}
func webFetchParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"url": map[string]any{"type": "string"}}, "required": []string{"url"}}
}
func webSearchParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}, "base_url": map[string]any{"type": "string"}}, "required": []string{"query"}}
}
func webExtractParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"url": map[string]any{"type": "string"}, "max_chars": map[string]any{"type": "integer"}}, "required": []string{"url"}}
}
func stubParams() map[string]any {
	// Keep schema footprint minimal while still being callable by name.
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func visionAnalyzeParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":     map[string]any{"type": "string", "description": "Local image file path (within workdir)."},
			"question": map[string]any{"type": "string", "description": "Optional specific question about the image."},
		},
		"required": []string{"path"},
	}
}
func imageGenerateParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt":      map[string]any{"type": "string"},
			"output_path": map[string]any{"type": "string"},
			"size": map[string]any{
				"type":        "string",
				"description": "Optional image size for real backends (e.g. 512x512, 1024x1024).",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Optional image model override for real backends.",
			},
			"caption":     map[string]any{"type": "string", "description": "Optional caption used when deliver=true."},
			"deliver": map[string]any{
				"type":        "boolean",
				"description": "If true and running under a gateway context, deliver the generated image to the current chat (requires adapter media support).",
			},
		},
		"required": []string{"prompt"},
	}
}
func textToSpeechParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text":        map[string]any{"type": "string"},
			"output_path": map[string]any{"type": "string"},
			"format": map[string]any{
				"type":        "string",
				"description": "Audio format hint for real backends (mp3,wav,opus,aac).",
			},
			"deliver": map[string]any{
				"type":        "boolean",
				"description": "If true and running under a gateway context, immediately deliver the generated file to the current chat (requires adapter media support).",
			},
		},
		"required": []string{"text"},
	}
}
func browserNavigateParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"url": map[string]any{"type": "string"}}, "required": []string{"url"}}
}
func browserBackParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func browserSnapshotParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"full":      map[string]any{"type": "boolean", "description": "Compatibility hint; ignored by lightweight browser."},
		"max_chars": map[string]any{"type": "integer", "description": "Maximum snapshot text length (default 120000, max 300000)."},
	}}
}
func browserClickParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"ref":          map[string]any{"type": "string", "description": "Ref ID like '@e5' from browser_snapshot."},
		"text":         map[string]any{"type": "string"},
		"href_contains": map[string]any{"type": "string"},
	}}
}
func browserTypeParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"ref":   map[string]any{"type": "string", "description": "Ref ID like '@e3' for an input field from browser_snapshot."},
		"field": map[string]any{"type": "string"},
		"text":  map[string]any{"type": "string"},
	}}
}
func browserPressParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"key": map[string]any{"type": "string", "description": "Key to press (default: unknown). Enter triggers best-effort GET form submit in lightweight mode."}}}
}
func browserScrollParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"direction": map[string]any{"type": "string", "enum": []string{"up", "down", "left", "right"}, "description": "Scroll direction (up/down/left/right, default down)."},
		"amount":    map[string]any{"type": "integer", "description": "Scroll amount units (default 1, max 100)."},
	}}
}
func browserGetImagesParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"limit": map[string]any{"type": "integer", "description": "Maximum number of images to return (default 200, max 1000)."},
	}}
}
func browserConsoleParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"limit": map[string]any{"type": "integer", "description": "Maximum number of latest console entries to return (default 200)."},
	}}
}
func browserVisionParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"limit": map[string]any{"type": "integer", "description": "Maximum number of images to inspect (default 5, max 20)."}}}
}
func browserCDPParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"include_html": map[string]any{"type": "boolean", "description": "Include page HTML in response (default false)."}}}
}
func browserDialogParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Dialog action (accept|dismiss, default: inspect pending dialog).",
				"enum":        []string{"accept", "dismiss"},
			},
			"prompt_text": map[string]any{
				"type":        "string",
				"description": "Optional text for prompt dialogs (action=accept).",
			},
		},
	}
}
func mixtureOfAgentsParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"user_prompt":       map[string]any{"type": "string"},
			"reference_agents":  map[string]any{"type": "integer", "description": "Number of reference subagents (default 3, max 6)"},
			"max_iterations":    map[string]any{"type": "integer"},
			"timeout_seconds":   map[string]any{"type": "integer"},
		},
		"required": []string{"user_prompt"},
	}
}
func clarifyParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{"type": "string"},
			"allow_freeform": map[string]any{"type": "boolean", "description": "Whether user may answer outside the provided options"},
			"options": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label":       map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
					},
					"required": []string{"label"},
				},
			},
		},
		"required": []string{"question"},
	}
}
func delegateTaskParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"goal": map[string]any{"type": "string"}, "context": map[string]any{"type": "string"}, "max_iterations": map[string]any{"type": "integer"}, "max_concurrency": map[string]any{"type": "integer"}, "timeout_seconds": map[string]any{"type": "integer"}, "fail_fast": map[string]any{"type": "boolean"}, "tasks": map[string]any{"type": "array"}}}
}
func approvalParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"action": map[string]any{"type": "string", "enum": []string{"status", "grant", "revoke", "confirm"}, "description": "Action to perform (default: status)"}, "scope": map[string]any{"type": "string", "enum": []string{"session", "pattern"}, "description": "Approval scope: session (default) or pattern (category-specific)"}, "pattern": map[string]any{"type": "string", "description": "Dangerous command category when scope=pattern (e.g. recursive_delete, world_writable, root_ownership, remote_pipe_shell, service_lifecycle)"}, "ttl_seconds": map[string]any{"type": "integer"}, "approval_id": map[string]any{"type": "string", "description": "Pending approval ID for action=confirm"}, "approve": map[string]any{"type": "boolean", "description": "Approve (true) or deny (false) for action=confirm"}}}
}
func skillListParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}}
}
func skillSearchParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Search query for skills"},
			"repo":  map[string]any{"type": "string", "description": "GitHub repo (owner/name), default: anthropics/skills"},
		},
		"required": []string{"query"},
	}
}
func skillViewParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}, "path": map[string]any{"type": "string"}}, "required": []string{"name"}}
}
func skillManageParams() map[string]any {
		return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":       map[string]any{"type": "string", "enum": []string{"create", "edit", "patch", "delete", "write_file", "remove_file", "sync"}},
			"name":         map[string]any{"type": "string"},
			"content":      map[string]any{"type": "string"},
			"old_string":   map[string]any{"type": "string"},
			"new_string":   map[string]any{"type": "string"},
			"replace_all":  map[string]any{"type": "boolean"},
			"file_path":    map[string]any{"type": "string"},
			"file_content": map[string]any{"type": "string"},
			"path":         map[string]any{"type": "string"},
			"source":       map[string]any{"type": "string", "enum": []string{"github", "url"}, "description": "Sync source: github (GitHub repo) or url (direct URL)"},
			"url":          map[string]any{"type": "string", "description": "URL for source=url sync"},
			"repo":         map[string]any{"type": "string", "description": "GitHub repo (owner/name) for source=github sync"},
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

func fetchHTTPBytes(ctx context.Context, url string) ([]byte, error) {
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
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	return body, nil
}

var ddgResultLinkRE = regexp.MustCompile(`(?is)<a[^>]+class="[^"]*result__a[^"]*"[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)

func parseDuckDuckGoHTMLResults(htmlBody string, limit int) []map[string]any {
	matches := ddgResultLinkRE.FindAllStringSubmatch(htmlBody, -1)
	out := make([]map[string]any, 0, limit)
	for _, m := range matches {
		if len(out) >= limit {
			break
		}
		rawHref := html.UnescapeString(strings.TrimSpace(m[1]))
		title := strings.TrimSpace(htmlToText(m[2]))
		u, err := url.Parse(rawHref)
		finalURL := rawHref
		if err == nil {
			// DuckDuckGo wraps outbound links in /l/?uddg=<encoded>
			if uddg := u.Query().Get("uddg"); uddg != "" {
				if decoded, derr := url.QueryUnescape(uddg); derr == nil && strings.HasPrefix(decoded, "http") {
					finalURL = decoded
				}
			}
		}
		out = append(out, map[string]any{
			"title": title,
			"url":   finalURL,
		})
	}
	return out
}

func htmlToText(input string) string {
	// Strip script/style blocks.
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	s := reScript.ReplaceAllString(input, " ")
	s = reStyle.ReplaceAllString(s, " ")

	// Convert some block separators to newlines.
	reBR := regexp.MustCompile(`(?i)<br\s*/?>`)
	reP := regexp.MustCompile(`(?i)</p\s*>`)
	reDiv := regexp.MustCompile(`(?i)</div\s*>`)
	s = reBR.ReplaceAllString(s, "\n")
	s = reP.ReplaceAllString(s, "\n")
	s = reDiv.ReplaceAllString(s, "\n")

	// Remove all tags.
	reTags := regexp.MustCompile(`(?s)<[^>]+>`)
	s = reTags.ReplaceAllString(s, " ")

	s = html.UnescapeString(s)
	// Collapse whitespace.
	lines := strings.Split(s, "\n")
	outLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(line, " "))
		if line != "" {
			outLines = append(outLines, line)
		}
	}
	return strings.Join(outLines, "\n")
}

func syncGitHubSkill(ctx context.Context, localRoot, repo, subPath string) ([]string, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("repo must be owner/name")
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", parts[0], parts[1], subPath)
	bs, err := fetchHTTPBytes(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	var entries []map[string]any
	if err := json.Unmarshal(bs, &entries); err != nil {
		var single map[string]any
		if err2 := json.Unmarshal(bs, &single); err2 != nil {
			return nil, fmt.Errorf("unexpected github response: %w", err)
		}
		entries = []map[string]any{single}
	}

	var synced []string
	for _, entry := range entries {
		entryName, _ := entry["name"].(string)
		entryType, _ := entry["type"].(string)
		if entryType != "dir" {
			continue
		}
		if strings.TrimSpace(entryName) == "" {
			continue
		}
		skillSubPath := subPath + "/" + entryName
		skillMDURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s/SKILL.md", parts[0], parts[1], skillSubPath)
		skillBS, err := fetchHTTPBytes(ctx, skillMDURL)
		if err != nil {
			continue
		}
		var fileInfo map[string]any
		if err := json.Unmarshal(skillBS, &fileInfo); err != nil {
			continue
		}
		downloadURL, _ := fileInfo["download_url"].(string)
		if downloadURL == "" {
			continue
		}
		content, err := fetchHTTPBytes(ctx, downloadURL)
		if err != nil {
			continue
		}
		skillDir := filepath.Join(localRoot, entryName)
		skillMD := filepath.Join(skillDir, "SKILL.md")
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			continue
		}
		if err := os.WriteFile(skillMD, content, 0o644); err != nil {
			continue
		}
		synced = append(synced, entryName)
	}
	return synced, nil
}
