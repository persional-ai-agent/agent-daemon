package tools

import (
	"context"
	"strings"
	"testing"
)

func TestBuildForegroundCommandLocal(t *testing.T) {
	cmd, err := buildForegroundCommand(context.Background(), "echo hi", "/tmp", ForegroundBackendOptions{Backend: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path == "" || !strings.Contains(strings.ToLower(cmd.Path), "bash") {
		t.Fatalf("expected bash command path, got %q", cmd.Path)
	}
	if cmd.Dir != "/tmp" {
		t.Fatalf("expected dir /tmp, got %q", cmd.Dir)
	}
}

func TestBuildForegroundCommandDocker(t *testing.T) {
	cmd, err := buildForegroundCommand(context.Background(), "echo hi", "/work", ForegroundBackendOptions{
		Backend:     "docker",
		DockerImage: "alpine:3.20",
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "docker run") {
		t.Fatalf("unexpected docker args: %v", cmd.Args)
	}
	if !strings.Contains(joined, "alpine:3.20") {
		t.Fatalf("expected image in args: %v", cmd.Args)
	}
}

func TestBuildForegroundCommandUnsupported(t *testing.T) {
	_, err := buildForegroundCommand(context.Background(), "echo hi", "", ForegroundBackendOptions{Backend: "ssh"})
	if err == nil {
		t.Fatal("expected unsupported backend error")
	}
}
