package tools

import "strings"

func BuildCollectionPayload(key string, count int, items any) map[string]any {
	k := strings.TrimSpace(strings.ToLower(key))
	if k == "" {
		k = "items"
	}
	return map[string]any{
		"count": count,
		k:       items,
	}
}

func BuildMemoryContentPayload(target string, content any) map[string]any {
	return map[string]any{
		"target":  strings.TrimSpace(strings.ToLower(target)),
		"content": content,
	}
}

func BuildMemorySnapshotPayload(snapshot any) map[string]any {
	return map[string]any{
		"memory": snapshot,
	}
}

func BuildPersonalityPayload(mode, systemPrompt string) map[string]any {
	payload := map[string]any{
		"system_prompt": systemPrompt,
	}
	mode = strings.TrimSpace(strings.ToLower(mode))
	switch mode {
	case "reset":
		payload["reset"] = true
	case "set", "update":
		payload["updated"] = true
	}
	return payload
}

func BuildObjectPayload(key string, value any) map[string]any {
	k := strings.TrimSpace(strings.ToLower(key))
	if k == "" {
		k = "data"
	}
	return map[string]any{k: value}
}
