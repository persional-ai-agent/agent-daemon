package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

const DefaultConfigFile = "config/config.ini"

type ConfigEntry struct {
	Key   string
	Value string
}

func ParseConfigKey(name string) (string, string, error) {
	name = strings.TrimSpace(name)
	idx := strings.LastIndex(name, ".")
	if idx <= 0 || idx == len(name)-1 {
		return "", "", fmt.Errorf("config key must use section.key format: %q", name)
	}
	section := strings.TrimSpace(name[:idx])
	key := strings.TrimSpace(name[idx+1:])
	if section == "" || key == "" {
		return "", "", fmt.Errorf("config key must use section.key format: %q", name)
	}
	return section, key, nil
}

func ConfigFilePath(path string) string {
	if strings.TrimSpace(path) != "" {
		return path
	}
	if envPath := strings.TrimSpace(os.Getenv("AGENT_CONFIG_FILE")); envPath != "" {
		return envPath
	}
	return DefaultConfigFile
}

func LoadConfigFile(path string) (*ini.File, error) {
	path = ConfigFilePath(path)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ini.Empty(), nil
		}
		return nil, err
	}
	return ini.Load(path)
}

func SaveConfigValue(path, dottedKey, value string) error {
	section, key, err := ParseConfigKey(dottedKey)
	if err != nil {
		return err
	}
	path = ConfigFilePath(path)
	f, err := LoadConfigFile(path)
	if err != nil {
		return err
	}
	f.Section(section).Key(key).SetValue(value)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return f.SaveTo(path)
}

func ReadConfigValue(path, dottedKey string) (string, bool, error) {
	section, key, err := ParseConfigKey(dottedKey)
	if err != nil {
		return "", false, err
	}
	f, err := LoadConfigFile(path)
	if err != nil {
		return "", false, err
	}
	sec := f.Section(section)
	if sec == nil || !sec.HasKey(key) {
		return "", false, nil
	}
	return sec.Key(key).String(), true, nil
}

func ListConfigValues(path string) ([]ConfigEntry, error) {
	f, err := LoadConfigFile(path)
	if err != nil {
		return nil, err
	}
	entries := make([]ConfigEntry, 0)
	for _, sec := range f.Sections() {
		if sec.Name() == ini.DefaultSection {
			continue
		}
		for _, key := range sec.Keys() {
			entries = append(entries, ConfigEntry{
				Key:   sec.Name() + "." + key.Name(),
				Value: key.String(),
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries, nil
}

func RedactConfigValue(key, value string) string {
	lower := strings.ToLower(key)
	for _, marker := range []string{"api_key", "token", "secret", "password"} {
		if strings.Contains(lower, marker) {
			if strings.TrimSpace(value) == "" {
				return ""
			}
			return "********"
		}
	}
	return value
}
