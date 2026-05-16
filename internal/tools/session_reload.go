package tools

import (
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func BuildSessionReloadPayload(sessionID string, messages []core.Message) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"count":      len(messages),
		"messages":   messages,
	}
}

