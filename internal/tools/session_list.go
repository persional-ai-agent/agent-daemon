package tools

func BuildSessionListPayload(limit int, sessions []map[string]any) map[string]any {
	return map[string]any{
		"count":    len(sessions),
		"limit":    limit,
		"sessions": sessions,
	}
}

