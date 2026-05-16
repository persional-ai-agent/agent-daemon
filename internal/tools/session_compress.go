package tools

import "strings"

func BuildSessionCompressPayload(sessionID string, before, after, tailMessages, summarizedMessages int, compacted bool, reason string) map[string]any {
	payload := map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"compacted":  compacted,
		"before":     before,
		"after":      after,
		"dropped":    before - after,
	}
	if tailMessages > 0 {
		payload["tail_messages"] = tailMessages
	}
	if summarizedMessages > 0 {
		payload["summarized_messages"] = summarizedMessages
	}
	if strings.TrimSpace(reason) != "" {
		payload["reason"] = strings.TrimSpace(reason)
	}
	return payload
}

