package tools

import (
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func BuildSessionShowPayload(sessionID string, offset, limit int, messages []core.Message) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"offset":     offset,
		"limit":      limit,
		"count":      len(messages),
		"messages":   messages,
	}
}

