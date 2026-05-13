package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func discordBotToken() (string, bool) {
	tok := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN"))
	return tok, tok != ""
}

func discordDo(ctx context.Context, method, path, token string, body any) ([]byte, int, error) {
	base := "https://discord.com/api/v10"
	var r io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		r = bytes.NewReader(bs)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, r)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	bs, _ := io.ReadAll(resp.Body)
	return bs, resp.StatusCode, nil
}

func (b *BuiltinTools) discordAdmin(ctx context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	token, ok := discordBotToken()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "discord_admin not configured (missing env: DISCORD_BOT_TOKEN)"}, nil
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
	case "create_text_channel":
		guildID := strings.TrimSpace(strArg(args, "guild_id"))
		name := strings.TrimSpace(strArg(args, "name"))
		if guildID == "" || name == "" {
			return nil, errors.New("guild_id and name required")
		}
		// 0 = GUILD_TEXT
		payload := map[string]any{"name": name, "type": 0}
		if topic := strings.TrimSpace(strArg(args, "topic")); topic != "" {
			payload["topic"] = topic
		}
		bs, code, err := discordDo(ctx, http.MethodPost, "/guilds/"+guildID+"/channels", token, payload)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "guild_id": guildID, "channel": data}, nil
	case "delete_channel":
		channelID := strings.TrimSpace(strArg(args, "channel_id"))
		if channelID == "" {
			return nil, errors.New("channel_id required")
		}
		bs, code, err := discordDo(ctx, http.MethodDelete, "/channels/"+channelID, token, nil)
		if err != nil {
			return nil, err
		}
		if code < 200 || code >= 300 {
			return map[string]any{"success": false, "status_code": code, "error": string(bs)}, nil
		}
		var data any
		_ = json.Unmarshal(bs, &data)
		return map[string]any{"success": true, "channel_id": channelID, "deleted": true, "channel": data}, nil
	default:
		return nil, fmt.Errorf("unsupported discord_admin action: %s", action)
	}
}

func discordAdminParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":     map[string]any{"type": "string", "enum": []string{"list_guilds", "list_channels", "create_text_channel", "delete_channel"}},
			"guild_id":   map[string]any{"type": "string"},
			"channel_id": map[string]any{"type": "string"},
			"name":       map[string]any{"type": "string"},
			"topic":      map[string]any{"type": "string"},
		},
	}
}
