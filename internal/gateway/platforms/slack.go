package platforms

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

type SlackAdapter struct {
	client  *slack.Client
	sm      *socketmode.Client
	handler gateway.MessageHandler
}

func NewSlackAdapter(botToken, appToken string) (*SlackAdapter, error) {
	client := slack.New(botToken, slack.OptionAppLevelToken(appToken))
	sm := socketmode.New(client,
		socketmode.OptionDebug(false),
	)
	return &SlackAdapter{client: client, sm: sm}, nil
}

func (s *SlackAdapter) Name() string { return "slack" }

func (s *SlackAdapter) Connect(ctx context.Context) error {
	go func() {
		for evt := range s.sm.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				apiEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}
				inner := apiEvent.InnerEvent
				msgEvent, ok := inner.Data.(*slackevents.MessageEvent)
				if !ok || msgEvent == nil {
					continue
				}
				if msgEvent.BotID != "" || msgEvent.User == "" {
					continue
				}
				if s.handler == nil {
					continue
				}

				chatType := "dm"
				if strings.HasPrefix(msgEvent.Channel, "C") {
					chatType = "group"
				}

				event := gateway.MessageEvent{
					Text:      msgEvent.Text,
					MessageID: msgEvent.TimeStamp,
					ChatID:    msgEvent.Channel,
					ChatType:  chatType,
					UserID:    msgEvent.User,
					UserName:  msgEvent.User,
					ThreadID:  msgEvent.ThreadTimeStamp,
				}
				s.handler(ctx, event)
				s.sm.Ack(*evt.Request)
			case socketmode.EventTypeInteractive:
				callback, ok := evt.Data.(slack.InteractionCallback)
				if !ok {
					continue
				}
				s.sm.Ack(*evt.Request)
				if s.handler == nil || callback.Type != slack.InteractionTypeBlockActions {
					continue
				}
				if len(callback.ActionCallback.BlockActions) == 0 {
					continue
				}
				action := callback.ActionCallback.BlockActions[0]
				if action == nil {
					continue
				}
				customID := strings.TrimSpace(action.ActionID)
				if !strings.HasPrefix(customID, "/") {
					continue
				}
				chatType := "dm"
				if strings.HasPrefix(callback.Channel.ID, "C") {
					chatType = "group"
				}
				s.handler(ctx, gateway.MessageEvent{
					Text:      customID,
					MessageID: callback.Container.MessageTs,
					ChatID:    callback.Channel.ID,
					ChatType:  chatType,
					UserID:    callback.User.ID,
					UserName:  callback.User.Name,
					ThreadID:  callback.Container.ThreadTs,
					ReplyToID: callback.Container.MessageTs,
					IsCommand: true,
				})
			case socketmode.EventTypeSlashCommand:
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					continue
				}
				_ = s.sm.Ack(*evt.Request, map[string]any{
					"response_type": "ephemeral",
					"text":          "Accepted. Check the next bot reply.",
				})
				if s.handler == nil {
					continue
				}
				text := renderSlackSlashCommand(cmd)
				if strings.TrimSpace(text) == "" {
					continue
				}
				chatType := "dm"
				if strings.HasPrefix(cmd.ChannelID, "C") {
					chatType = "group"
				}
				s.handler(ctx, gateway.MessageEvent{
					Text:      text,
					MessageID: cmd.TriggerID,
					ChatID:    cmd.ChannelID,
					ChatType:  chatType,
					UserID:    cmd.UserID,
					UserName:  cmd.UserName,
					IsCommand: true,
				})
			}
		}
	}()

	go func() {
		if err := s.sm.RunContext(ctx); err != nil {
			log.Printf("[gateway:slack] socket mode: %v", err)
		}
	}()

	log.Printf("[gateway:slack] connected via socket mode")

	go func() {
		<-ctx.Done()
		log.Printf("[gateway:slack] disconnected")
	}()

	return nil
}

func (s *SlackAdapter) Disconnect(_ context.Context) error {
	return nil
}

func (s *SlackAdapter) Send(_ context.Context, chatID, content, replyTo string) (gateway.SendResult, error) {
	return s.SendText(context.Background(), chatID, content, replyTo, nil)
}

func (s *SlackAdapter) SendText(_ context.Context, chatID, content, replyTo string, meta map[string]any) (gateway.SendResult, error) {
	opts := []slack.MsgOption{
		slack.MsgOptionText(truncateSlack(content), false),
	}
	if blocks := approvalBlocks(meta); len(blocks) > 0 {
		opts = append(opts, slack.MsgOptionBlocks(blocks...))
	}
	if replyTo != "" {
		opts = append(opts, slack.MsgOptionTS(replyTo))
	}
	_, ts, err := s.client.PostMessage(chatID, opts...)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	return gateway.SendResult{Success: true, MessageID: ts}, nil
}

func (s *SlackAdapter) SendMedia(_ context.Context, chatID, path, caption, replyTo string) (gateway.SendResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}

	params := slack.UploadFileParameters{
		File:           path,
		Filename:       filepath.Base(path),
		FileSize:       int(info.Size()),
		Title:          "",
		InitialComment: truncateSlack(caption),
		Channel:        chatID,
		ThreadTimestamp: func() string {
			if strings.TrimSpace(replyTo) == "" {
				return ""
			}
			return strings.TrimSpace(replyTo)
		}(),
	}
	file, err := s.client.UploadFile(params)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	if file == nil {
		return gateway.SendResult{Success: false, Error: "slack: upload returned nil file"}, nil
	}
	return gateway.SendResult{Success: true, MessageID: file.ID}, nil
}

func (s *SlackAdapter) EditMessage(_ context.Context, chatID, messageID, content string) error {
	_, _, _, err := s.client.UpdateMessage(chatID, messageID, slack.MsgOptionText(truncateSlack(content), false))
	return err
}

func (s *SlackAdapter) SendTyping(_ context.Context, _ string) error {
	return nil
}

func (s *SlackAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	s.handler = handler
}

func truncateSlack(s string) string {
	limit := 40000
	if len(s) <= limit {
		return s
	}
	return s[:limit-3] + "..."
}

func approvalBlocks(meta map[string]any) []slack.Block {
	if len(meta) == 0 {
		return nil
	}
	approvalID, _ := meta["approval_id"].(string)
	approvalID = strings.TrimSpace(approvalID)
	if approvalID == "" {
		return nil
	}
	approve := slack.NewButtonBlockElement("/approve "+approvalID, approvalID, slack.NewTextBlockObject("plain_text", "Approve", false, false))
	approve.Style = slack.StylePrimary
	deny := slack.NewButtonBlockElement("/deny "+approvalID, approvalID, slack.NewTextBlockObject("plain_text", "Deny", false, false))
	deny.Style = slack.StyleDanger
	return []slack.Block{
		slack.NewActionBlock("approval_actions_"+approvalID, approve, deny),
	}
}

func renderSlackSlashCommand(cmd slack.SlashCommand) string {
	command := strings.TrimSpace(cmd.Command)
	if command == "" {
		return ""
	}
	text := strings.TrimSpace(cmd.Text)
	if strings.HasPrefix(text, "/") {
		return text
	}
	if text == "" {
		return command
	}
	base := strings.TrimPrefix(command, "/")
	if isBuiltInGatewaySlashCommand(base) {
		return command + " " + text
	}
	return "/" + text
}

func isBuiltInGatewaySlashCommand(name string) bool {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "pair", "unpair", "cancel", "queue", "status", "pending", "approvals", "grant", "revoke", "approve", "deny", "help":
		return true
	default:
		return false
	}
}
