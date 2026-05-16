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

func BuildSlashModePayloadWithExtra(slash, mode string, extra map[string]any) map[string]any {
	return mergeSlashExtra(BuildSlashModePayload(slash, mode), extra)
}

func BuildSlashSubcommandPayload(slash, subcommand string) map[string]any {
	payload := BuildSlashPayload(slash)
	if strings.TrimSpace(subcommand) != "" {
		payload["subcommand"] = strings.TrimSpace(subcommand)
	}
	return payload
}

func BuildSlashSubcommandPayloadWithExtra(slash, subcommand string, extra map[string]any) map[string]any {
	return mergeSlashExtra(BuildSlashSubcommandPayload(slash, subcommand), extra)
}

func AttachSlashPayload(payload map[string]any, slash string) map[string]any {
	out := BuildSlashPayload(slash)
	for k, v := range payload {
		out[k] = v
	}
	return out
}

func BuildAuthPayload(status string) map[string]any {
	return map[string]any{
		"auth": strings.TrimSpace(status),
	}
}

func mergeSlashExtra(base map[string]any, extra map[string]any) map[string]any {
	for k, v := range extra {
		base[k] = v
	}
	return base
}
