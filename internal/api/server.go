package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type Server struct {
	Engine *agent.Engine
	// Optional UI helpers for dashboard pages.
	ConfigSnapshotFn func() map[string]any
	GatewayStatusFn  func() map[string]any
	ConfigUpdateFn   func(key, value string) (map[string]any, error)
	GatewayActionFn  func(action string) (map[string]any, error)
	mu               sync.Mutex
	active           map[string]activeRun
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

type chatResponsePayload struct {
	SessionID         string         `json:"session_id"`
	FinalResponse     string         `json:"final_response"`
	Messages          []core.Message `json:"messages"`
	TurnsUsed         int            `json:"turns_used"`
	FinishedNaturally bool           `json:"finished_naturally"`
	Summary           map[string]any `json:"summary"`
}

type activeRun struct {
	token  string
	cancel context.CancelFunc
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
	mux.HandleFunc("/v1/ui/tools", s.handleUITools)
	mux.HandleFunc("/v1/ui/tools/", s.handleUIToolSchema)
	mux.HandleFunc("/v1/ui/sessions", s.handleUISessions)
	mux.HandleFunc("/v1/ui/sessions/", s.handleUISessionDetail)
	mux.HandleFunc("/v1/ui/config", s.handleUIConfig)
	mux.HandleFunc("/v1/ui/config/set", s.handleUIConfigSet)
	mux.HandleFunc("/v1/ui/gateway/status", s.handleUIGatewayStatus)
	mux.HandleFunc("/v1/ui/gateway/action", s.handleUIGatewayAction)
	mux.HandleFunc("/v1/ui/approval/confirm", s.handleUIApprovalConfirm)
	return mux
}

type recentSessionsStore interface {
	ListRecentSessions(limit int) ([]map[string]any, error)
}

type sessionDetailStore interface {
	LoadMessagesPage(sessionID string, offset, limit int) ([]core.Message, error)
	SessionStats(sessionID string) (map[string]any, error)
}

const (
	uiAPIVersion = "v1"
	uiCompat     = "2026-05-13"
)

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
		"session_id":  sessionID,
		"status":      "error",
		"error_code":  code,
		"error":       message,
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

type uiGatewayActionRequest struct {
	Action string `json:"action"`
}

type uiApprovalConfirmRequest struct {
	SessionID  string `json:"session_id"`
	ApprovalID string `json:"approval_id"`
	Approve    bool   `json:"approve"`
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
	s.active[sessionID] = activeRun{token: token, cancel: cancel}
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
