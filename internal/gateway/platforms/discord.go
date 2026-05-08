package platforms

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"github.com/dingjingmaster/agent-daemon/internal/gateway"
)

type DiscordAdapter struct {
	session *discordgo.Session
	handler gateway.MessageHandler
}

func NewDiscordAdapter(token string) (*DiscordAdapter, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discord: create session: %w", err)
	}
	return &DiscordAdapter{session: session}, nil
}

func (d *DiscordAdapter) Name() string { return "discord" }

func (d *DiscordAdapter) Connect(ctx context.Context) error {
	d.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if d.handler == nil {
			return
		}
		if m.Author == nil || m.Author.Bot {
			return
		}

		chatType := "dm"
		if m.GuildID != "" {
			chatType = "group"
		}
		threadID := ""
		if m.Thread != nil {
			threadID = m.Thread.ID
		}

		event := gateway.MessageEvent{
			Text:      m.ContentWithMentionsReplaced(),
			MessageID: m.ID,
			ChatID:    m.ChannelID,
			ChatType:  chatType,
			UserID:    m.Author.ID,
			UserName:  m.Author.Username,
			ThreadID:  threadID,
		}
		if m.ReferencedMessage != nil {
			event.ReplyToID = m.ReferencedMessage.ID
		}
		d.handler(ctx, event)
	})

	if err := d.session.Open(); err != nil {
		return fmt.Errorf("discord: connect: %w", err)
	}
	log.Printf("[gateway:discord] connected as @%s", d.session.State.User.Username)

	go func() {
		<-ctx.Done()
		_ = d.session.Close()
		log.Printf("[gateway:discord] disconnected")
	}()
	return nil
}

func (d *DiscordAdapter) Disconnect(_ context.Context) error {
	return d.session.Close()
}

func (d *DiscordAdapter) Send(_ context.Context, chatID, content, replyTo string) (gateway.SendResult, error) {
	msg := &discordgo.MessageSend{
		Content:   truncateDiscord(content, 2000),
		Reference: &discordgo.MessageReference{MessageID: replyTo},
	}
	if replyTo == "" {
		msg.Reference = nil
	}
	sent, err := d.session.ChannelMessageSendComplex(chatID, msg)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	return gateway.SendResult{Success: true, MessageID: sent.ID}, nil
}

func (d *DiscordAdapter) EditMessage(_ context.Context, chatID, messageID, content string) error {
	_, err := d.session.ChannelMessageEdit(chatID, messageID, truncateDiscord(content, 2000))
	return err
}

func (d *DiscordAdapter) SendTyping(_ context.Context, chatID string) error {
	return d.session.ChannelTyping(chatID)
}

func (d *DiscordAdapter) OnMessage(_ context.Context, handler gateway.MessageHandler) {
	d.handler = handler
}

func truncateDiscord(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit-3] + "..."
}
