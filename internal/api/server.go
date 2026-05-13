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
)

type Server struct {
	Engine *agent.Engine
	// Optional UI helpers for dashboard pages.
	ConfigSnapshotFn func() map[string]any
	GatewayStatusFn  func() map[string]any
	ConfigUpdateFn   func(key, value string) (map[string]any, error)
	GatewayActionFn  func(action string) (map[string]any, error)
	mu     sync.Mutex
	active map[string]activeRun
}

type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
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
	return mux
}

type recentSessionsStore interface {
	ListRecentSessions(limit int) ([]map[string]any, error)
}

type sessionDetailStore interface {
	LoadMessagesPage(sessionID string, offset, limit int) ([]core.Message, error)
	SessionStats(sessionID string) (map[string]any, error)
}

func (s *Server) handleUITools(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.Registry == nil {
		http.Error(w, "engine unavailable", http.StatusInternalServerError)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	names := s.Engine.Registry.Names()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"count": namesCount(names),
		"tools": names,
	})
}

func (s *Server) handleUIToolSchema(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.Registry == nil {
		http.Error(w, "engine unavailable", http.StatusInternalServerError)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "tool name required", http.StatusBadRequest)
		return
	}
	for _, schema := range s.Engine.Registry.Schemas() {
		if schema.Function.Name == name {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(schema)
			return
		}
	}
	http.Error(w, "tool not found", http.StatusNotFound)
}

func (s *Server) handleUISessions(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		http.Error(w, "session store unavailable", http.StatusInternalServerError)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "session listing not supported", http.StatusNotImplemented)
		return
	}
	rows, err := lister.ListRecentSessions(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"count":    len(rows),
		"sessions": rows,
	})
}

func (s *Server) handleUISessionDetail(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil || s.Engine.SessionStore == nil {
		http.Error(w, "session store unavailable", http.StatusInternalServerError)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/ui/sessions/"))
	if sessionID == "" {
		http.Error(w, "session id required", http.StatusBadRequest)
		return
	}
	detailer, ok := s.Engine.SessionStore.(sessionDetailStore)
	if !ok {
		http.Error(w, "session detail not supported", http.StatusNotImplemented)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stats, err := detailer.SessionStats(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.ConfigSnapshotFn == nil {
		http.Error(w, "config snapshot unavailable", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.ConfigSnapshotFn())
}

type uiConfigSetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *Server) handleUIConfigSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.ConfigUpdateFn == nil {
		http.Error(w, "config update unavailable", http.StatusNotImplemented)
		return
	}
	var req uiConfigSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	res, err := s.ConfigUpdateFn(req.Key, req.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleUIGatewayStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.GatewayStatusFn == nil {
		http.Error(w, "gateway status unavailable", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.GatewayStatusFn())
}

type uiGatewayActionRequest struct {
	Action string `json:"action"`
}

func (s *Server) handleUIGatewayAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.GatewayActionFn == nil {
		http.Error(w, "gateway action unavailable", http.StatusNotImplemented)
		return
	}
	var req uiGatewayActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	if req.Action == "" {
		http.Error(w, "action required", http.StatusBadRequest)
		return
	}
	res, err := s.GatewayActionFn(req.Action)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		http.Error(w, "engine unavailable", http.StatusInternalServerError)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}
	history, err := s.loadHistory(req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, "request cancelled", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(buildChatResponsePayload(res))
}

func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		http.Error(w, "engine unavailable", http.StatusInternalServerError)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}
	history, err := s.loadHistory(req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
					_ = writeSSE(w, "cancelled", map[string]any{"session_id": req.SessionID, "status": "cancelled", "reason": "request cancelled"})
				}
			} else if !errorSeen {
				_ = writeSSE(w, "error", map[string]any{"session_id": req.SessionID, "status": "error", "error": ctx.Err().Error()})
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
						_ = writeSSE(w, "cancelled", map[string]any{"session_id": req.SessionID, "status": "cancelled", "reason": "request cancelled"})
					}
				} else if !errorSeen {
					_ = writeSSE(w, "error", map[string]any{"session_id": req.SessionID, "status": "error", "error": res.Err.Error()})
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req cancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.SessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if !s.cancelActiveRun(req.SessionID) {
		http.Error(w, "active session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"session_id": req.SessionID, "cancelled": true})
}

var wsUpgrader = websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}

func (s *Server) handleChatWS(w http.ResponseWriter, r *http.Request) {
	if s.Engine == nil {
		http.Error(w, "engine unavailable", http.StatusInternalServerError)
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
		_ = conn.WriteJSON(map[string]any{"type": "error", "error": "invalid request: " + err.Error()})
		return
	}
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}

	history, err := s.loadHistory(req.SessionID)
	if err != nil {
		_ = conn.WriteJSON(map[string]any{"type": "error", "error": err.Error()})
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

	_ = conn.WriteJSON(map[string]any{"type": "session", "session_id": req.SessionID})

	go func() {
		res, runErr := eng.Run(ctx, req.SessionID, req.Message, agent.DefaultSystemPrompt(), history)
		done <- runResponse{Result: res, Err: runErr}
	}()

	cancelled := false
	for {
		select {
		case <-ctx.Done():
			if !cancelled {
				_ = conn.WriteJSON(map[string]any{"type": "cancelled", "session_id": req.SessionID, "reason": "request cancelled"})
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
							_ = conn.WriteJSON(map[string]any{"type": "error", "session_id": req.SessionID, "error": res.Err.Error()})
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
