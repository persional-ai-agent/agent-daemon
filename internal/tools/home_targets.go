package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var homeTargetsMu sync.Mutex

func homeTargetsPath(workdir string) string {
	return filepath.Join(strings.TrimSpace(workdir), ".agent-daemon", "home_targets.json")
}

func SetHomeTarget(workdir, platform, chatID string) error {
	platform = strings.ToLower(strings.TrimSpace(platform))
	chatID = strings.TrimSpace(chatID)
	if platform == "" || chatID == "" || strings.TrimSpace(workdir) == "" {
		return nil
	}
	homeTargetsMu.Lock()
	defer homeTargetsMu.Unlock()
	path := homeTargetsPath(workdir)
	m := map[string]string{}
	if bs, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(bs, &m)
	}
	m[platform] = chatID
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func GetHomeTarget(workdir, platform string) (string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "" || strings.TrimSpace(workdir) == "" {
		return "", nil
	}
	homeTargetsMu.Lock()
	defer homeTargetsMu.Unlock()
	bs, err := os.ReadFile(homeTargetsPath(workdir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	m := map[string]string{}
	if err := json.Unmarshal(bs, &m); err != nil {
		return "", err
	}
	return strings.TrimSpace(m[platform]), nil
}

func ResolveHomeTarget(workdir, platform string) string {
	if v := strings.TrimSpace(os.Getenv(HomeTargetEnvVar(platform))); v != "" {
		return v
	}
	v, _ := GetHomeTarget(workdir, platform)
	return strings.TrimSpace(v)
}

func ParseSetHomeArgs(args []string) (platform string, chatID string, err error) {
	if len(args) == 1 {
		p, c, pErr := ParseDeliveryTarget(strings.TrimSpace(args[0]))
		if pErr != nil || strings.TrimSpace(c) == "" {
			return "", "", fmt.Errorf("invalid sethome target")
		}
		return strings.ToLower(strings.TrimSpace(p)), strings.TrimSpace(c), nil
	}
	if len(args) == 2 {
		p := strings.ToLower(strings.TrimSpace(args[0]))
		c := strings.TrimSpace(args[1])
		if p == "" || c == "" {
			return "", "", fmt.Errorf("invalid sethome target")
		}
		return p, c, nil
	}
	return "", "", fmt.Errorf("invalid sethome args")
}
