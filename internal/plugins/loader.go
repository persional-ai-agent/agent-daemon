package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Manifest struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Version     string            `json:"version,omitempty"`
	Description string            `json:"description,omitempty"`
	Entry       string            `json:"entry,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Tool        *ToolManifest     `json:"tool,omitempty"`
	Provider    *ProviderManifest `json:"provider,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	File        string            `json:"file,omitempty"`
}

type ToolManifest struct {
	Command        string         `json:"command"`
	Args           []string       `json:"args,omitempty"`
	Schema         map[string]any `json:"schema"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
	PassContext    bool           `json:"pass_context,omitempty"`
}

type ProviderManifest struct {
	Command        string   `json:"command"`
	Args           []string `json:"args,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
	Model          string   `json:"model,omitempty"`
}

func (m Manifest) IsEnabled() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}

func ValidateManifest(m Manifest) error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	switch strings.ToLower(strings.TrimSpace(m.Type)) {
	case "tool":
	case "provider":
	default:
		return fmt.Errorf("unsupported plugin type: %q", m.Type)
	}
	if strings.EqualFold(strings.TrimSpace(m.Type), "tool") {
		if m.Tool == nil {
			return fmt.Errorf("tool spec is required for type=tool")
		}
		if strings.TrimSpace(m.Tool.Command) == "" {
			return fmt.Errorf("tool.command is required")
		}
		if len(m.Tool.Schema) == 0 {
			return fmt.Errorf("tool.schema is required")
		}
	}
	if strings.EqualFold(strings.TrimSpace(m.Type), "provider") {
		if m.Provider == nil {
			return fmt.Errorf("provider spec is required for type=provider")
		}
		if strings.TrimSpace(m.Provider.Command) == "" {
			return fmt.Errorf("provider.command is required")
		}
	}
	return nil
}

func DefaultDirs(workdir string) []string {
	dirs := []string{}
	if strings.TrimSpace(workdir) != "" {
		dirs = append(dirs, filepath.Join(workdir, ".agent-daemon", "plugins"))
		dirs = append(dirs, filepath.Join(workdir, "plugins"))
	}
	return dirs
}

func LoadFromDirs(dirs []string) ([]Manifest, error) {
	out := make([]Manifest, 0)
	for _, dir := range dirs {
		entries, err := os.ReadDir(strings.TrimSpace(dir))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				continue
			}
			full := filepath.Join(dir, e.Name())
			bs, err := os.ReadFile(full)
			if err != nil {
				return nil, err
			}
			var m Manifest
			if err := json.Unmarshal(bs, &m); err != nil {
				return nil, err
			}
			if err := ValidateManifest(m); err != nil {
				continue
			}
			m.File = full
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type == out[j].Type {
			return out[i].Name < out[j].Name
		}
		return out[i].Type < out[j].Type
	})
	return out, nil
}
