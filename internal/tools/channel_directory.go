package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type ChannelDirectoryEntry struct {
	Platform   string `json:"platform"`
	ChatID     string `json:"chat_id"`
	ChatType   string `json:"chat_type,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	UserName   string `json:"user_name,omitempty"`
	GlobalID   string `json:"global_id,omitempty"`
	HomeTarget string `json:"home_target,omitempty"`
	LastSeenAt string `json:"last_seen_at"`
}

var channelDirectoryMu sync.Mutex

func channelDirectoryPath(workdir string) string {
	return filepath.Join(strings.TrimSpace(workdir), ".agent-daemon", "channel_directory.json")
}

func ListChannelDirectory(workdir string) ([]ChannelDirectoryEntry, error) {
	channelDirectoryMu.Lock()
	defer channelDirectoryMu.Unlock()
	return listChannelDirectoryLocked(workdir)
}

func listChannelDirectoryLocked(workdir string) ([]ChannelDirectoryEntry, error) {
	path := channelDirectoryPath(workdir)
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ChannelDirectoryEntry{}, nil
		}
		return nil, err
	}
	var items []ChannelDirectoryEntry
	if err := json.Unmarshal(bs, &items); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastSeenAt > items[j].LastSeenAt
	})
	return items, nil
}

func UpsertChannelDirectory(workdir string, entry ChannelDirectoryEntry) error {
	channelDirectoryMu.Lock()
	defer channelDirectoryMu.Unlock()

	entry.Platform = strings.ToLower(strings.TrimSpace(entry.Platform))
	entry.ChatID = strings.TrimSpace(entry.ChatID)
	entry.ChatType = strings.ToLower(strings.TrimSpace(entry.ChatType))
	entry.UserID = strings.TrimSpace(entry.UserID)
	entry.UserName = strings.TrimSpace(entry.UserName)
	entry.GlobalID = strings.TrimSpace(entry.GlobalID)
	entry.HomeTarget = strings.TrimSpace(entry.HomeTarget)
	if entry.Platform == "" || entry.ChatID == "" {
		return nil
	}
	if entry.LastSeenAt == "" {
		entry.LastSeenAt = time.Now().Format(time.RFC3339Nano)
	}

	items, err := listChannelDirectoryLocked(workdir)
	if err != nil {
		return err
	}

	found := false
	for i := range items {
		if items[i].Platform == entry.Platform && items[i].ChatID == entry.ChatID {
			items[i].ChatType = nonEmpty(entry.ChatType, items[i].ChatType)
			items[i].UserID = nonEmpty(entry.UserID, items[i].UserID)
			items[i].UserName = nonEmpty(entry.UserName, items[i].UserName)
			items[i].GlobalID = nonEmpty(entry.GlobalID, items[i].GlobalID)
			items[i].HomeTarget = nonEmpty(entry.HomeTarget, items[i].HomeTarget)
			items[i].LastSeenAt = entry.LastSeenAt
			found = true
			break
		}
	}
	if !found {
		items = append(items, entry)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastSeenAt > items[j].LastSeenAt
	})

	if err := os.MkdirAll(filepath.Dir(channelDirectoryPath(workdir)), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(channelDirectoryPath(workdir), out, 0o644)
}

func ClearChannelDirectoryGlobalID(workdir, platform, chatID string) error {
	channelDirectoryMu.Lock()
	defer channelDirectoryMu.Unlock()
	platform = strings.ToLower(strings.TrimSpace(platform))
	chatID = strings.TrimSpace(chatID)
	if platform == "" || chatID == "" {
		return nil
	}
	items, err := listChannelDirectoryLocked(workdir)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].Platform == platform && items[i].ChatID == chatID {
			items[i].GlobalID = ""
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastSeenAt > items[j].LastSeenAt
	})
	if err := os.MkdirAll(filepath.Dir(channelDirectoryPath(workdir)), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(channelDirectoryPath(workdir), out, 0o644)
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
