package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

func envGate(tool string, required ...string) map[string]any {
	missing := make([]string, 0)
	for _, k := range required {
		if strings.TrimSpace(os.Getenv(k)) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return map[string]any{
			"success":   false,
			"available": false,
			"tool":      tool,
			"error":     fmt.Sprintf("%s not configured (missing env: %s)", tool, strings.Join(missing, ", ")),
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// RL training tools (Hermes: rl_*)
// ---------------------------------------------------------------------------

func (b *BuiltinTools) rlNotImplemented(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	action := strings.TrimSpace(strArg(args, "action"))
	if action == "" {
		action = "unknown"
	}
	return map[string]any{"success": false, "available": false, "error": "rl_* tools not implemented in agent-daemon", "action": action}, nil
}

func rlParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"action": map[string]any{"type": "string"}}}
}

// ---------------------------------------------------------------------------
// Spotify tools (Hermes: spotify_*)
// ---------------------------------------------------------------------------

func (b *BuiltinTools) spotifyNotImplemented(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	if gated := envGate("spotify", "SPOTIFY_ACCESS_TOKEN"); gated != nil {
		return gated, nil
	}
	tool := strings.TrimSpace(strArg(args, "tool"))
	if tool == "" {
		tool = "spotify"
	}
	return map[string]any{"success": false, "available": false, "tool": tool, "error": "spotify integration not implemented in agent-daemon (token detected)"}, nil
}

func spotifyParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}, "tool": map[string]any{"type": "string"}}}
}

// ---------------------------------------------------------------------------
// Yuanbao tools (Hermes: yb_*)
// ---------------------------------------------------------------------------

func (b *BuiltinTools) ybNotImplemented(_ context.Context, args map[string]any, _ ToolContext) (map[string]any, error) {
	if gated := envGate("yb", "YUANBAO_TOKEN"); gated != nil {
		return gated, nil
	}
	tool := strings.TrimSpace(strArg(args, "tool"))
	if tool == "" {
		tool = "yb"
	}
	return map[string]any{"success": false, "available": false, "tool": tool, "error": "yuanbao integration not implemented in agent-daemon (token detected)"}, nil
}

func ybParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"tool": map[string]any{"type": "string"}}}
}

var _ = errors.New
