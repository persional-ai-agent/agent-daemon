package plugins

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDirs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	okJSON := `{"name":"demo-tool","type":"tool","version":"1.0.0","tool":{"command":"./tool.sh","schema":{"type":"object"}},"enabled":true}`
	if err := os.WriteFile(filepath.Join(dir, "tool.json"), []byte(okJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invalid.json"), []byte(`{"type":"tool"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := LoadFromDirs([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(items))
	}
	if items[0].Name != "demo-tool" || items[0].Type != "tool" {
		t.Fatalf("unexpected manifest: %+v", items[0])
	}
}

func TestLoadFromDirsSupportsYAMLNestedPlugin(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "plugins", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	yml := `
name: demo
kind: standalone
version: 1.0.0
description: demo plugin
provides_tools:
  - declared_only
commands:
  - name: hello
    command: ./hello.sh
dashboard:
  label: Demo
  entry: dist/index.js
tools:
  - name: demo_tool
    command: ./tool.sh
    schema:
      type: object
providers:
  - name: demo_provider
    command: ./provider.sh
`
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := LoadFromDirs([]string{filepath.Join(root, "plugins")})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one plugin, got %#v", items)
	}
	if items[0].Type != "plugin" || items[0].Kind != "standalone" {
		t.Fatalf("unexpected manifest: %#v", items[0])
	}
	if len(items[0].Commands) != 1 || len(items[0].Tools) != 1 || len(items[0].Providers) != 1 {
		t.Fatalf("capabilities not loaded: %#v", items[0])
	}
	expanded := ExpandRuntimeManifests(items)
	if len(expanded) != 2 {
		t.Fatalf("expected tool+provider runtime manifests, got %#v", expanded)
	}
}

func TestValidateManifest(t *testing.T) {
	m := Manifest{
		Name: "demo",
		Type: "tool",
		Tool: &ToolManifest{
			Command: "./tool.sh",
			Schema:  map[string]any{"type": "object"},
		},
	}
	if err := ValidateManifest(m); err != nil {
		t.Fatalf("expected valid manifest: %v", err)
	}
	m.Tool.Command = ""
	if err := ValidateManifest(m); err == nil {
		t.Fatal("expected missing command error")
	}

	pm := Manifest{
		Name: "demo_provider",
		Type: "provider",
		Provider: &ProviderManifest{
			Command: "./provider.sh",
		},
	}
	if err := ValidateManifest(pm); err != nil {
		t.Fatalf("expected valid provider manifest: %v", err)
	}

	meta := Manifest{Name: "spotify", Kind: "backend", ProvidesTools: []string{"spotify_search"}}
	if err := ValidateManifest(meta); err != nil {
		t.Fatalf("expected metadata-only plugin manifest: %v", err)
	}
}

func TestVerifyManifestChecksSignatureAndFiles(t *testing.T) {
	root := t.TempDir()
	asset := filepath.Join(root, "asset.txt")
	if err := os.WriteFile(asset, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := HashFile(asset)
	if err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := Manifest{
		Name: "signed",
		Kind: "backend",
		File: filepath.Join(root, "plugin.yaml"),
		Security: &SecurityManifest{
			PublicKey: base64.StdEncoding.EncodeToString(pub),
			Files: []FileChecksum{{
				Path:   "asset.txt",
				SHA256: sum,
			}},
		},
	}
	payload, err := ManifestSignaturePayload(m)
	if err != nil {
		t.Fatal(err)
	}
	m.Security.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))
	if err := VerifyManifest(m); err != nil {
		t.Fatalf("expected signed manifest to verify: %v", err)
	}
	m.Description = "tampered"
	if err := VerifyManifest(m); err == nil {
		t.Fatal("expected tampered manifest signature failure")
	}
}
