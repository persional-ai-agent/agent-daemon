package main

import (
	"path/filepath"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/config"
	"github.com/dingjingmaster/agent-daemon/internal/model"
)

func TestBuildModelClientWithoutFallback(t *testing.T) {
	cfg := config.Config{
		ModelProvider: "openai",
		ModelBaseURL:  "https://api.openai.com/v1",
		ModelName:     "gpt-4o-mini",
	}
	client := buildModelClient(cfg)
	oc, ok := client.(*model.OpenAIClient)
	if !ok {
		t.Fatalf("expected OpenAIClient, got %T", client)
	}
	if oc.UseStreaming {
		t.Fatalf("expected UseStreaming=false by default")
	}
}

func TestBuildModelClientWithFallback(t *testing.T) {
	cfg := config.Config{
		ModelProvider:         "openai",
		ModelFallbackProvider: "anthropic",
		ModelBaseURL:          "https://api.openai.com/v1",
		ModelName:             "gpt-4o-mini",
		AnthropicBaseURL:      "https://api.anthropic.com/v1",
		AnthropicModel:        "claude-3-5-haiku-latest",
	}
	client := buildModelClient(cfg)
	if _, ok := client.(*model.FallbackClient); !ok {
		t.Fatalf("expected FallbackClient, got %T", client)
	}
}

func TestBuildModelClientOpenAIStreamingEnabled(t *testing.T) {
	cfg := config.Config{
		ModelProvider:     "openai",
		ModelBaseURL:      "https://api.openai.com/v1",
		ModelName:         "gpt-4o-mini",
		ModelUseStreaming: true,
	}
	client := buildModelClient(cfg)
	oc, ok := client.(*model.OpenAIClient)
	if !ok {
		t.Fatalf("expected OpenAIClient, got %T", client)
	}
	if !oc.UseStreaming {
		t.Fatalf("expected UseStreaming=true")
	}
}

func TestBuildModelClientAnthropicStreamingEnabled(t *testing.T) {
	cfg := config.Config{
		ModelProvider:     "anthropic",
		AnthropicBaseURL:  "https://api.anthropic.com/v1",
		AnthropicModel:    "claude-3-5-haiku-latest",
		ModelUseStreaming: true,
	}
	client := buildModelClient(cfg)
	ac, ok := client.(*model.AnthropicClient)
	if !ok {
		t.Fatalf("expected AnthropicClient, got %T", client)
	}
	if !ac.UseStreaming {
		t.Fatalf("expected UseStreaming=true")
	}
}

func TestBuildModelClientCodexStreamingEnabled(t *testing.T) {
	cfg := config.Config{
		ModelProvider:     "codex",
		CodexBaseURL:      "https://api.openai.com/v1",
		CodexModel:        "gpt-5-codex",
		ModelUseStreaming: true,
	}
	client := buildModelClient(cfg)
	cc, ok := client.(*model.CodexClient)
	if !ok {
		t.Fatalf("expected CodexClient, got %T", client)
	}
	if !cc.UseStreaming {
		t.Fatalf("expected UseStreaming=true")
	}
}

func TestParseModelSetArgs(t *testing.T) {
	provider, modelName, err := parseModelSetArgs([]string{"anthropic:claude-3-5-haiku-latest"})
	if err != nil {
		t.Fatal(err)
	}
	if provider != "anthropic" || modelName != "claude-3-5-haiku-latest" {
		t.Fatalf("parsed %s %s", provider, modelName)
	}

	provider, modelName, err = parseModelSetArgs([]string{"codex", "gpt-5-codex"})
	if err != nil {
		t.Fatal(err)
	}
	if provider != "codex" || modelName != "gpt-5-codex" {
		t.Fatalf("parsed %s %s", provider, modelName)
	}

	if _, _, err := parseModelSetArgs([]string{"unknown", "model"}); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestSaveModelSelectionWritesProviderSpecificKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.ini")
	if err := saveModelSelection(path, "anthropic", "claude-test", "https://anthropic.example/v1"); err != nil {
		t.Fatal(err)
	}

	assertConfigValue(t, path, "api.type", "anthropic")
	assertConfigValue(t, path, "api.anthropic.model", "claude-test")
	assertConfigValue(t, path, "api.anthropic.base_url", "https://anthropic.example/v1")

	if err := saveModelSelection(path, "openai", "gpt-test", ""); err != nil {
		t.Fatal(err)
	}
	assertConfigValue(t, path, "api.type", "openai")
	assertConfigValue(t, path, "api.model", "gpt-test")
}

func assertConfigValue(t *testing.T, path, key, want string) {
	t.Helper()
	got, ok, err := config.ReadConfigValue(path, key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != want {
		t.Fatalf("%s = %q, %v; want %q, true", key, got, ok, want)
	}
}
