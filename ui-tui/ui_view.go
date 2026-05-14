package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func helpLines() []string {
	return []string{
		"commands:",
		"/help                 show help",
		"/session              show current session id",
		"/session <id>         switch session id",
		"/api                  show websocket endpoint",
		"/api <ws-url>         switch websocket endpoint",
		"/http                 show http api base",
		"/http <http-url>      switch http api base",
		"/tools                list tools",
		"/tool <name>          show tool schema",
		"/sessions [n]         list recent sessions",
		"/pick <index>         switch session from last /sessions result",
		"/show [sid] [o] [l]   show session messages",
		"/next                 show next page (based on last /show)",
		"/prev                 show previous page (based on last /show)",
		"/stats [sid]          show session stats",
		"/gateway status       show gateway status",
		"/gateway enable       enable gateway",
		"/gateway disable      disable gateway",
		"/config get           show config snapshot",
		"/config set k v       set config key/value",
		"/config tui           show effective [ui-tui] config and source",
		"/pretty on|off        enable/disable pretty json",
		"/view human|json      switch display mode",
		"/last                 print last json payload",
		"/save <file>          save last json payload",
		"/status               show last command status",
		"/health               check backend health endpoint",
		"/cancel               cancel current session run",
		"/history [n]          show local command history",
		"/timeline [n]         show recent conversation timeline",
		"/rerun <index>        rerun command from history",
		"/events [n]           show recent runtime events",
		"/events save <file> [json|ndjson] [since=<RFC3339>] [until=<RFC3339>]",
		"/bookmark add <name>  save current session/api profile",
		"/bookmark list        list bookmarks",
		"/bookmark use <name>  restore session/api profile",
		"/workbench save <name> save current workbench profile",
		"/workbench list       list workbench profiles",
		"/workbench load <name> load workbench profile",
		"/workbench delete <name> delete workbench profile",
		"/workflow save <name> <cmd1;cmd2;...>",
		"/workflow list",
		"/workflow run <name> [dry]",
		"/workflow delete <name>",
		"/pending [n]          show latest pending approval(s) in session",
		"/approve [id]         approve pending approval id (default latest)",
		"/deny [id]            deny pending approval id (default latest)",
		"/reload-config        reload [ui-tui] config from config.ini",
		"/doctor               run backend capability checks",
		"/actions              open quick action palette",
		"/panel [name]         switch fullscreen panel (overview/dashboard/sessions/tools/approvals/gateway/diag)",
		"/panel list           list available fullscreen panels",
		"/panel next|prev      cycle fullscreen panels",
		"/panel status         show panel runtime status",
		"/panel auto on|off    toggle panel auto refresh",
		"/panel interval <sec> set panel auto refresh interval",
		"/open <index>         open item from current panel (sessions/tools/approvals)",
		"/refresh              refresh current fullscreen panel data",
		"/version              show ui-tui build metadata",
		"/reconnect status     show reconnect status",
		"/reconnect on|off     enable/disable auto reconnect",
		"/reconnect now        probe websocket endpoint immediately",
		"/reconnect timeout wait|reconnect|cancel",
		"/diag                 show realtime diagnostics",
		"/diag export <file>   export diagnostics bundle",
		"/fullscreen           show fullscreen mode status",
		"/fullscreen on|off    toggle fullscreen dashboard mode",
		"/quit                 exit",
		"aliases: :q, quit, ls, show, gw, cfg, h",
	}
}

func actionMenuItems(s *appState) []string {
	fullscreenAction := "fullscreen on"
	if s != nil && s.fullscreen {
		fullscreenAction = "fullscreen off"
	}
	return []string{
		"/tools",
		"/sessions 20",
		"/show",
		"/gateway status",
		"/config get",
		"/doctor",
		"/diag",
		"/reconnect status",
		"/pending 5",
		"/panel next",
		"/panel status",
		"/workbench list",
		"/workflow list",
		"/" + fullscreenAction,
		"/help",
	}
}

func actionCommandByIndex(s *appState, idx int) (string, bool) {
	items := actionMenuItems(s)
	if idx <= 0 || idx > len(items) {
		return "", false
	}
	return items[idx-1], true
}

func panelNames() []string {
	return []string{"overview", "dashboard", "sessions", "tools", "approvals", "gateway", "diag"}
}

func nextPanel(current string) string {
	names := panelNames()
	idx := 0
	for i, name := range names {
		if name == current {
			idx = i
			break
		}
	}
	return names[(idx+1)%len(names)]
}

func prevPanel(current string) string {
	names := panelNames()
	idx := 0
	for i, name := range names {
		if name == current {
			idx = i
			break
		}
	}
	if idx == 0 {
		return names[len(names)-1]
	}
	return names[idx-1]
}

func printEvent(evt map[string]any, emit bool) string {
	evtType := ""
	if v, ok := evt["type"].(string); ok {
		evtType = v
	}
	if evtType == "" {
		if v, ok := evt["Type"].(string); ok {
			evtType = v
		}
	}
	switch evtType {
	case "assistant_message":
		if emit {
			fmt.Printf("[assistant] %v\n", evt["content"])
		}
		return fmt.Sprintf("assistant: %v", evt["content"])
	case "tool_started", "tool_finished":
		toolName := evt["tool_name"]
		if toolName == nil {
			toolName = evt["ToolName"]
		}
		if emit {
			fmt.Printf("[%s] %v\n", evtType, toolName)
		}
		return fmt.Sprintf("%s: %v", evtType, toolName)
	case "result":
		if emit {
			fmt.Printf("[result] %v\n", evt["final_response"])
		}
		return fmt.Sprintf("result: %v", evt["final_response"])
	case "error":
		if emit {
			fmt.Printf("[error] %v\n", evt["error"])
		}
		return fmt.Sprintf("error: %v", evt["error"])
	default:
		bs, _ := json.Marshal(evt)
		if emit {
			fmt.Printf("[%s] %s\n", evtType, string(bs))
		}
		return fmt.Sprintf("%s: %s", evtType, string(bs))
	}
}

func printJSONMode(v any, pretty bool) {
	var bs []byte
	if pretty {
		bs, _ = json.MarshalIndent(v, "", "  ")
	} else {
		bs, _ = json.Marshal(v)
	}
	fmt.Println(string(bs))
}

func (s *appState) addChatLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if len(line) > 300 {
		line = line[:300] + "..."
	}
	s.chatLog = append(s.chatLog, line)
	if len(s.chatLog) > s.chatMaxLines {
		s.chatLog = append([]string(nil), s.chatLog[len(s.chatLog)-s.chatMaxLines:]...)
	}
}

func (s *appState) timelineSlice(limit int) []string {
	if limit <= 0 {
		limit = 20
	}
	if limit > len(s.chatLog) {
		limit = len(s.chatLog)
	}
	if limit <= 0 {
		return nil
	}
	start := len(s.chatLog) - limit
	return append([]string(nil), s.chatLog[start:]...)
}
