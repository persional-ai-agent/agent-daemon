package tools

import "strings"

func BuildSessionUndoPayload(sessionID string, removedMessages, messagesInContext int) map[string]any {
	return map[string]any{
		"session_id":          strings.TrimSpace(sessionID),
		"removed_messages":    removedMessages,
		"messages_in_context": messagesInContext,
	}
}

