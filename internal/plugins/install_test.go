package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallAndUninstallLocalDirectoryPlugin(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "plugin.yaml"), []byte("name: Demo Plugin\nkind: standalone\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "asset.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	destRoot := filepath.Join(t.TempDir(), "plugins")
	manifest, installedPath, err := InstallLocal(src, destRoot)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "Demo Plugin" {
		t.Fatalf("manifest=%#v", manifest)
	}
	if _, err := os.Stat(filepath.Join(installedPath, "asset.txt")); err != nil {
		t.Fatalf("asset not copied: %v", err)
	}
	items, err := LoadFromDirs([]string{destRoot})
	if err != nil {
		t.Fatal(err)
	}
	removed, err := UninstallLocal(items, "demo plugin", []string{destRoot})
	if err != nil {
		t.Fatal(err)
	}
	if removed != installedPath {
		t.Fatalf("removed=%q want %q", removed, installedPath)
	}
	if _, err := os.Stat(installedPath); !os.IsNotExist(err) {
		t.Fatalf("plugin directory should be removed, err=%v", err)
	}
}

func TestInstallLocalManifestFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "demo.json")
	if err := os.WriteFile(src, []byte(`{"name":"demo","kind":"backend"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	destRoot := filepath.Join(t.TempDir(), "plugins")
	manifest, installedPath, err := InstallLocal(src, destRoot)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "demo" {
		t.Fatalf("manifest=%#v", manifest)
	}
	if filepath.Base(installedPath) != "demo.json" {
		t.Fatalf("installedPath=%q", installedPath)
	}
}

func TestInstallFromMarketplace(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "plugin.yaml"), []byte("name: demo\nkind: backend\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := HashPath(src)
	if err != nil {
		t.Fatal(err)
	}
	indexDir := t.TempDir()
	index := filepath.Join(indexDir, "marketplace.json")
	body := `{"plugins":[{"name":"demo","version":"1.0.0","source":"` + filepath.ToSlash(src) + `","sha256":"` + sum + `"}]}`
	if err := os.WriteFile(index, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "plugins")
	manifest, installedPath, err := InstallFromMarketplace(index, "demo", dest)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "demo" {
		t.Fatalf("manifest=%#v", manifest)
	}
	if _, err := os.Stat(filepath.Join(installedPath, "plugin.yaml")); err != nil {
		t.Fatalf("plugin not installed: %v", err)
	}
}
