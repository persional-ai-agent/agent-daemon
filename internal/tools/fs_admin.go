package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (b *BuiltinTools) mkdir(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "path": path, "created": true}, nil
}

func (b *BuiltinTools) listDir(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	root := strArg(args, "path")
	if strings.TrimSpace(root) == "" {
		root = tc.Workdir
	}
	path, err := resolvePathWithinWorkdir(tc.Workdir, root)
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		item := map[string]any{
			"name":  e.Name(),
			"isDir": e.IsDir(),
		}
		if info != nil {
			item["mode"] = info.Mode().String()
			item["size"] = info.Size()
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return fmt.Sprint(out[i]["name"]) < fmt.Sprint(out[j]["name"]) })
	return map[string]any{"success": true, "path": path, "entries": out, "count": len(out)}, nil
}

func (b *BuiltinTools) deleteFile(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	path, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, path); err != nil {
		return nil, err
	}
	recursive := boolArg(args, "recursive", false)
	if recursive {
		// Only allow deleting directories within workdir; still reject symlink components.
		if err := os.RemoveAll(path); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "path": path, "deleted": true, "recursive": true}, nil
	}
	if err := rejectNonRegularFile(path); err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "path": path, "deleted": true}, nil
}

func (b *BuiltinTools) moveFile(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	src, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "src"))
	if err != nil {
		return nil, err
	}
	dst, err := resolvePathWithinWorkdir(tc.Workdir, strArg(args, "dst"))
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, src); err != nil {
		return nil, err
	}
	if err := rejectSymlinkEscape(tc.Workdir, dst); err != nil {
		return nil, err
	}
	if err := rejectNonRegularFile(src); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(dst); err == nil {
		return nil, errors.New("destination already exists")
	}
	if err := os.Rename(src, dst); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "src": src, "dst": dst, "moved": true}, nil
}

