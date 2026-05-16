package tools

import "strings"

func BuildSessionOverviewPayload(sessionID, routeSession string, messagesInContext, toolsCount int) map[string]any {
	payload := map[string]any{
		"session_id": strings.TrimSpace(sessionID),
	}
	if strings.TrimSpace(routeSession) != "" {
		payload["route_session"] = strings.TrimSpace(routeSession)
	}
	if messagesInContext >= 0 {
		payload["messages_in_context"] = messagesInContext
	}
	if toolsCount >= 0 {
		payload["tools"] = toolsCount
	}
	return payload
}

