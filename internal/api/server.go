package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/platform"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type Server struct {
	Engine *agent.Engine
	// Optional UI helpers for dashboard pages.
	ConfigSnapshotFn     func() map[string]any
	GatewayStatusFn      func() map[string]any
	GatewayDiagnosticsFn func() map[string]any
	ModelInfoFn          func() map[string]any
	ModelProvidersFn     func() []string
	ModelSetFn           func(provider, model, baseURL string) (map[string]any, error)
	PluginDashboardsFn   func() ([]map[string]any, error)
	ConfigUpdateFn       func(key, value string) (map[string]any, error)
	GatewayActionFn      func(action string) (map[string]any, error)
	SkillListFn          func() ([]map[string]any, error)
	SkillsReloadFn       func() (map[string]any, error)
	VoiceStatusFn        func() (map[string]any, error)
	VoiceToggleFn        func(action string) (map[string]any, error)
	VoiceRecordFn        func(action string) (map[string]any, error)
	VoiceTTSFn           func(text string) (map[string]any, error)
	mu                   sync.Mutex
	active               map[string]activeRun
	voiceEnabled         bool
	voiceRecording       bool
	voiceTTSEnabled      bool
}

type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	TurnID    string `json:"turn_id,omitempty"`
	Resume    bool   `json:"resume,omitempty"`
}

type cancelRequest struct {
	SessionID string `json:"session_id"`
}

type acpSessionCreateRequest struct {
	SessionID string `json:"session_id,omitempty"`
}

type acpMessageRequest struct {
	SessionID string `json:"session_id"`
	Input     string `json:"input"`
	TurnID    string `json:"turn_id,omitempty"`
	Resume    bool   `json:"resume,omitempty"`
}

type chatResponsePayload struct {
	SessionID         string         `json:"session_id"`
	FinalResponse     string         `json:"final_response"`
	Messages          []core.Message `json:"messages"`
	TurnsUsed         int            `json:"turns_used"`
	FinishedNaturally bool           `json:"finished_naturally"`
	Summary           map[string]any `json:"summary"`
}

type activeRun struct {
	token     string
	cancel    context.CancelFunc
	startedAt time.Time
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/v1/chat", s.handleChat)
	mux.HandleFunc("/v1/chat/stream", s.handleChatStream)
	mux.HandleFunc("/v1/chat/ws", s.handleChatWS)
	mux.HandleFunc("/v1/chat/cancel", s.handleCancel)
	mux.HandleFunc("/v1/gateway/matrix/webhook", s.handleGatewayMatrixWebhook)
	mux.HandleFunc("/v1/gateway/feishu/webhook", s.handleGatewayFeishuWebhook)
	mux.HandleFunc("/v1/gateway/dingtalk/webhook", s.handleGatewayDingTalkWebhook)
	mux.HandleFunc("/v1/gateway/wecom/webhook", s.handleGatewayWeComWebhook)
	mux.HandleFunc("/v1/gateway/mattermost/webhook", s.handleGatewayMattermostWebhook)
	mux.HandleFunc("/v1/gateway/sms/webhook", s.handleGatewaySMSWebhook)
	mux.HandleFunc("/v1/gateway/bluebubbles/webhook", s.handleGatewayBlueBubblesWebhook)
	mux.HandleFunc("/v1/gateway/signal/webhook", s.handleGatewaySignalWebhook)
	mux.HandleFunc("/v1/gateway/email/webhook", s.handleGatewayEmailWebhook)
	mux.HandleFunc("/v1/gateway/homeassistant/webhook", s.handleGatewayHomeAssistantWebhook)
	mux.HandleFunc("/v1/gateway/whatsapp/webhook", s.handleGatewayWhatsAppWebhook)
	mux.HandleFunc("/v1/gateway/webhook/inbound", s.handleGatewayWebhookInbound)
	mux.HandleFunc("/v1/acp/sessions", s.handleACPSessions)
	mux.HandleFunc("/v1/acp/message", s.handleACPMessage)
	mux.HandleFunc("/v1/acp/message/stream", s.handleACPMessageStream)
	mux.HandleFunc("/v1/acp/cancel", s.handleACPCancel)
	mux.HandleFunc("/v1/ui/tools", s.handleUITools)
	mux.HandleFunc("/v1/ui/tools/", s.handleUIToolSchema)
	mux.HandleFunc("/v1/ui/sessions", s.handleUISessions)
	mux.HandleFunc("/v1/ui/sessions/", s.handleUISessionDetail)
	mux.HandleFunc("/v1/ui/sessions/branch", s.handleUISessionBranch)
	mux.HandleFunc("/v1/ui/sessions/resume", s.handleUISessionResume)
	mux.HandleFunc("/v1/ui/sessions/compress", s.handleUISessionCompress)
	mux.HandleFunc("/v1/ui/sessions/replay", s.handleUISessionReplay)
	mux.HandleFunc("/v1/ui/config", s.handleUIConfig)
	mux.HandleFunc("/v1/ui/config/set", s.handleUIConfigSet)
	mux.HandleFunc("/v1/ui/model", s.handleUIModel)
	mux.HandleFunc("/v1/ui/model/providers", s.handleUIModelProviders)
	mux.HandleFunc("/v1/ui/model/set", s.handleUIModelSet)
	mux.HandleFunc("/v1/ui/gateway/status", s.handleUIGatewayStatus)
	mux.HandleFunc("/v1/ui/gateway/diagnostics", s.handleUIGatewayDiagnostics)
	mux.HandleFunc("/v1/ui/gateway/action", s.handleUIGatewayAction)
	mux.HandleFunc("/v1/ui/plugins/dashboards", s.handleUIPluginDashboards)
	mux.HandleFunc("/v1/ui/cron/jobs/action", s.handleUICronJobAction)
	mux.HandleFunc("/v1/ui/cron/jobs/", s.handleUICronJobDetail)
	mux.HandleFunc("/v1/ui/cron/jobs", s.handleUICronJobs)
	mux.HandleFunc("/v1/ui/approval/confirm", s.handleUIApprovalConfirm)
	mux.HandleFunc("/v1/ui/complete/slash", s.handleUICompleteSlash)
	mux.HandleFunc("/v1/ui/complete/path", s.handleUICompletePath)
	mux.HandleFunc("/v1/ui/agents", s.handleUIAgents)
	mux.HandleFunc("/v1/ui/agents/active", s.handleUIAgentsActive)
	mux.HandleFunc("/v1/ui/agents/detail", s.handleUIAgentsDetail)
	mux.HandleFunc("/v1/ui/agents/interrupt", s.handleUIAgentsInterrupt)
	mux.HandleFunc("/v1/ui/agents/history", s.handleUIAgentsHistory)
	mux.HandleFunc("/v1/ui/skills", s.handleUISkills)
	mux.HandleFunc("/v1/ui/skills/detail", s.handleUISkillDetail)
	mux.HandleFunc("/v1/ui/skills/manage", s.handleUISkillManage)
	mux.HandleFunc("/v1/ui/skills/reload", s.handleUISkillsReload)
	mux.HandleFunc("/v1/ui/skills/search", s.handleUISkillsSearch)
	mux.HandleFunc("/v1/ui/skills/sync", s.handleUISkillsSync)
	mux.HandleFunc("/v1/ui/voice/status", s.handleUIVoiceStatus)
	mux.HandleFunc("/v1/ui/voice/toggle", s.handleUIVoiceToggle)
	mux.HandleFunc("/v1/ui/voice/record", s.handleUIVoiceRecord)
	mux.HandleFunc("/v1/ui/voice/tts", s.handleUIVoiceTTS)
	return mux
}

func (s *Server) handleGatewayWhatsAppWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("whatsapp")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "whatsapp gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "whatsapp webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewaySignalWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("signal")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "signal gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "signal webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayMatrixWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("matrix")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "matrix gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "matrix webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayFeishuWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("feishu")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "feishu gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "feishu webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayDingTalkWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("dingtalk")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "dingtalk gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "dingtalk webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayWeComWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("wecom")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "wecom gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "wecom webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayMattermostWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("mattermost")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "mattermost gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "mattermost webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewaySMSWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("sms")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "sms gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "sms webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayBlueBubblesWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("bluebubbles")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "bluebubbles gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "bluebubbles webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayEmailWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("email")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "email gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "email webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayHomeAssistantWebhook(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("homeassistant")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "homeassistant gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "homeassistant webhook is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleGatewayWebhookInbound(w http.ResponseWriter, r *http.Request) {
	adapter, ok := platform.Get("webhook")
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "gateway_unavailable", "webhook gateway adapter not connected")
		return
	}
	handler, ok := adapter.(interface {
		HandleWebhook(http.ResponseWriter, *http.Request)
	})
	if !ok {
		writeAPIError(w, http.StatusNotImplemented, "not_supported", "webhook inbound is not supported by adapter")
		return
	}
	handler.HandleWebhook(w, r)
}

func (s *Server) handleACPSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req acpSessionCreateRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"session_id": sessionID,
		"result": map[string]any{
			"session_id": sessionID,
		},
	})
}

func (s *Server) handleACPMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req acpMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	chatReq := chatRequest{
		SessionID: strings.TrimSpace(req.SessionID),
		Message:   req.Input,
		TurnID:    req.TurnID,
		Resume:    req.Resume,
	}
	s.handleChat(w, cloneRequestWithJSONBody(r, chatReq))
}

func (s *Server) handleACPMessageStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req acpMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	chatReq := chatRequest{
		SessionID: strings.TrimSpace(req.SessionID),
		Message:   req.Input,
		TurnID:    req.TurnID,
		Resume:    req.Resume,
	}
	s.handleChatStream(w, cloneRequestWithJSONBody(r, chatReq))
}

func (s *Server) handleACPCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req cancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	s.handleCancel(w, cloneRequestWithJSONBody(r, req))
}

func cloneRequestWithJSONBody(r *http.Request, v any) *http.Request {
	bs, _ := json.Marshal(v)
	req := r.Clone(r.Context())
	req.Body = io.NopCloser(strings.NewReader(string(bs)))
	req.ContentLength = int64(len(bs))
	req.Header = req.Header.Clone()
	req.Header.Set("Content-Type", "application/json")
	return req
}

type recentSessionsStore interface {
	ListRecentSessions(limit int) ([]map[string]any, error)
}

type sessionDetailStore interface {
	LoadMessagesPage(sessionID string, offset, limit int) ([]core.Message, error)
	SessionStats(sessionID string) (map[string]any, error)
}

type sessionBranchStore interface {
	LoadMessages(sessionID string, limit int) ([]core.Message, error)
	AppendMessage(sessionID string, msg core.Message) error
}

type agentListItem struct {
	SessionID      string `json:"session_id"`
	DelegateCount  int    `json:"delegate_count"`
	LastGoal       string `json:"last_goal,omitempty"`
	LastToolCallID string `json:"last_tool_call_id,omitempty"`
}

type agentHistoryItem struct {
	SessionID     string   `json:"session_id"`
	DelegateCount int      `json:"delegate_count"`
	Goals         []string `json:"goals,omitempty"`
}

type agentDetailItem struct {
	SessionID      string   `json:"session_id"`
	Running        bool     `json:"running"`
	DelegateCount  int      `json:"delegate_count"`
	LastGoal       string   `json:"last_goal,omitempty"`
	LastToolCallID string   `json:"last_tool_call_id,omitempty"`
	Goals          []string `json:"goals,omitempty"`
}

type agentActiveItem struct {
	SessionID      string `json:"session_id"`
	Running        bool   `json:"running"`
	StartedAt      string `json:"started_at"`
	DurationSec    int64  `json:"duration_sec"`
	DelegateCount  int    `json:"delegate_count"`
	LastGoal       string `json:"last_goal,omitempty"`
	LastToolCallID string `json:"last_tool_call_id,omitempty"`
}

const (
	uiAPIVersion = "v1"
	uiCompat     = "2026-05-13"
)

var apiProcessStartedAt = time.Now()

func writeUIHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Agent-UI-API-Version", uiAPIVersion)
	w.Header().Set("X-Agent-UI-API-Compat", uiCompat)
}

func writeUIJSON(w http.ResponseWriter, status int, payload map[string]any) {
	writeUIHeaders(w)
	w.WriteHeader(status)
	if payload == nil {
		payload = map[string]any{}
	}
	payload["api_version"] = uiAPIVersion
	payload["compat"] = uiCompat
	_ = json.NewEncoder(w).Encode(payload)
}

func writeUIError(w http.ResponseWriter, status int, code, message string) {
	writeUIJSON(w, status, map[string]any{
		"ok": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeUIHeaders(w)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":          false,
		"api_version": uiAPIVersion,
		"compat":      uiCompat,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func runtimeErrorPayload(sessionID, code, message string) map[string]any {
	return map[string]any{
		"session_id": sessionID,
		"status":     "error",
		"error_code": code,
		"error":      message,
		"error_detail": map[string]any{
			"code":    code,
			"message": message,
		},
		"api_version": uiAPIVersion,
		"compat":      uiCompat,
	}
}

func cancelledPayload(sessionID, reason string) map[string]any {
	return map[string]any{
		"session_id":  sessionID,
		"status":      "cancelled",
		"reason":      reason,
		"error_code":  "cancelled",
		"api_version": uiAPIVersion,
		"compat":      uiCompat,
	}
}

func resumedPayload(sessionID, turnID, transport string) map[string]any {
	return map[string]any{
		"type":        "resumed",
		"session_id":  sessionID,
		"turn_id":     turnID,
		"resumed":     true,
		"transport":   transport,
		"api_version": uiAPIVersion,
		"compat":      uiCompat,
	}
}

func runtimeErrorCode(err error) string {
	if err == nil {
		return "internal_error"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	return "internal_error"
}

func (s *Server) handleUITools(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.Registry == nil {
		writeUIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	names := s.Engine.Registry.Names()
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"count": namesCount(names),
		"tools": names,
	})
}

func (s *Server) handleUIToolSchema(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.Registry == nil {
		writeUIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1/ui/tools/")
	if !strings.HasSuffix(path, "/schema") {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimSuffix(path, "/schema")
	name = strings.TrimSpace(strings.Trim(name, "/"))
	if name == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "tool name required")
		return
	}
	for _, schema := range s.Engine.Registry.Schemas() {
		if schema.Function.Name == name {
			writeUIJSON(w, http.StatusOK, map[string]any{
				"ok":     true,
				"schema": schema,
			})
			return
		}
	}
	writeUIError(w, http.StatusNotFound, "not_found", "tool not found")
}

func (s *Server) handleUISessions(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	lister, ok := s.Engine.SessionStore.(recentSessionsStore)
	if !ok {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "session listing not supported")
		return
	}
	rows, err := lister.ListRecentSessions(limit)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"count":    len(rows),
		"sessions": rows,
	})
}

func (s *Server) handleUISessionDetail(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	sessionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/ui/sessions/"))
	if sessionID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "session id required")
		return
	}
	detailer, ok := s.Engine.SessionStore.(sessionDetailStore)
	if !ok {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "session detail not supported")
		return
	}
	offset := 0
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	msgs, err := detailer.LoadMessagesPage(sessionID, offset, limit)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	stats, err := detailer.SessionStats(sessionID)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"session_id": sessionID,
		"offset":     offset,
		"limit":      limit,
		"count":      len(msgs),
		"messages":   msgs,
		"stats":      stats,
	})
}

func (s *Server) handleUIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.ConfigSnapshotFn == nil {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "config snapshot unavailable")
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"snapshot": s.ConfigSnapshotFn(),
	})
}

type uiConfigSetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type uiSessionBranchRequest struct {
	SessionID    string `json:"session_id"`
	NewSessionID string `json:"new_session_id,omitempty"`
	LastN        int    `json:"last_n,omitempty"`
}

type uiSessionResumeRequest struct {
	SessionID string `json:"session_id"`
	TurnID    string `json:"turn_id,omitempty"`
}

type uiSessionCompressRequest struct {
	SessionID string `json:"session_id"`
	KeepLastN int    `json:"keep_last_n,omitempty"`
}

type uiSessionReplayRequest struct {
	SessionID string `json:"session_id"`
	Offset    int    `json:"offset,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func (s *Server) handleUIConfigSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.ConfigUpdateFn == nil {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "config update unavailable")
		return
	}
	var req uiConfigSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "key required")
		return
	}
	res, err := s.ConfigUpdateFn(req.Key, req.Value)
	if err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"result": res,
	})
}

func (s *Server) handleUIModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.ModelInfoFn == nil {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "model info unavailable")
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"model": s.ModelInfoFn(),
	})
}

func (s *Server) handleUIModelProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.ModelProvidersFn == nil {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "model providers unavailable")
		return
	}
	providers := s.ModelProvidersFn()
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"count":     len(providers),
		"providers": providers,
	})
}

func (s *Server) handleUIModelSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.ModelSetFn == nil {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "model update unavailable")
		return
	}
	var req uiModelSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.Model = strings.TrimSpace(req.Model)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	if req.Provider == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "provider required")
		return
	}
	if req.Model == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "model required")
		return
	}
	result, err := s.ModelSetFn(req.Provider, req.Model, req.BaseURL)
	if err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"result": result,
	})
}

func (s *Server) handleUISessionBranch(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiSessionBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.NewSessionID = strings.TrimSpace(req.NewSessionID)
	if req.SessionID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "session_id required")
		return
	}
	if req.NewSessionID == "" {
		req.NewSessionID = uuid.NewString()
	}
	if req.NewSessionID == req.SessionID {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "new_session_id must differ from session_id")
		return
	}
	ss, ok := s.Engine.SessionStore.(sessionBranchStore)
	if !ok {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "session branching not supported")
		return
	}
	msgs, err := ss.LoadMessages(req.SessionID, 500)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if req.LastN > 0 && req.LastN < len(msgs) {
		msgs = msgs[len(msgs)-req.LastN:]
	}
	for _, msg := range msgs {
		if err := ss.AppendMessage(req.NewSessionID, msg); err != nil {
			writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"result": map[string]any{
			"source_session_id": req.SessionID,
			"new_session_id":    req.NewSessionID,
			"copied_messages":   len(msgs),
		},
	})
}

func (s *Server) handleUISessionResume(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiSessionResumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.TurnID = strings.TrimSpace(req.TurnID)
	if req.SessionID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "session_id required")
		return
	}
	_, err := s.Engine.SessionStore.LoadMessages(req.SessionID, 1)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"result": map[string]any{
			"session_id": req.SessionID,
			"turn_id":    req.TurnID,
			"resumed":    true,
			"transport":  "http",
		},
	})
}

func (s *Server) handleUISessionCompress(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiSessionCompressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "session_id required")
		return
	}
	keep := req.KeepLastN
	if keep <= 0 {
		keep = 20
	}
	msgs, err := s.Engine.SessionStore.LoadMessages(req.SessionID, 500)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	before := len(msgs)
	after := before
	if keep < before {
		after = keep
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"result": map[string]any{
			"session_id":       req.SessionID,
			"compressed":       true,
			"before_messages":  before,
			"after_messages":   after,
			"dropped_messages": before - after,
			"keep_last_n":      keep,
		},
	})
}

func (s *Server) handleUISessionReplay(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiSessionReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "session_id required")
		return
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	detailer, ok := s.Engine.SessionStore.(sessionDetailStore)
	if !ok {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "session replay not supported")
		return
	}
	msgs, err := detailer.LoadMessagesPage(req.SessionID, offset, limit)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"result": map[string]any{
			"session_id": req.SessionID,
			"offset":     offset,
			"limit":      limit,
			"count":      len(msgs),
			"messages":   msgs,
			"replayed":   true,
		},
	})
}

func (s *Server) handleUIGatewayStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.GatewayStatusFn == nil {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "gateway status unavailable")
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"status": s.GatewayStatusFn(),
	})
}

func (s *Server) handleUIGatewayDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.GatewayDiagnosticsFn != nil {
		writeUIJSON(w, http.StatusOK, map[string]any{
			"ok":          true,
			"diagnostics": s.GatewayDiagnosticsFn(),
		})
		return
	}
	active := s.activeRunsSnapshot()
	sessionIDs := make([]string, 0, len(active))
	for sid := range active {
		sessionIDs = append(sessionIDs, sid)
	}
	sort.Strings(sessionIDs)
	uptimeSec := int64(time.Since(apiProcessStartedAt).Seconds())
	if uptimeSec < 0 {
		uptimeSec = 0
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"diagnostics": map[string]any{
			"uptime_sec":              uptimeSec,
			"active_run_count":        len(active),
			"active_session_ids":      sessionIDs,
			"status_endpoint_enabled": s.GatewayStatusFn != nil,
			"action_endpoint_enabled": s.GatewayActionFn != nil,
		},
	})
}

func (s *Server) handleUIPluginDashboards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.PluginDashboardsFn == nil {
		writeUIJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"dashboards": []any{},
			"count":      0,
		})
		return
	}
	items, err := s.PluginDashboardsFn()
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"dashboards": items,
		"count":      len(items),
	})
}

func (s *Server) handleUICronJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		out, ok := s.dispatchUICron(w, r, map[string]any{"action": "list"})
		if !ok {
			return
		}
		writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "result": out})
	case http.MethodPost:
		var req uiCronJobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		args := map[string]any{
			"action":          "create",
			"name":            strings.TrimSpace(req.Name),
			"prompt":          strings.TrimSpace(req.Prompt),
			"schedule":        strings.TrimSpace(req.Schedule),
			"delivery_target": strings.TrimSpace(req.DeliveryTarget),
			"deliver_on":      strings.TrimSpace(req.DeliverOn),
			"context_mode":    strings.TrimSpace(req.ContextMode),
			"run_mode":        strings.TrimSpace(req.RunMode),
			"script_command":  strings.TrimSpace(req.ScriptCommand),
			"script_cwd":      strings.TrimSpace(req.ScriptCWD),
		}
		if req.ScriptTimeout > 0 {
			args["script_timeout"] = req.ScriptTimeout
		}
		if req.ChainContext != nil {
			args["chain_context"] = *req.ChainContext
		}
		if req.Repeat != nil {
			args["repeat"] = *req.Repeat
		}
		out, ok := s.dispatchUICron(w, r, args)
		if !ok {
			return
		}
		writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "result": out})
	default:
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (s *Server) handleUICronJobDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	jobID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/ui/cron/jobs/"))
	if jobID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "job_id required")
		return
	}
	out, ok := s.dispatchUICron(w, r, map[string]any{"action": "get", "job_id": jobID})
	if !ok {
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "result": out})
}

func (s *Server) handleUICronJobAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiCronJobActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "action required")
		return
	}
	args := map[string]any{"action": action}
	if v := strings.TrimSpace(req.JobID); v != "" {
		args["job_id"] = v
	}
	if v := strings.TrimSpace(req.RunID); v != "" {
		args["run_id"] = v
	}
	if v := strings.TrimSpace(req.Name); v != "" {
		args["name"] = v
	}
	if v := strings.TrimSpace(req.Prompt); v != "" {
		args["prompt"] = v
	}
	if v := strings.TrimSpace(req.Schedule); v != "" {
		args["schedule"] = v
	}
	if v := strings.TrimSpace(req.DeliveryTarget); v != "" {
		args["delivery_target"] = v
	}
	if v := strings.TrimSpace(req.DeliverOn); v != "" {
		args["deliver_on"] = v
	}
	if v := strings.TrimSpace(req.ContextMode); v != "" {
		args["context_mode"] = v
	}
	if v := strings.TrimSpace(req.RunMode); v != "" {
		args["run_mode"] = v
	}
	if v := strings.TrimSpace(req.ScriptCommand); v != "" {
		args["script_command"] = v
	}
	if v := strings.TrimSpace(req.ScriptCWD); v != "" {
		args["script_cwd"] = v
	}
	if req.ScriptTimeout > 0 {
		args["script_timeout"] = req.ScriptTimeout
	}
	if req.ChainContext != nil {
		args["chain_context"] = *req.ChainContext
	}
	if req.Repeat != nil {
		args["repeat"] = *req.Repeat
	}
	if req.Paused != nil {
		args["paused"] = *req.Paused
	}
	if req.Limit > 0 {
		args["limit"] = req.Limit
	}
	out, ok := s.dispatchUICron(w, r, args)
	if !ok {
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "result": out})
}

func (s *Server) dispatchUICron(w http.ResponseWriter, r *http.Request, args map[string]any) (map[string]any, bool) {
	if s.Engine == nil || s.Engine.Registry == nil {
		writeUIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return nil, false
	}
	raw := s.Engine.Registry.Dispatch(r.Context(), "cronjob", args, tools.ToolContext{
		SessionStore:   s.Engine.SearchStore,
		MemoryStore:    s.Engine.MemoryStore,
		TodoStore:      s.Engine.TodoStore,
		ApprovalStore:  s.Engine.ApprovalStore,
		DelegateRunner: s.Engine,
		Workdir:        s.Engine.Workdir,
	})
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return nil, false
	}
	if e, ok := out["error"].(string); ok && strings.TrimSpace(e) != "" {
		writeUIError(w, http.StatusBadRequest, "tool_error", e)
		return nil, false
	}
	return out, true
}

type uiGatewayActionRequest struct {
	Action string `json:"action"`
}

type uiModelSetRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url,omitempty"`
}

type uiCompleteSlashRequest struct {
	Text string `json:"text"`
}

type uiCompletePathRequest struct {
	Path string `json:"path"`
}

type uiApprovalConfirmRequest struct {
	SessionID  string `json:"session_id"`
	ApprovalID string `json:"approval_id"`
	Approve    bool   `json:"approve"`
}

type uiCronJobRequest struct {
	Name           string `json:"name,omitempty"`
	Prompt         string `json:"prompt,omitempty"`
	Schedule       string `json:"schedule,omitempty"`
	Repeat         *int   `json:"repeat,omitempty"`
	DeliveryTarget string `json:"delivery_target,omitempty"`
	DeliverOn      string `json:"deliver_on,omitempty"`
	ContextMode    string `json:"context_mode,omitempty"`
	ChainContext   *bool  `json:"chain_context,omitempty"`
	RunMode        string `json:"run_mode,omitempty"`
	ScriptCommand  string `json:"script_command,omitempty"`
	ScriptCWD      string `json:"script_cwd,omitempty"`
	ScriptTimeout  int    `json:"script_timeout,omitempty"`
}

type uiCronJobActionRequest struct {
	Action         string `json:"action"`
	JobID          string `json:"job_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	Name           string `json:"name,omitempty"`
	Prompt         string `json:"prompt,omitempty"`
	Schedule       string `json:"schedule,omitempty"`
	Repeat         *int   `json:"repeat,omitempty"`
	DeliveryTarget string `json:"delivery_target,omitempty"`
	DeliverOn      string `json:"deliver_on,omitempty"`
	ContextMode    string `json:"context_mode,omitempty"`
	ChainContext   *bool  `json:"chain_context,omitempty"`
	RunMode        string `json:"run_mode,omitempty"`
	ScriptCommand  string `json:"script_command,omitempty"`
	ScriptCWD      string `json:"script_cwd,omitempty"`
	ScriptTimeout  int    `json:"script_timeout,omitempty"`
	Paused         *bool  `json:"paused,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

type uiAgentsInterruptRequest struct {
	SessionID string `json:"session_id"`
}

type uiVoiceToggleRequest struct {
	Action string `json:"action"`
}

type uiVoiceRecordRequest struct {
	Action string `json:"action"`
}

type uiVoiceTTSRequest struct {
	Text string `json:"text"`
}

type uiSkillDetailRequest struct {
	Name string `json:"name"`
}

type uiSkillManageRequest struct {
	Action     string `json:"action"`
	Name       string `json:"name"`
	Content    string `json:"content,omitempty"`
	OldString  string `json:"old_string,omitempty"`
	NewString  string `json:"new_string,omitempty"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

type uiSkillsSearchRequest struct {
	Query string `json:"query"`
	Repo  string `json:"repo,omitempty"`
}

type uiSkillsSyncRequest struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	URL    string `json:"url,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Path   string `json:"path,omitempty"`
}

var apiSkillNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

var uiSlashCommands = []string{
	"/help", "/tools", "/tool", "/sessions", "/session", "/show", "/pick", "/open", "/stats",
	"/gateway", "/config", "/panel", "/refresh", "/fullscreen", "/diag", "/doctor", "/pending",
	"/approve", "/deny", "/bookmark", "/workbench", "/workflow", "/history", "/timeline",
	"/events", "/rerun", "/save", "/pretty", "/api", "/http", "/version", "/reconnect", "/quit",
}

func (s *Server) handleUICompleteSlash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiCompleteSlashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeUIJSON(w, http.StatusOK, map[string]any{
			"ok":           true,
			"replace_from": 1,
			"items":        uiSlashCommands,
		})
		return
	}
	if !strings.HasPrefix(text, "/") {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "slash text must start with /")
		return
	}
	first := text
	if idx := strings.IndexAny(first, " \t"); idx >= 0 {
		first = first[:idx]
	}
	first = strings.ToLower(first)
	items := make([]string, 0, len(uiSlashCommands))
	for _, cmd := range uiSlashCommands {
		if strings.HasPrefix(cmd, first) {
			items = append(items, cmd)
		}
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"replace_from": 1,
		"items":        items,
	})
}

func (s *Server) handleUICompletePath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiCompletePathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	raw := strings.TrimSpace(req.Path)
	if raw == "" {
		raw = "."
	}
	dir, base := filepath.Split(raw)
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	items := make([]string, 0, 32)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, base) {
			continue
		}
		candidate := filepath.Join(dir, name)
		if e.IsDir() {
			candidate += string(filepath.Separator)
		}
		items = append(items, candidate)
	}
	sort.Strings(items)
	if len(items) > 100 {
		items = items[:100]
	}
	replaceFrom := len(raw) - len(base) + 1
	if replaceFrom < 1 {
		replaceFrom = 1
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"replace_from": replaceFrom,
		"items":        items,
	})
}

func (s *Server) handleUIGatewayAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.GatewayActionFn == nil {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "gateway action unavailable")
		return
	}
	var req uiGatewayActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	if req.Action == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "action required")
		return
	}
	res, err := s.GatewayActionFn(req.Action)
	if err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"result": res,
	})
}

func (s *Server) handleUIApprovalConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.Engine == nil || s.Engine.Registry == nil {
		writeUIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return
	}
	var req uiApprovalConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ApprovalID = strings.TrimSpace(req.ApprovalID)
	if req.SessionID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "session_id required")
		return
	}
	if req.ApprovalID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "approval_id required")
		return
	}
	raw := s.Engine.Registry.Dispatch(r.Context(), "approval", map[string]any{
		"action":      "confirm",
		"approval_id": req.ApprovalID,
		"approve":     req.Approve,
	}, tools.ToolContext{
		SessionID:      req.SessionID,
		SessionStore:   s.Engine.SearchStore,
		MemoryStore:    s.Engine.MemoryStore,
		TodoStore:      s.Engine.TodoStore,
		ApprovalStore:  s.Engine.ApprovalStore,
		DelegateRunner: s.Engine,
		Workdir:        s.Engine.Workdir,
	})
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if e, ok := out["error"].(string); ok && strings.TrimSpace(e) != "" {
		writeUIError(w, http.StatusBadRequest, "tool_error", e)
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"result": out,
	})
}

func (s *Server) handleUIAgents(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	ids := make([]string, 0, limit)
	if sessionID != "" {
		ids = append(ids, sessionID)
	} else {
		lister, ok := s.Engine.SessionStore.(recentSessionsStore)
		if !ok {
			writeUIError(w, http.StatusNotImplemented, "not_supported", "session listing not supported")
			return
		}
		rows, err := lister.ListRecentSessions(limit)
		if err != nil {
			writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		for _, row := range rows {
			id, _ := row["session_id"].(string)
			id = strings.TrimSpace(id)
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	agents := make([]agentListItem, 0, len(ids))
	for _, sid := range ids {
		msgs, err := s.Engine.SessionStore.LoadMessages(sid, 500)
		if err != nil {
			continue
		}
		item := agentListItem{SessionID: sid}
		for _, msg := range msgs {
			for _, tc := range msg.ToolCalls {
				if strings.TrimSpace(tc.Function.Name) != "delegate_task" {
					continue
				}
				item.DelegateCount++
				item.LastToolCallID = tc.ID
				args := tools.ParseJSONArgs(tc.Function.Arguments)
				goal, _ := args["goal"].(string)
				goal = strings.TrimSpace(goal)
				if goal != "" {
					item.LastGoal = goal
				}
			}
		}
		if item.DelegateCount > 0 {
			agents = append(agents, item)
		}
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"count":  len(agents),
		"agents": agents,
	})
}

func (s *Server) handleUIAgentsActive(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	filterSessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	now := time.Now()
	runs := s.activeRunsSnapshot()
	items := make([]agentActiveItem, 0, len(runs))
	for sid, run := range runs {
		if filterSessionID != "" && sid != filterSessionID {
			continue
		}
		msgs, err := s.Engine.SessionStore.LoadMessages(sid, 500)
		if err != nil {
			continue
		}
		item := agentActiveItem{
			SessionID:   sid,
			Running:     true,
			StartedAt:   run.startedAt.UTC().Format(time.RFC3339),
			DurationSec: int64(now.Sub(run.startedAt).Seconds()),
		}
		for _, msg := range msgs {
			for _, tc := range msg.ToolCalls {
				if strings.TrimSpace(tc.Function.Name) != "delegate_task" {
					continue
				}
				item.DelegateCount++
				item.LastToolCallID = tc.ID
				args := tools.ParseJSONArgs(tc.Function.Arguments)
				goal, _ := args["goal"].(string)
				goal = strings.TrimSpace(goal)
				if goal != "" {
					item.LastGoal = goal
				}
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SessionID < items[j].SessionID })
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"count":  len(items),
		"active": items,
	})
}

func (s *Server) handleUIAgentsDetail(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "session_id required")
		return
	}
	msgs, err := s.Engine.SessionStore.LoadMessages(sessionID, 500)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	detail := agentDetailItem{SessionID: sessionID, Running: s.isActiveRun(sessionID)}
	goalSet := map[string]struct{}{}
	for _, msg := range msgs {
		for _, tc := range msg.ToolCalls {
			if strings.TrimSpace(tc.Function.Name) != "delegate_task" {
				continue
			}
			detail.DelegateCount++
			detail.LastToolCallID = tc.ID
			args := tools.ParseJSONArgs(tc.Function.Arguments)
			goal, _ := args["goal"].(string)
			goal = strings.TrimSpace(goal)
			if goal == "" {
				continue
			}
			detail.LastGoal = goal
			if _, ok := goalSet[goal]; ok {
				continue
			}
			goalSet[goal] = struct{}{}
			detail.Goals = append(detail.Goals, goal)
		}
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"session": detail,
	})
}

func (s *Server) handleUIAgentsInterrupt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiAgentsInterruptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "session_id required")
		return
	}
	if !s.cancelActiveRun(req.SessionID) {
		writeUIError(w, http.StatusNotFound, "not_found", "active session not found")
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"session_id":  req.SessionID,
		"interrupted": true,
		"result": map[string]any{
			"session_id":  req.SessionID,
			"interrupted": true,
		},
	})
}

func (s *Server) handleUIAgentsHistory(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		writeUIError(w, http.StatusInternalServerError, "session_store_unavailable", "session store unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	lister, ok := s.Engine.SessionStore.(recentSessionsStore)
	if !ok {
		writeUIError(w, http.StatusNotImplemented, "not_supported", "session listing not supported")
		return
	}
	rows, err := lister.ListRecentSessions(limit)
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	history := make([]agentHistoryItem, 0, len(rows))
	for _, row := range rows {
		sid, _ := row["session_id"].(string)
		sid = strings.TrimSpace(sid)
		if sid == "" {
			continue
		}
		msgs, err := s.Engine.SessionStore.LoadMessages(sid, 500)
		if err != nil {
			continue
		}
		item := agentHistoryItem{SessionID: sid}
		goalSet := map[string]struct{}{}
		for _, msg := range msgs {
			for _, tc := range msg.ToolCalls {
				if strings.TrimSpace(tc.Function.Name) != "delegate_task" {
					continue
				}
				item.DelegateCount++
				args := tools.ParseJSONArgs(tc.Function.Arguments)
				goal, _ := args["goal"].(string)
				goal = strings.TrimSpace(goal)
				if goal == "" {
					continue
				}
				if _, exists := goalSet[goal]; exists {
					continue
				}
				goalSet[goal] = struct{}{}
				item.Goals = append(item.Goals, goal)
			}
		}
		if item.DelegateCount > 0 {
			history = append(history, item)
		}
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"count":   len(history),
		"history": history,
	})
}

func (s *Server) handleUISkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var (
		skills []map[string]any
		err    error
	)
	if s.SkillListFn != nil {
		skills, err = s.SkillListFn()
	} else {
		workdir := "."
		if s.Engine != nil && strings.TrimSpace(s.Engine.Workdir) != "" {
			workdir = s.Engine.Workdir
		}
		skills, err = listLocalSkills(workdir)
	}
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"count":  len(skills),
		"skills": skills,
	})
}

func (s *Server) handleUISkillDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "name required")
		return
	}
	if !apiSkillNameRE.MatchString(name) {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "invalid skill name")
		return
	}
	workdir := "."
	if s.Engine != nil && strings.TrimSpace(s.Engine.Workdir) != "" {
		workdir = s.Engine.Workdir
	}
	path := filepath.Join(workdir, "skills", name, "SKILL.md")
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeUIError(w, http.StatusNotFound, "not_found", "skill not found")
			return
		}
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"skill": map[string]any{
			"name":    name,
			"path":    path,
			"content": string(bs),
		},
	})
}

func (s *Server) handleUISkillManage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiSkillManageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	name := strings.TrimSpace(req.Name)
	if name == "" || !apiSkillNameRE.MatchString(name) {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "invalid skill name")
		return
	}
	workdir := "."
	if s.Engine != nil && strings.TrimSpace(s.Engine.Workdir) != "" {
		workdir = s.Engine.Workdir
	}
	skillDir := filepath.Join(workdir, "skills", name)
	skillMD := filepath.Join(skillDir, "SKILL.md")
	result := map[string]any{"action": action, "name": name}
	switch action {
	case "create":
		if strings.TrimSpace(req.Content) == "" {
			writeUIError(w, http.StatusBadRequest, "invalid_argument", "content required")
			return
		}
		if _, err := os.Stat(skillMD); err == nil {
			writeUIError(w, http.StatusBadRequest, "invalid_argument", "skill already exists")
			return
		}
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if err := os.WriteFile(skillMD, []byte(req.Content), 0o644); err != nil {
			writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		result["path"] = skillMD
	case "edit":
		if strings.TrimSpace(req.Content) == "" {
			writeUIError(w, http.StatusBadRequest, "invalid_argument", "content required")
			return
		}
		if _, err := os.Stat(skillMD); err != nil {
			writeUIError(w, http.StatusNotFound, "not_found", "skill not found")
			return
		}
		if err := os.WriteFile(skillMD, []byte(req.Content), 0o644); err != nil {
			writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		result["path"] = skillMD
	case "patch":
		if req.OldString == "" {
			writeUIError(w, http.StatusBadRequest, "invalid_argument", "old_string required")
			return
		}
		bs, err := os.ReadFile(skillMD)
		if err != nil {
			writeUIError(w, http.StatusNotFound, "not_found", "skill not found")
			return
		}
		content := string(bs)
		matchCount := strings.Count(content, req.OldString)
		if matchCount == 0 {
			writeUIError(w, http.StatusBadRequest, "invalid_argument", "old_string not found")
			return
		}
		if !req.ReplaceAll && matchCount != 1 {
			writeUIError(w, http.StatusBadRequest, "invalid_argument", "old_string matched multiple times; set replace_all=true")
			return
		}
		replacements := 1
		updated := strings.Replace(content, req.OldString, req.NewString, 1)
		if req.ReplaceAll {
			replacements = matchCount
			updated = strings.ReplaceAll(content, req.OldString, req.NewString)
		}
		if err := os.WriteFile(skillMD, []byte(updated), 0o644); err != nil {
			writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		result["path"] = skillMD
		result["replacements"] = replacements
	case "delete":
		if _, err := os.Stat(skillDir); err != nil {
			writeUIError(w, http.StatusNotFound, "not_found", "skill not found")
			return
		}
		if err := os.RemoveAll(skillDir); err != nil {
			writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		result["path"] = skillDir
	default:
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "unsupported action")
		return
	}
	result["success"] = true
	writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (s *Server) handleUISkillsReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var (
		result map[string]any
		err    error
	)
	if s.SkillsReloadFn != nil {
		result, err = s.SkillsReloadFn()
	} else {
		workdir := "."
		if s.Engine != nil && strings.TrimSpace(s.Engine.Workdir) != "" {
			workdir = s.Engine.Workdir
		}
		skills, listErr := listLocalSkills(workdir)
		if listErr != nil {
			err = listErr
		} else {
			result = map[string]any{
				"success": true,
				"count":   len(skills),
			}
		}
	}
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"result": result,
	})
}

func (s *Server) handleUISkillsSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.Engine == nil || s.Engine.Registry == nil {
		writeUIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return
	}
	var req uiSkillsSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	req.Repo = strings.TrimSpace(req.Repo)
	if req.Query == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "query required")
		return
	}
	args := map[string]any{"query": req.Query}
	if req.Repo != "" {
		args["repo"] = req.Repo
	}
	raw := s.Engine.Registry.Dispatch(r.Context(), "skill_search", args, tools.ToolContext{
		Workdir: s.engineWorkdir(),
	})
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if e, ok := out["error"].(string); ok && strings.TrimSpace(e) != "" {
		writeUIError(w, http.StatusBadRequest, "tool_error", e)
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"result": out,
	})
}

func (s *Server) handleUISkillsSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if s.Engine == nil || s.Engine.Registry == nil {
		writeUIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return
	}
	var req uiSkillsSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Source = strings.TrimSpace(strings.ToLower(req.Source))
	req.URL = strings.TrimSpace(req.URL)
	req.Repo = strings.TrimSpace(req.Repo)
	req.Path = strings.TrimSpace(req.Path)
	if req.Name == "" || !apiSkillNameRE.MatchString(req.Name) {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "invalid skill name")
		return
	}
	if req.Source == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "source required")
		return
	}
	args := map[string]any{
		"action": "sync",
		"name":   req.Name,
		"source": req.Source,
	}
	switch req.Source {
	case "url":
		if req.URL == "" {
			writeUIError(w, http.StatusBadRequest, "invalid_argument", "url required")
			return
		}
		args["url"] = req.URL
	case "github":
		if req.Repo == "" || req.Path == "" {
			writeUIError(w, http.StatusBadRequest, "invalid_argument", "repo and path required")
			return
		}
		args["repo"] = req.Repo
		args["sub_path"] = req.Path
	default:
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "unsupported source")
		return
	}
	raw := s.Engine.Registry.Dispatch(r.Context(), "skill_manage", args, tools.ToolContext{
		Workdir: s.engineWorkdir(),
	})
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if e, ok := out["error"].(string); ok && strings.TrimSpace(e) != "" {
		writeUIError(w, http.StatusBadRequest, "tool_error", e)
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"result": out,
	})
}

func (s *Server) engineWorkdir() string {
	if s.Engine != nil && strings.TrimSpace(s.Engine.Workdir) != "" {
		return s.Engine.Workdir
	}
	return "."
}

func listLocalSkills(workdir string) ([]map[string]any, error) {
	root := filepath.Join(workdir, "skills")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	skills := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		skillMD := filepath.Join(root, name, "SKILL.md")
		if _, statErr := os.Stat(skillMD); statErr != nil {
			continue
		}
		skills = append(skills, map[string]any{
			"name": name,
			"path": skillMD,
		})
	}
	sort.Slice(skills, func(i, j int) bool {
		li, _ := skills[i]["name"].(string)
		lj, _ := skills[j]["name"].(string)
		return li < lj
	})
	return skills, nil
}

func (s *Server) handleUIVoiceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var (
		status map[string]any
		err    error
	)
	if s.VoiceStatusFn != nil {
		status, err = s.VoiceStatusFn()
	} else {
		status = s.voiceStatus()
	}
	if err != nil {
		writeUIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "status": status})
}

func (s *Server) handleUIVoiceToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiVoiceToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Action = strings.TrimSpace(strings.ToLower(req.Action))
	if req.Action == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "action required")
		return
	}
	var (
		result map[string]any
		err    error
	)
	if s.VoiceToggleFn != nil {
		result, err = s.VoiceToggleFn(req.Action)
	} else {
		result, err = s.voiceToggle(req.Action)
	}
	if err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (s *Server) handleUIVoiceRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiVoiceRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Action = strings.TrimSpace(strings.ToLower(req.Action))
	if req.Action == "" {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", "action required")
		return
	}
	var (
		result map[string]any
		err    error
	)
	if s.VoiceRecordFn != nil {
		result, err = s.VoiceRecordFn(req.Action)
	} else {
		result, err = s.voiceRecord(req.Action)
	}
	if err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (s *Server) handleUIVoiceTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req uiVoiceTTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	var (
		result map[string]any
		err    error
	)
	if s.VoiceTTSFn != nil {
		result, err = s.VoiceTTSFn(req.Text)
	} else {
		result, err = s.voiceSpeak(req.Text)
	}
	if err != nil {
		writeUIError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (s *Server) voiceStatus() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"enabled":   s.voiceEnabled,
		"recording": s.voiceRecording,
		"tts":       s.voiceTTSEnabled,
	}
}

func (s *Server) voiceToggle(action string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch action {
	case "on":
		s.voiceEnabled = true
	case "off":
		s.voiceEnabled = false
		s.voiceRecording = false
	case "tts":
		if !s.voiceEnabled {
			return nil, errors.New("enable voice mode first")
		}
		s.voiceTTSEnabled = !s.voiceTTSEnabled
	case "status":
	default:
		return nil, fmt.Errorf("unknown voice action: %s", action)
	}
	return map[string]any{
		"enabled":   s.voiceEnabled,
		"recording": s.voiceRecording,
		"tts":       s.voiceTTSEnabled,
		"action":    action,
	}, nil
}

func (s *Server) voiceRecord(action string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch action {
	case "start":
		if !s.voiceEnabled {
			return nil, errors.New("voice mode is off")
		}
		s.voiceRecording = true
	case "stop":
		s.voiceRecording = false
	default:
		return nil, fmt.Errorf("unknown voice action: %s", action)
	}
	return map[string]any{
		"enabled":   s.voiceEnabled,
		"recording": s.voiceRecording,
		"tts":       s.voiceTTSEnabled,
		"action":    action,
	}, nil
}

func (s *Server) voiceSpeak(text string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.voiceEnabled {
		return nil, errors.New("voice mode is off")
	}
	if !s.voiceTTSEnabled {
		return nil, errors.New("tts is disabled")
	}
	return map[string]any{
		"spoken":  true,
		"text":    text,
		"length":  len([]rune(text)),
		"enabled": s.voiceEnabled,
		"tts":     s.voiceTTSEnabled,
	}, nil
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		writeAPIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return
	}
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}
	history, err := s.loadHistory(req.SessionID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	token := s.registerActiveRun(req.SessionID, cancel)
	defer func() {
		s.unregisterActiveRun(req.SessionID, token)
		cancel()
	}()
	res, err := s.Engine.Run(ctx, req.SessionID, req.Message, agent.DefaultSystemPrompt(), history)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			writeAPIError(w, http.StatusConflict, "cancelled", "request cancelled")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	payload := buildChatResponsePayload(res)
	out := map[string]any{
		"ok":          true,
		"api_version": uiAPIVersion,
		"compat":      uiCompat,
		"result":      payload,
		// Backward-compat: keep legacy top-level fields while introducing result envelope.
		"session_id":         payload.SessionID,
		"final_response":     payload.FinalResponse,
		"messages":           payload.Messages,
		"turns_used":         payload.TurnsUsed,
		"finished_naturally": payload.FinishedNaturally,
		"summary":            payload.Summary,
	}
	writeUIHeaders(w)
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		writeAPIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return
	}
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "streaming_unsupported", "streaming unsupported")
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}
	history, err := s.loadHistory(req.SessionID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	token := s.registerActiveRun(req.SessionID, cancel)
	defer func() {
		s.unregisterActiveRun(req.SessionID, token)
		cancel()
	}()

	events := make(chan core.AgentEvent, 64)
	type runResponse struct {
		Result *core.RunResult
		Err    error
	}
	done := make(chan runResponse, 1)

	eng := *s.Engine
	eng.EventSink = func(event core.AgentEvent) {
		select {
		case events <- event:
		case <-ctx.Done():
		}
	}

	go func() {
		res, err := eng.Run(ctx, req.SessionID, req.Message, agent.DefaultSystemPrompt(), history)
		done <- runResponse{Result: res, Err: err}
	}()

	if err := writeSSE(w, "session", map[string]any{"session_id": req.SessionID}); err != nil {
		return
	}
	if req.Resume {
		if err := writeSSE(w, "resumed", resumedPayload(req.SessionID, req.TurnID, "sse")); err != nil {
			return
		}
	}
	flusher.Flush()
	cancelledSeen := false
	errorSeen := false

	for {
		select {
		case <-ctx.Done():
			for {
				select {
				case event := <-events:
					if event.Type == "cancelled" {
						cancelledSeen = true
					}
					if event.Type == "error" {
						errorSeen = true
					}
					if err := writeSSE(w, event.Type, event); err != nil {
						return
					}
					flusher.Flush()
				default:
					goto doneCtx
				}
			}
		doneCtx:
			if errors.Is(ctx.Err(), context.Canceled) {
				if !cancelledSeen {
					_ = writeSSE(w, "cancelled", cancelledPayload(req.SessionID, "request cancelled"))
				}
			} else if !errorSeen {
				code := runtimeErrorCode(ctx.Err())
				_ = writeSSE(w, "error", runtimeErrorPayload(req.SessionID, code, ctx.Err().Error()))
			}
			flusher.Flush()
			return
		case event := <-events:
			if event.Type == "cancelled" {
				cancelledSeen = true
			}
			if event.Type == "error" {
				errorSeen = true
			}
			if err := writeSSE(w, event.Type, event); err != nil {
				return
			}
			flusher.Flush()
		case res := <-done:
			if res.Err != nil {
				for {
					select {
					case event := <-events:
						if event.Type == "cancelled" {
							cancelledSeen = true
						}
						if event.Type == "error" {
							errorSeen = true
						}
						if err := writeSSE(w, event.Type, event); err != nil {
							return
						}
						flusher.Flush()
					default:
						goto doneErr
					}
				}
			doneErr:
				if errors.Is(res.Err, context.Canceled) {
					if !cancelledSeen {
						_ = writeSSE(w, "cancelled", cancelledPayload(req.SessionID, "request cancelled"))
					}
				} else if !errorSeen {
					code := runtimeErrorCode(res.Err)
					_ = writeSSE(w, "error", runtimeErrorPayload(req.SessionID, code, res.Err.Error()))
				}
				flusher.Flush()
				return
			}
			for {
				select {
				case event := <-events:
					if event.Type == "cancelled" {
						cancelledSeen = true
					}
					if event.Type == "error" {
						errorSeen = true
					}
					if err := writeSSE(w, event.Type, event); err != nil {
						return
					}
					flusher.Flush()
				default:
					if err := writeSSE(w, "result", res.Result); err != nil {
						return
					}
					flusher.Flush()
					return
				}
			}
		}
	}
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req cancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.SessionID == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_argument", "session_id is required")
		return
	}
	if !s.cancelActiveRun(req.SessionID) {
		writeAPIError(w, http.StatusNotFound, "not_found", "active session not found")
		return
	}
	writeUIJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"result":     map[string]any{"session_id": req.SessionID, "cancelled": true},
		"session_id": req.SessionID,
		"cancelled":  true, // backward-compat
	})
}

var wsUpgrader = websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}

func (s *Server) handleChatWS(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		writeAPIError(w, http.StatusInternalServerError, "engine_unavailable", "engine unavailable")
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	var req chatRequest
	if err := conn.ReadJSON(&req); err != nil {
		payload := runtimeErrorPayload("", "invalid_json", "invalid request: "+err.Error())
		payload["type"] = "error"
		_ = conn.WriteJSON(payload)
		return
	}
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}

	history, err := s.loadHistory(req.SessionID)
	if err != nil {
		payload := runtimeErrorPayload(req.SessionID, "internal_error", err.Error())
		payload["type"] = "error"
		_ = conn.WriteJSON(payload)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	token := s.registerActiveRun(req.SessionID, cancel)
	defer func() {
		s.unregisterActiveRun(req.SessionID, token)
		cancel()
	}()

	events := make(chan core.AgentEvent, 64)
	type runResponse struct {
		Result *core.RunResult
		Err    error
	}
	done := make(chan runResponse, 1)

	eng := *s.Engine
	eng.EventSink = func(event core.AgentEvent) {
		select {
		case events <- event:
		case <-ctx.Done():
		}
	}

	_ = conn.WriteJSON(map[string]any{"type": "session", "session_id": req.SessionID, "api_version": uiAPIVersion, "compat": uiCompat})
	if req.Resume {
		_ = conn.WriteJSON(resumedPayload(req.SessionID, req.TurnID, "ws"))
	}

	go func() {
		res, runErr := eng.Run(ctx, req.SessionID, req.Message, agent.DefaultSystemPrompt(), history)
		done <- runResponse{Result: res, Err: runErr}
	}()

	cancelled := false
	for {
		select {
		case <-ctx.Done():
			if !cancelled {
				payload := cancelledPayload(req.SessionID, "request cancelled")
				payload["type"] = "cancelled"
				_ = conn.WriteJSON(payload)
			}
			return
		case event := <-events:
			_ = conn.WriteJSON(event)
			if event.Type == "cancelled" {
				cancelled = true
			}
		case res := <-done:
			close(done)
			for {
				select {
				case event := <-events:
					_ = conn.WriteJSON(event)
					if event.Type == "cancelled" {
						cancelled = true
					}
				default:
					if res.Err != nil {
						if !cancelled {
							code := runtimeErrorCode(res.Err)
							payload := runtimeErrorPayload(req.SessionID, code, res.Err.Error())
							payload["type"] = "error"
							_ = conn.WriteJSON(payload)
						}
						return
					}
					_ = conn.WriteJSON(map[string]any{
						"type":               "result",
						"session_id":         res.Result.SessionID,
						"final_response":     res.Result.FinalResponse,
						"turns_used":         res.Result.TurnsUsed,
						"finished_naturally": res.Result.FinishedNaturally,
						"summary":            summarizeRunResult(res.Result),
					})
					return
				}
			}
		}
	}
}

func (s *Server) loadHistory(sessionID string) ([]core.Message, error) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		return nil, nil
	}
	return s.Engine.SessionStore.LoadMessages(sessionID, 500)
}

func writeSSE(w http.ResponseWriter, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	return nil
}

func buildChatResponsePayload(res *core.RunResult) chatResponsePayload {
	return chatResponsePayload{
		SessionID:         res.SessionID,
		FinalResponse:     res.FinalResponse,
		Messages:          res.Messages,
		TurnsUsed:         res.TurnsUsed,
		FinishedNaturally: res.FinishedNaturally,
		Summary:           summarizeRunResult(res),
	}
}

func summarizeRunResult(res *core.RunResult) map[string]any {
	assistantCount := 0
	toolCount := 0
	delegateCount := 0
	toolNames := make([]string, 0)
	seenTools := make(map[string]struct{})
	for _, msg := range res.Messages {
		switch msg.Role {
		case "assistant":
			assistantCount++
		case "tool":
			toolCount++
			if msg.Name == "delegate_task" {
				delegateCount++
			}
			if msg.Name != "" {
				if _, ok := seenTools[msg.Name]; !ok {
					seenTools[msg.Name] = struct{}{}
					toolNames = append(toolNames, msg.Name)
				}
			}
		}
	}
	sort.Strings(toolNames)
	status := "completed"
	if !res.FinishedNaturally {
		status = "max_iterations_reached"
	}
	return map[string]any{
		"status":                  status,
		"message_count":           len(res.Messages),
		"assistant_message_count": assistantCount,
		"tool_call_count":         toolCount,
		"delegate_count":          delegateCount,
		"tool_names":              toolNames,
	}
}

func namesCount(items []string) int {
	return len(items)
}

func (s *Server) registerActiveRun(sessionID string, cancel context.CancelFunc) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil {
		s.active = make(map[string]activeRun)
	}
	token := uuid.NewString()
	s.active[sessionID] = activeRun{token: token, cancel: cancel, startedAt: time.Now()}
	return token
}

func (s *Server) unregisterActiveRun(sessionID, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.active[sessionID]
	if !ok || run.token != token {
		return
	}
	delete(s.active, sessionID)
}

func (s *Server) cancelActiveRun(sessionID string) bool {
	s.mu.Lock()
	run, ok := s.active[sessionID]
	s.mu.Unlock()
	if !ok {
		return false
	}
	run.cancel()
	return true
}

func (s *Server) isActiveRun(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil {
		return false
	}
	_, ok := s.active[sessionID]
	return ok
}

func (s *Server) activeRunsSnapshot() map[string]activeRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]activeRun, len(s.active))
	for sid, run := range s.active {
		out[sid] = run
	}
	return out
}
