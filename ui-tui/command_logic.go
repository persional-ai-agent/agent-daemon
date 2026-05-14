package main

import (
	"fmt"
	"strings"
)

func handleTUICommand(s *appState, text string, onEvent func(map[string]any)) (lines []string, err error, quit bool) {
	switch {
	case text == "/quit" || text == "/exit":
		return nil, nil, true
	case text == "/help":
		return []string{
			"命令：/help /quit /clear /status /view human|json",
			"其他输入会作为消息发送给 Agent。",
		}, nil, false
	case text == "/clear":
		return nil, nil, false
	case text == "/status":
		return []string{
			fmt.Sprintf("status=%s code=%s detail=%s", s.lastStatus, s.lastCode, s.lastDetail),
			fmt.Sprintf("reconnect=%v state=%s count=%d", s.reconnectEnabled, s.reconnectState, s.reconnectCount),
		}, nil, false
	case strings.HasPrefix(text, "/view "):
		mode := strings.TrimSpace(strings.TrimPrefix(text, "/view "))
		if mode != "human" && mode != "json" {
			return nil, fmt.Errorf("usage: /view human|json"), false
		}
		s.viewMode = mode
		return []string{"view mode: " + mode}, nil, false
	default:
		s.addChatLine("user: " + text)
		out := make([]string, 0, 16)
		if onEvent == nil {
			onEvent = func(map[string]any) {}
		}
		wrapped := func(evt map[string]any) {
			onEvent(evt)
			line := printEvent(evt, false)
			if strings.TrimSpace(line) != "" {
				out = append(out, line)
			}
		}
		runErr := s.sendTurn(text, wrapped)
		return out, runErr, false
	}
}
