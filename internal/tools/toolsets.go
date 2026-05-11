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
	"file": {
		Description: "File read/write/search tools",
		Tools:       []string{"read_file", "write_file", "patch", "search_files"},
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
	"vision": {
		Description: "Vision analysis tools (stub)",
		Tools:       []string{"vision_analyze"},
	},
	"video": {
		Description: "Video analysis tools (ffprobe if available)",
		Tools:       []string{"video_analyze"},
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
		},
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
		Includes:    []string{"web", "terminal", "file", "vision", "image_gen", "browser", "tts", "planning", "memory", "session_search", "clarify", "skills", "approval", "code_execution", "delegation", "cronjob", "messaging", "homeassistant", "kanban"},
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
