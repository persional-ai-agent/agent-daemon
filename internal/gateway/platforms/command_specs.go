package platforms

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/bwmarrin/discordgo"
)

type gatewayPlatformCommandSpec struct {
	Name        string
	Description string
	Telegram    bool
	Discord     bool
	DiscordOpts []*discordgo.ApplicationCommandOption
}

func gatewayPlatformCommandSpecs() []gatewayPlatformCommandSpec {
	return []gatewayPlatformCommandSpec{
		{
			Name:        "pair",
			Description: "pair with gateway using a code",
			Telegram:    true,
			Discord:     true,
			DiscordOpts: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "code", Description: "pair code", Required: true},
			},
		},
		{Name: "unpair", Description: "remove current gateway pairing", Telegram: true, Discord: true},
		{Name: "cancel", Description: "cancel the running task", Telegram: true, Discord: true},
		{Name: "queue", Description: "show queued task count", Telegram: true, Discord: true},
		{Name: "status", Description: "show current session status", Telegram: true, Discord: true},
		{Name: "pending", Description: "show latest pending approval", Telegram: true, Discord: true},
		{Name: "approvals", Description: "show active approvals", Telegram: true, Discord: true},
		{
			Name:        "grant",
			Description: "grant session or pattern approval",
			Telegram:    true,
			Discord:     true,
			DiscordOpts: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "pattern", Description: "pattern name for pattern approval", Required: false},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "ttl", Description: "ttl seconds", Required: false},
			},
		},
		{
			Name:        "revoke",
			Description: "revoke session or pattern approval",
			Telegram:    true,
			Discord:     true,
			DiscordOpts: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "pattern", Description: "pattern name for pattern approval", Required: false},
			},
		},
		{
			Name:        "approve",
			Description: "approve a pending approval id",
			Telegram:    true,
			Discord:     true,
			DiscordOpts: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "id", Description: "approval id", Required: false},
			},
		},
		{
			Name:        "deny",
			Description: "deny a pending approval id",
			Telegram:    true,
			Discord:     true,
			DiscordOpts: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "id", Description: "approval id", Required: false},
			},
		},
		{Name: "help", Description: "show supported commands", Telegram: true, Discord: true},
	}
}

func telegramCommandsFromSpecs(specs []gatewayPlatformCommandSpec) []tgbotapi.BotCommand {
	out := make([]tgbotapi.BotCommand, 0, len(specs))
	for _, s := range specs {
		if !s.Telegram {
			continue
		}
		out = append(out, tgbotapi.BotCommand{
			Command:     s.Name,
			Description: s.Description,
		})
	}
	return out
}

func discordCommandsFromSpecs(specs []gatewayPlatformCommandSpec) []*discordgo.ApplicationCommand {
	out := make([]*discordgo.ApplicationCommand, 0, len(specs))
	for _, s := range specs {
		if !s.Discord {
			continue
		}
		out = append(out, &discordgo.ApplicationCommand{
			Name:        s.Name,
			Description: s.Description,
			Options:     s.DiscordOpts,
		})
	}
	return out
}
