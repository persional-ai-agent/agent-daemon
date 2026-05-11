package tools

import (
	"fmt"
	"sort"
	"strings"
)

type Toolset struct {
	Description string
	Tools       []string
	Includes    []string
}

// Minimal Hermes-aligned toolsets for agent-daemon.
// These primarily exist to (optionally) shrink the exposed tool schema surface.
var Toolsets = map[string]Toolset{
	"terminal": {
		Description: "Terminal/process execution tools",
		Tools:       []string{"terminal", "process"},
	},
	// Hermes parity alias
	"todo": {
		Description: "Todo/planning tools",
		Tools:       []string{"todo"},
	},
	"file": {
		Description: "File read/write/search tools",
		Tools:       []string{"read_file", "write_file", "patch", "search_files"},
	},
	"fs_admin": {
		Description: "Filesystem admin tools (mkdir/list_dir/delete_file/move_file)",
		Tools:       []string{"mkdir", "list_dir", "delete_file", "move_file"},
	},
	"planning": {
		Description: "Planning tools (todo)",
		Tools:       []string{"todo"},
	},
	"memory": {
		Description: "Persistent memory tools",
		Tools:       []string{"memory"},
	},
	"session_search": {
		Description: "Search previous session messages",
		Tools:       []string{"session_search"},
	},
	"clarify": {
		Description: "Ask the user clarifying questions",
		Tools:       []string{"clarify"},
	},
	"web": {
		Description: "Web research tools (search + extract)",
		Tools:       []string{"web_search", "web_extract"},
	},
	// Hermes parity: web_search only
	"search": {
		Description: "Web search only",
		Tools:       []string{"web_search"},
	},
	"vision": {
		Description: "Vision analysis tools (stub)",
		Tools:       []string{"vision_analyze"},
	},
	"video": {
		Description: "Video analysis tools (ffprobe if available)",
		Tools:       []string{"video_analyze"},
	},
	"discord_admin": {
		Description: "Discord admin tools (requires DISCORD_BOT_TOKEN; placeholder)",
		Tools:       []string{"discord_admin"},
	},
	"discord": {
		Description: "Discord tools (requires DISCORD_BOT_TOKEN)",
		Tools:       []string{"discord"},
	},
	"feishu": {
		Description: "Feishu/Lark doc & drive tools (requires FEISHU_APP_ID/FEISHU_APP_SECRET; placeholder)",
		Tools: []string{
			"feishu_doc_read",
			"feishu_drive_list_comments",
			"feishu_drive_list_comment_replies",
			"feishu_drive_add_comment",
			"feishu_drive_reply_comment",
		},
	},
	"spotify": {
		Description: "Spotify tools (requires SPOTIFY_ACCESS_TOKEN; placeholder)",
		Tools: []string{
			"spotify_search", "spotify_devices", "spotify_playback", "spotify_queue",
			"spotify_playlists", "spotify_albums", "spotify_library",
		},
	},
	"rl": {
		Description: "RL training tools (placeholder)",
		Tools: []string{
			"rl_list_environments", "rl_select_environment", "rl_get_current_config", "rl_edit_config",
			"rl_start_training", "rl_stop_training", "rl_check_status", "rl_get_results", "rl_list_runs", "rl_test_inference",
		},
	},
	"yuanbao": {
		Description: "Yuanbao tools (requires gateway + YUANBAO_TOKEN or YUANBAO_APP_ID/YUANBAO_APP_SECRET)",
		Tools: []string{
			"yb_send_dm", "yb_send_sticker", "yb_search_sticker", "yb_query_group_info", "yb_query_group_members",
		},
	},
	"image_gen": {
		Description: "Image generation tools (stub)",
		Tools:       []string{"image_generate"},
	},
	"browser": {
		Description: "Browser automation tools (stub)",
		Tools: []string{
			"browser_navigate", "browser_snapshot", "browser_click",
			"browser_type", "browser_scroll", "browser_back",
			"browser_press", "browser_get_images",
			"browser_vision", "browser_console", "browser_cdp", "browser_dialog",
			"web_search",
		},
	},
	"browser-cdp": {
		Description: "Browser CDP tools (lightweight shim)",
		Tools:       []string{"browser_cdp", "browser_dialog"},
	},
	"tts": {
		Description: "Text-to-speech tools (stub)",
		Tools:       []string{"text_to_speech"},
	},
	"delegation": {
		Description: "Spawn subagent(s) for subtasks",
		Tools:       []string{"delegate_task"},
	},
	"moa": {
		Description: "Mixture-of-agents reasoning tool (stub)",
		Tools:       []string{"mixture_of_agents"},
	},
	"code_execution": {
		Description: "Execute short scripts (python)",
		Tools:       []string{"execute_code"},
	},
	"approval": {
		Description: "Dangerous command approvals",
		Tools:       []string{"approval"},
	},
	"skills": {
		Description: "List/view/manage skills",
		Tools:       []string{"skill_list", "skills_list", "skill_view", "skill_manage", "skill_search"},
	},
	"cronjob": {
		Description: "Cronjob management",
		Tools:       []string{"cronjob"},
	},
	"messaging": {
		Description: "Cross-platform messaging via gateway adapters",
		Tools:       []string{"send_message"},
	},
	"homeassistant": {
		Description: "Home Assistant control (requires HASS_URL/HASS_TOKEN)",
		Tools:       []string{"ha_list_entities", "ha_get_state", "ha_list_services", "ha_call_service"},
	},
	"kanban": {
		Description: "Local kanban coordination tools",
		Tools: []string{
			"kanban_show", "kanban_complete", "kanban_block", "kanban_heartbeat",
			"kanban_comment", "kanban_create", "kanban_link",
		},
	},
	"core": {
		Description: "Default core toolset (terminal + file + memory + web + skills + approvals + delegation + cronjob + messaging)",
		Tools:       []string{},
		Includes:    []string{"web", "terminal", "file", "vision", "image_gen", "browser", "tts", "todo", "memory", "session_search", "clarify", "skills", "approval", "code_execution", "delegation", "cronjob", "messaging", "homeassistant", "kanban"},
	},
	"debugging": {
		Description: "Debug bundle (file + terminal + web)",
		Tools:       []string{},
		Includes:    []string{"file", "terminal", "web"},
	},
	"safe": {
		Description: "Read-only research + media generation (no file write, no terminal)",
		Tools:       []string{},
		Includes:    []string{"image_gen", "vision", "web"},
	},
	// Hermes platform toolset names (composites) — kept for config/toolset-name parity.
	"hermes-cli": {
		Description: "Hermes CLI platform toolset (alias of core)",
		Tools:       []string{},
		Includes:    []string{"core"},
	},
	"hermes-api-server": {
		Description: "Hermes API server platform toolset (core minus interactive tools; approximated)",
		Tools:       []string{},
		Includes:    []string{"web", "terminal", "file", "vision", "image_gen", "browser", "skills", "approval", "code_execution", "delegation", "cronjob", "homeassistant", "kanban", "session_search"},
	},
	"hermes-acp": {
		Description: "Hermes ACP platform toolset (coding-focused; approximated)",
		Tools:       []string{},
		Includes:    []string{"terminal", "file", "skills", "approval", "code_execution", "delegation", "web", "session_search", "memory"},
	},
	"hermes-cron": {
		Description: "Hermes cron platform toolset (alias of core)",
		Tools:       []string{},
		Includes:    []string{"core"},
	},
	"hermes-telegram": {
		Description: "Hermes Telegram platform toolset (alias of core)",
		Tools:       []string{},
		Includes:    []string{"core"},
	},
	"hermes-discord": {
		Description: "Hermes Discord platform toolset (core + discord + discord_admin)",
		Tools:       []string{},
		Includes:    []string{"core", "discord", "discord_admin"},
	},
	"hermes-slack": {
		Description: "Hermes Slack platform toolset (alias of core)",
		Tools:       []string{},
		Includes:    []string{"core"},
	},
	"hermes-yuanbao": {
		Description: "Hermes Yuanbao platform toolset (core + yuanbao tools)",
		Tools:       []string{},
		Includes:    []string{"core", "yuanbao"},
	},
}

func ListToolsets() []map[string]any {
	names := make([]string, 0, len(Toolsets))
	for name := range Toolsets {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		ts := Toolsets[name]
		out = append(out, map[string]any{
			"name":        name,
			"description": ts.Description,
			"tools":       append([]string{}, ts.Tools...),
			"includes":    append([]string{}, ts.Includes...),
		})
	}
	return out
}

func GetToolset(name string) (map[string]any, bool) {
	name = strings.TrimSpace(name)
	ts, ok := Toolsets[name]
	if !ok {
		return nil, false
	}
	return map[string]any{
		"name":        name,
		"description": ts.Description,
		"tools":       append([]string{}, ts.Tools...),
		"includes":    append([]string{}, ts.Includes...),
	}, true
}

func ResolveToolset(names []string) (map[string]struct{}, error) {
	allowed := make(map[string]struct{})
	seen := make(map[string]struct{})
	var visit func(name string) error
	visit = func(name string) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		if _, ok := seen[name]; ok {
			return nil
		}
		seen[name] = struct{}{}
		ts, ok := Toolsets[name]
		if !ok {
			return fmt.Errorf("unknown toolset: %s", name)
		}
		for _, tool := range ts.Tools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				allowed[tool] = struct{}{}
			}
		}
		for _, inc := range ts.Includes {
			if err := visit(inc); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return allowed, nil
}
