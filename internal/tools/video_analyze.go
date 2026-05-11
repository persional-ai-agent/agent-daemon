package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (b *BuiltinTools) videoAnalyze(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	if err := rejectNonRegularFile(path); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	if _, err := exec.LookPath("ffprobe"); err != nil {
		return map[string]any{
			"success":   false,
			"available": false,
			"error":     "ffprobe not found; video_analyze requires ffprobe in PATH",
		}, nil
	}

	timeout := intArg(args, "timeout", 30)
	if timeout <= 0 {
		timeout = 30
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, "ffprobe", "-v", "error", "-print_format", "json", "-show_format", "-show_streams", path)
	out, err := cmd.CombinedOutput()
	if cctx.Err() != nil {
		return map[string]any{"success": false, "error": "ffprobe timeout"}, nil
	}
	if err != nil {
		return map[string]any{"success": false, "error": strings.TrimSpace(string(out))}, nil
	}
	var data any
	if jerr := json.Unmarshal(out, &data); jerr != nil {
		return map[string]any{"success": false, "error": "failed to parse ffprobe output"}, nil
	}
	return map[string]any{"success": true, "path": path, "analysis": data}, nil
}

func videoAnalyzeParams() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string"},
			"timeout": map[string]any{"type": "integer"},
		},
		"required": []string{"path"},
	}
}

var _ = errors.New

