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

type WebhookAdapter struct {
	outboundURL   string
	inboundSecret string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewWebhookAdapter(outboundURL, inboundSecret string) (*WebhookAdapter, error) {
	outboundURL = strings.TrimSpace(outboundURL)
	if outboundURL == "" {
		return nil, errors.New("webhook outbound url is required")
	}
	return &WebhookAdapter{
		outboundURL:   outboundURL,
		inboundSecret: strings.TrimSpace(inboundSecret),
		httpClient:    http.DefaultClient,
	}, nil
}

func (w *WebhookAdapter) Name() string { return "webhook" }

func (w *WebhookAdapter) Connect(_ context.Context) error { return nil }

func (w *WebhookAdapter) Disconnect(_ context.Context) error { return nil }

func (w *WebhookAdapter) Send(ctx context.Context, chatID, content, replyTo string) (gateway.SendResult, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return gateway.SendResult{Success: false, Error: "chat_id required"}, nil
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}
	body := map[string]any{
		"platform": "webhook",
		"chat_id":  chatID,
		"content":  content,
	}
	if rt := strings.TrimSpace(replyTo); rt != "" {
		body["reply_to"] = rt
	}
	bs, err := json.Marshal(body)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.outboundURL, bytes.NewReader(bs))
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.inboundSecret != "" {
		req.Header.Set("X-Agent-Webhook-Secret", w.inboundSecret)
	}
	client := w.httpClient
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
		return gateway.SendResult{Success: false, Error: fmt.Sprintf("webhook send failed: %s", msg)}, nil
	}
	messageID := ""
	var out struct {
		MessageID string `json:"message_id"`
	}
	if json.Unmarshal(respBody, &out) == nil {
		messageID = strings.TrimSpace(out.MessageID)
	}
	return gateway.SendResult{Success: true, MessageID: messageID}, nil
}

func (w *WebhookAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("webhook does not support edit message")
}

func (w *WebhookAdapter) SendTyping(_ context.Context, _ string) error { return nil }

func (w *WebhookAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	w.handler = handler
}

func (w *WebhookAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if w.inboundSecret != "" {
		if strings.TrimSpace(req.Header.Get("X-Agent-Webhook-Secret")) != w.inboundSecret {
			http.Error(resp, "forbidden", http.StatusForbidden)
			return
		}
	}
	defer req.Body.Close()
	bs, err := io.ReadAll(io.LimitReader(req.Body, 2<<20))
	if err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	var payload struct {
		Text      string   `json:"text"`
		MessageID string   `json:"message_id"`
		ChatID    string   `json:"chat_id"`
		ChatType  string   `json:"chat_type"`
		UserID    string   `json:"user_id"`
		UserName  string   `json:"user_name"`
		MediaURLs []string `json:"media_urls"`
		ReplyToID string   `json:"reply_to_id"`
		ThreadID  string   `json:"thread_id"`
		IsCommand *bool    `json:"is_command"`
	}
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	chatID := strings.TrimSpace(payload.ChatID)
	if chatID == "" {
		http.Error(resp, "chat_id required", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(payload.Text)
	if text == "" && len(payload.MediaURLs) == 0 {
		http.Error(resp, "text or media_urls required", http.StatusBadRequest)
		return
	}
	chatType := strings.TrimSpace(payload.ChatType)
	if chatType == "" {
		chatType = "dm"
	}
	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		userID = "webhook"
	}
	userName := strings.TrimSpace(payload.UserName)
	if userName == "" {
		userName = userID
	}
	isCommand := strings.HasPrefix(text, "/")
	if payload.IsCommand != nil {
		isCommand = *payload.IsCommand
	}
	if w.handler != nil {
		w.handler(req.Context(), gateway.MessageEvent{
			Text:      text,
			MessageID: strings.TrimSpace(payload.MessageID),
			ChatID:    chatID,
			ChatType:  chatType,
			UserID:    userID,
			UserName:  userName,
			MediaURLs: payload.MediaURLs,
			ReplyToID: strings.TrimSpace(payload.ReplyToID),
			ThreadID:  strings.TrimSpace(payload.ThreadID),
			IsCommand: isCommand,
		})
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(`{"ok":true}`))
}
