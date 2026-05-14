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

type FeishuAdapter struct {
	webhookURL    string
	inboundSecret string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewFeishuAdapter(webhookURL, inboundSecret string) (*FeishuAdapter, error) {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return nil, errors.New("feishu webhook url is required")
	}
	return &FeishuAdapter{
		webhookURL:    webhookURL,
		inboundSecret: strings.TrimSpace(inboundSecret),
		httpClient:    http.DefaultClient,
	}, nil
}

func (f *FeishuAdapter) Name() string                       { return "feishu" }
func (f *FeishuAdapter) Connect(_ context.Context) error    { return nil }
func (f *FeishuAdapter) Disconnect(_ context.Context) error { return nil }

func (f *FeishuAdapter) Send(ctx context.Context, chatID, content, _ string) (gateway.SendResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}
	payload := map[string]any{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}
	bs, err := json.Marshal(payload)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.webhookURL, bytes.NewReader(bs))
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if f.inboundSecret != "" {
		req.Header.Set("X-Agent-Feishu-Secret", f.inboundSecret)
	}
	client := f.httpClient
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
		return gateway.SendResult{Success: false, Error: fmt.Sprintf("feishu send failed: %s", msg)}, nil
	}
	return gateway.SendResult{Success: true}, nil
}

func (f *FeishuAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("feishu edit is not supported in minimal mode")
}
func (f *FeishuAdapter) SendTyping(_ context.Context, _ string) error { return nil }
func (f *FeishuAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	f.handler = handler
}

func (f *FeishuAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if f.inboundSecret != "" && strings.TrimSpace(req.Header.Get("X-Agent-Feishu-Secret")) != f.inboundSecret {
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
		Event struct {
			Message struct {
				MessageID string `json:"message_id"`
				ChatID    string `json:"chat_id"`
				Content   string `json:"content"`
			} `json:"message"`
			Sender struct {
				SenderID struct {
					OpenID string `json:"open_id"`
				} `json:"sender_id"`
			} `json:"sender"`
		} `json:"event"`
	}
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	chatID := strings.TrimSpace(payload.Event.Message.ChatID)
	userID := strings.TrimSpace(payload.Event.Sender.SenderID.OpenID)
	raw := strings.TrimSpace(payload.Event.Message.Content)
	if chatID == "" || userID == "" || raw == "" {
		http.Error(resp, "invalid payload", http.StatusBadRequest)
		return
	}
	text := raw
	var msg map[string]any
	if json.Unmarshal([]byte(raw), &msg) == nil {
		if v, ok := msg["text"].(string); ok && strings.TrimSpace(v) != "" {
			text = strings.TrimSpace(v)
		}
	}
	if f.handler != nil {
		f.handler(req.Context(), gateway.MessageEvent{
			Text:      text,
			MessageID: strings.TrimSpace(payload.Event.Message.MessageID),
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
