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
		Tools:       []string{"terminal", "process_status", "stop_process"},
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
	"web": {
		Description: "Fetch URLs over HTTP",
		Tools:       []string{"web_fetch"},
	},
	"delegation": {
		Description: "Spawn subagent(s) for subtasks",
		Tools:       []string{"delegate_task"},
	},
	"approval": {
		Description: "Dangerous command approvals",
		Tools:       []string{"approval"},
	},
	"skills": {
		Description: "List/view/manage skills",
		Tools:       []string{"skill_list", "skill_view", "skill_manage", "skill_search"},
	},
	"cronjob": {
		Description: "Cronjob management",
		Tools:       []string{"cronjob"},
	},
	"messaging": {
		Description: "Cross-platform messaging via gateway adapters",
		Tools:       []string{"send_message"},
	},
	"core": {
		Description: "Default core toolset (terminal + file + memory + web + skills + approvals + delegation + cronjob + messaging)",
		Tools:       []string{},
		Includes:    []string{"terminal", "file", "planning", "memory", "session_search", "web", "skills", "approval", "delegation", "cronjob", "messaging"},
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
