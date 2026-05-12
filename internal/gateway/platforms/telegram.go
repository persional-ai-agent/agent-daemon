package platforms

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
	"github.com/dingjingmaster/agent-daemon/internal/platform"
)

type TelegramAdapter struct {
	bot     *tgbotapi.BotAPI
	handler gateway.MessageHandler
}

func NewTelegramAdapter(token string) (*TelegramAdapter, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("telegram: create bot: %w", err)
	}
	return &TelegramAdapter{bot: bot}, nil
}

func (t *TelegramAdapter) Name() string { return "telegram" }

func (t *TelegramAdapter) Connect(ctx context.Context) error {
	log.Printf("[gateway:telegram] connected as @%s", t.bot.Self.UserName)
	if err := t.registerCommands(); err != nil {
		log.Printf("[gateway:telegram] register commands: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := t.bot.GetUpdatesChan(u)

	go func() {
		defer log.Printf("[gateway:telegram] update loop stopped")
		for {
			select {
			case <-ctx.Done():
				return
			case update, ok := <-updates:
				if !ok {
					return
				}
				if update.CallbackQuery != nil && t.handler != nil && update.CallbackQuery.Message != nil && update.CallbackQuery.From != nil {
					cb := update.CallbackQuery
					chatID := fmt.Sprintf("%d", cb.Message.Chat.ID)
					chatType := "dm"
					if cb.Message.Chat.IsGroup() || cb.Message.Chat.IsSuperGroup() {
						chatType = "group"
					}
					event := gateway.MessageEvent{
						Text:      strings.TrimSpace(cb.Data),
						MessageID: fmt.Sprintf("%d", cb.Message.MessageID),
						ChatID:    chatID,
						ChatType:  chatType,
						UserID:    fmt.Sprintf("%d", cb.From.ID),
						UserName:  cb.From.UserName,
						IsCommand: strings.HasPrefix(strings.TrimSpace(cb.Data), "/"),
					}
					if _, err := t.bot.Request(tgbotapi.NewCallback(cb.ID, "")); err != nil {
						log.Printf("[gateway:telegram] answer callback: %v", err)
					}
					t.handler(ctx, event)
					continue
				}
				if update.Message == nil || t.handler == nil {
					continue
				}
				msg := update.Message
				chatID := fmt.Sprintf("%d", msg.Chat.ID)
				chatType := "dm"
				if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
					chatType = "group"
				}
				event := gateway.MessageEvent{
					Text:      msg.Text,
					MessageID: fmt.Sprintf("%d", msg.MessageID),
					ChatID:    chatID,
					ChatType:  chatType,
					UserID:    fmt.Sprintf("%d", msg.From.ID),
					UserName:  msg.From.UserName,
					IsCommand: msg.IsCommand(),
				}
				if msg.ReplyToMessage != nil {
					event.ReplyToID = fmt.Sprintf("%d", msg.ReplyToMessage.MessageID)
				}
				t.handler(ctx, event)
			}
		}
	}()
	return nil
}

func (t *TelegramAdapter) Disconnect(_ context.Context) error {
	t.bot.StopReceivingUpdates()
	return nil
}

func (t *TelegramAdapter) Send(_ context.Context, chatID, content, replyTo string) (gateway.SendResult, error) {
	return t.SendText(context.Background(), chatID, content, replyTo, nil)
}

func (t *TelegramAdapter) SendText(_ context.Context, chatID, content, replyTo string, meta map[string]any) (gateway.SendResult, error) {
	chatIDInt, err := parseChatID(chatID)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	msg := tgbotapi.NewMessage(chatIDInt, content)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if markup := approvalKeyboard(meta); markup != nil {
		msg.ReplyMarkup = markup
	}
	if replyTo != "" {
		if id, err2 := parseChatID(replyTo); err2 == nil {
			msg.ReplyToMessageID = int(id)
		}
	}
	sent, err := t.bot.Send(msg)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	return gateway.SendResult{Success: true, MessageID: fmt.Sprintf("%d", sent.MessageID)}, nil
}

func (t *TelegramAdapter) EditMessage(_ context.Context, chatID, messageID, content string) error {
	chatIDInt, err := parseChatID(chatID)
	if err != nil {
		return err
	}
	msgIDInt, err := parseChatID(messageID)
	if err != nil {
		return err
	}
	edit := tgbotapi.NewEditMessageText(chatIDInt, int(msgIDInt), content)
	edit.ParseMode = tgbotapi.ModeMarkdown
	_, err = t.bot.Send(edit)
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	return err
}

func (t *TelegramAdapter) SendTyping(_ context.Context, chatID string) error {
	chatIDInt, err := parseChatID(chatID)
	if err != nil {
		return err
	}
	action := tgbotapi.NewChatAction(chatIDInt, tgbotapi.ChatTyping)
	_, err = t.bot.Send(action)
	return err
}

func (t *TelegramAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	t.handler = handler
}

func (t *TelegramAdapter) SendMedia(_ context.Context, chatID, path, caption, replyTo string) (platform.SendResult, error) {
	chatIDInt, err := parseChatID(chatID)
	if err != nil {
		return platform.SendResult{Success: false, Error: err.Error()}, err
	}
	if st, err := os.Stat(path); err != nil || st == nil || st.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a file: %s", path)
		}
		return platform.SendResult{Success: false, Error: err.Error()}, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	file := tgbotapi.FilePath(path)
	var msg tgbotapi.Chattable
	switch ext {
	case ".ogg", ".opus":
		v := tgbotapi.NewVoice(chatIDInt, file)
		v.Caption = caption
		if replyTo != "" {
			if id, err2 := parseChatID(replyTo); err2 == nil {
				v.ReplyToMessageID = int(id)
			}
		}
		msg = v
	case ".mp3", ".wav", ".m4a", ".aac":
		a := tgbotapi.NewAudio(chatIDInt, file)
		a.Caption = caption
		if replyTo != "" {
			if id, err2 := parseChatID(replyTo); err2 == nil {
				a.ReplyToMessageID = int(id)
			}
		}
		msg = a
	default:
		d := tgbotapi.NewDocument(chatIDInt, file)
		d.Caption = caption
		if replyTo != "" {
			if id, err2 := parseChatID(replyTo); err2 == nil {
				d.ReplyToMessageID = int(id)
			}
		}
		msg = d
	}
	sent, err := t.bot.Send(msg)
	if err != nil {
		return platform.SendResult{Success: false, Error: err.Error()}, err
	}
	_ = sent
	// Not all Chattable responses include MessageID in a consistent way; keep empty.
	return platform.SendResult{Success: true}, nil
}

func parseChatID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid chat id %q: %w", s, err)
	}
	return id, nil
}

func approvalKeyboard(meta map[string]any) any {
	if len(meta) == 0 {
		return nil
	}
	approvalID, _ := meta["approval_id"].(string)
	approvalID = strings.TrimSpace(approvalID)
	if approvalID == "" {
		return nil
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Approve", "/approve "+approvalID),
			tgbotapi.NewInlineKeyboardButtonData("Deny", "/deny "+approvalID),
		),
	)
}

func (t *TelegramAdapter) registerCommands() error {
	_, err := t.bot.Request(tgbotapi.NewSetMyCommands(TelegramCommands()...))
	return err
}

func TelegramCommands() []tgbotapi.BotCommand {
	return []tgbotapi.BotCommand{
		{Command: "pair", Description: "pair with gateway using a code"},
		{Command: "unpair", Description: "remove current gateway pairing"},
		{Command: "cancel", Description: "cancel the running task"},
		{Command: "queue", Description: "show queued task count"},
		{Command: "status", Description: "show current session status"},
		{Command: "pending", Description: "show latest pending approval"},
		{Command: "approvals", Description: "show active approvals"},
		{Command: "grant", Description: "grant session or pattern approval"},
		{Command: "revoke", Description: "revoke session or pattern approval"},
		{Command: "approve", Description: "approve a pending approval id"},
		{Command: "deny", Description: "deny a pending approval id"},
		{Command: "help", Description: "show supported commands"},
	}
}
