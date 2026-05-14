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
	verifyToken   string
	baseURL       string
	apiVersion    string
	httpClient    *http.Client
	handler       gateway.MessageHandler
}

func NewWhatsAppAdapter(accessToken, phoneNumberID, verifyToken string) (*WhatsAppAdapter, error) {
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
		verifyToken:   strings.TrimSpace(verifyToken),
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

func (w *WhatsAppAdapter) HandleWebhook(resp http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		mode := strings.TrimSpace(req.URL.Query().Get("hub.mode"))
		verifyToken := strings.TrimSpace(req.URL.Query().Get("hub.verify_token"))
		challenge := strings.TrimSpace(req.URL.Query().Get("hub.challenge"))
		if mode != "subscribe" || challenge == "" || strings.TrimSpace(w.verifyToken) == "" || verifyToken != strings.TrimSpace(w.verifyToken) {
			http.Error(resp, "forbidden", http.StatusForbidden)
			return
		}
		resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte(challenge))
		return
	case http.MethodPost:
		// continue below
	default:
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer req.Body.Close()
	bs, err := io.ReadAll(io.LimitReader(req.Body, 2<<20))
	if err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	var payload whatsAppWebhookPayload
	if err := json.Unmarshal(bs, &payload); err != nil {
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}
	if w.handler != nil {
		for _, entry := range payload.Entry {
			for _, change := range entry.Changes {
				for _, msg := range change.Value.Messages {
					text := strings.TrimSpace(primaryWebhookText(msg))
					userID := strings.TrimSpace(msg.From)
					if userID == "" {
						continue
					}
					mediaURLs := w.resolveWebhookMediaURLs(req.Context(), msg)
					if text == "" && len(mediaURLs) == 0 {
						continue
					}
					replyTo := strings.TrimSpace(msg.Context.ID)
					w.handler(req.Context(), gateway.MessageEvent{
						Text:      text,
						MessageID: strings.TrimSpace(msg.ID),
						ChatID:    userID,
						ChatType:  "dm",
						UserID:    userID,
						UserName:  userID,
						MediaURLs: mediaURLs,
						ReplyToID: replyTo,
						IsCommand: strings.HasPrefix(text, "/"),
					})
				}
			}
		}
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	_, _ = resp.Write([]byte(`{"ok":true}`))
}

type whatsAppWebhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
					Image struct {
						ID      string `json:"id"`
						Caption string `json:"caption"`
					} `json:"image"`
					Video struct {
						ID      string `json:"id"`
						Caption string `json:"caption"`
					} `json:"video"`
					Document struct {
						ID      string `json:"id"`
						Caption string `json:"caption"`
					} `json:"document"`
					Audio struct {
						ID string `json:"id"`
					} `json:"audio"`
					Sticker struct {
						ID string `json:"id"`
					} `json:"sticker"`
					Context struct {
						ID string `json:"id"`
					} `json:"context"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func primaryWebhookText(msg struct {
	From string `json:"from"`
	ID   string `json:"id"`
	Type string `json:"type"`
	Text struct {
		Body string `json:"body"`
	} `json:"text"`
	Image struct {
		ID      string `json:"id"`
		Caption string `json:"caption"`
	} `json:"image"`
	Video struct {
		ID      string `json:"id"`
		Caption string `json:"caption"`
	} `json:"video"`
	Document struct {
		ID      string `json:"id"`
		Caption string `json:"caption"`
	} `json:"document"`
	Audio struct {
		ID string `json:"id"`
	} `json:"audio"`
	Sticker struct {
		ID string `json:"id"`
	} `json:"sticker"`
	Context struct {
		ID string `json:"id"`
	} `json:"context"`
}) string {
	if text := strings.TrimSpace(msg.Text.Body); text != "" {
		return text
	}
	if caption := strings.TrimSpace(msg.Image.Caption); caption != "" {
		return caption
	}
	if caption := strings.TrimSpace(msg.Video.Caption); caption != "" {
		return caption
	}
	if caption := strings.TrimSpace(msg.Document.Caption); caption != "" {
		return caption
	}
	if strings.TrimSpace(msg.Type) != "" {
		return "[whatsapp_" + strings.ToLower(strings.TrimSpace(msg.Type)) + "_message]"
	}
	return ""
}

func (w *WhatsAppAdapter) resolveWebhookMediaURLs(ctx context.Context, msg struct {
	From string `json:"from"`
	ID   string `json:"id"`
	Type string `json:"type"`
	Text struct {
		Body string `json:"body"`
	} `json:"text"`
	Image struct {
		ID      string `json:"id"`
		Caption string `json:"caption"`
	} `json:"image"`
	Video struct {
		ID      string `json:"id"`
		Caption string `json:"caption"`
	} `json:"video"`
	Document struct {
		ID      string `json:"id"`
		Caption string `json:"caption"`
	} `json:"document"`
	Audio struct {
		ID string `json:"id"`
	} `json:"audio"`
	Sticker struct {
		ID string `json:"id"`
	} `json:"sticker"`
	Context struct {
		ID string `json:"id"`
	} `json:"context"`
}) []string {
	mediaIDs := []string{
		strings.TrimSpace(msg.Image.ID),
		strings.TrimSpace(msg.Video.ID),
		strings.TrimSpace(msg.Document.ID),
		strings.TrimSpace(msg.Audio.ID),
		strings.TrimSpace(msg.Sticker.ID),
	}
	out := make([]string, 0, len(mediaIDs))
	for _, mediaID := range mediaIDs {
		if mediaID == "" {
			continue
		}
		mediaURL, err := w.fetchMediaURL(ctx, mediaID)
		if err != nil || strings.TrimSpace(mediaURL) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(mediaURL))
	}
	return out
}

func (w *WhatsAppAdapter) fetchMediaURL(ctx context.Context, mediaID string) (string, error) {
	endpoint := strings.TrimRight(w.baseURL, "/") + "/" + strings.Trim(strings.TrimSpace(w.apiVersion), "/") + "/" + strings.TrimSpace(mediaID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	client := w.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bs, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("whatsapp media metadata failed: %s", strings.TrimSpace(string(bs)))
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(bs, &out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.URL), nil
}
