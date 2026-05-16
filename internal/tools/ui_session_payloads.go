package tools

import (
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func BuildUISessionBranchPayload(sourceSessionID, newSessionID string, copiedMessages int) map[string]any {
	return map[string]any{
		"source_session_id": strings.TrimSpace(sourceSessionID),
		"new_session_id":    strings.TrimSpace(newSessionID),
		"copied_messages":   copiedMessages,
	}
}

func BuildUISessionResumePayload(sessionID, turnID, transport string) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"turn_id":    strings.TrimSpace(turnID),
		"resumed":    true,
		"transport":  strings.TrimSpace(transport),
	}
}

func BuildUISessionCompressPayload(sessionID string, before, after, keepLastN int) map[string]any {
	return map[string]any{
		"session_id":       strings.TrimSpace(sessionID),
		"compressed":       true,
		"before_messages":  before,
		"after_messages":   after,
		"dropped_messages": before - after,
		"keep_last_n":      keepLastN,
	}
}

func BuildUISessionUndoPayload(sourceSessionID, newSessionID string, copiedMessages, removedMessages int, undone bool, reason string) map[string]any {
	payload := map[string]any{
		"source_session_id": strings.TrimSpace(sourceSessionID),
		"new_session_id":    strings.TrimSpace(newSessionID),
		"copied_messages":   copiedMessages,
		"removed_messages":  removedMessages,
		"undone":            undone,
	}
	if strings.TrimSpace(reason) != "" {
		payload["reason"] = strings.TrimSpace(reason)
	}
	return payload
}

func BuildUISessionReplayPayload(sessionID string, offset, limit int, messages []core.Message) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"offset":     offset,
		"limit":      limit,
		"count":      len(messages),
		"messages":   messages,
		"replayed":   true,
	}
}

func BuildUISessionRenamePayload(sourceSessionID, newSessionID string, copiedMessages int) map[string]any {
	return map[string]any{
		"source_session_id": strings.TrimSpace(sourceSessionID),
		"new_session_id":    strings.TrimSpace(newSessionID),
		"copied_messages":   copiedMessages,
		"renamed":           true,
	}
}

func BuildUISessionDeletePayload(sessionID string, deleted bool) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"deleted":    deleted,
	}
}

func BuildUISessionExportPayload(sessionID string, format string, messages []core.Message, content string) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"format":     strings.TrimSpace(format),
		"count":      len(messages),
		"messages":   messages,
		"content":    content,
		"exported":   true,
	}
}
