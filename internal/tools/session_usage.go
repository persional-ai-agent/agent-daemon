package tools

import "strings"

func BuildSessionUsagePayload(sessionID string, stats map[string]any) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"usage":      stats,
	}
}

func BuildSessionStatsPayload(sessionID string, stats map[string]any) map[string]any {
	return map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"stats":      stats,
	}
}

