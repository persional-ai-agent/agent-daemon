package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/platform"
)

type SendMessageTool struct{}

func NewSendMessageTool() *SendMessageTool { return &SendMessageTool{} }

func (t *SendMessageTool) Name() string { return "send_message" }

func (t *SendMessageTool) Schema() core.ToolSchema {
	return core.ToolSchema{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:        t.Name(),
			Description: "Send a message via the running gateway adapters, or list connected platforms. Actions: send, list.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": "Action to perform (default: send)",
						"enum":        []string{"send", "list"},
					},
					"target": map[string]any{
						"type":        "string",
						"description": "Hermes-style target string like 'telegram:123', 'discord:channel_id', or 'slack:channel_id'. If set, overrides platform/chat_id.",
					},
					"platform": map[string]any{
						"type":        "string",
						"description": "Platform name (signal/email/homeassistant/telegram/discord/slack/whatsapp/webhook/yuanbao)",
					},
					"chat_id": map[string]any{
						"type":        "string",
						"description": "Target chat/channel id for the platform adapter",
					},
					"message": map[string]any{
						"type":        "string",
						"description": "Message content to send",
					},
					"media_path": map[string]any{
						"type":        "string",
						"description": "Optional local file path to send as an attachment (requires adapter support). If set, message is used as caption.",
					},
					"reply_to": map[string]any{
						"type":        "string",
						"description": "Optional message id to reply to",
					},
				},
			},
		},
	}
}

func (t *SendMessageTool) Call(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	if action == "" {
		action = "send"
	}
	switch action {
	case "list":
		names := platform.Names()
		sort.Strings(names)
		return map[string]any{"success": true, "platforms": names}, nil
	case "send":
		p := strings.ToLower(strings.TrimSpace(strArg(args, "platform")))
		chatID := strings.TrimSpace(strArg(args, "chat_id"))
		if target := strings.TrimSpace(strArg(args, "target")); target != "" {
			parts := strings.SplitN(target, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid target %q (expected platform:chat_id)", target)
			}
			p = strings.ToLower(strings.TrimSpace(parts[0]))
			chatID = strings.TrimSpace(parts[1])
		}
		if p == "" && strings.TrimSpace(tc.GatewayPlatform) != "" {
			p = strings.ToLower(strings.TrimSpace(tc.GatewayPlatform))
		}
		if chatID == "" && strings.TrimSpace(tc.GatewayChatID) != "" {
			chatID = strings.TrimSpace(tc.GatewayChatID)
		}
		if chatID == "" && p != "" {
			if v := strings.TrimSpace(os.Getenv(homeTargetEnvVar(p))); v != "" {
				chatID = v
			}
		}
		if p == "" {
			return nil, errors.New("platform required (or set target)")
		}
		if chatID == "" {
			return nil, errors.New("chat_id required (or set target)")
		}
		msg := strArg(args, "message")
		mediaPath := strings.TrimSpace(strArg(args, "media_path"))
		if strings.TrimSpace(msg) == "" && mediaPath == "" {
			return nil, errors.New("message required (or set media_path)")
		}
		replyTo := strings.TrimSpace(strArg(args, "reply_to"))
		a, ok := platform.Get(p)
		if !ok {
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("platform adapter not connected: %s (enable gateway and configure credentials)", p),
			}, nil
		}
		// Hermes-style: if message starts with "MEDIA:" treat it as a media send.
		if mediaPath == "" {
			m := strings.TrimSpace(msg)
			if strings.HasPrefix(strings.ToUpper(m), "MEDIA:") {
				mediaPath = strings.TrimSpace(m[len("MEDIA:"):])
				msg = ""
			}
		}
		if mediaPath != "" {
			if ms, ok := a.(platform.MediaSender); ok {
				res, err := ms.SendMedia(ctx, chatID, mediaPath, msg, replyTo)
				if err != nil {
					return map[string]any{"success": false, "error": err.Error()}, nil
				}
				if !res.Success && strings.TrimSpace(res.Error) != "" {
					return map[string]any{"success": false, "error": res.Error}, nil
				}
				return map[string]any{"success": true, "platform": p, "chat_id": chatID, "media_path": mediaPath}, nil
			}
			return map[string]any{"success": false, "available": false, "error": "platform adapter does not support media delivery", "platform": p}, nil
		}
		res, err := a.Send(ctx, chatID, msg, replyTo)
		if err != nil {
			return map[string]any{"success": false, "error": err.Error()}, nil
		}
		if !res.Success && strings.TrimSpace(res.Error) != "" {
			return map[string]any{"success": false, "error": res.Error}, nil
		}
		return map[string]any{"success": true, "platform": p, "chat_id": chatID, "message_id": res.MessageID}, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func homeTargetEnvVar(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "telegram":
		return "TELEGRAM_HOME_CHANNEL"
	case "discord":
		return "DISCORD_HOME_CHANNEL"
	case "slack":
		return "SLACK_HOME_CHANNEL"
	case "yuanbao":
		return "YUANBAO_HOME_CHANNEL"
	default:
		return strings.ToUpper(platform) + "_HOME_CHANNEL"
	}
}
