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

type WhatsAppAdapter struct {
	accessToken   string
	phoneNumberID string
	baseURL       string
	apiVersion    string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewWhatsAppAdapter(accessToken, phoneNumberID string) (*WhatsAppAdapter, error) {
	accessToken = strings.TrimSpace(accessToken)
	phoneNumberID = strings.TrimSpace(phoneNumberID)
	if accessToken == "" {
		return nil, errors.New("whatsapp access token is required")
	}
	if phoneNumberID == "" {
		return nil, errors.New("whatsapp phone number id is required")
	}
	return &WhatsAppAdapter{
		accessToken:   accessToken,
		phoneNumberID: phoneNumberID,
		baseURL:       "https://graph.facebook.com",
		apiVersion:    "v21.0",
		httpClient:    http.DefaultClient,
	}, nil
}

func (w *WhatsAppAdapter) Name() string { return "whatsapp" }

func (w *WhatsAppAdapter) Connect(_ context.Context) error { return nil }

func (w *WhatsAppAdapter) Disconnect(_ context.Context) error { return nil }

func (w *WhatsAppAdapter) Send(ctx context.Context, chatID, content, replyTo string) (gateway.SendResult, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return gateway.SendResult{Success: false, Error: "chat_id required"}, nil
	}
	bodyText := strings.TrimSpace(content)
	if bodyText == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}

	reqBody := map[string]any{
		"messaging_product": "whatsapp",
		"to":                chatID,
		"type":              "text",
		"text": map[string]any{
			"preview_url": false,
			"body":        bodyText,
		},
	}
	if rt := strings.TrimSpace(replyTo); rt != "" {
		reqBody["context"] = map[string]any{"message_id": rt}
	}
	bs, err := json.Marshal(reqBody)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	endpoint := strings.TrimRight(w.baseURL, "/") + "/" + strings.Trim(strings.TrimSpace(w.apiVersion), "/") + "/" + w.phoneNumberID + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bs))
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req.Header.Set("Authorization", "Bearer "+w.accessToken)
	req.Header.Set("Content-Type", "application/json")

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
		return gateway.SendResult{Success: false, Error: fmt.Sprintf("whatsapp send failed: %s", msg)}, nil
	}
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return gateway.SendResult{Success: true}, nil
	}
	messageID := ""
	if len(out.Messages) > 0 {
		messageID = strings.TrimSpace(out.Messages[0].ID)
	}
	return gateway.SendResult{Success: true, MessageID: messageID}, nil
}

func (w *WhatsAppAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("whatsapp does not support edit message")
}

func (w *WhatsAppAdapter) SendTyping(_ context.Context, _ string) error { return nil }

func (w *WhatsAppAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	w.handler = handler
}
