package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Store struct {
	BaseDir string
}

type memoryEntry struct {
	ID            string  `json:"id"`
	Target        string  `json:"target"`
	Content       string  `json:"content"`
	SourceSession string  `json:"source_session_id,omitempty"`
	SourceTurn    string  `json:"source_turn_id,omitempty"`
	Confidence    float64 `json:"confidence,omitempty"`
	Provider      string  `json:"provider,omitempty"`
	Revoked       bool    `json:"revoked"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type memoryState struct {
	ExternalProvider string `json:"external_provider"`
	ExternalEnabled  bool   `json:"external_enabled"`
	UpdatedAt        string `json:"updated_at"`
}

func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &Store{BaseDir: baseDir}, nil
}

func (s *Store) Manage(action, target, content, oldText string) (map[string]any, error) {
	return s.ManageWithContext(action, target, content, oldText, nil)
}

func (s *Store) ManageWithContext(action, target, content, oldText string, extra map[string]any) (map[string]any, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		target = "memory"
	}
	if target == "status" {
		target = "memory"
	}

	switch action {
	case "status":
		st, err := s.loadState()
		if err != nil {
			return nil, err
		}
		entries, _ := s.loadEntries(target)
		active := 0
		for _, e := range entries {
			if !e.Revoked {
				active++
			}
		}
		return map[string]any{"success": true, "target": target, "external_provider": st.ExternalProvider, "external_enabled": st.ExternalEnabled, "active_entries": active, "total_entries": len(entries), "updated_at": st.UpdatedAt}, nil
	case "off":
		st, err := s.loadState()
		if err != nil {
			return nil, err
		}
		st.ExternalEnabled = false
		st.UpdatedAt = time.Now().Format(time.RFC3339)
		if err := s.saveState(st); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "action": "off", "external_provider": st.ExternalProvider, "external_enabled": st.ExternalEnabled}, nil
	case "on":
		st, err := s.loadState()
		if err != nil {
			return nil, err
		}
		st.ExternalEnabled = true
		st.UpdatedAt = time.Now().Format(time.RFC3339)
		if err := s.saveState(st); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "action": "on", "external_provider": st.ExternalProvider, "external_enabled": st.ExternalEnabled}, nil
	case "reset":
		if err := os.WriteFile(filepath.Join(s.BaseDir, s.fileForTarget(target)), []byte(""), 0o644); err != nil {
			return nil, err
		}
		if err := os.WriteFile(s.entriesPath(target), []byte("[]\n"), 0o644); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "action": "reset", "target": target, "reset": true}, nil
	case "list":
		entries, err := s.loadEntries(target)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			if e.Revoked {
				continue
			}
			items = append(items, map[string]any{"id": e.ID, "content": e.Content, "confidence": e.Confidence, "source_session_id": e.SourceSession, "source_turn_id": e.SourceTurn, "provider": e.Provider, "created_at": e.CreatedAt, "updated_at": e.UpdatedAt})
		}
		return map[string]any{"success": true, "target": target, "count": len(items), "entries": items}, nil
	case "revoke":
		id := strings.TrimSpace(content)
		if id == "" {
			return map[string]any{"success": false, "error": "content must be entry id for revoke"}, nil
		}
		entries, err := s.loadEntries(target)
		if err != nil {
			return nil, err
		}
		revoked := false
		now := time.Now().Format(time.RFC3339)
		for i := range entries {
			if entries[i].ID == id && !entries[i].Revoked {
				entries[i].Revoked = true
				entries[i].UpdatedAt = now
				revoked = true
				break
			}
		}
		if !revoked {
			return map[string]any{"success": false, "error": "entry id not found"}, nil
		}
		if err := s.saveEntries(target, entries); err != nil {
			return nil, err
		}
		if err := s.syncMarkdownFromEntries(target, entries); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "target": target, "revoked": true, "id": id}, nil
	case "insights":
		entries, err := s.loadEntries(target)
		if err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "target": target, "insights": buildInsights(entries)}, nil
	}

	file := s.fileForTarget(target)
	if file == "" {
		return nil, fmt.Errorf("unknown memory target: %s", target)
	}
	fullPath := filepath.Join(s.BaseDir, file)
	entries, err := s.loadEntries(target)
	if err != nil {
		return nil, err
	}

	sourceSession := strings.TrimSpace(strExtra(extra, "session_id"))
	sourceTurn := strings.TrimSpace(strExtra(extra, "turn_id"))
	provider := strings.TrimSpace(strExtra(extra, "provider"))
	if provider == "" {
		provider = "local"
	}
	confidence := floatExtra(extra, "confidence", 0.8)
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}
	now := time.Now().Format(time.RFC3339)

	switch action {
	case "add":
		line := strings.TrimSpace(content)
		if line == "" {
			return map[string]any{"success": false, "error": "empty content"}, nil
		}
		entries = append(entries, memoryEntry{ID: newEntryID(), Target: target, Content: line, SourceSession: sourceSession, SourceTurn: sourceTurn, Confidence: confidence, Provider: provider, CreatedAt: now, UpdatedAt: now})
	case "replace", "update":
		if strings.TrimSpace(oldText) == "" {
			return map[string]any{"success": false, "error": "old_text required for replace"}, nil
		}
		hit := false
		for i := range entries {
			if entries[i].Revoked {
				continue
			}
			if strings.Contains(entries[i].Content, oldText) {
				entries[i].Content = strings.Replace(entries[i].Content, oldText, content, 1)
				entries[i].SourceSession = sourceSession
				entries[i].SourceTurn = sourceTurn
				entries[i].Confidence = confidence
				entries[i].Provider = provider
				entries[i].UpdatedAt = now
				hit = true
				break
			}
		}
		if !hit {
			return map[string]any{"success": false, "error": "old_text not found"}, nil
		}
	case "delete", "remove":
		if strings.TrimSpace(content) == "" {
			return map[string]any{"success": false, "error": "content required for delete"}, nil
		}
		for i := range entries {
			if entries[i].Revoked {
				continue
			}
			if strings.Contains(entries[i].Content, content) {
				entries[i].Revoked = true
				entries[i].UpdatedAt = now
			}
		}
	case "extract":
		candidates := extractMemoryCandidates(content, 12)
		added := []string{}
		skipped := []string{}
		seen := map[string]struct{}{}
		for _, e := range entries {
			if e.Revoked {
				continue
			}
			seen[normalizeMemoryLine(e.Content)] = struct{}{}
		}
		for _, c := range candidates {
			key := normalizeMemoryLine(c)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				skipped = append(skipped, c)
				continue
			}
			entries = append(entries, memoryEntry{ID: newEntryID(), Target: target, Content: strings.TrimSpace(c), SourceSession: sourceSession, SourceTurn: sourceTurn, Confidence: confidence, Provider: provider, CreatedAt: now, UpdatedAt: now})
			seen[key] = struct{}{}
			added = append(added, c)
		}
		if err := s.saveEntries(target, entries); err != nil {
			return nil, err
		}
		if err := s.syncMarkdownFromEntries(target, entries); err != nil {
			return nil, err
		}
		return map[string]any{"success": true, "target": target, "file": fullPath, "added": added, "skipped": skipped, "candidates": candidates}, nil
	default:
		return nil, fmt.Errorf("unsupported memory action: %s", action)
	}

	if err := s.saveEntries(target, entries); err != nil {
		return nil, err
	}
	if err := s.syncMarkdownFromEntries(target, entries); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "target": target, "file": fullPath}, nil
}

func (s *Store) Snapshot() (map[string]string, error) {
	out := map[string]string{"memory": "", "user": ""}
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
	b, err := os.ReadFile(filepath.Join(s.BaseDir, file))
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

func (s *Store) entriesPath(target string) string {
	return filepath.Join(s.BaseDir, strings.ToUpper(strings.TrimSuffix(s.fileForTarget(target), ".md"))+".entries.json")
}

func (s *Store) loadEntries(target string) ([]memoryEntry, error) {
	path := s.entriesPath(target)
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []memoryEntry{}, nil
		}
		return nil, err
	}
	var out []memoryEntry
	if err := json.Unmarshal(bs, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) saveEntries(target string, entries []memoryEntry) error {
	path := s.entriesPath(target)
	bs, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func (s *Store) syncMarkdownFromEntries(target string, entries []memoryEntry) error {
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Revoked {
			continue
		}
		lines = append(lines, "- "+strings.TrimSpace(e.Content))
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(filepath.Join(s.BaseDir, s.fileForTarget(target)), []byte(content), 0o644)
}

func (s *Store) statePath() string {
	return filepath.Join(s.BaseDir, "MEMORY.state.json")
}

func (s *Store) loadState() (memoryState, error) {
	defaultProvider := strings.TrimSpace(os.Getenv("AGENT_MEMORY_PROVIDER"))
	if defaultProvider == "" {
		defaultProvider = "local"
	}
	out := memoryState{ExternalProvider: defaultProvider, ExternalEnabled: true, UpdatedAt: time.Now().Format(time.RFC3339)}
	bs, err := os.ReadFile(s.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return memoryState{}, err
	}
	if err := json.Unmarshal(bs, &out); err != nil {
		return memoryState{}, err
	}
	if strings.TrimSpace(out.ExternalProvider) == "" {
		out.ExternalProvider = defaultProvider
	}
	return out, nil
}

func (s *Store) saveState(st memoryState) error {
	bs, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.statePath(), bs, 0o644)
}

func buildInsights(entries []memoryEntry) map[string]any {
	preferences := []string{}
	facts := []string{}
	topicsCount := map[string]int{}
	for _, e := range entries {
		if e.Revoked {
			continue
		}
		text := strings.TrimSpace(e.Content)
		lower := strings.ToLower(text)
		if strings.Contains(lower, "prefer") || strings.Contains(lower, "喜欢") || strings.Contains(lower, "希望") {
			preferences = append(preferences, text)
		} else {
			facts = append(facts, text)
		}
		for _, w := range strings.Fields(lower) {
			w = strings.Trim(w, ",.!?;:()[]{}\"'“”‘’")
			if len(w) < 4 {
				continue
			}
			topicsCount[w]++
		}
	}
	type kv struct {
		K string
		V int
	}
	pairs := make([]kv, 0, len(topicsCount))
	for k, v := range topicsCount {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].V == pairs[j].V {
			return pairs[i].K < pairs[j].K
		}
		return pairs[i].V > pairs[j].V
	})
	topics := make([]string, 0, 8)
	for i := 0; i < len(pairs) && i < 8; i++ {
		topics = append(topics, pairs[i].K)
	}
	return map[string]any{"preferences": preferences, "facts": facts, "recent_topics": topics, "count": len(preferences) + len(facts)}
}

func newEntryID() string { return fmt.Sprintf("mem_%d", time.Now().UnixNano()) }

func strExtra(extra map[string]any, key string) string {
	if extra == nil {
		return ""
	}
	if v, ok := extra[key]; ok {
		s, _ := v.(string)
		return strings.TrimSpace(s)
	}
	return ""
}

func floatExtra(extra map[string]any, key string, def float64) float64 {
	if extra == nil {
		return def
	}
	v, ok := extra[key]
	if !ok {
		return def
	}
	switch vv := v.(type) {
	case float64:
		return vv
	case int:
		return float64(vv)
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(vv), 64); err == nil {
			return f
		}
	}
	return def
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
	patterns := []string{"i prefer", "i like", "i want", "my name", "my project", "we use", "we need", "project uses", "user prefers", "user likes", "remember that", "please remember", "我喜欢", "我希望", "我需要", "我的", "我们使用", "项目使用", "用户偏好", "用户喜欢", "请记住", "记住"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func looksSensitiveMemoryCandidate(lower string) bool {
	patterns := []string{"password", "passwd", "token", "api key", "apikey", "secret", "private key", "access key", "sk-", "密码", "令牌", "密钥", "私钥"}
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

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }
