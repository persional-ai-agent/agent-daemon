package platforms

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/platform"
	"github.com/dingjingmaster/agent-daemon/internal/yuanbao"
)

type YuanbaoAdapter struct {
	cfg yuanbao.EnvConfig

	clientMu sync.Mutex
	client   *yuanbao.Client

	onMessage platform.MessageHandler
}

func NewYuanbaoAdapterFromEnv() (*YuanbaoAdapter, error) {
	cfg, err := yuanbao.ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return &YuanbaoAdapter{cfg: cfg}, nil
}

func (a *YuanbaoAdapter) Name() string { return "yuanbao" }

func (a *YuanbaoAdapter) Connect(ctx context.Context) error {
	a.clientMu.Lock()
	defer a.clientMu.Unlock()
	if a.client != nil {
		return nil
	}

	token := a.cfg.Token
	botID := a.cfg.BotID
	if token == "" {
		st, err := yuanbao.FetchSignToken(ctx, a.cfg.AppID, a.cfg.AppSecret, a.cfg.APIDomain)
		if err != nil {
			return err
		}
		token = st.Token
		if botID == "" {
			botID = st.BotID
		}
	}
	if strings.TrimSpace(botID) == "" {
		return errors.New("YUANBAO_BOT_ID required (or sign-token must return bot_id)")
	}

	c, err := yuanbao.NewClient(yuanbao.ClientOptions{
		WSURL:            a.cfg.WSURL,
		Token:            token,
		BotID:            botID,
		RouteEnv:         a.cfg.RouteEnv,
		AppVersion:       "agent-daemon",
		OperationSystem:  "linux",
		BotVersion:       "agent-daemon",
		OnPush: func(msg yuanbao.ConnMsg) {
			a.handlePush(botID, msg)
		},
	})
	if err != nil {
		return err
	}
	if err := c.Connect(ctx); err != nil {
		return err
	}
	a.client = c
	return nil
}

func (a *YuanbaoAdapter) Disconnect(_ context.Context) error {
	a.clientMu.Lock()
	c := a.client
	a.client = nil
	a.clientMu.Unlock()
	if c != nil {
		return c.Close()
	}
	return nil
}

func (a *YuanbaoAdapter) Send(ctx context.Context, chatID, content, _ string) (platform.SendResult, error) {
	a.clientMu.Lock()
	c := a.client
	a.clientMu.Unlock()
	if c == nil {
		return platform.SendResult{Success: false, Error: "yuanbao adapter not connected"}, nil
	}

	kind, id := parseYuanbaoChatID(chatID)
	if id == "" {
		return platform.SendResult{Success: false, Error: "chat_id required"}, nil
	}

	if kind == "group" {
		_, err := c.SendGroupText(ctx, id, content, "")
		if err != nil {
			return platform.SendResult{Success: false, Error: err.Error()}, nil
		}
		return platform.SendResult{Success: true, MessageID: ""}, nil
	}
	_, err := c.SendC2C(ctx, id, content)
	if err != nil {
		return platform.SendResult{Success: false, Error: err.Error()}, nil
	}
	return platform.SendResult{Success: true, MessageID: ""}, nil
}

func (a *YuanbaoAdapter) SendMedia(ctx context.Context, chatID, path, caption, replyTo string) (platform.SendResult, error) {
	a.clientMu.Lock()
	c := a.client
	cfg := a.cfg
	a.clientMu.Unlock()
	if c == nil {
		return platform.SendResult{Success: false, Error: "yuanbao adapter not connected"}, nil
	}

	kind, id := parseYuanbaoChatID(chatID)
	if id == "" {
		return platform.SendResult{Success: false, Error: "chat_id required"}, nil
	}
	filename := filepath.Base(strings.TrimSpace(path))
	if filename == "" {
		filename = "file"
	}
	info, err := yuanbao.GetUploadInfo(ctx, cfg, filename, "")
	if err != nil {
		return platform.SendResult{Success: false, Error: err.Error()}, nil
	}
	up, err := yuanbao.UploadToCOS(ctx, info, path)
	if err != nil {
		return platform.SendResult{Success: false, Error: err.Error()}, nil
	}

	// Best-effort: treat images as TIMImageElem, otherwise TIMFileElem.
	isImage := strings.HasPrefix(strings.ToLower(up.MimeType), "image/")
	if kind == "group" {
		if strings.TrimSpace(caption) != "" {
			// Send caption as a separate text message before the media (Yuanbao media elems may not support captions).
			_, _ = c.SendGroupText(ctx, id, caption, replyTo)
		}
		if isImage {
			imgFmt := yuanbao.ImageFormatFromMime(up.MimeType)
			_, err := c.SendGroupImage(ctx, id, replyTo, up.UUID, imgFmt, yuanbao.ImageInfo{
				Type:   1,
				Size:   uint32(up.Size),
				Width:  uint32(up.Width),
				Height: uint32(up.Height),
				URL:    up.URL,
			})
			if err != nil {
				return platform.SendResult{Success: false, Error: err.Error()}, nil
			}
			return platform.SendResult{Success: true, MessageID: ""}, nil
		}
		_, err := c.SendGroupFile(ctx, id, replyTo, up.UUID, up.FileName, uint32(up.Size), up.URL)
		if err != nil {
			return platform.SendResult{Success: false, Error: err.Error()}, nil
		}
		return platform.SendResult{Success: true, MessageID: ""}, nil
	}

	if strings.TrimSpace(caption) != "" {
		_, _ = c.SendC2C(ctx, id, caption)
	}
	if isImage {
		imgFmt := yuanbao.ImageFormatFromMime(up.MimeType)
		_, err := c.SendC2CImage(ctx, id, up.UUID, imgFmt, yuanbao.ImageInfo{
			Type:   1,
			Size:   uint32(up.Size),
			Width:  uint32(up.Width),
			Height: uint32(up.Height),
			URL:    up.URL,
		})
		if err != nil {
			return platform.SendResult{Success: false, Error: err.Error()}, nil
		}
		return platform.SendResult{Success: true, MessageID: ""}, nil
	}
	_, err = c.SendC2CFile(ctx, id, up.UUID, up.FileName, uint32(up.Size), up.URL)
	if err != nil {
		return platform.SendResult{Success: false, Error: err.Error()}, nil
	}
	return platform.SendResult{Success: true, MessageID: ""}, nil
}

func (a *YuanbaoAdapter) EditMessage(_ context.Context, _ string, _ string, _ string) error {
	// Yuanbao does not provide a stable message edit API for bots (minimal adapter: no-op).
	return nil
}

func (a *YuanbaoAdapter) SendTyping(_ context.Context, _ string) error { return nil }

func (a *YuanbaoAdapter) OnMessage(_ context.Context, handler platform.MessageHandler) {
	// Best-effort inbound events via WS push decode (TIMTextElem only).
	a.onMessage = handler
}

func (a *YuanbaoAdapter) handlePush(botID string, msg yuanbao.ConnMsg) {
	if a.onMessage == nil {
		return
	}
	// Best-effort: attempt to parse inbound message push.
	push, err := yuanbao.DecodeInboundPush(msg.Data)
	if err != nil {
		return
	}
	// Ignore empty events.
	if strings.TrimSpace(push.FromAccount) == "" || strings.TrimSpace(push.MsgID) == "" {
		return
	}
	// Ignore bot echo.
	if push.FromAccount == botID {
		return
	}
	chatType := "direct"
	chatID := "direct:" + push.FromAccount
	if strings.TrimSpace(push.GroupCode) != "" {
		chatType = "group"
		chatID = "group:" + push.GroupCode
	}
	text := renderInboundText(push.MsgBody)
	if strings.TrimSpace(text) == "" {
		return
	}
	a.onMessage(context.Background(), platform.MessageEvent{
		Text:      text,
		MessageID: push.MsgID,
		ChatID:    chatID,
		ChatType:  chatType,
		UserID:    push.FromAccount,
		UserName:  push.SenderNickname,
		IsCommand: strings.HasPrefix(strings.TrimSpace(text), "/"),
	})
}

func renderInboundText(body []yuanbao.MsgBodyElement) string {
	var sb strings.Builder
	for _, el := range body {
		if el.MsgType == "TIMTextElem" {
			if s, ok := el.MsgContent["text"].(string); ok {
				sb.WriteString(s)
			}
		}
	}
	return sb.String()
}

// Extra helpers for tools (not part of platform.Adapter)

func (a *YuanbaoAdapter) SendSticker(ctx context.Context, chatID string, stickerJSON string, replyTo string) (map[string]any, error) {
	a.clientMu.Lock()
	c := a.client
	a.clientMu.Unlock()
	if c == nil {
		return nil, errors.New("yuanbao adapter not connected")
	}
	kind, id := parseYuanbaoChatID(chatID)
	if id == "" {
		return nil, errors.New("chat_id required")
	}
	if kind == "group" {
		return c.SendGroupSticker(ctx, id, stickerJSON, replyTo)
	}
	return c.SendC2CSticker(ctx, id, stickerJSON)
}

func (a *YuanbaoAdapter) QueryGroupInfo(ctx context.Context, groupCode string) (map[string]any, error) {
	a.clientMu.Lock()
	c := a.client
	a.clientMu.Unlock()
	if c == nil {
		return nil, errors.New("yuanbao adapter not connected")
	}
	return c.QueryGroupInfo(ctx, groupCode)
}

func (a *YuanbaoAdapter) QueryGroupMembers(ctx context.Context, groupCode string, offset, limit uint32) (map[string]any, error) {
	a.clientMu.Lock()
	c := a.client
	a.clientMu.Unlock()
	if c == nil {
		return nil, errors.New("yuanbao adapter not connected")
	}
	return c.GetGroupMemberList(ctx, groupCode, offset, limit)
}

func parseYuanbaoChatID(chatID string) (kind string, id string) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", ""
	}
	if strings.HasPrefix(chatID, "direct:") {
		return "direct", strings.TrimPrefix(chatID, "direct:")
	}
	if strings.HasPrefix(chatID, "group:") {
		return "group", strings.TrimPrefix(chatID, "group:")
	}
	// Default: treat bare id as direct message target.
	return "direct", chatID
}

func BuildStickerJSON(stickerID, packageID, name string, width, height int, formats string) (string, error) {
	if formats == "" {
		formats = "png"
	}
	payload := map[string]any{
		"sticker_id": stickerID,
		"package_id": packageID,
		"width":      width,
		"height":     height,
		"formats":    formats,
		"name":       name,
	}
	bs, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

var _ = time.Second
