package platforms

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

type EmailAdapter struct {
	smtpHost      string
	smtpPort      string
	smtpUsername  string
	smtpPassword  string
	fromAddress   string
	inboundSecret string
	handler       gateway.MessageHandler
	sendMail      func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

func NewEmailAdapter(smtpHost, smtpPort, smtpUsername, smtpPassword, fromAddress, inboundSecret string) (*EmailAdapter, error) {
	smtpHost = strings.TrimSpace(smtpHost)
	smtpPort = strings.TrimSpace(smtpPort)
	smtpUsername = strings.TrimSpace(smtpUsername)
	smtpPassword = strings.TrimSpace(smtpPassword)
	fromAddress = strings.TrimSpace(fromAddress)
	if smtpHost == "" {
		return nil, errors.New("email smtp host is required")
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if smtpUsername == "" || smtpPassword == "" {
		return nil, errors.New("email smtp username/password are required")
	}
	if fromAddress == "" {
		return nil, errors.New("email from address is required")
	}
	return &EmailAdapter{
		smtpHost:      smtpHost,
		smtpPort:      smtpPort,
		smtpUsername:  smtpUsername,
		smtpPassword:  smtpPassword,
		fromAddress:   fromAddress,
		inboundSecret: strings.TrimSpace(inboundSecret),
		sendMail:      smtp.SendMail,
	}, nil
}

func (e *EmailAdapter) Name() string { return "email" }

func (e *EmailAdapter) Connect(_ context.Context) error { return nil }

func (e *EmailAdapter) Disconnect(_ context.Context) error { return nil }

func (e *EmailAdapter) Send(_ context.Context, chatID, content, _ string) (gateway.SendResult, error) {
	to := strings.TrimSpace(chatID)
	content = strings.TrimSpace(content)
	if to == "" {
		return gateway.SendResult{Success: false, Error: "chat_id required (recipient email)"}, nil
	}
	if content == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}
	lines := strings.Split(content, "\n")
	subject := "Agent message"
	body := content
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		subject = strings.TrimSpace(lines[0])
		if len(lines) > 1 {
			body = strings.Join(lines[1:], "\n")
		}
	}
	msg := strings.Join([]string{
		"From: " + e.fromAddress,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)
	addr := e.smtpHost + ":" + e.smtpPort
	if err := e.sendMail(addr, auth, e.fromAddress, []string{to}, []byte(msg)); err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, nil
	}
	return gateway.SendResult{Success: true}, nil
}

func (e *EmailAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("email does not support edit message")
}

func (e *EmailAdapter) SendTyping(_ context.Context, _ string) error { return nil }

func (e *EmailAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	e.handler = handler
}

func (e *EmailAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if e.inboundSecret != "" && strings.TrimSpace(req.Header.Get("X-Agent-Email-Secret")) != e.inboundSecret {
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
		From    string `json:"from"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Text    string `json:"text"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	from := strings.TrimSpace(payload.From)
	text := strings.TrimSpace(payload.Text)
	if from == "" || text == "" {
		http.Error(resp, "invalid payload", http.StatusBadRequest)
		return
	}
	messageText := text
	if subject := strings.TrimSpace(payload.Subject); subject != "" {
		messageText = subject + "\n" + text
	}
	if e.handler != nil {
		e.handler(req.Context(), gateway.MessageEvent{
			Text:      messageText,
			MessageID: strings.TrimSpace(payload.ID),
			ChatID:    from,
			ChatType:  "dm",
			UserID:    from,
			UserName:  from,
			IsCommand: strings.HasPrefix(strings.TrimSpace(text), "/"),
		})
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(`{"ok":true}`))
}
