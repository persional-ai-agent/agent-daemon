package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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
	action = strings.ToLower(strings.TrimSpace(action))
	file := s.fileForTarget(target)
	if file == "" {
		return nil, fmt.Errorf("unknown memory target: %s", target)
	}
	fullPath := filepath.Join(s.BaseDir, file)

	current := ""
	if b, err := os.ReadFile(fullPath); err == nil {
		current = string(b)
	}

	switch action {
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
	case "extract":
		candidates := extractMemoryCandidates(content, 12)
		updated, added, skipped := appendUniqueBullets(current, candidates)
		if len(added) == 0 {
			return map[string]any{"success": true, "target": target, "file": fullPath, "added": added, "skipped": skipped, "candidates": candidates}, nil
		}
		current = updated
		if err := os.WriteFile(fullPath, []byte(current), 0o644); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "target": target, "file": fullPath, "added": added, "skipped": skipped, "candidates": candidates}, nil
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

func appendUniqueBullets(current string, candidates []string) (string, []string, []string) {
	seen := map[string]struct{}{}
	for _, line := range strings.Split(current, "\n") {
		key := normalizeMemoryLine(line)
		if key != "" {
			seen[key] = struct{}{}
		}
	}

	updated := current
	added := make([]string, 0, len(candidates))
	skipped := make([]string, 0)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(strings.TrimPrefix(candidate, "-"))
		if candidate == "" {
			continue
		}
		key := normalizeMemoryLine(candidate)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			skipped = append(skipped, candidate)
			continue
		}
		if updated != "" && !strings.HasSuffix(updated, "\n") {
			updated += "\n"
		}
		updated += "- " + candidate + "\n"
		seen[key] = struct{}{}
		added = append(added, candidate)
	}
	return updated, added, skipped
}

func extractMemoryCandidates(text string, limit int) []string {
	sentences := splitSentences(text)
	out := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		lower := strings.ToLower(sentence)
		if !looksLikeMemoryCandidate(lower) || looksSensitiveMemoryCandidate(lower) {
			continue
		}
		candidate := trimRunes(oneLine(sentence), 220)
		key := normalizeMemoryLine(candidate)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, candidate)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func splitSentences(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？' || r == ';' || r == '；'
	})
}

func looksLikeMemoryCandidate(lower string) bool {
	patterns := []string{
		"i prefer", "i like", "i want", "my name", "my project", "we use", "we need", "project uses", "user prefers", "user likes", "remember that", "please remember",
		"我喜欢", "我希望", "我需要", "我的", "我们使用", "项目使用", "用户偏好", "用户喜欢", "请记住", "记住",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func looksSensitiveMemoryCandidate(lower string) bool {
	patterns := []string{
		"password", "passwd", "token", "api key", "apikey", "secret", "private key", "access key", "sk-",
		"密码", "令牌", "密钥", "私钥",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func normalizeMemoryLine(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "-*• \t")
	s = strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("。.!！?？;；,，", r)
	})
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func trimRunes(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "..."
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
