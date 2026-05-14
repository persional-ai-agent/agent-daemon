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

type MattermostAdapter struct {
	webhookURL    string
	inboundSecret string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewMattermostAdapter(webhookURL, inboundSecret string) (*MattermostAdapter, error) {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return nil, errors.New("mattermost webhook url is required")
	}
	return &MattermostAdapter{
		webhookURL:    webhookURL,
		inboundSecret: strings.TrimSpace(inboundSecret),
		httpClient:    http.DefaultClient,
	}, nil
}

func (m *MattermostAdapter) Name() string                       { return "mattermost" }
func (m *MattermostAdapter) Connect(_ context.Context) error    { return nil }
func (m *MattermostAdapter) Disconnect(_ context.Context) error { return nil }

func (m *MattermostAdapter) Send(ctx context.Context, chatID, content, _ string) (gateway.SendResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}
	payload := map[string]any{"text": content}
	bs, err := json.Marshal(payload)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.webhookURL, bytes.NewReader(bs))
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.inboundSecret != "" {
		req.Header.Set("X-Agent-Mattermost-Secret", m.inboundSecret)
	}
	client := m.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return gateway.SendResult{Success: false, Error: fmt.Sprintf("mattermost send failed: %s", msg)}, nil
	}
	return gateway.SendResult{Success: true}, nil
}

func (m *MattermostAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("mattermost edit is not supported in minimal mode")
}
func (m *MattermostAdapter) SendTyping(_ context.Context, _ string) error { return nil }
func (m *MattermostAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	m.handler = handler
}

func (m *MattermostAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.inboundSecret != "" && strings.TrimSpace(req.Header.Get("X-Agent-Mattermost-Secret")) != m.inboundSecret {
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
		UserID    string `json:"user_id"`
		ChannelID string `json:"channel_id"`
		Text      string `json:"text"`
		PostID    string `json:"post_id"`
	}
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	userID := strings.TrimSpace(payload.UserID)
	chatID := strings.TrimSpace(payload.ChannelID)
	text := strings.TrimSpace(payload.Text)
	if userID == "" || chatID == "" || text == "" {
		http.Error(resp, "invalid payload", http.StatusBadRequest)
		return
	}
	if m.handler != nil {
		m.handler(req.Context(), gateway.MessageEvent{
			Text:      text,
			MessageID: strings.TrimSpace(payload.PostID),
			ChatID:    chatID,
			ChatType:  "group",
			UserID:    userID,
			UserName:  userID,
			IsCommand: strings.HasPrefix(text, "/"),
		})
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(`{"ok":true}`))
}
