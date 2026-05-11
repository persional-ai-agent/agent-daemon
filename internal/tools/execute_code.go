package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/google/uuid"
)

type ExecuteCodeTool struct{}

func NewExecuteCodeTool() *ExecuteCodeTool { return &ExecuteCodeTool{} }

func (t *ExecuteCodeTool) Name() string { return "execute_code" }

func (t *ExecuteCodeTool) Schema() core.ToolSchema {
	return core.ToolSchema{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:        t.Name(),
			Description: "Execute a short Python script in an isolated subprocess (workdir-scoped). Returns stdout/stderr and exit code.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"code": map[string]any{
						"type":        "string",
						"description": "Python code to run",
					},
					"timeout_seconds": map[string]any{
						"type":        "integer",
						"description": "Timeout in seconds (default 30)",
					},
					"workdir": map[string]any{
						"type":        "string",
						"description": "Optional working directory relative to AGENT_WORKDIR",
					},
				},
				"required": []string{"code"},
			},
		},
	}
}

func (t *ExecuteCodeTool) Call(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	code := strArg(args, "code")
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("code required")
	}
	timeout := intArg(args, "timeout_seconds", 30)
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 300 {
		timeout = 300
	}

	wd := tc.Workdir
	if v := strings.TrimSpace(strArg(args, "workdir")); v != "" {
		var err error
		wd, err = resolvePathWithinWorkdir(tc.Workdir, v)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		wd, err = normalizedWorkdir(tc.Workdir)
		if err != nil {
			return nil, err
		}
	}

	tmpDir := filepath.Join(wd, ".agent-daemon", "execute_code")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, err
	}
	scriptPath := filepath.Join(tmpDir, "script-"+uuid.NewString()+".py")
	if err := os.WriteFile(scriptPath, []byte(code), 0o600); err != nil {
		return nil, err
	}
	defer os.Remove(scriptPath)

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "python3", "-I", scriptPath)
	cmd.Dir = wd
	cmd.Env = []string{
		"PYTHONUNBUFFERED=1",
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee := (&exec.ExitError{}); errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return map[string]any{
				"success":    false,
				"exit_code":  124,
				"stdout":     stdout.String(),
				"stderr":     stderr.String(),
				"timeout":    true,
				"workdir":    wd,
				"script_path": scriptPath,
			}, nil
		} else {
			return nil, fmt.Errorf("execute python: %w", err)
		}
	}
	success := err == nil && exitCode == 0
	return map[string]any{
		"success":     success,
		"exit_code":   exitCode,
		"stdout":      stdout.String(),
		"stderr":      stderr.String(),
		"timeout":     false,
		"workdir":     wd,
		"script_path": scriptPath,
	}, nil
}

