package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type ForegroundBackendOptions struct {
	Backend     string
	DockerImage string
}

type ProcessSession struct {
	ID         string
	Command    string
	StartedAt  time.Time
	Done       bool
	ExitCode   int
	OutputFile string
	Err        string
	cmd        *exec.Cmd
	stdinMu    sync.Mutex
	stdin      io.WriteCloser
}

type ProcessRegistry struct {
	mu      sync.Mutex
	baseDir string
	procs   map[string]*ProcessSession
}

func NewProcessRegistry(baseDir string) *ProcessRegistry {
	_ = os.MkdirAll(baseDir, 0o755)
	return &ProcessRegistry{baseDir: baseDir, procs: map[string]*ProcessSession{}}
}

func (r *ProcessRegistry) StartBackground(ctx context.Context, command, cwd string) (*ProcessSession, error) {
	id := uuid.NewString()
	outputFile := filepath.Join(r.baseDir, fmt.Sprintf("proc-%s.log", id))
	f, err := os.Create(outputFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	if cwd != "" {
		cmd.Dir = cwd
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		_ = stdin.Close()
		return nil, err
	}
	s := &ProcessSession{ID: id, Command: command, StartedAt: time.Now(), OutputFile: outputFile, cmd: cmd, stdin: stdin}

	r.mu.Lock()
	r.procs[id] = s
	r.mu.Unlock()

	go func() {
		err := cmd.Wait()
		_ = logFile.Close()
		s.stdinMu.Lock()
		if s.stdin != nil {
			_ = s.stdin.Close()
			s.stdin = nil
		}
		s.stdinMu.Unlock()
		r.mu.Lock()
		defer r.mu.Unlock()
		s.Done = true
		if err != nil {
			s.Err = err.Error()
			if ee, ok := err.(*exec.ExitError); ok {
				s.ExitCode = ee.ExitCode()
			} else {
				s.ExitCode = -1
			}
		} else {
			s.ExitCode = 0
		}
	}()
	return s, nil
}

func (r *ProcessRegistry) Poll(id string) (*ProcessSession, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.procs[id]
	return s, ok
}

func (r *ProcessRegistry) Stop(id string) error {
	// Backwards-compatible: Stop behaves like a hard kill.
	return r.Kill(id)
}

func (r *ProcessRegistry) Terminate(id string) error {
	r.mu.Lock()
	s, ok := r.procs[id]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown process: %s", id)
	}
	if s.Done || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	_ = s.cmd.Process.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		done := s.Done
		r.mu.Unlock()
		if done {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return s.cmd.Process.Kill()
}

func (r *ProcessRegistry) Kill(id string) error {
	r.mu.Lock()
	s, ok := r.procs[id]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown process: %s", id)
	}
	if s.Done || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	return s.cmd.Process.Kill()
}

func (r *ProcessRegistry) Write(id string, input string) error {
	r.mu.Lock()
	s, ok := r.procs[id]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown process: %s", id)
	}
	if s.Done {
		return fmt.Errorf("process already finished: %s", id)
	}
	s.stdinMu.Lock()
	defer s.stdinMu.Unlock()
	if s.stdin == nil {
		return fmt.Errorf("stdin not available: %s", id)
	}
	_, err := io.WriteString(s.stdin, input)
	return err
}

func (r *ProcessRegistry) List(includeDone bool, limit int) []ProcessSession {
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ProcessSession, 0, len(r.procs))
	for _, s := range r.procs {
		if s == nil {
			continue
		}
		if !includeDone && s.Done {
			continue
		}
		out = append(out, ProcessSession{
			ID:         s.ID,
			Command:    s.Command,
			StartedAt:  s.StartedAt,
			Done:       s.Done,
			ExitCode:   s.ExitCode,
			OutputFile: s.OutputFile,
			Err:        s.Err,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func RunForeground(ctx context.Context, command, cwd string, timeoutSec int) (string, int, error) {
	return RunForegroundWithOptions(ctx, command, cwd, timeoutSec, ForegroundBackendOptions{Backend: "local"})
}

func RunForegroundWithOptions(ctx context.Context, command, cwd string, timeoutSec int, opt ForegroundBackendOptions) (string, int, error) {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd, err := buildForegroundCommand(cmdCtx, command, cwd, opt)
	if err != nil {
		return "", -1, err
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if cmdCtx.Err() == context.DeadlineExceeded {
		return out.String(), 124, fmt.Errorf("command timed out after %d seconds", timeoutSec)
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return out.String(), ee.ExitCode(), nil
		}
		return out.String(), -1, err
	}
	return out.String(), 0, nil
}

func buildForegroundCommand(ctx context.Context, command, cwd string, opt ForegroundBackendOptions) (*exec.Cmd, error) {
	backend := strings.ToLower(strings.TrimSpace(opt.Backend))
	if backend == "" {
		backend = "local"
	}
	switch backend {
	case "local":
		cmd := exec.CommandContext(ctx, "bash", "-lc", command)
		if cwd != "" {
			cmd.Dir = cwd
		}
		return cmd, nil
	case "docker":
		image := strings.TrimSpace(opt.DockerImage)
		if image == "" {
			image = "alpine:3.20"
		}
		workdir := "/workspace"
		args := []string{"run", "--rm", "-i"}
		if strings.TrimSpace(cwd) != "" {
			args = append(args, "-v", fmt.Sprintf("%s:%s", cwd, workdir), "-w", workdir)
		}
		args = append(args, image, "sh", "-lc", command)
		return exec.CommandContext(ctx, "docker", args...), nil
	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}
}
