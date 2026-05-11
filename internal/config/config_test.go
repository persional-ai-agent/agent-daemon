package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromINI(t *testing.T) {
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "config.ini")
	os.WriteFile(iniPath, []byte(`
[api]
type = anthropic
base_url = https://custom-api.example.com/v1
api_key = sk-test-key
model = claude-3-opus
streaming = true

[agent]
max_iterations = 10
listen_addr = :9090

[provider]
fallback = openai
race_enabled = true
`), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg := Load()

	if cfg.ModelProvider != "anthropic" {
		t.Errorf("ModelProvider = %q, want anthropic", cfg.ModelProvider)
	}
	if cfg.ModelBaseURL != "https://custom-api.example.com/v1" {
		t.Errorf("ModelBaseURL = %q, want custom", cfg.ModelBaseURL)
	}
	if cfg.ModelAPIKey != "sk-test-key" {
		t.Errorf("ModelAPIKey = %q, want sk-test-key", cfg.ModelAPIKey)
	}
	if cfg.ModelName != "claude-3-opus" {
		t.Errorf("ModelName = %q, want claude-3-opus", cfg.ModelName)
	}
	if !cfg.ModelUseStreaming {
		t.Error("ModelUseStreaming should be true")
	}
	if cfg.MaxIterations != 10 {
		t.Errorf("MaxIterations = %d, want 10", cfg.MaxIterations)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want :9090", cfg.ListenAddr)
	}
	if cfg.ModelFallbackProvider != "openai" {
		t.Errorf("ModelFallbackProvider = %q, want openai", cfg.ModelFallbackProvider)
	}
	if !cfg.ModelRaceEnabled {
		t.Error("ModelRaceEnabled should be true")
	}
}

func TestEnvOverridesINI(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.ini"), []byte(`
[api]
type = anthropic
base_url = https://ini.example.com
api_key = ini-key
`), 0o644)

	os.Setenv("AGENT_MODEL_PROVIDER", "openai")
	os.Setenv("OPENAI_BASE_URL", "https://env.example.com")
	os.Setenv("OPENAI_API_KEY", "env-key")
	defer func() {
		os.Unsetenv("AGENT_MODEL_PROVIDER")
		os.Unsetenv("OPENAI_BASE_URL")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg := Load()

	if cfg.ModelProvider != "openai" {
		t.Errorf("env should override ini: ModelProvider = %q, want openai", cfg.ModelProvider)
	}
	if cfg.ModelBaseURL != "https://env.example.com" {
		t.Errorf("env should override ini: ModelBaseURL = %q", cfg.ModelBaseURL)
	}
	if cfg.ModelAPIKey != "env-key" {
		t.Errorf("env should override ini: ModelAPIKey = %q", cfg.ModelAPIKey)
	}
}

func TestDefaultsWhenNoINI(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cfg := Load()

	if cfg.ModelProvider != "openai" {
		t.Errorf("default ModelProvider = %q, want openai", cfg.ModelProvider)
	}
	if cfg.ModelBaseURL != "https://api.openai.com/v1" {
		t.Errorf("default ModelBaseURL = %q", cfg.ModelBaseURL)
	}
	if cfg.MaxIterations != 30 {
		t.Errorf("default MaxIterations = %d, want 30", cfg.MaxIterations)
	}
}
