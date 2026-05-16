package tools

import "strings"

func BuildSessionHistoryPayload(sessionID string, totalMessages, limit int) map[string]any {
	payload := map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"limit":      limit,
	}
	if totalMessages >= 0 {
		payload["total_messages"] = totalMessages
		count := limit
		if totalMessages < count {
			count = totalMessages
		}
		if count < 0 {
			count = 0
		}
		payload["count"] = count
	}
	return payload
}

