package main

import "strings"

func rootSlashCommands() []string {
	return []string{
		"/help", "/session", "/api", "/http", "/tools", "/tool", "/sessions", "/pick", "/show", "/next", "/prev",
		"/stats", "/gateway", "/config", "/pretty", "/view", "/last", "/save", "/status", "/health", "/cancel",
		"/history", "/timeline", "/rerun", "/events", "/bookmark", "/workbench", "/workflow", "/pending",
		"/approve", "/deny", "/reload-config", "/doctor", "/actions", "/panel", "/open", "/refresh", "/version",
		"/reconnect", "/recover", "/diag", "/fullscreen", "/quit", "/exit", "/new", "/reset",
	}
}

func commandSecondTokens(cmd string) []string {
	switch strings.ToLower(strings.TrimSpace(cmd)) {
	case "/gateway":
		return []string{"status", "enable", "disable"}
	case "/config":
		return []string{"get", "set", "tui"}
	case "/pretty":
		return []string{"on", "off"}
	case "/view":
		return []string{"human", "json"}
	case "/bookmark":
		return []string{"add", "list", "use"}
	case "/workbench":
		return []string{"save", "list", "load", "delete"}
	case "/workflow":
		return []string{"save", "list", "run", "delete"}
	case "/panel":
		return []string{"list", "next", "prev", "status", "auto", "interval", "overview", "dashboard", "sessions", "tools", "approvals", "gateway", "diag"}
	case "/reconnect":
		return []string{"status", "on", "off", "now", "timeout"}
	case "/fullscreen":
		return []string{"on", "off"}
	case "/recover":
		return []string{"context"}
	}
	return nil
}

func commandThirdTokens(cmd, second string) []string {
	switch strings.ToLower(strings.TrimSpace(cmd)) {
	case "/reconnect":
		if strings.EqualFold(strings.TrimSpace(second), "timeout") {
			return []string{"wait", "reconnect", "cancel"}
		}
	case "/panel":
		if strings.EqualFold(strings.TrimSpace(second), "auto") {
			return []string{"on", "off"}
		}
	}
	return nil
}

func slashCompletions(input string) []string {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil
	}
	if !strings.HasPrefix(raw, "/") {
		return nil
	}
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return nil
	}
	trailingSpace := strings.HasSuffix(input, " ")
	if len(parts) == 1 && !trailingSpace {
		return completeRoot(parts[0])
	}
	cmd := strings.ToLower(strings.TrimSpace(parts[0]))
	if len(parts) == 1 && trailingSpace {
		return completeSub(raw, cmd, "", commandSecondTokens(cmd))
	}
	if len(parts) == 2 {
		prefix := ""
		if !trailingSpace {
			prefix = parts[1]
		}
		return completeSub(raw, cmd, prefix, commandSecondTokens(cmd))
	}
	if len(parts) == 3 {
		prefix := ""
		if !trailingSpace {
			prefix = parts[2]
		}
		return completeSub(raw, cmd, prefix, commandThirdTokens(cmd, parts[1]))
	}
	return nil
}

func completeRoot(prefix string) []string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	roots := rootSlashCommands()
	out := make([]string, 0, len(roots))
	for _, cmd := range roots {
		if strings.HasPrefix(cmd, prefix) {
			out = append(out, cmd)
		}
	}
	return out
}

func completeSub(raw, cmd, prefix string, options []string) []string {
	if len(options) == 0 {
		return nil
	}
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	baseParts := strings.Fields(strings.TrimSpace(raw))
	if len(baseParts) == 0 {
		return nil
	}
	head := cmd
	if len(baseParts) >= 2 {
		head = strings.Join(baseParts[:len(baseParts)-1], " ")
		if strings.HasSuffix(raw, " ") {
			head = strings.TrimSpace(raw)
		}
	} else {
		head = strings.TrimSpace(raw)
	}
	out := make([]string, 0, len(options))
	for _, opt := range options {
		if prefix == "" || strings.HasPrefix(strings.ToLower(opt), prefix) {
			if len(baseParts) == 1 || strings.HasSuffix(raw, " ") {
				out = append(out, strings.TrimSpace(head+" "+opt))
			} else {
				out = append(out, strings.TrimSpace(strings.Join(append(baseParts[:len(baseParts)-1], opt), " ")))
			}
		}
	}
	return out
}
