package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
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
		action = "list_guilds"
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
			return map[string]any{"success": false, "error": "guild_id required"}, nil
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
			return map[string]any{"success": false, "error": "guild_id required"}, nil
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

	case "fetch_channel":
		channelID := strings.TrimSpace(strArg(args, "channel_id"))
		if channelID == "" {
			return map[string]any{"success": false, "error": "channel_id required"}, nil
		}
		bs, code, err := discordDo(ctx, http.MethodGet, "/channels/"+channelID, token, nil)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "channel_id": channelID, "channel": data}, nil

	case "fetch_messages":
		channelID := strings.TrimSpace(strArg(args, "channel_id"))
		if channelID == "" {
			return map[string]any{"success": false, "error": "channel_id required"}, nil
		}
		limit := intArg(args, "limit", 50)
		if limit <= 0 {
			limit = 50
		}
		if limit > 100 {
			limit = 100
		}
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if v := strings.TrimSpace(strArg(args, "before")); v != "" {
			q.Set("before", v)
		}
		if v := strings.TrimSpace(strArg(args, "after")); v != "" {
			q.Set("after", v)
		}
		if v := strings.TrimSpace(strArg(args, "around")); v != "" {
			q.Set("around", v)
		}
		path := "/channels/" + channelID + "/messages"
		if qs := q.Encode(); qs != "" {
			path += "?" + qs
		}
		bs, code, err := discordDo(ctx, http.MethodGet, path, token, nil)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "channel_id": channelID, "limit": limit, "messages": data}, nil

	case "send_message":
		channelID := strings.TrimSpace(strArg(args, "channel_id"))
		if channelID == "" {
			return map[string]any{"success": false, "error": "channel_id required"}, nil
		}
		content := strArg(args, "content")
		if strings.TrimSpace(content) == "" {
			content = strArg(args, "message")
		}
		if strings.TrimSpace(content) == "" {
			return map[string]any{"success": false, "error": "content/message required"}, nil
		}
		payload := map[string]any{"content": content}
		bs, code, err := discordDo(ctx, http.MethodPost, "/channels/"+channelID+"/messages", token, payload)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "channel_id": channelID, "message": data}, nil

	case "react":
		channelID := strings.TrimSpace(strArg(args, "channel_id"))
		messageID := strings.TrimSpace(strArg(args, "message_id"))
		emoji := strings.TrimSpace(strArg(args, "emoji"))
		if channelID == "" || messageID == "" || emoji == "" {
			return map[string]any{"success": false, "error": "channel_id, message_id, emoji required"}, nil
		}
		// Emoji must be URL-encoded per Discord API.
		emojiEnc := url.PathEscape(emoji)
		bs, code, err := discordDo(ctx, http.MethodPut, "/channels/"+channelID+"/messages/"+messageID+"/reactions/"+emojiEnc+"/@me", token, nil)
		if err != nil {
			return nil, err
		}
		if code != 204 && (code < 200 || code >= 300) {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		return map[string]any{"success": true, "channel_id": channelID, "message_id": messageID, "emoji": emoji}, nil

	default:
		return map[string]any{"success": false, "error": "unsupported action"}, nil
	}
}

func discordToolParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{
				"list_guilds",
				"server_info",
				"list_channels",
				"fetch_channel",
				"fetch_messages",
				"send_message",
				"react",
			}, "description": "Action to perform (default: list_guilds)"},
			"guild_id":   map[string]any{"type": "string"},
			"channel_id": map[string]any{"type": "string"},
			"message_id": map[string]any{"type": "string"},
			"emoji":      map[string]any{"type": "string"},
			"limit":      map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "description": "Maximum messages to fetch (default 50, max 100)."},
			"before":     map[string]any{"type": "string"},
			"after":      map[string]any{"type": "string"},
			"around":     map[string]any{"type": "string"},
			"content":    map[string]any{"type": "string"},
			"message":    map[string]any{"type": "string"},
		},
	}
}
