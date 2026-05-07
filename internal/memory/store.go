package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	BaseDir string
}

func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &Store{BaseDir: baseDir}, nil
}

func (s *Store) Manage(action, target, content, oldText string) (map[string]any, error) {
	file := s.fileForTarget(target)
	if file == "" {
		return nil, fmt.Errorf("unknown memory target: %s", target)
	}
	fullPath := filepath.Join(s.BaseDir, file)

	current := ""
	if b, err := os.ReadFile(fullPath); err == nil {
		current = string(b)
	}

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "add":
		line := strings.TrimSpace(content)
		if line == "" {
			return map[string]any{"success": false, "error": "empty content"}, nil
		}
		if current != "" && !strings.HasSuffix(current, "\n") {
			current += "\n"
		}
		current += "- " + line + "\n"
	case "replace", "update":
		if strings.TrimSpace(oldText) == "" {
			return map[string]any{"success": false, "error": "old_text required for replace"}, nil
		}
		if !strings.Contains(current, oldText) {
			return map[string]any{"success": false, "error": "old_text not found"}, nil
		}
		current = strings.Replace(current, oldText, content, 1)
	case "delete", "remove":
		if strings.TrimSpace(content) == "" {
			return map[string]any{"success": false, "error": "content required for delete"}, nil
		}
		current = strings.ReplaceAll(current, content, "")
	default:
		return nil, fmt.Errorf("unsupported memory action: %s", action)
	}

	if err := os.WriteFile(fullPath, []byte(current), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "target": target, "file": fullPath}, nil
}

func (s *Store) Snapshot() (map[string]string, error) {
	out := map[string]string{
		"memory": "",
		"user":   "",
	}
	for target := range out {
		content, err := s.readTarget(target)
		if err != nil {
			return nil, err
		}
		out[target] = content
	}
	return out, nil
}

func (s *Store) readTarget(target string) (string, error) {
	file := s.fileForTarget(target)
	if file == "" {
		return "", fmt.Errorf("unknown memory target: %s", target)
	}
	fullPath := filepath.Join(s.BaseDir, file)
	b, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

func (s *Store) fileForTarget(target string) string {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "memory", "memory.md":
		return "MEMORY.md"
	case "user", "user.md":
		return "USER.md"
	default:
		return ""
	}
}
