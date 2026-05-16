package tools

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SkillMetadata struct {
	Name            string `json:"name"`
	Source          string `json:"source"`
	Version         int    `json:"version"`
	UsageCount      int    `json:"usage_count"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	LastAction      string `json:"last_action,omitempty"`
	LastTriggerTask string `json:"last_trigger_task,omitempty"`
}

type SkillSnapshot struct {
	VersionID   string `json:"version_id"`
	Name        string `json:"name"`
	Action      string `json:"action"`
	FilePath    string `json:"file_path"`
	Content     string `json:"content"`
	CreatedAt   string `json:"created_at"`
	SessionID   string `json:"session_id,omitempty"`
	ContentHash string `json:"content_hash"`
}

func skillMetaRoot(workdir string) string {
	return filepath.Join(workdir, ".agent-daemon", "skills")
}

func skillMetaPath(workdir, name string) string {
	return filepath.Join(skillMetaRoot(workdir), "metadata", name+".json")
}

func skillAuditPath(workdir string) string {
	return filepath.Join(skillMetaRoot(workdir), "audit.log")
}

func skillHistoryPath(workdir, name, versionID string) string {
	return filepath.Join(skillMetaRoot(workdir), "history", name, versionID+".json")
}

func LoadSkillMetadata(workdir, name string) (SkillMetadata, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return SkillMetadata{}, errors.New("skill name required")
	}
	path := skillMetaPath(workdir, name)
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SkillMetadata{Name: name, Source: "local", Version: 0, UsageCount: 0}, nil
		}
		return SkillMetadata{}, err
	}
	var meta SkillMetadata
	if err := json.Unmarshal(bs, &meta); err != nil {
		return SkillMetadata{}, err
	}
	if strings.TrimSpace(meta.Name) == "" {
		meta.Name = name
	}
	if meta.Version < 0 {
		meta.Version = 0
	}
	if strings.TrimSpace(meta.Source) == "" {
		meta.Source = "local"
	}
	return meta, nil
}

func SaveSkillMetadata(workdir string, meta SkillMetadata) error {
	meta.Name = strings.TrimSpace(meta.Name)
	if meta.Name == "" {
		return errors.New("skill name required")
	}
	now := time.Now().Format(time.RFC3339)
	if strings.TrimSpace(meta.CreatedAt) == "" {
		meta.CreatedAt = now
	}
	meta.UpdatedAt = now
	if strings.TrimSpace(meta.Source) == "" {
		meta.Source = "local"
	}
	path := skillMetaPath(workdir, meta.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func TrackSkillUsage(workdir, name, triggerTask string) (SkillMetadata, error) {
	meta, err := LoadSkillMetadata(workdir, name)
	if err != nil {
		return SkillMetadata{}, err
	}
	meta.UsageCount++
	meta.LastAction = "view"
	meta.LastTriggerTask = strings.TrimSpace(triggerTask)
	if err := SaveSkillMetadata(workdir, meta); err != nil {
		return SkillMetadata{}, err
	}
	_ = AppendSkillAudit(workdir, map[string]any{"name": name, "action": "view", "usage_count": meta.UsageCount, "trigger_task": triggerTask})
	return meta, nil
}

func UpsertSkillMetadata(workdir, name, action, source, triggerTask string, bumpVersion bool) (SkillMetadata, error) {
	meta, err := LoadSkillMetadata(workdir, name)
	if err != nil {
		return SkillMetadata{}, err
	}
	meta.Name = name
	if bumpVersion {
		meta.Version++
	}
	if strings.TrimSpace(source) != "" {
		meta.Source = strings.TrimSpace(source)
	}
	meta.LastAction = strings.TrimSpace(action)
	meta.LastTriggerTask = strings.TrimSpace(triggerTask)
	if err := SaveSkillMetadata(workdir, meta); err != nil {
		return SkillMetadata{}, err
	}
	return meta, nil
}

func AppendSkillAudit(workdir string, entry map[string]any) error {
	if entry == nil {
		entry = map[string]any{}
	}
	entry["timestamp"] = time.Now().Format(time.RFC3339)
	path := skillAuditPath(workdir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	bs, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(bs, '\n'))
	return err
}

func SaveSkillSnapshot(workdir, name, action, filePath, content, sessionID string) (SkillSnapshot, error) {
	versionID := fmt.Sprintf("%d", time.Now().UnixNano())
	hash := sha1.Sum([]byte(content))
	snap := SkillSnapshot{
		VersionID:   versionID,
		Name:        name,
		Action:      action,
		FilePath:    filePath,
		Content:     content,
		CreatedAt:   time.Now().Format(time.RFC3339),
		SessionID:   strings.TrimSpace(sessionID),
		ContentHash: hex.EncodeToString(hash[:]),
	}
	path := skillHistoryPath(workdir, name, versionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return SkillSnapshot{}, err
	}
	bs, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return SkillSnapshot{}, err
	}
	if err := os.WriteFile(path, bs, 0o644); err != nil {
		return SkillSnapshot{}, err
	}
	_ = AppendSkillAudit(workdir, map[string]any{"name": name, "action": "snapshot", "version_id": versionID, "file_path": filePath})
	return snap, nil
}

func LoadSkillSnapshot(workdir, name, versionID string) (SkillSnapshot, error) {
	path := skillHistoryPath(workdir, name, versionID)
	bs, err := os.ReadFile(path)
	if err != nil {
		return SkillSnapshot{}, err
	}
	var snap SkillSnapshot
	if err := json.Unmarshal(bs, &snap); err != nil {
		return SkillSnapshot{}, err
	}
	return snap, nil
}

func ListSkillSnapshots(workdir, name string, limit int) ([]SkillSnapshot, error) {
	root := filepath.Join(skillMetaRoot(workdir), "history", name)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []SkillSnapshot{}, nil
		}
		return nil, err
	}
	items := make([]SkillSnapshot, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		bs, err := os.ReadFile(filepath.Join(root, e.Name()))
		if err != nil {
			continue
		}
		var snap SkillSnapshot
		if err := json.Unmarshal(bs, &snap); err != nil {
			continue
		}
		items = append(items, snap)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].VersionID > items[j].VersionID })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
