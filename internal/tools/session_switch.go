package tools

import "strings"

func BuildSessionSwitchPayload(previousSessionID, sessionID string, reset bool, loadedMessages int) map[string]any {
	payload := map[string]any{
		"previous_session_id": strings.TrimSpace(previousSessionID),
		"session_id":          strings.TrimSpace(sessionID),
	}
	if reset {
		payload["reset"] = true
	}
	if loadedMessages >= 0 {
		payload["loaded_messages"] = loadedMessages
	}
	return payload
}

