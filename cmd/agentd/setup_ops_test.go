package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMigrationPlanAndApply(t *testing.T) {
	src := t.TempDir()
	wd := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "config.ini"), []byte("[api]\nprovider=openai\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "MEMORY.md"), []byte("mem"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := buildMigrationPlan(src, wd, data, "minimal", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) == 0 {
		t.Fatalf("expected plan")
	}
	checkpoint, copied, skipped, err := applyMigrationPlan(plan, false, data)
	if err != nil {
		t.Fatal(err)
	}
	if copied == 0 {
		t.Fatalf("expected copied files")
	}
	if skipped != 0 {
		t.Fatalf("unexpected skipped: %d", skipped)
	}
	if checkpoint != "" {
		t.Fatalf("checkpoint should be empty when no existing targets")
	}
}

func TestCompletionInstallUninstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := installShellCompletion("bash", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	removed, err := uninstallShellCompletion("bash")
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d", removed)
	}
}
