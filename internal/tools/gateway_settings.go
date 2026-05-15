package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var gatewaySettingsMu sync.Mutex

func gatewaySettingsPath(workdir string) string {
	return filepath.Join(strings.TrimSpace(workdir), ".agent-daemon", "gateway_settings.json")
}

func SetGatewaySetting(workdir, key, value string) error {
	workdir = strings.TrimSpace(workdir)
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if workdir == "" || key == "" {
		return nil
	}
	gatewaySettingsMu.Lock()
	defer gatewaySettingsMu.Unlock()
	m := map[string]string{}
	path := gatewaySettingsPath(workdir)
	if bs, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(bs, &m)
	}
	m[key] = value
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func GetGatewaySetting(workdir, key string) (string, error) {
	workdir = strings.TrimSpace(workdir)
	key = strings.TrimSpace(key)
	if workdir == "" || key == "" {
		return "", nil
	}
	gatewaySettingsMu.Lock()
	defer gatewaySettingsMu.Unlock()
	path := gatewaySettingsPath(workdir)
	bs, err := os.ReadFile(path)
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
	return strings.TrimSpace(m[key]), nil
}
