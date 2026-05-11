package tools

import (
	"context"
	"errors"
	"fmt"
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
						"description": "Action to perform",
						"enum":        []string{"send", "list"},
					},
					"platform": map[string]any{
						"type":        "string",
						"description": "Platform name (telegram/discord/slack)",
					},
					"chat_id": map[string]any{
						"type":        "string",
						"description": "Target chat/channel id for the platform adapter",
					},
					"message": map[string]any{
						"type":        "string",
						"description": "Message content to send",
					},
					"reply_to": map[string]any{
						"type":        "string",
						"description": "Optional message id to reply to",
					},
				},
				"required": []string{"action"},
			},
		},
	}
}

func (t *SendMessageTool) Call(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
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
		if p == "" {
			return nil, errors.New("platform required")
		}
		chatID := strings.TrimSpace(strArg(args, "chat_id"))
		if chatID == "" {
			return nil, errors.New("chat_id required")
		}
		msg := strArg(args, "message")
		if strings.TrimSpace(msg) == "" {
			return nil, errors.New("message required")
		}
		replyTo := strings.TrimSpace(strArg(args, "reply_to"))
		a, ok := platform.Get(p)
		if !ok {
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("platform adapter not connected: %s (enable gateway and configure credentials)", p),
			}, nil
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

