package platforms

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	d.session.AddHandler(func(s *discordgo.Session, ic *discordgo.InteractionCreate) {
		if d.handler == nil || ic == nil || ic.Type != discordgo.InteractionMessageComponent || ic.Member == nil && ic.User == nil {
			return
		}
		data := ic.MessageComponentData()
		customID := strings.TrimSpace(data.CustomID)
		if !strings.HasPrefix(customID, "/") {
			return
		}
		if err := s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		}); err != nil {
			log.Printf("[gateway:discord] acknowledge interaction: %v", err)
		}
		userID := ""
		userName := ""
		if ic.Member != nil && ic.Member.User != nil {
			userID = ic.Member.User.ID
			userName = ic.Member.User.Username
		} else if ic.User != nil {
			userID = ic.User.ID
			userName = ic.User.Username
		}
		chatType := "dm"
		if strings.TrimSpace(ic.GuildID) != "" {
			chatType = "group"
		}
		threadID := ""
		if ic.Message != nil && ic.Message.Thread != nil {
			threadID = ic.Message.Thread.ID
		}
		messageID := ""
		replyToID := ""
		if ic.Message != nil {
			messageID = ic.Message.ID
			replyToID = ic.Message.ID
		}
		d.handler(ctx, gateway.MessageEvent{
			Text:      customID,
			MessageID: messageID,
			ChatID:    ic.ChannelID,
			ChatType:  chatType,
			UserID:    userID,
			UserName:  userName,
			ThreadID:  threadID,
			ReplyToID: replyToID,
			IsCommand: true,
		})
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
	return d.SendText(context.Background(), chatID, content, replyTo, nil)
}

func (d *DiscordAdapter) SendText(_ context.Context, chatID, content, replyTo string, meta map[string]any) (gateway.SendResult, error) {
	msg := &discordgo.MessageSend{
		Content:   truncateDiscord(content, 2000),
		Reference: &discordgo.MessageReference{MessageID: replyTo},
	}
	if components := approvalComponents(meta); len(components) > 0 {
		msg.Components = components
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

func (d *DiscordAdapter) SendMedia(_ context.Context, chatID, path, caption, replyTo string) (gateway.SendResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return gateway.SendResult{Success: false, Error: err.Error()}, err
	}
	defer f.Close()

	msg := &discordgo.MessageSend{
		Content:   truncateDiscord(caption, 2000),
		Reference: &discordgo.MessageReference{MessageID: replyTo},
		Files: []*discordgo.File{
			{
				Name:   filepath.Base(path),
				Reader: f,
			},
		},
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

func approvalComponents(meta map[string]any) []discordgo.MessageComponent {
	if len(meta) == 0 {
		return nil
	}
	approvalID, _ := meta["approval_id"].(string)
	approvalID = strings.TrimSpace(approvalID)
	if approvalID == "" {
		return nil
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Approve",
					Style:    discordgo.SuccessButton,
					CustomID: "/approve " + approvalID,
				},
				discordgo.Button{
					Label:    "Deny",
					Style:    discordgo.DangerButton,
					CustomID: "/deny " + approvalID,
				},
			},
		},
	}
}
