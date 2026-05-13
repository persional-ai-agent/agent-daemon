package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUITUIBinaryFromEnv(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "ui-tui-bin")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := os.Getenv("AGENT_UI_TUI_BIN")
	defer func() { _ = os.Setenv("AGENT_UI_TUI_BIN", old) }()
	if err := os.Setenv("AGENT_UI_TUI_BIN", bin); err != nil {
		t.Fatal(err)
	}
	got, err := resolveUITUIBinary()
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Fatalf("resolveUITUIBinary=%q, want %q", got, bin)
	}
}

func TestResolveUITUIBinaryFromLocalCandidate(t *testing.T) {
	oldEnv := os.Getenv("AGENT_UI_TUI_BIN")
	defer func() { _ = os.Setenv("AGENT_UI_TUI_BIN", oldEnv) }()
	_ = os.Unsetenv("AGENT_UI_TUI_BIN")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	tmp := t.TempDir()
	candidateDir := filepath.Join(tmp, "ui-tui")
	if err := os.MkdirAll(candidateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(candidateDir, "tui.run")
	if err := os.WriteFile(candidate, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	got, err := resolveUITUIBinary()
	if err != nil {
		t.Fatal(err)
	}
	if got != candidate {
		t.Fatalf("resolveUITUIBinary=%q, want %q", got, candidate)
	}
}

