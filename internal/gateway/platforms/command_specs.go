package platforms

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dingjingmaster/agent-daemon/internal/gateway"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type gatewayPlatformCommandSpec struct {
	Name        string
	Description string
	Telegram    bool
	Discord     bool
	DiscordOpts []*discordgo.ApplicationCommandOption
}

func GatewayPlatformCommandSpecs() []gatewayPlatformCommandSpec {
	order := gateway.GatewayCommandOrder()
	desc := gateway.GatewayCommandDescriptions()
	out := make([]gatewayPlatformCommandSpec, 0, len(order))
	for _, name := range order {
		out = append(out, gatewayPlatformCommandSpec{
			Name:        name,
			Description: desc[name],
			Telegram:    true,
			Discord:     true,
			DiscordOpts: discordCommandOptions(name),
		})
	}
	return out
}

func discordCommandOptions(name string) []*discordgo.ApplicationCommandOption {
	switch name {
	case "pair":
		return []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "code", Description: "pair code", Required: true},
		}
	case "grant":
		return []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "pattern", Description: "pattern name for pattern approval", Required: false},
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "ttl", Description: "ttl seconds", Required: false},
		}
	case "revoke":
		return []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "pattern", Description: "pattern name for pattern approval", Required: false},
		}
	case "approve", "deny":
		return []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "id", Description: "approval id", Required: false},
		}
	default:
		return nil
	}
}

func gatewayPlatformCommandSpecs() []gatewayPlatformCommandSpec {
	return GatewayPlatformCommandSpecs()
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
