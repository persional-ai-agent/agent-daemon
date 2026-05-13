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
	_, err := buildForegroundCommand(context.Background(), "echo hi", "", ForegroundBackendOptions{Backend: "invalid"})
	if err == nil {
		t.Fatal("expected unsupported backend error")
	}
}

func TestBuildForegroundCommandSSH(t *testing.T) {
	cmd, err := buildForegroundCommand(context.Background(), "echo hi", "/work", ForegroundBackendOptions{
		Backend:          "ssh",
		SSHHost:          "example.com",
		SSHUser:          "root",
		SSHPort:          2222,
		SSHKeyPath:       "/tmp/key",
		SSHStrictHostKey: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "ssh") || !strings.Contains(joined, "root@example.com") {
		t.Fatalf("unexpected ssh args: %v", cmd.Args)
	}
	if !strings.Contains(joined, "-p 2222") {
		t.Fatalf("ssh port not set: %v", cmd.Args)
	}
}

func TestBuildForegroundCommandPodman(t *testing.T) {
	cmd, err := buildForegroundCommand(context.Background(), "echo hi", "/work", ForegroundBackendOptions{
		Backend:     "podman",
		PodmanImage: "alpine:3.20",
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "podman run") {
		t.Fatalf("unexpected podman args: %v", cmd.Args)
	}
	if !strings.Contains(joined, "alpine:3.20") {
		t.Fatalf("expected image in args: %v", cmd.Args)
	}
}

func TestBuildForegroundCommandSingularityRequiresImage(t *testing.T) {
	_, err := buildForegroundCommand(context.Background(), "echo hi", "/work", ForegroundBackendOptions{
		Backend: "singularity",
	})
	if err == nil || !strings.Contains(err.Error(), "requires singularity_image") {
		t.Fatalf("expected singularity image required error, got %v", err)
	}
}

func TestBuildForegroundCommandDaytonaRequiresWorkspace(t *testing.T) {
	_, err := buildForegroundCommand(context.Background(), "echo hi", "", ForegroundBackendOptions{
		Backend: "daytona",
	})
	if err == nil || !strings.Contains(err.Error(), "requires daytona_workspace") {
		t.Fatalf("expected daytona workspace required error, got %v", err)
	}
}

func TestBuildForegroundCommandVercelRequiresSandboxID(t *testing.T) {
	_, err := buildForegroundCommand(context.Background(), "echo hi", "", ForegroundBackendOptions{
		Backend: "vercel",
	})
	if err == nil || !strings.Contains(err.Error(), "requires vercel_sandbox_id") {
		t.Fatalf("expected vercel sandbox required error, got %v", err)
	}
}

func TestBuildForegroundCommandModalRequiresRef(t *testing.T) {
	_, err := buildForegroundCommand(context.Background(), "echo hi", "", ForegroundBackendOptions{
		Backend: "modal",
	})
	if err == nil || !strings.Contains(err.Error(), "requires modal_ref") {
		t.Fatalf("expected modal ref required error, got %v", err)
	}
}
