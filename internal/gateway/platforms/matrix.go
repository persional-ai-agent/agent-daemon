package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

type MatrixAdapter struct {
	baseURL       string
	accessToken   string
	inboundSecret string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewMatrixAdapter(baseURL, accessToken, inboundSecret string) (*MatrixAdapter, error) {
	baseURL = strings.TrimSpace(baseURL)
	accessToken = strings.TrimSpace(accessToken)
	if baseURL == "" {
		return nil, errors.New("matrix base url is required")
	}
	if accessToken == "" {
		return nil, errors.New("matrix access token is required")
	}
	return &MatrixAdapter{
		baseURL:       strings.TrimRight(baseURL, "/"),
		accessToken:   accessToken,
		inboundSecret: strings.TrimSpace(inboundSecret),
		httpClient:    http.DefaultClient,
	}, nil
}

func (m *MatrixAdapter) Name() string { return "matrix" }

func (m *MatrixAdapter) Connect(_ context.Context) error { return nil }

func (m *MatrixAdapter) Disconnect(_ context.Context) error { return nil }

func (m *MatrixAdapter) Send(ctx context.Context, chatID, content, _ string) (gateway.SendResult, error) {
	roomID := strings.TrimSpace(chatID)
	content = strings.TrimSpace(content)
	if roomID == "" {
		return gateway.SendResult{Success: false, Error: "chat_id required (matrix room id)"}, nil
	}
	if content == "" {
		return gateway.SendResult{Success: false, Error: "content required"}, nil
	}
	txnID := fmt.Sprintf("agent-%d", time.Now().UnixNano())
	payload := map[string]any{
		"msgtype": "m.text",
		"body":    content,
	}
	bs, err := json.Marshal(payload)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	endpoint := m.baseURL + "/_matrix/client/v3/rooms/" + url.PathEscape(roomID) + "/send/m.room.message/" + url.PathEscape(txnID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(bs))
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.accessToken)
	if m.inboundSecret != "" {
		req.Header.Set("X-Agent-Matrix-Secret", m.inboundSecret)
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
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return gateway.SendResult{Success: false, Error: fmt.Sprintf("matrix send failed: %s", msg)}, nil
	}
	var out struct {
		EventID string `json:"event_id"`
	}
	_ = json.Unmarshal(respBody, &out)
	return gateway.SendResult{Success: true, MessageID: strings.TrimSpace(out.EventID)}, nil
}

func (m *MatrixAdapter) EditMessage(_ context.Context, _, _, _ string) error {
	return errors.New("matrix edit message is not supported in minimal mode")
}

func (m *MatrixAdapter) SendTyping(_ context.Context, _ string) error { return nil }

func (m *MatrixAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	m.handler = handler
}

func (m *MatrixAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.inboundSecret != "" && strings.TrimSpace(req.Header.Get("X-Agent-Matrix-Secret")) != m.inboundSecret {
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
		Sender  string `json:"sender"`
		RoomID  string `json:"room_id"`
		EventID string `json:"event_id"`
		Content struct {
			Body string `json:"body"`
		} `json:"content"`
	}
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(payload.Content.Body)
	userID := strings.TrimSpace(payload.Sender)
	roomID := strings.TrimSpace(payload.RoomID)
	if text == "" || userID == "" || roomID == "" {
		http.Error(resp, "invalid payload", http.StatusBadRequest)
		return
	}
	if m.handler != nil {
		m.handler(req.Context(), gateway.MessageEvent{
			Text:      text,
			MessageID: strings.TrimSpace(payload.EventID),
			ChatID:    roomID,
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
