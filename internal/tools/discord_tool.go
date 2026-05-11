package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// Hermes parity: `discord` tool (server introspection/management).
// Minimal mapping to our existing Discord REST helper.

func (b *BuiltinTools) discordTool(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := discordBotToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "discord not configured (missing env: DISCORD_BOT_TOKEN)"}, nil
	}
	action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
	if action == "" {
		return nil, errors.New("action required")
	}

	switch action {
	case "list_guilds":
		bs, code, err := discordDo(ctx, http.MethodGet, "/users/@me/guilds", token, nil)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "guilds": data}, nil

	case "server_info":
		guildID := strings.TrimSpace(strArg(args, "guild_id"))
		if guildID == "" {
			return nil, errors.New("guild_id required")
		}
		bs, code, err := discordDo(ctx, http.MethodGet, "/guilds/"+guildID+"?with_counts=true", token, nil)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "guild_id": guildID, "server": data}, nil

	case "list_channels":
		guildID := strings.TrimSpace(strArg(args, "guild_id"))
		if guildID == "" {
			return nil, errors.New("guild_id required")
		}
		bs, code, err := discordDo(ctx, http.MethodGet, "/guilds/"+guildID+"/channels", token, nil)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "guild_id": guildID, "channels": data}, nil

	default:
		return nil, errors.New("unsupported action (supported: list_guilds, server_info, list_channels)")
	}
}

func discordToolParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":   map[string]any{"type": "string", "enum": []string{"list_guilds", "server_info", "list_channels"}},
			"guild_id": map[string]any{"type": "string"},
		},
		"required": []string{"action"},
	}
}

