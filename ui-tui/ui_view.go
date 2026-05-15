package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/slashcmd"
)

func helpLines() []string {
	lines := []string{"commands:"}
	for _, entry := range slashcmd.TUIHelpEntries() {
		if strings.TrimSpace(entry.Description) == "" {
			lines = append(lines, entry.Command)
			continue
		}
		lines = append(lines, fmt.Sprintf("%-22s %s", entry.Command, entry.Description))
	}
	lines = append(lines, "aliases: :q, quit, ls, show, gw, cfg, h")
	return lines
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
	case "session", "turn_started", "completed":
		return ""
	case "user_message":
		return ""
	case "assistant_message":
		if text, _ := evt["content"].(string); strings.TrimSpace(text) != "" {
			return fmt.Sprintf("assistant: %s", text)
		}
		return ""
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
	case "model_stream_event":
		return ""
	case "error":
		errText := extractEventErrorText(evt)
		if errText == "" || errText == "<nil>" {
			return ""
		}
		if emit {
			fmt.Printf("[error] %s\n", errText)
		}
		return fmt.Sprintf("error: %s", errText)
	default:
		bs, _ := json.Marshal(evt)
		if emit {
			fmt.Printf("[%s] %s\n", evtType, string(bs))
		}
		return fmt.Sprintf("%s: %s", evtType, string(bs))
	}
}

func extractEventErrorText(evt map[string]any) string {
	if evt == nil {
		return ""
	}
	if text, _ := evt["error"].(string); strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text)
	}
	if text, _ := evt["content"].(string); strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text)
	}
	if data, _ := evt["data"].(map[string]any); data != nil {
		if text, _ := data["error"].(string); strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return strings.TrimSpace(fmt.Sprintf("%v", evt["error"]))
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
