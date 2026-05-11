package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/platform"
)

func (b *BuiltinTools) ybSearchSticker(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	query := strings.TrimSpace(strArg(args, "query"))
	limit := intArg(args, "limit", 10)
	results := searchYuanbaoStickers(query, limit)
	out := make([]map[string]any, 0, len(results))
	for _, s := range results {
		out = append(out, map[string]any{
			"sticker_id":  s.StickerID,
			"package_id":  s.PackageID,
			"name":        s.Name,
			"description": s.Description,
		})
	}
	return map[string]any{"success": true, "query": query, "count": len(out), "results": out}, nil
}

func ybSearchStickerParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}}}
}

func ybSendParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"chat_id":   map[string]any{"type": "string"},
		"sticker":   map[string]any{"type": "string"},
		"text":      map[string]any{"type": "string"},
		"message":   map[string]any{"type": "string"},
		"reply_to":  map[string]any{"type": "string"},
	}}
}

var _ = errors.New

// ---------------------------------------------------------------------------
// Yuanbao gateway-backed tools (yb_*)
// ---------------------------------------------------------------------------

type yuanbaoGateway interface {
	SendSticker(ctx context.Context, chatID string, stickerJSON string, replyTo string) (map[string]any, error)
	QueryGroupInfo(ctx context.Context, groupCode string) (map[string]any, error)
	QueryGroupMembers(ctx context.Context, groupCode string, offset, limit uint32) (map[string]any, error)
}

func getYuanbaoGateway() (yuanbaoGateway, bool) {
	a, ok := platform.Get("yuanbao")
	if !ok {
		return nil, false
	}
	gw, ok := a.(yuanbaoGateway)
	return gw, ok
}

func (b *BuiltinTools) ybSendDM(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	chatID := strings.TrimSpace(strArg(args, "chat_id"))
	if chatID == "" && strings.EqualFold(strings.TrimSpace(tc.GatewayPlatform), "yuanbao") {
		chatID = strings.TrimSpace(tc.GatewayChatID)
	}
	text := strArg(args, "text")
	if strings.TrimSpace(text) == "" {
		text = strArg(args, "message")
	}
	if strings.TrimSpace(chatID) == "" {
		return nil, errors.New("chat_id required")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text/message required")
	}
	a, ok := platform.Get("yuanbao")
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "yuanbao adapter not connected (enable gateway and set YUANBAO_*)"}, nil
	}
	res, err := a.Send(ctx, "direct:"+chatID, text, "")
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	if !res.Success {
		return map[string]any{"success": false, "error": res.Error}, nil
	}
	return map[string]any{"success": true}, nil
}

func (b *BuiltinTools) ybSendSticker(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	chatID := strings.TrimSpace(strArg(args, "chat_id"))
	if chatID == "" && strings.EqualFold(strings.TrimSpace(tc.GatewayPlatform), "yuanbao") {
		chatID = strings.TrimSpace(tc.GatewayChatID)
	}
	if chatID == "" {
		return nil, errors.New("chat_id required")
	}
	replyTo := strings.TrimSpace(strArg(args, "reply_to"))
	nameOrID := strings.TrimSpace(strArg(args, "sticker"))
	var s *yuanbaoSticker
	if nameOrID != "" {
		s = findStickerByNameOrID(nameOrID)
		if s == nil {
			return map[string]any{"success": false, "error": fmt.Sprintf("sticker not found: %q (call yb_search_sticker first)", nameOrID)}, nil
		}
	} else {
		// Default: pick the first sticker in our minimal catalogue.
		if len(yuanbaoStickers) == 0 {
			return map[string]any{"success": false, "error": "no built-in stickers available"}, nil
		}
		s = &yuanbaoStickers[0]
	}

	stickerJSON, err := buildStickerJSON(*s)
	if err != nil {
		return nil, err
	}
	gw, ok := getYuanbaoGateway()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "yuanbao adapter not connected (enable gateway and set YUANBAO_*)"}, nil
	}
	out, err := gw.SendSticker(ctx, chatID, stickerJSON, replyTo)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	if out == nil {
		out = map[string]any{}
	}
	out["success"] = true
	out["sticker_id"] = s.StickerID
	out["name"] = s.Name
	return out, nil
}

func (b *BuiltinTools) ybQueryGroupInfo(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	groupCode := strings.TrimSpace(strArg(args, "group_code"))
	if groupCode == "" {
		groupCode = strings.TrimSpace(strArg(args, "chat_id"))
	}
	if groupCode == "" && strings.EqualFold(strings.TrimSpace(tc.GatewayPlatform), "yuanbao") && strings.EqualFold(strings.TrimSpace(tc.GatewayChatType), "group") {
		groupCode = strings.TrimSpace(tc.GatewayChatID)
	}
	if groupCode == "" {
		return nil, errors.New("group_code required")
	}
	gw, ok := getYuanbaoGateway()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "yuanbao adapter not connected (enable gateway and set YUANBAO_*)"}, nil
	}
	out, err := gw.QueryGroupInfo(ctx, groupCode)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	if out == nil {
		out = map[string]any{}
	}
	out["success"] = true
	return out, nil
}

func (b *BuiltinTools) ybQueryGroupMembers(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	groupCode := strings.TrimSpace(strArg(args, "group_code"))
	if groupCode == "" {
		groupCode = strings.TrimSpace(strArg(args, "chat_id"))
	}
	if groupCode == "" && strings.EqualFold(strings.TrimSpace(tc.GatewayPlatform), "yuanbao") && strings.EqualFold(strings.TrimSpace(tc.GatewayChatType), "group") {
		groupCode = strings.TrimSpace(tc.GatewayChatID)
	}
	if groupCode == "" {
		return nil, errors.New("group_code required")
	}
	offset := uint32(intArg(args, "offset", 0))
	limit := uint32(intArg(args, "limit", 200))
	gw, ok := getYuanbaoGateway()
	if !ok {
		return map[string]any{"success": false, "available": false, "error": "yuanbao adapter not connected (enable gateway and set YUANBAO_*)"}, nil
	}
	out, err := gw.QueryGroupMembers(ctx, groupCode, offset, limit)
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, nil
	}
	if out == nil {
		out = map[string]any{}
	}
	out["success"] = true
	return out, nil
}

func ybQueryGroupInfoParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"group_code": map[string]any{"type": "string"}, "chat_id": map[string]any{"type": "string"}}, "required": []string{}}
}

func ybQueryGroupMembersParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"group_code": map[string]any{"type": "string"}, "chat_id": map[string]any{"type": "string"}, "offset": map[string]any{"type": "integer"}, "limit": map[string]any{"type": "integer"}}, "required": []string{}}
}

func findStickerByNameOrID(nameOrID string) *yuanbaoSticker {
	q := strings.TrimSpace(nameOrID)
	if q == "" {
		return nil
	}
	for i := range yuanbaoStickers {
		s := &yuanbaoStickers[i]
		if s.StickerID == q || strings.EqualFold(s.Name, q) {
			return s
		}
	}
	return nil
}

func buildStickerJSON(s yuanbaoSticker) (string, error) {
	payload := map[string]any{
		"sticker_id": s.StickerID,
		"package_id": s.PackageID,
		"width":      128,
		"height":     128,
		"formats":    "png",
		"name":       s.Name,
	}
	bs, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}
