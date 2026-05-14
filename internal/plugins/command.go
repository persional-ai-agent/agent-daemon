package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CommandInfo struct {
	Plugin      string `json:"plugin"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	File        string `json:"file,omitempty"`
}

type DashboardInfo struct {
	Plugin      string         `json:"plugin"`
	Name        string         `json:"name"`
	Label       string         `json:"label,omitempty"`
	Description string         `json:"description,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	Entry       string         `json:"entry,omitempty"`
	CSS         string         `json:"css,omitempty"`
	API         string         `json:"api,omitempty"`
	Tab         map[string]any `json:"tab,omitempty"`
	File        string         `json:"file,omitempty"`
}

func CommandInfos(manifests []Manifest) []CommandInfo {
	out := make([]CommandInfo, 0)
	for _, m := range manifests {
		if !m.IsEnabled() {
			continue
		}
		for _, cmd := range m.Commands {
			name := strings.TrimSpace(cmd.Name)
			if name == "" {
				continue
			}
			out = append(out, CommandInfo{
				Plugin:      strings.TrimSpace(m.Name),
				Name:        name,
				Description: strings.TrimSpace(cmd.Description),
				File:        m.File,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Plugin == out[j].Plugin {
			return out[i].Name < out[j].Name
		}
		return out[i].Plugin < out[j].Plugin
	})
	return out
}

func DashboardInfos(manifests []Manifest) []DashboardInfo {
	out := make([]DashboardInfo, 0)
	for _, m := range manifests {
		if !m.IsEnabled() {
			continue
		}
		if m.Dashboard != nil {
			out = append(out, dashboardInfo(m, *m.Dashboard))
		}
		for _, d := range m.Dashboards {
			out = append(out, dashboardInfo(m, d))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Plugin == out[j].Plugin {
			return out[i].Name < out[j].Name
		}
		return out[i].Plugin < out[j].Plugin
	})
	return out
}

func dashboardInfo(m Manifest, d DashboardManifest) DashboardInfo {
	name := strings.TrimSpace(d.Name)
	if name == "" {
		name = strings.TrimSpace(m.Name)
	}
	label := strings.TrimSpace(d.Label)
	if label == "" {
		label = name
	}
	desc := strings.TrimSpace(d.Description)
	if desc == "" {
		desc = strings.TrimSpace(m.Description)
	}
	return DashboardInfo{
		Plugin:      strings.TrimSpace(m.Name),
		Name:        name,
		Label:       label,
		Description: desc,
		Icon:        strings.TrimSpace(d.Icon),
		Entry:       resolvePluginPath(m, d.Entry),
		CSS:         resolvePluginPath(m, d.CSS),
		API:         resolvePluginPath(m, d.API),
		Tab:         d.Tab,
		File:        m.File,
	}
}

func RunCommand(ctx context.Context, manifests []Manifest, name string, argv []string, workdir string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	for _, m := range manifests {
		if !m.IsEnabled() {
			continue
		}
		for _, spec := range m.Commands {
			if strings.EqualFold(strings.TrimSpace(spec.Name), name) {
				return runCommandSpec(ctx, m, spec, argv, workdir)
			}
		}
	}
	return nil, fmt.Errorf("plugin command %q not found", name)
}

func runCommandSpec(ctx context.Context, manifest Manifest, spec CommandManifest, argv []string, workdir string) (map[string]any, error) {
	callCtx := ctx
	var cancel context.CancelFunc
	timeout := time.Duration(spec.TimeoutSeconds) * time.Second
	if spec.TimeoutSeconds <= 0 {
		timeout = 30 * time.Second
	}
	if timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmdPath := resolvePluginPath(manifest, spec.Command)
	args := append([]string{}, spec.Args...)
	args = append(args, argv...)
	payload := map[string]any{"args": argv}
	if spec.PassContext {
		payload["context"] = map[string]any{"workdir": workdir}
	}
	in, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(callCtx, cmdPath, args...)
	cmd.Stdin = strings.NewReader(string(in))
	if err := prepareCommand(cmd, manifest, cmdPath, workdir); err != nil {
		return nil, err
	}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return nil, fmt.Errorf("plugin command %s failed: %v: %s", spec.Name, err, truncatePluginOutput(text))
	}
	if text == "" {
		return map[string]any{"success": true}, nil
	}
	var parsed map[string]any
	if jerr := json.Unmarshal([]byte(text), &parsed); jerr == nil {
		return parsed, nil
	}
	return map[string]any{"success": true, "output": text}, nil
}

func resolvePluginPath(m Manifest, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) || strings.TrimSpace(m.File) == "" {
		return path
	}
	return filepath.Join(filepath.Dir(strings.TrimSpace(m.File)), path)
}
