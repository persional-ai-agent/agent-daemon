package tools

import "strings"

func BuildSlashPayload(slash string) map[string]any {
	return map[string]any{
		"slash": strings.TrimSpace(slash),
	}
}

func BuildSlashModePayload(slash, mode string) map[string]any {
	payload := BuildSlashPayload(slash)
	if strings.TrimSpace(mode) != "" {
		payload["mode"] = strings.TrimSpace(mode)
	}
	return payload
}

