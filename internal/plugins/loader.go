package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Manifest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version,omitempty"`
	Entry   string `json:"entry,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
	File    string `json:"file,omitempty"`
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
			if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Type) == "" {
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
