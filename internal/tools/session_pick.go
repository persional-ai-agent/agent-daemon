package tools

import "strings"

func BuildSessionPickPayload(previousSessionID, sessionID string, index int) map[string]any {
	return map[string]any{
		"previous_session_id": strings.TrimSpace(previousSessionID),
		"session_id":          strings.TrimSpace(sessionID),
		"index":               index,
	}
}

