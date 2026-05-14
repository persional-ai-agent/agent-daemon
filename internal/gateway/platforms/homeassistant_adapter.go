package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

type HomeAssistantAdapter struct {
	baseURL       string
	token         string
	inboundSecret string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewHomeAssistantAdapter(baseURL, token, inboundSecret string) (*HomeAssistantAdapter, error) {
	baseURL = strings.TrimSpace(baseURL)
	token = strings.TrimSpace(token)
	if baseURL == "" {
		return nil, errors.New("homeassistant base url is required")
	}
	if token == "" {
		return nil, errors.New("homeassistant token is required")
	}
	return &HomeAssistantAdapter{
		baseURL:       strings.TrimRight(baseURL, "/"),
		token:         token,
		inboundSecret: strings.TrimSpace(inboundSecret),
		httpClient:    http.DefaultClient,
	}, nil
}

func (h *HomeAssistantAdapter) Name() string { return "homeassistant" }

func (h *HomeAssistantAdapter) Connect(_ context.Context) error { return nil }

func (h *HomeAssistantAdapter) Disconnect(_ context.Context) error { return nil }

func (h *HomeAssistantAdapter) Send(ctx context.Context, chatID, content, _ string) (gateway.SendResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}
	title := strings.TrimSpace(chatID)
	if title == "" {
		title = "Agent"
	}
	body := map[string]any{
		"title":   title,
		"message": content,
	}
	bs, err := json.Marshal(body)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+"/api/services/persistent_notification/create", bytes.NewReader(bs))
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.token)
	if h.inboundSecret != "" {
		req.Header.Set("X-Agent-HA-Secret", h.inboundSecret)
	}
	client := h.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return gateway.SendResult{Success: false, Error: fmt.Sprintf("homeassistant send failed: %s", msg)}, nil
	}
	return gateway.SendResult{Success: true}, nil
}

func (h *HomeAssistantAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("homeassistant does not support edit message")
}

func (h *HomeAssistantAdapter) SendTyping(_ context.Context, _ string) error { return nil }

func (h *HomeAssistantAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	h.handler = handler
}

func (h *HomeAssistantAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.inboundSecret != "" && strings.TrimSpace(req.Header.Get("X-Agent-HA-Secret")) != h.inboundSecret {
		http.Error(resp, "forbidden", http.StatusForbidden)
		return
	}
	defer req.Body.Close()
	bs, err := io.ReadAll(io.LimitReader(req.Body, 2<<20))
	if err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	var payload struct {
		Text   string `json:"text"`
		UserID string `json:"user_id"`
		ChatID string `json:"chat_id"`
		Event  struct {
			Data struct {
				Message string `json:"message"`
				UserID  string `json:"user_id"`
			} `json:"data"`
		} `json:"event"`
	}
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(payload.Text)
	if text == "" {
		text = strings.TrimSpace(payload.Event.Data.Message)
	}
	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		userID = strings.TrimSpace(payload.Event.Data.UserID)
	}
	if userID == "" {
		userID = "homeassistant"
	}
	chatID := strings.TrimSpace(payload.ChatID)
	if chatID == "" {
		chatID = userID
	}
	if text == "" {
		http.Error(resp, "text required", http.StatusBadRequest)
		return
	}
	if h.handler != nil {
		h.handler(req.Context(), gateway.MessageEvent{
			Text:      text,
			ChatID:    chatID,
			ChatType:  "dm",
			UserID:    userID,
			UserName:  userID,
			IsCommand: strings.HasPrefix(text, "/"),
		})
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(`{"ok":true}`))
}
