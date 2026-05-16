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

func BuildSlashSubcommandPayload(slash, subcommand string) map[string]any {
	payload := BuildSlashPayload(slash)
	if strings.TrimSpace(subcommand) != "" {
		payload["subcommand"] = strings.TrimSpace(subcommand)
	}
	return payload
}

func AttachSlashPayload(payload map[string]any, slash string) map[string]any {
	out := BuildSlashPayload(slash)
	for k, v := range payload {
		out[k] = v
	}
	return out
}
