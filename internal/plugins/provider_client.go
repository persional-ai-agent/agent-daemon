package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/model"
)

type providerCommandClient struct {
	name     string
	model    string
	command  string
	args     []string
	timeout  time.Duration
	manifest Manifest
}

func (c *providerCommandClient) ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req := map[string]any{
		"provider": c.name,
		"model":    c.model,
		"messages": messages,
		"tools":    tools,
	}
	in, err := json.Marshal(req)
	if err != nil {
		return core.Message{}, err
	}
	cmd := exec.CommandContext(callCtx, c.command, c.args...)
	cmd.Stdin = strings.NewReader(string(in))
	if err := prepareCommand(cmd, c.manifest, c.command, ""); err != nil {
		return core.Message{}, err
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return core.Message{}, fmt.Errorf("provider plugin %s failed: %v: %s", c.name, err, truncatePluginText(string(out), 500))
	}
	// Compatible shapes:
	// 1) {"message": {...core.Message...}}
	// 2) {"choices":[{"message": {...}}]} (OpenAI-like)
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		return core.Message{}, fmt.Errorf("provider plugin %s returned non-json: %s", c.name, truncatePluginText(string(out), 500))
	}
	if msgRaw, ok := payload["message"]; ok {
		b, _ := json.Marshal(msgRaw)
		var msg core.Message
		if err := json.Unmarshal(b, &msg); err != nil {
			return core.Message{}, fmt.Errorf("provider plugin %s invalid message: %w", c.name, err)
		}
		if strings.TrimSpace(msg.Role) == "" {
			msg.Role = "assistant"
		}
		return msg, nil
	}
	if choicesRaw, ok := payload["choices"]; ok {
		b, _ := json.Marshal(choicesRaw)
		var choices []struct {
			Message core.Message `json:"message"`
		}
		if err := json.Unmarshal(b, &choices); err == nil && len(choices) > 0 {
			msg := choices[0].Message
			if strings.TrimSpace(msg.Role) == "" {
				msg.Role = "assistant"
			}
			return msg, nil
		}
	}
	if errText, _ := payload["error"].(string); strings.TrimSpace(errText) != "" {
		return core.Message{}, fmt.Errorf("provider plugin %s error: %s", c.name, errText)
	}
	return core.Message{}, fmt.Errorf("provider plugin %s response missing message", c.name)
}

func FindProviderManifest(providerName string, manifests []Manifest) (Manifest, bool) {
	key := strings.ToLower(strings.TrimSpace(providerName))
	for _, m := range ExpandRuntimeManifests(manifests) {
		if !m.IsEnabled() || !strings.EqualFold(strings.TrimSpace(m.Type), "provider") {
			continue
		}
		if strings.ToLower(strings.TrimSpace(m.Name)) == key {
			return m, true
		}
	}
	return Manifest{}, false
}

func NewProviderClient(providerName, selectedModel string, manifests []Manifest) (model.Client, bool, error) {
	m, ok := FindProviderManifest(providerName, manifests)
	if !ok {
		return nil, false, nil
	}
	if err := ValidateManifest(m); err != nil {
		return nil, true, err
	}
	spec := m.Provider
	cmdPath := strings.TrimSpace(spec.Command)
	if !filepath.IsAbs(cmdPath) && strings.TrimSpace(m.File) != "" {
		cmdPath = filepath.Join(filepath.Dir(strings.TrimSpace(m.File)), cmdPath)
	}
	timeout := 120 * time.Second
	if spec.TimeoutSeconds > 0 {
		timeout = time.Duration(spec.TimeoutSeconds) * time.Second
	}
	modelName := strings.TrimSpace(selectedModel)
	if modelName == "" {
		modelName = strings.TrimSpace(spec.Model)
	}
	return &providerCommandClient{
		name:     strings.TrimSpace(m.Name),
		model:    modelName,
		command:  cmdPath,
		args:     append([]string{}, spec.Args...),
		timeout:  timeout,
		manifest: m,
	}, true, nil
}

func ProviderNames(manifests []Manifest) []string {
	names := make([]string, 0, len(manifests))
	seen := map[string]bool{}
	for _, m := range ExpandRuntimeManifests(manifests) {
		if !m.IsEnabled() || !strings.EqualFold(strings.TrimSpace(m.Type), "provider") {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(m.Name))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

func truncatePluginText(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max || max <= 0 {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
