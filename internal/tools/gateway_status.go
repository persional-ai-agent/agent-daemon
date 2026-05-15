package tools

import (
	"fmt"
	"sort"
	"strings"
)

type GatewayStatusSnapshot struct {
	Platform       string
	RouteSession   string
	ActiveSession  string
	QueueLen       int
	Paired         bool
	ContinuityMode string
	MappedSession  string
	MessageCount   any
	Running        bool
	LastApprovalID string
}

func (s GatewayStatusSnapshot) IsZero() bool {
	return strings.TrimSpace(s.Platform) == "" &&
		strings.TrimSpace(s.RouteSession) == "" &&
		strings.TrimSpace(s.ActiveSession) == "" &&
		s.QueueLen == 0 &&
		!s.Paired &&
		strings.TrimSpace(s.ContinuityMode) == "" &&
		strings.TrimSpace(s.MappedSession) == "" &&
		s.MessageCount == nil &&
		!s.Running &&
		strings.TrimSpace(s.LastApprovalID) == ""
}

func ExtractGatewayStatusSnapshot(status map[string]any) GatewayStatusSnapshot {
	out := GatewayStatusSnapshot{}
	if status == nil {
		return out
	}
	if v, ok := status["platform"].(string); ok {
		out.Platform = strings.TrimSpace(v)
	}
	if v, ok := status["route_session"].(string); ok {
		out.RouteSession = strings.TrimSpace(v)
	}
	if v, ok := status["active_session"].(string); ok {
		out.ActiveSession = strings.TrimSpace(v)
	}
	if v, ok := status["queue"].(int); ok {
		out.QueueLen = v
	} else if vf, ok := status["queue"].(float64); ok {
		out.QueueLen = int(vf)
	}
	if v, ok := status["paired"].(bool); ok {
		out.Paired = v
	}
	if v, ok := status["continuity_mode"].(string); ok {
		out.ContinuityMode = strings.TrimSpace(v)
	}
	if v, ok := status["mapped_session"].(string); ok {
		out.MappedSession = strings.TrimSpace(v)
	}
	if v, ok := status["message_count"]; ok {
		out.MessageCount = v
	}
	if v, ok := status["running"].(bool); ok {
		out.Running = v
	}
	if v, ok := status["last_approval_id"].(string); ok {
		out.LastApprovalID = strings.TrimSpace(v)
	}
	return out
}

func BuildGatewayStatusPayload(s GatewayStatusSnapshot) map[string]any {
	out := map[string]any{
		"platform":        s.Platform,
		"route_session":   s.RouteSession,
		"active_session":  s.ActiveSession,
		"queue":           s.QueueLen,
		"paired":          s.Paired,
		"continuity_mode": s.ContinuityMode,
		"running":         s.Running,
	}
	if strings.TrimSpace(s.MappedSession) != "" {
		out["mapped_session"] = s.MappedSession
	}
	if s.MessageCount != nil {
		out["message_count"] = s.MessageCount
	}
	if strings.TrimSpace(s.LastApprovalID) != "" {
		out["last_approval_id"] = s.LastApprovalID
	}
	return out
}

func NormalizeGatewayStatusMap(raw map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range raw {
		out[k] = v
	}
	snap := ExtractGatewayStatusSnapshot(raw)
	if snap.IsZero() {
		return out
	}
	for k, v := range BuildGatewayStatusPayload(snap) {
		if _, exists := out[k]; !exists {
			out[k] = v
		}
	}
	return out
}

func RenderGatewayStatusText(s GatewayStatusSnapshot) string {
	lines := []string{
		"platform: " + s.Platform,
		"route_session: " + s.RouteSession,
		"active_session: " + s.ActiveSession,
		"queue: " + fmt.Sprintf("%d", s.QueueLen),
		"paired: " + map[bool]string{true: "yes", false: "no"}[s.Paired],
		"continuity_mode: " + s.ContinuityMode,
	}
	if strings.TrimSpace(s.MappedSession) != "" {
		lines = append(lines, "mapped_session: "+s.MappedSession)
	}
	if s.MessageCount != nil {
		lines = append(lines, "message_count: "+fmt.Sprintf("%v", s.MessageCount))
	}
	lines = append(lines, "running: "+map[bool]string{true: "yes", false: "no"}[s.Running])
	if strings.TrimSpace(s.LastApprovalID) != "" {
		lines = append(lines, "last_approval_id: "+s.LastApprovalID)
	}
	return strings.Join(lines, "\n")
}

func BuildGatewayDiagnosticsFallback(activeSessionIDs []string, uptimeSec int64, statusEnabled bool, actionEnabled bool) map[string]any {
	ids := append([]string(nil), activeSessionIDs...)
	sort.Strings(ids)
	if uptimeSec < 0 {
		uptimeSec = 0
	}
	return map[string]any{
		"uptime_sec":              uptimeSec,
		"active_run_count":        len(ids),
		"active_session_ids":      ids,
		"status_endpoint_enabled": statusEnabled,
		"action_endpoint_enabled": actionEnabled,
	}
}
