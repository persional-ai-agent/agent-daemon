package platforms

import (
	"context"
	"log"
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
			if evt.Type != socketmode.EventTypeEventsAPI {
				continue
			}
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
	opts := []slack.MsgOption{
		slack.MsgOptionText(truncateSlack(content), false),
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
