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

type SignalAdapter struct {
	baseURL       string
	account       string
	inboundSecret string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewSignalAdapter(baseURL, account, inboundSecret string) (*SignalAdapter, error) {
	baseURL = strings.TrimSpace(baseURL)
	account = strings.TrimSpace(account)
	if baseURL == "" {
		return nil, errors.New("signal base url is required")
	}
	if account == "" {
		return nil, errors.New("signal account is required")
	}
	return &SignalAdapter{
		baseURL:       strings.TrimRight(baseURL, "/"),
		account:       account,
		inboundSecret: strings.TrimSpace(inboundSecret),
		httpClient:    http.DefaultClient,
	}, nil
}

func (s *SignalAdapter) Name() string { return "signal" }

func (s *SignalAdapter) Connect(_ context.Context) error { return nil }

func (s *SignalAdapter) Disconnect(_ context.Context) error { return nil }

func (s *SignalAdapter) Send(ctx context.Context, chatID, content, _ string) (gateway.SendResult, error) {
	chatID = strings.TrimSpace(chatID)
	content = strings.TrimSpace(content)
	if chatID == "" {
		return gateway.SendResult{Success: false, Error: "chat_id required"}, nil
	}
	if content == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}
	reqBody := map[string]any{
		"message":    content,
		"number":     s.account,
		"recipients": []string{chatID},
	}
	bs, err := json.Marshal(reqBody)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v2/send", bytes.NewReader(bs))
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.inboundSecret != "" {
		req.Header.Set("X-Agent-Signal-Secret", s.inboundSecret)
	}
	client := s.httpClient
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
		return gateway.SendResult{Success: false, Error: fmt.Sprintf("signal send failed: %s", msg)}, nil
	}
	return gateway.SendResult{Success: true}, nil
}

func (s *SignalAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("signal does not support edit message")
}

func (s *SignalAdapter) SendTyping(_ context.Context, _ string) error { return nil }

func (s *SignalAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	s.handler = handler
}

func (s *SignalAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.inboundSecret != "" && strings.TrimSpace(req.Header.Get("X-Agent-Signal-Secret")) != s.inboundSecret {
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
		Envelope struct {
			Source string `json:"source"`
		} `json:"envelope"`
		DataMessage struct {
			Message string `json:"message"`
		} `json:"dataMessage"`
	}
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	userID := strings.TrimSpace(payload.Envelope.Source)
	text := strings.TrimSpace(payload.DataMessage.Message)
	if userID == "" || text == "" {
		http.Error(resp, "invalid payload", http.StatusBadRequest)
		return
	}
	if s.handler != nil {
		s.handler(req.Context(), gateway.MessageEvent{
			Text:      text,
			ChatID:    userID,
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
