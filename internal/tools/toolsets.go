package tools

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type Toolset struct {
	Description string
	Tools       []string
	Includes    []string
	Excludes    []string
	Conflicts   []string
}

type ToolsetResolveOptions struct {
	Env           map[string]string
	DisabledTools map[string]struct{}
}

type ToolsetDetail struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Tools              []string `json:"tools"`
	Includes           []string `json:"includes,omitempty"`
	Excludes           []string `json:"excludes,omitempty"`
	Conflicts          []string `json:"conflicts,omitempty"`
	Available          bool     `json:"available"`
	UnavailableReasons []string `json:"unavailable_reasons,omitempty"`
}

type ToolResolution struct {
	Requested          []string            `json:"requested"`
	ResolvedTools      []string            `json:"resolved_tools"`
	ToolSources        map[string][]string `json:"tool_sources"`
	UnavailableToolset []ToolsetDetail     `json:"unavailable_toolsets,omitempty"`
	ExcludedByToolset  []string            `json:"excluded_by_toolset,omitempty"`
	Conflicts          []string            `json:"conflicts,omitempty"`
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
		Description: "Vision analysis tools (minimal implementation)",
		Tools:       []string{"vision_analyze"},
	},
	"video": {
		Description: "Video analysis tools (ffprobe if available)",
		Tools:       []string{"video_analyze"},
	},
	"discord_admin": {
		Description: "Discord admin tools (requires DISCORD_BOT_TOKEN)",
		Tools:       []string{"discord_admin"},
	},
	"discord": {
		Description: "Discord tools (requires DISCORD_BOT_TOKEN)",
		Tools:       []string{"discord"},
	},
	"feishu": {
		Description: "Feishu/Lark doc & drive tools (requires FEISHU credentials)",
		Tools: []string{
			"feishu_doc_read",
			"feishu_drive_list_comments",
			"feishu_drive_list_comment_replies",
			"feishu_drive_add_comment",
			"feishu_drive_reply_comment",
		},
	},
	"spotify": {
		Description: "Spotify tools (requires SPOTIFY_ACCESS_TOKEN)",
		Tools: []string{
			"spotify_search", "spotify_devices", "spotify_playback", "spotify_queue",
			"spotify_playlists", "spotify_albums", "spotify_library",
		},
	},
	"rl": {
		Description: "RL training tools (minimal local runner)",
		Tools: []string{
			"rl_list_environments", "rl_select_environment", "rl_get_current_config", "rl_edit_config",
			"rl_start_training", "rl_stop_training", "rl_check_status", "rl_get_results", "rl_list_runs", "rl_test_inference",
		},
	},
	"yuanbao": {
		Description: "Yuanbao tools (requires gateway + YUANBAO creds)",
		Tools: []string{
			"yb_send_dm", "yb_send_sticker", "yb_search_sticker", "yb_query_group_info", "yb_query_group_members",
		},
	},
	"image_gen": {
		Description: "Image generation tools (minimal implementation)",
		Tools:       []string{"image_generate"},
	},
	"browser": {
		Description: "Browser automation tools (lightweight implementation)",
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
		Description: "Text-to-speech tools (minimal implementation)",
		Tools:       []string{"text_to_speech"},
	},
	"delegation": {
		Description: "Spawn subagent(s) for subtasks",
		Tools:       []string{"delegate_task"},
	},
	"moa": {
		Description: "Mixture-of-agents reasoning tool",
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
		Description: "Default core toolset",
		Tools:       []string{},
		Includes:    []string{"web", "terminal", "file", "vision", "image_gen", "browser", "tts", "todo", "memory", "session_search", "clarify", "skills", "approval", "code_execution", "delegation", "cronjob", "messaging", "homeassistant", "kanban"},
	},
	"debugging": {
		Description: "Debug bundle (file + terminal + web)",
		Tools:       []string{},
		Includes:    []string{"file", "terminal", "web"},
	},
	"safe": {
		Description: "Read-only research + media generation",
		Tools:       []string{},
		Includes:    []string{"image_gen", "vision", "web"},
		Excludes:    []string{"terminal", "file", "code_execution", "approval", "delegation", "fs_admin"},
		Conflicts:   []string{"core", "debugging", "hermes-cli", "hermes-api-server", "hermes-acp", "hermes-cron"},
	},
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
	return ListToolsetsWithEnv(nil)
}

func ListToolsetsWithEnv(env map[string]string) []map[string]any {
	names := make([]string, 0, len(Toolsets))
	for name := range Toolsets {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		ts := Toolsets[name]
		d := detailForToolset(name, ts, env)
		out = append(out, map[string]any{
			"name":                d.Name,
			"description":         d.Description,
			"tools":               d.Tools,
			"includes":            d.Includes,
			"excludes":            d.Excludes,
			"conflicts":           d.Conflicts,
			"available":           d.Available,
			"unavailable_reasons": d.UnavailableReasons,
		})
	}
	return out
}

func GetToolset(name string) (map[string]any, bool) {
	return GetToolsetWithEnv(name, nil)
}

func GetToolsetWithEnv(name string, env map[string]string) (map[string]any, bool) {
	name = strings.TrimSpace(name)
	ts, ok := Toolsets[name]
	if !ok {
		return nil, false
	}
	d := detailForToolset(name, ts, env)
	return map[string]any{
		"name":                d.Name,
		"description":         d.Description,
		"tools":               d.Tools,
		"includes":            d.Includes,
		"excludes":            d.Excludes,
		"conflicts":           d.Conflicts,
		"available":           d.Available,
		"unavailable_reasons": d.UnavailableReasons,
	}, true
}

func ResolveToolset(names []string) (map[string]struct{}, error) {
	res, err := ResolveToolsetDetailed(names, ToolsetResolveOptions{})
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]struct{}, len(res.ResolvedTools))
	for _, n := range res.ResolvedTools {
		allowed[n] = struct{}{}
	}
	return allowed, nil
}

func ResolveToolsetDetailed(names []string, opt ToolsetResolveOptions) (ToolResolution, error) {
	env := envOrCurrent(opt.Env)
	allowed := make(map[string]struct{})
	seen := make(map[string]struct{})
	toolSources := make(map[string]map[string]struct{})
	unavailable := make(map[string]ToolsetDetail)
	excludedTools := make(map[string]struct{})
	conflicts := make(map[string]struct{})
	requested := make([]string, 0, len(names))
	requestedSet := map[string]struct{}{}

	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		requested = append(requested, name)
		requestedSet[name] = struct{}{}
	}

	var visit func(name, source string) error
	visit = func(name, source string) error {
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
		d := detailForToolset(name, ts, env)
		if !d.Available {
			unavailable[name] = d
			return nil
		}
		for _, cf := range ts.Conflicts {
			cf = strings.TrimSpace(cf)
			if cf != "" {
				if _, req := requestedSet[cf]; req {
					conflicts[name+" conflicts with "+cf] = struct{}{}
				}
			}
		}
		for _, tool := range ts.Tools {
			tool = strings.TrimSpace(tool)
			if tool == "" {
				continue
			}
			allowed[tool] = struct{}{}
			if _, ok := toolSources[tool]; !ok {
				toolSources[tool] = map[string]struct{}{}
			}
			toolSources[tool][name] = struct{}{}
			if strings.TrimSpace(source) != "" {
				toolSources[tool][source] = struct{}{}
			}
		}
		for _, inc := range ts.Includes {
			if err := visit(inc, name); err != nil {
				return err
			}
		}
		for _, exc := range ts.Excludes {
			exc = strings.TrimSpace(exc)
			if exc == "" {
				continue
			}
			if exSet, ok := Toolsets[exc]; ok {
				for _, t := range exSet.Tools {
					if tt := strings.TrimSpace(t); tt != "" {
						excludedTools[tt] = struct{}{}
					}
				}
			}
		}
		return nil
	}

	for _, name := range requested {
		if err := visit(name, ""); err != nil {
			return ToolResolution{}, err
		}
	}

	for t := range excludedTools {
		delete(allowed, t)
	}
	for t := range opt.DisabledTools {
		delete(allowed, strings.TrimSpace(t))
	}

	resolvedTools := mapKeys(allowed)
	sort.Strings(resolvedTools)

	toolSourcesOut := make(map[string][]string, len(toolSources))
	for tool, srcSet := range toolSources {
		if _, ok := allowed[tool]; !ok {
			continue
		}
		src := mapKeys(srcSet)
		sort.Strings(src)
		toolSourcesOut[tool] = src
	}

	unavailOut := make([]ToolsetDetail, 0, len(unavailable))
	for _, d := range unavailable {
		unavailOut = append(unavailOut, d)
	}
	sort.Slice(unavailOut, func(i, j int) bool { return unavailOut[i].Name < unavailOut[j].Name })

	excludedOut := mapKeys(excludedTools)
	sort.Strings(excludedOut)
	conflictOut := mapKeys(conflicts)
	sort.Strings(conflictOut)

	return ToolResolution{
		Requested:          requested,
		ResolvedTools:      resolvedTools,
		ToolSources:        toolSourcesOut,
		UnavailableToolset: unavailOut,
		ExcludedByToolset:  excludedOut,
		Conflicts:          conflictOut,
	}, nil
}

func detailForToolset(name string, ts Toolset, env map[string]string) ToolsetDetail {
	reasons := toolsetUnavailableReasons(name, envOrCurrent(env))
	return ToolsetDetail{
		Name:               name,
		Description:        ts.Description,
		Tools:              append([]string{}, ts.Tools...),
		Includes:           append([]string{}, ts.Includes...),
		Excludes:           append([]string{}, ts.Excludes...),
		Conflicts:          append([]string{}, ts.Conflicts...),
		Available:          len(reasons) == 0,
		UnavailableReasons: reasons,
	}
}

func toolsetUnavailableReasons(name string, env map[string]string) []string {
	reasons := make([]string, 0, 2)
	hasAny := func(keys ...string) bool {
		for _, k := range keys {
			if strings.TrimSpace(env[k]) != "" {
				return true
			}
		}
		return false
	}
	hasAll := func(keys ...string) bool {
		for _, k := range keys {
			if strings.TrimSpace(env[k]) == "" {
				return false
			}
		}
		return true
	}
	switch name {
	case "video":
		if strings.TrimSpace(env["FFPROBE_PATH"]) == "" {
			if _, err := exec.LookPath("ffprobe"); err != nil {
				reasons = append(reasons, "ffprobe not found in PATH")
			}
		}
	case "discord", "discord_admin":
		if !hasAny("AGENT_DISCORD_BOT_TOKEN", "DISCORD_BOT_TOKEN") {
			reasons = append(reasons, "missing AGENT_DISCORD_BOT_TOKEN or DISCORD_BOT_TOKEN")
		}
	case "feishu":
		if !(hasAll("FEISHU_APP_ID", "FEISHU_APP_SECRET") || hasAll("AGENT_FEISHU_APP_ID", "AGENT_FEISHU_APP_SECRET") || hasAny("AGENT_FEISHU_WEBHOOK_URL")) {
			reasons = append(reasons, "missing FEISHU app credentials or AGENT_FEISHU_WEBHOOK_URL")
		}
	case "spotify":
		if !hasAny("SPOTIFY_ACCESS_TOKEN") {
			reasons = append(reasons, "missing SPOTIFY_ACCESS_TOKEN")
		}
	case "yuanbao", "hermes-yuanbao":
		if !(hasAny("YUANBAO_TOKEN") || hasAll("YUANBAO_APP_ID", "YUANBAO_APP_SECRET")) {
			reasons = append(reasons, "missing YUANBAO_TOKEN or YUANBAO_APP_ID/YUANBAO_APP_SECRET")
		}
	case "homeassistant":
		if !(hasAny("AGENT_HOMEASSISTANT_URL", "HASS_URL", "HOMEASSISTANT_URL") && hasAny("AGENT_HOMEASSISTANT_TOKEN", "HASS_TOKEN", "HOMEASSISTANT_TOKEN")) {
			reasons = append(reasons, "missing Home Assistant URL/token")
		}
	case "browser-cdp":
		if !hasAny("BROWSER_CDP_URL") {
			reasons = append(reasons, "missing BROWSER_CDP_URL")
		}
	}
	return reasons
}

func envOrCurrent(env map[string]string) map[string]string {
	if env != nil {
		return env
	}
	out := map[string]string{}
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}

func mapKeys[V any](in map[string]V) []string {
	out := make([]string, 0, len(in))
	for k := range in {
		if strings.TrimSpace(k) == "" {
			continue
		}
		out = append(out, k)
	}
	return out
}

func UnavailableToolsByEnv(env map[string]string) map[string][]string {
	out := map[string][]string{}
	for name, ts := range Toolsets {
		reasons := toolsetUnavailableReasons(name, envOrCurrent(env))
		if len(reasons) == 0 || len(ts.Tools) == 0 {
			continue
		}
		for _, tool := range ts.Tools {
			tool = strings.TrimSpace(tool)
			if tool == "" {
				continue
			}
			out[tool] = append([]string{}, reasons...)
		}
	}
	return out
}
