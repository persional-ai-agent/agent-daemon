package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type RuntimeToolPlugin struct {
	manifest Manifest
	schema   core.ToolSchema
	command  string
	args     []string
	env      map[string]string
	timeout  time.Duration
}

func BuildToolPlugin(m Manifest) (*RuntimeToolPlugin, error) {
	if err := ValidateManifest(m); err != nil {
		return nil, err
	}
	spec := m.Tool
	schema := core.ToolSchema{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:        strings.TrimSpace(m.Name),
			Description: strings.TrimSpace(m.Description),
			Parameters:  spec.Schema,
		},
	}
	if v, ok := spec.Schema["description"].(string); ok && strings.TrimSpace(schema.Function.Description) == "" {
		schema.Function.Description = strings.TrimSpace(v)
	}
	if schema.Function.Description == "" {
		schema.Function.Description = "plugin tool: " + strings.TrimSpace(m.Name)
	}
	cmdPath := strings.TrimSpace(spec.Command)
	if !filepath.IsAbs(cmdPath) && strings.TrimSpace(m.File) != "" {
		cmdPath = filepath.Join(filepath.Dir(strings.TrimSpace(m.File)), cmdPath)
	}
	timeout := time.Duration(spec.TimeoutSeconds) * time.Second
	if spec.TimeoutSeconds <= 0 {
		timeout = 30 * time.Second
	}
	return &RuntimeToolPlugin{
		manifest: m,
		schema:   schema,
		command:  cmdPath,
		args:     append([]string{}, spec.Args...),
		env:      m.Env,
		timeout:  timeout,
	}, nil
}

func (p *RuntimeToolPlugin) Name() string {
	return p.schema.Function.Name
}

func (p *RuntimeToolPlugin) Schema() core.ToolSchema {
	return p.schema
}

func (p *RuntimeToolPlugin) Call(ctx context.Context, args map[string]any, tc tools.ToolContext) (map[string]any, error) {
	callCtx := ctx
	var cancel context.CancelFunc
	if p.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}
	payload := map[string]any{"args": args}
	if p.manifest.Tool != nil && p.manifest.Tool.PassContext {
		payload["context"] = map[string]any{
			"session_id": tc.SessionID,
			"workdir":    tc.Workdir,
			"gateway": map[string]any{
				"platform":   tc.GatewayPlatform,
				"chat_id":    tc.GatewayChatID,
				"chat_type":  tc.GatewayChatType,
				"user_id":    tc.GatewayUserID,
				"user_name":  tc.GatewayUserName,
				"message_id": tc.GatewayMessageID,
				"thread_id":  tc.GatewayThreadID,
			},
		}
	}
	in, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(callCtx, p.command, p.args...)
	cmd.Stdin = strings.NewReader(string(in))
	if strings.TrimSpace(tc.Workdir) != "" {
		cmd.Dir = strings.TrimSpace(tc.Workdir)
	}
	cmd.Env = os.Environ()
	for k, v := range p.env {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		cmd.Env = append(cmd.Env, key+"="+v)
	}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return nil, fmt.Errorf("plugin %s failed: %v: %s", p.Name(), err, truncatePluginOutput(text))
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

func truncatePluginOutput(s string) string {
	if len(s) <= 400 {
		return s
	}
	return s[:400] + "..."
}

func RegisterToolPlugins(registry *tools.Registry, manifests []Manifest) (int, error) {
	count := 0
	for _, m := range manifests {
		if !m.IsEnabled() || !strings.EqualFold(strings.TrimSpace(m.Type), "tool") {
			continue
		}
		tp, err := BuildToolPlugin(m)
		if err != nil {
			return count, fmt.Errorf("plugin %s: %w", strings.TrimSpace(m.Name), err)
		}
		registry.Register(tp)
		count++
	}
	return count, nil
}
