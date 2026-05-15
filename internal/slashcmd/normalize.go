package slashcmd

import "strings"

// NormalizeInput normalizes chat/tui command aliases and command root casing.
func NormalizeInput(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "/") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) == 1 {
			text = strings.ToLower(parts[0])
		} else {
			text = strings.ToLower(parts[0]) + " " + parts[1]
		}
	}
	lower := strings.ToLower(text)
	switch lower {
	case ":q", "quit":
		return "/quit"
	case "/q":
		return "/quit"
	case "ls":
		return "/tools"
	case "sessions":
		return "/sessions"
	case "show":
		return "/show"
	case "next":
		return "/next"
	case "prev":
		return "/prev"
	case "stats":
		return "/stats"
	case "gateway", "gw":
		return "/gateway status"
	case "config", "cfg":
		return "/config get"
	case "h":
		return "/help"
	case "/h":
		return "/help"
	case "stop", "abort", "/stop", "/abort":
		return "/cancel"
	}
	if strings.HasPrefix(lower, "/gw ") && !strings.HasPrefix(lower, "/gateway ") {
		return "/gateway " + strings.TrimSpace(text[len("/gw "):])
	}
	if strings.HasPrefix(lower, "/cfg ") && !strings.HasPrefix(lower, "/config ") {
		return "/config " + strings.TrimSpace(text[len("/cfg "):])
	}
	if strings.HasPrefix(lower, "/sess ") && !strings.HasPrefix(lower, "/sessions ") {
		return "/sessions " + strings.TrimSpace(text[len("/sess "):])
	}
	if strings.HasPrefix(lower, "/wb ") && !strings.HasPrefix(lower, "/workbench ") {
		return "/workbench " + strings.TrimSpace(text[len("/wb "):])
	}
	if strings.HasPrefix(lower, "/wf ") && !strings.HasPrefix(lower, "/workflow ") {
		return "/workflow " + strings.TrimSpace(text[len("/wf "):])
	}
	if strings.HasPrefix(lower, "/bm ") && !strings.HasPrefix(lower, "/bookmark ") {
		return "/bookmark " + strings.TrimSpace(text[len("/bm "):])
	}
	if strings.HasPrefix(lower, "show ") && !strings.HasPrefix(lower, "/show ") {
		return "/show " + strings.TrimSpace(text[len("show "):])
	}
	if strings.HasPrefix(lower, "sessions ") && !strings.HasPrefix(lower, "/sessions ") {
		return "/sessions " + strings.TrimSpace(text[len("sessions "):])
	}
	if strings.HasPrefix(lower, "tool ") && !strings.HasPrefix(lower, "/tool ") {
		return "/tool " + strings.TrimSpace(text[len("tool "):])
	}
	if strings.HasPrefix(lower, "gw ") && !strings.HasPrefix(lower, "/gateway ") {
		return "/gateway " + strings.TrimSpace(text[len("gw "):])
	}
	if strings.HasPrefix(lower, "cfg ") && !strings.HasPrefix(lower, "/config ") {
		return "/config " + strings.TrimSpace(text[len("cfg "):])
	}
	return text
}
