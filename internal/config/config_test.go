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

func TestConfigManagementReadWriteList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.ini")

	if err := SaveConfigValue(path, "api.model", "gpt-test"); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfigValue(path, "api.api_key", "secret-key"); err != nil {
		t.Fatal(err)
	}

	value, ok, err := ReadConfigValue(path, "api.model")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || value != "gpt-test" {
		t.Fatalf("api.model = %q, %v; want gpt-test, true", value, ok)
	}

	entries, err := ListConfigValues(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2: %#v", len(entries), entries)
	}
	if entries[0].Key != "api.api_key" || entries[1].Key != "api.model" {
		t.Fatalf("entries not sorted or missing: %#v", entries)
	}
	if RedactConfigValue("api.api_key", "secret-key") != "********" {
		t.Fatal("api key should be redacted")
	}
}

func TestLoadUsesAgentConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.ini")
	if err := os.WriteFile(path, []byte("[api]\ntype = codex\nmodel = custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("AGENT_CONFIG_FILE", path)
	defer os.Unsetenv("AGENT_CONFIG_FILE")

	cfg := Load()
	if cfg.ModelProvider != "codex" {
		t.Fatalf("ModelProvider = %q, want codex", cfg.ModelProvider)
	}
	if cfg.ModelName != "custom" {
		t.Fatalf("ModelName = %q, want custom", cfg.ModelName)
	}
}

func TestLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.ini")
	if err := os.WriteFile(path, []byte("[api]\ntype = anthropic\n[api.anthropic]\nmodel = claude-test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ModelProvider != "anthropic" {
		t.Fatalf("ModelProvider = %q, want anthropic", cfg.ModelProvider)
	}
	if cfg.AnthropicModel != "claude-test" {
		t.Fatalf("AnthropicModel = %q, want claude-test", cfg.AnthropicModel)
	}
}
