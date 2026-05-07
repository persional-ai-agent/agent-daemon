package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ProcessSession struct {
	ID         string
	Command    string
	StartedAt  time.Time
	Done       bool
	ExitCode   int
	OutputFile string
	Err        string
	cmd        *exec.Cmd
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
	logFile, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	s := &ProcessSession{ID: id, Command: command, StartedAt: time.Now(), OutputFile: outputFile, cmd: cmd}

	r.mu.Lock()
	r.procs[id] = s
	r.mu.Unlock()

	go func() {
		err := cmd.Wait()
		_ = logFile.Close()
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

func RunForeground(ctx context.Context, command, cwd string, timeoutSec int) (string, int, error) {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-lc", command)
	if cwd != "" {
		cmd.Dir = cwd
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
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
