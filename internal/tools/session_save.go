package tools

import "strings"

func BuildSessionSavePayload(sessionID, path string, messages int) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"path":       strings.TrimSpace(path),
		"messages":   messages,
	}
}

