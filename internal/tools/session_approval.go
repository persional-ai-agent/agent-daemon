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

func ApprovalFailedEN(errText string) string { return "Approval failed: " + errText }
func ApprovalDeniedEN(command string) string {
	if strings.TrimSpace(command) == "" {
		return "Denied."
	}
	return "Denied: " + command
}
func ApprovalApprovedExecutedEN(output string) string { return "Approved and executed.\n" + output }
func ApprovalApprovedEN(command string) string {
	if strings.TrimSpace(command) == "" {
		return "Approved."
	}
	return "Approved: " + command
}
func ApprovalStatusFailedEN(errText string) string { return "Approval status failed: " + errText }
func NoActiveApprovalsEN() string                  { return "No active approvals." }
func SessionApprovalActiveEN() string              { return "Session approval: active" }
func NoPendingApprovalEN() string                  { return "No pending approval." }
func GrantFailedEN(errText string) string          { return "Grant failed: " + errText }
func GrantedPatternApprovalEN(pattern, expiresAt string) string {
	if strings.TrimSpace(expiresAt) != "" {
		return "Granted pattern approval: " + pattern + " until " + expiresAt
	}
	return "Granted pattern approval: " + pattern
}
func GrantedSessionApprovalEN(expiresAt string) string {
	if strings.TrimSpace(expiresAt) != "" {
		return "Granted session approval until " + expiresAt
	}
	return "Granted session approval."
}
func RevokeFailedEN(errText string) string { return "Revoke failed: " + errText }
func RevokedPatternApprovalEN(pattern string) string {
	return "Revoked pattern approval: " + pattern
}
func RevokedSessionApprovalEN() string { return "Revoked session approval." }
func NoActiveSessionApprovalEN() string {
	return "No active session approval."
}
