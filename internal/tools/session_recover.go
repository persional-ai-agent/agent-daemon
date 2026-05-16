package tools

import "strings"

func BuildSessionRecoverPayload(previousSessionID, sessionID string, replay bool) map[string]any {
	payload := map[string]any{
		"recovered":           true,
		"previous_session_id": strings.TrimSpace(previousSessionID),
		"session_id":          strings.TrimSpace(sessionID),
	}
	if replay {
		payload["replay"] = true
	}
	return payload
}

