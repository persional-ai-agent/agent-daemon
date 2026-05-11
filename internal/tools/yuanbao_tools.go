package tools

import (
	"context"
	"errors"
	"os"
	"strings"
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

func (b *BuiltinTools) ybSendNotImplemented(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	if strings.TrimSpace(os.Getenv("YUANBAO_TOKEN")) == "" &&
		strings.TrimSpace(os.Getenv("YUANBAO_APP_ID")) == "" {
		return map[string]any{"success": false, "available": false, "error": "yuanbao not configured (set YUANBAO_* env and gateway adapter)"}, nil
	}
	_ = args
	return map[string]any{"success": false, "available": false, "error": "yuanbao gateway adapter not implemented in agent-daemon"}, nil
}

func ybSearchStickerParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}}}
}

func ybSendParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"chat_id": map[string]any{"type": "string"}, "sticker": map[string]any{"type": "string"}, "text": map[string]any{"type": "string"}}}
}

var _ = errors.New

