package tools

import "strings"

func BuildApprovalCommandPayload(slash, approvalID string) map[string]any {
	payload := map[string]any{
		"slash": strings.TrimSpace(slash),
	}
	if strings.TrimSpace(approvalID) != "" {
		payload["approval_id"] = strings.TrimSpace(approvalID)
	}
	return payload
}

