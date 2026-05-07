package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type Server struct {
	Engine *agent.Engine
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
	mux.HandleFunc("/v1/chat/cancel", s.handleCancel)
	return mux
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

	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				if !cancelledSeen {
					_ = writeSSE(w, "cancelled", map[string]any{"session_id": req.SessionID, "reason": "request cancelled"})
				}
			} else {
				_ = writeSSE(w, "error", map[string]any{"error": ctx.Err().Error(), "session_id": req.SessionID})
			}
			flusher.Flush()
			return
		case event := <-events:
			if event.Type == "cancelled" {
				cancelledSeen = true
			}
			if err := writeSSE(w, event.Type, event); err != nil {
				return
			}
			flusher.Flush()
		case res := <-done:
			if res.Err != nil {
				if errors.Is(res.Err, context.Canceled) {
					if !cancelledSeen {
						_ = writeSSE(w, "cancelled", map[string]any{"session_id": req.SessionID, "reason": "request cancelled"})
					}
				} else {
					_ = writeSSE(w, "error", map[string]any{"error": res.Err.Error(), "session_id": req.SessionID})
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
