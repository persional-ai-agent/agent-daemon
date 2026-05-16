package tools

import "strings"

func BuildSessionClearPayload(previousSessionID, sessionID string, cleared bool) map[string]any {
	return map[string]any{
		"previous_session_id": strings.TrimSpace(previousSessionID),
		"session_id":          strings.TrimSpace(sessionID),
		"cleared":             cleared,
	}
}

