package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func printHelp() {
	fmt.Println("commands:")
	fmt.Println("/help                 show help")
	fmt.Println("/session              show current session id")
	fmt.Println("/session <id>         switch session id")
	fmt.Println("/api                  show websocket endpoint")
	fmt.Println("/api <ws-url>         switch websocket endpoint")
	fmt.Println("/http                 show http api base")
	fmt.Println("/http <http-url>      switch http api base")
	fmt.Println("/tools                list tools")
	fmt.Println("/tool <name>          show tool schema")
	fmt.Println("/sessions [n]         list recent sessions")
	fmt.Println("/pick <index>         switch session from last /sessions result")
	fmt.Println("/show [sid] [o] [l]   show session messages")
	fmt.Println("/next                 show next page (based on last /show)")
	fmt.Println("/prev                 show previous page (based on last /show)")
	fmt.Println("/stats [sid]          show session stats")
	fmt.Println("/gateway status       show gateway status")
	fmt.Println("/gateway enable       enable gateway")
	fmt.Println("/gateway disable      disable gateway")
	fmt.Println("/config get           show config snapshot")
	fmt.Println("/config set k v       set config key/value")
	fmt.Println("/config tui           show effective [ui-tui] config and source")
	fmt.Println("/pretty on|off        enable/disable pretty json")
	fmt.Println("/view human|json      switch display mode")
	fmt.Println("/last                 print last json payload")
	fmt.Println("/save <file>          save last json payload")
	fmt.Println("/status               show last command status")
	fmt.Println("/health               check backend health endpoint")
	fmt.Println("/cancel               cancel current session run")
	fmt.Println("/history [n]          show local command history")
	fmt.Println("/timeline [n]         show recent conversation timeline")
	fmt.Println("/rerun <index>        rerun command from history")
	fmt.Println("/events [n]           show recent runtime events")
	fmt.Println("/events save <file> [json|ndjson] [since=<RFC3339>] [until=<RFC3339>]")
	fmt.Println("/bookmark add <name>  save current session/api profile")
	fmt.Println("/bookmark list        list bookmarks")
	fmt.Println("/bookmark use <name>  restore session/api profile")
	fmt.Println("/workbench save <name> save current workbench profile")
	fmt.Println("/workbench list       list workbench profiles")
	fmt.Println("/workbench load <name> load workbench profile")
	fmt.Println("/workbench delete <name> delete workbench profile")
	fmt.Println("/workflow save <name> <cmd1;cmd2;...>")
	fmt.Println("/workflow list")
	fmt.Println("/workflow run <name> [dry]")
	fmt.Println("/workflow delete <name>")
	fmt.Println("/pending [n]          show latest pending approval(s) in session")
	fmt.Println("/approve [id]         approve pending approval id (default latest)")
	fmt.Println("/deny [id]            deny pending approval id (default latest)")
	fmt.Println("/reload-config        reload [ui-tui] config from config.ini")
	fmt.Println("/doctor               run backend capability checks")
	fmt.Println("/actions              open quick action palette")
	fmt.Println("/panel [name]         switch fullscreen panel (overview/dashboard/sessions/tools/approvals/gateway/diag)")
	fmt.Println("/panel list           list available fullscreen panels")
	fmt.Println("/panel next|prev      cycle fullscreen panels")
	fmt.Println("/panel status         show panel runtime status")
	fmt.Println("/panel auto on|off    toggle panel auto refresh")
	fmt.Println("/panel interval <sec> set panel auto refresh interval")
	fmt.Println("/open <index>         open item from current panel (sessions/tools/approvals)")
	fmt.Println("/refresh              refresh current fullscreen panel data")
	fmt.Println("/version              show ui-tui build metadata")
	fmt.Println("/reconnect status     show reconnect status")
	fmt.Println("/reconnect on|off     enable/disable auto reconnect")
	fmt.Println("/reconnect now        probe websocket endpoint immediately")
	fmt.Println("/reconnect timeout wait|reconnect|cancel")
	fmt.Println("/diag                 show realtime diagnostics")
	fmt.Println("/diag export <file>   export diagnostics bundle")
	fmt.Println("/fullscreen           show fullscreen mode status")
	fmt.Println("/fullscreen on|off    toggle fullscreen dashboard mode")
	fmt.Println("/quit                 exit")
	fmt.Println("aliases: :q, quit, ls, show, gw, cfg, h")
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

func (s *appState) renderFullscreenFrame() {
	if !s.fullscreen {
		return
	}
	clear := "\033[2J\033[H"
	fmt.Print(clear)
	fmt.Println("ui-tui fullscreen")
	fmt.Printf("session: %s\n", s.session)
	fmt.Printf("ws: %s\n", s.wsBase)
	fmt.Printf("http: %s\n", s.httpBase)
	fmt.Printf("status: %s/%s  detail: %s\n", s.lastStatus, s.lastCode, s.lastDetail)
	fmt.Printf("transport: %s  reconnect: %s(%d)  view: %s\n", s.activeTransport, s.reconnectState, s.reconnectCount, s.viewMode)
	fmt.Printf("panel: %s  auto=%v/%ds  hint: /panel next|prev|<name> /refresh /open <index> /actions /quit\n", s.fullscreenPanel, s.panelAutoRefresh, s.panelRefreshSec)
	if s.fullscreenPanel != "overview" {
		fmt.Println("---- panel data ----")
		if payload, ok := s.panelData[s.fullscreenPanel]; ok {
			s.printData(payload)
		} else {
			fmt.Println("(empty, run /refresh)")
		}
		fmt.Println("---- input ----")
		return
	}
	fmt.Println("---- recent events ----")
	start := len(s.eventLog) - 6
	if start < 0 {
		start = 0
	}
	for _, evt := range s.eventLog[start:] {
		typ, _ := evt["type"].(string)
		turn, _ := evt["turn"].(float64)
		if turn > 0 {
			fmt.Printf("- %s turn=%.0f\n", typ, turn)
		} else {
			fmt.Printf("- %s\n", typ)
		}
	}
	fmt.Println("---- timeline ----")
	lineStart := len(s.chatLog) - 10
	if lineStart < 0 {
		lineStart = 0
	}
	for _, ln := range s.chatLog[lineStart:] {
		fmt.Printf("%s\n", ln)
	}
	fmt.Println("---- input ----")
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
