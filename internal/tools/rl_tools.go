package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type rlState struct {
	SelectedEnvironment string                 `json:"selected_environment"`
	Config              map[string]any         `json:"config"`
	Training            map[string]any         `json:"training"`
	UpdatedAt           string                 `json:"updated_at"`
}

func rlStatePath(workdir string) (string, error) {
	root, err := normalizedWorkdir(workdir)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".agent-daemon", "rl_state.json"), nil
}

func loadRLState(workdir string) (rlState, string, error) {
	p, err := rlStatePath(workdir)
	if err != nil {
		return rlState{}, "", err
	}
	bs, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return rlState{Config: map[string]any{}, Training: map[string]any{}, UpdatedAt: time.Now().Format(time.RFC3339)}, p, nil
		}
		return rlState{}, "", err
	}
	var st rlState
	if err := json.Unmarshal(bs, &st); err != nil {
		return rlState{}, "", err
	}
	if st.Config == nil {
		st.Config = map[string]any{}
	}
	if st.Training == nil {
		st.Training = map[string]any{}
	}
	return st, p, nil
}

func saveRLState(p string, st rlState) error {
	st.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, bs, 0o644)
}

func envList() []string {
	raw := strings.TrimSpace(os.Getenv("RL_ENVIRONMENTS"))
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func (b *BuiltinTools) rlListEnvironments(_ context.Context, _ map[string]any, tc ToolContext) (map[string]any, error) {
	envs := envList()
	return map[string]any{"success": true, "environments": envs, "count": len(envs)}, nil
}

func (b *BuiltinTools) rlSelectEnvironment(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	env := strings.TrimSpace(strArg(args, "environment"))
	if env == "" {
		return nil, errors.New("environment required")
	}
	st, p, err := loadRLState(tc.Workdir)
	if err != nil {
		return nil, err
	}
	st.SelectedEnvironment = env
	if err := saveRLState(p, st); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "selected_environment": env, "written": p}, nil
}

func (b *BuiltinTools) rlGetCurrentConfig(_ context.Context, _ map[string]any, tc ToolContext) (map[string]any, error) {
	st, _, err := loadRLState(tc.Workdir)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "selected_environment": st.SelectedEnvironment, "config": st.Config}, nil
}

func (b *BuiltinTools) rlEditConfig(_ context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	key := strings.TrimSpace(strArg(args, "key"))
	if key == "" {
		return nil, errors.New("key required")
	}
	value := args["value"]
	st, p, err := loadRLState(tc.Workdir)
	if err != nil {
		return nil, err
	}
	if st.Config == nil {
		st.Config = map[string]any{}
	}
	st.Config[key] = value
	if err := saveRLState(p, st); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "key": key, "written": p}, nil
}

func (b *BuiltinTools) rlStartTraining(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	if b.proc == nil {
		return nil, errors.New("process registry unavailable")
	}
	cmdTmpl := strings.TrimSpace(os.Getenv("RL_TRAIN_COMMAND"))
	if cmdTmpl == "" {
		return map[string]any{"success": false, "available": false, "error": "RL_TRAIN_COMMAND not set"}, nil
	}
	st, p, err := loadRLState(tc.Workdir)
	if err != nil {
		return nil, err
	}
	env := st.SelectedEnvironment
	if v := strings.TrimSpace(strArg(args, "environment")); v != "" {
		env = v
	}
	command := strings.ReplaceAll(cmdTmpl, "{env}", env)
	s, err := b.proc.StartBackground(ctx, command, tc.Workdir)
	if err != nil {
		return nil, err
	}
	st.Training = map[string]any{
		"session_id":  s.ID,
		"command":     command,
		"started_at":  s.StartedAt.Format(time.RFC3339),
		"output_file": s.OutputFile,
	}
	if err := saveRLState(p, st); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "session_id": s.ID, "output_file": s.OutputFile, "command": command}, nil
}

func (b *BuiltinTools) rlStopTraining(_ context.Context, _ map[string]any, tc ToolContext) (map[string]any, error) {
	if b.proc == nil {
		return nil, errors.New("process registry unavailable")
	}
	st, p, err := loadRLState(tc.Workdir)
	if err != nil {
		return nil, err
	}
	id, _ := st.Training["session_id"].(string)
	if strings.TrimSpace(id) == "" {
		return map[string]any{"success": true, "stopped": false, "reason": "no active training session_id"}, nil
	}
	if err := b.proc.Stop(id); err != nil {
		return nil, err
	}
	st.Training["stopped_at"] = time.Now().Format(time.RFC3339)
	if err := saveRLState(p, st); err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "stopped": true, "session_id": id}, nil
}

func (b *BuiltinTools) rlCheckStatus(_ context.Context, _ map[string]any, tc ToolContext) (map[string]any, error) {
	if b.proc == nil {
		return nil, errors.New("process registry unavailable")
	}
	st, _, err := loadRLState(tc.Workdir)
	if err != nil {
		return nil, err
	}
	id, _ := st.Training["session_id"].(string)
	if strings.TrimSpace(id) == "" {
		return map[string]any{"success": true, "running": false}, nil
	}
	s, ok := b.proc.Poll(id)
	if !ok {
		return map[string]any{"success": true, "running": false, "session_id": id, "status": "unknown"}, nil
	}
	return map[string]any{"success": true, "running": !s.Done, "session_id": id, "status": statusFromDone(s.Done), "exit_code": s.ExitCode, "output_file": s.OutputFile, "error": s.Err}, nil
}

func (b *BuiltinTools) rlGetResults(_ context.Context, _ map[string]any, tc ToolContext) (map[string]any, error) {
	st, _, err := loadRLState(tc.Workdir)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "training": st.Training, "config": st.Config, "environment": st.SelectedEnvironment}, nil
}

// Compatibility no-ops for Hermes names we don't fully model.
func (b *BuiltinTools) rlListRuns(_ context.Context, _ map[string]any, _ ToolContext) (map[string]any, error) {
	return map[string]any{"success": true, "runs": []any{}, "count": 0}, nil
}

func (b *BuiltinTools) rlTestInference(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	cmdTmpl := strings.TrimSpace(os.Getenv("RL_INFER_COMMAND"))
	if cmdTmpl == "" {
		return map[string]any{"success": false, "available": false, "error": "RL_INFER_COMMAND not set"}, nil
	}
	st, _, err := loadRLState(tc.Workdir)
	if err != nil {
		return nil, err
	}
	env := st.SelectedEnvironment
	if v := strings.TrimSpace(strArg(args, "environment")); v != "" {
		env = v
	}
	command := strings.ReplaceAll(cmdTmpl, "{env}", env)
	timeout := intArg(args, "timeout", 180)
	out, code, runErr := RunForeground(ctx, command, tc.Workdir, timeout)
	res := map[string]any{"success": runErr == nil && code == 0, "exit_code": code, "output": out, "command": command, "error": nil}
	if runErr != nil {
		res["error"] = runErr.Error()
	}
	return res, nil
}

func rlSelectEnvParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"environment": map[string]any{"type": "string"}}, "required": []string{"environment"}}
}

func rlEditConfigParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"key": map[string]any{"type": "string"}, "value": map[string]any{}}, "required": []string{"key", "value"}}
}

func rlStartTrainingParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"environment": map[string]any{"type": "string"}}}
}

var _ = fmt.Sprintf
