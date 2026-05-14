package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func sandboxEnabled(m Manifest) bool {
	if m.Sandbox == nil || m.Sandbox.Enabled == nil {
		return true
	}
	return *m.Sandbox.Enabled
}

func prepareCommand(cmd *exec.Cmd, m Manifest, commandPath, requestedWorkdir string) error {
	if !sandboxEnabled(m) {
		if strings.TrimSpace(requestedWorkdir) != "" {
			cmd.Dir = strings.TrimSpace(requestedWorkdir)
		}
		cmd.Env = os.Environ()
		for k, v := range m.Env {
			if key := strings.TrimSpace(k); key != "" {
				cmd.Env = append(cmd.Env, key+"="+v)
			}
		}
		return nil
	}
	pluginDir := filepath.Dir(strings.TrimSpace(m.File))
	if strings.TrimSpace(m.File) != "" {
		if !pathInside(pluginDir, commandPath) && (m.Sandbox == nil || !m.Sandbox.AllowOutsideCommand) {
			return fmt.Errorf("plugin command escapes plugin directory: %s", commandPath)
		}
	}
	cmd.Dir = pluginDir
	if m.Sandbox != nil {
		switch strings.ToLower(strings.TrimSpace(m.Sandbox.Workdir)) {
		case "workdir":
			if strings.TrimSpace(requestedWorkdir) != "" {
				cmd.Dir = strings.TrimSpace(requestedWorkdir)
			}
		case "plugin", "":
		default:
			return fmt.Errorf("unsupported plugin sandbox workdir: %s", m.Sandbox.Workdir)
		}
	}
	cmd.Env = sandboxEnv(m)
	return nil
}

func sandboxEnv(m Manifest) []string {
	if m.Sandbox != nil && m.Sandbox.AllowHostEnv {
		env := os.Environ()
		for k, v := range m.Env {
			if key := strings.TrimSpace(k); key != "" {
				env = append(env, key+"="+v)
			}
		}
		return env
	}
	allowed := map[string]struct{}{"PATH": {}, "HOME": {}, "TMPDIR": {}, "TEMP": {}, "TMP": {}}
	if m.Sandbox != nil {
		for _, key := range m.Sandbox.EnvPassthrough {
			key = strings.TrimSpace(key)
			if key != "" {
				allowed[key] = struct{}{}
			}
		}
	}
	env := make([]string, 0, len(allowed)+len(m.Env))
	for key := range allowed {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}
	env = append(env, "AGENT_PLUGIN_NAME="+strings.TrimSpace(m.Name))
	env = append(env, "AGENT_PLUGIN_DIR="+filepath.Dir(strings.TrimSpace(m.File)))
	for k, v := range m.Env {
		if key := strings.TrimSpace(k); key != "" {
			env = append(env, key+"="+v)
		}
	}
	return env
}
