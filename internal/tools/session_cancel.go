package tools

import "strings"

func BuildSessionCancelPayload(sessionID string, cancelled bool, reason string) map[string]any {
	payload := map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"cancelled":  cancelled,
	}
	if strings.TrimSpace(reason) != "" {
		payload["reason"] = strings.TrimSpace(reason)
	}
	return payload
}
