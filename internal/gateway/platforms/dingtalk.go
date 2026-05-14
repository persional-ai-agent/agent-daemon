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

type DingTalkAdapter struct {
	webhookURL    string
	inboundSecret string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewDingTalkAdapter(webhookURL, inboundSecret string) (*DingTalkAdapter, error) {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return nil, errors.New("dingtalk webhook url is required")
	}
	return &DingTalkAdapter{
		webhookURL:    webhookURL,
		inboundSecret: strings.TrimSpace(inboundSecret),
		httpClient:    http.DefaultClient,
	}, nil
}

func (d *DingTalkAdapter) Name() string                       { return "dingtalk" }
func (d *DingTalkAdapter) Connect(_ context.Context) error    { return nil }
func (d *DingTalkAdapter) Disconnect(_ context.Context) error { return nil }

func (d *DingTalkAdapter) Send(ctx context.Context, chatID, content, _ string) (gateway.SendResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}
	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}
	bs, err := json.Marshal(payload)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(bs))
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if d.inboundSecret != "" {
		req.Header.Set("X-Agent-DingTalk-Secret", d.inboundSecret)
	}
	client := d.httpClient
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
		return gateway.SendResult{Success: false, Error: fmt.Sprintf("dingtalk send failed: %s", msg)}, nil
	}
	return gateway.SendResult{Success: true}, nil
}

func (d *DingTalkAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("dingtalk edit is not supported in minimal mode")
}
func (d *DingTalkAdapter) SendTyping(_ context.Context, _ string) error { return nil }
func (d *DingTalkAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	d.handler = handler
}

func (d *DingTalkAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if d.inboundSecret != "" && strings.TrimSpace(req.Header.Get("X-Agent-DingTalk-Secret")) != d.inboundSecret {
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
		SenderID string `json:"senderId"`
		ChatID   string `json:"conversationId"`
		Text     struct {
			Content string `json:"content"`
		} `json:"text"`
		MsgID string `json:"msgId"`
	}
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	userID := strings.TrimSpace(payload.SenderID)
	chatID := strings.TrimSpace(payload.ChatID)
	text := strings.TrimSpace(payload.Text.Content)
	if userID == "" || chatID == "" || text == "" {
		http.Error(resp, "invalid payload", http.StatusBadRequest)
		return
	}
	if d.handler != nil {
		d.handler(req.Context(), gateway.MessageEvent{
			Text:      text,
			MessageID: strings.TrimSpace(payload.MsgID),
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
