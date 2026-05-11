package main

import (
	"path/filepath"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/config"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/model"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
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

func TestFindToolSchema(t *testing.T) {
	schemas := []core.ToolSchema{
		{Type: "function", Function: core.ToolSchemaDetail{Name: "terminal"}},
		{Type: "function", Function: core.ToolSchemaDetail{Name: "read_file"}},
	}
	schema, ok := findToolSchema(schemas, "read_file")
	if !ok {
		t.Fatal("expected read_file schema")
	}
	if schema.Function.Name != "read_file" {
		t.Fatalf("schema name = %q", schema.Function.Name)
	}
	if _, ok := findToolSchema(schemas, "missing"); ok {
		t.Fatal("missing schema should not be found")
	}
}

func TestNameListHelpers(t *testing.T) {
	names := parseNameList(" terminal,web_fetch, terminal ,,read_file ")
	want := []string{"read_file", "terminal", "web_fetch"}
	if len(names) != len(want) {
		t.Fatalf("names = %#v, want %#v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %#v, want %#v", names, want)
		}
	}
	names = addName(names, "todo")
	names = removeName(names, "terminal")
	want = []string{"read_file", "todo", "web_fetch"}
	if len(names) != len(want) {
		t.Fatalf("names = %#v, want %#v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %#v, want %#v", names, want)
		}
	}
}

func TestApplyDisabledTools(t *testing.T) {
	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, tools.NewProcessRegistry(t.TempDir()))
	applyDisabledTools(registry, "terminal,web_fetch")
	if _, ok := findToolSchema(registry.Schemas(), "terminal"); ok {
		t.Fatal("terminal should be disabled")
	}
	if _, ok := findToolSchema(registry.Schemas(), "web_fetch"); ok {
		t.Fatal("web_fetch should be disabled")
	}
	if _, ok := findToolSchema(registry.Schemas(), "read_file"); !ok {
		t.Fatal("read_file should remain enabled")
	}
}

func TestDoctorChecksWarnOnMissingProviderKey(t *testing.T) {
	cfg := config.Config{
		ModelProvider:  "openai",
		ModelName:      "gpt-test",
		ModelBaseURL:   "https://api.openai.com/v1",
		Workdir:        t.TempDir(),
		DataDir:        filepath.Join(t.TempDir(), "data"),
		MCPTransport:   "http",
		GatewayEnabled: false,
	}
	checks := buildDoctorChecks(cfg)
	if hasDoctorError(checks) {
		t.Fatalf("unexpected doctor error: %#v", checks)
	}
	check := doctorCheckByName(checks, "provider_credentials")
	if check.Status != "warn" {
		t.Fatalf("provider_credentials status = %q, want warn", check.Status)
	}
}

func TestDoctorChecksErrorOnBadWorkdir(t *testing.T) {
	cfg := config.Config{
		ModelProvider: "openai",
		ModelName:     "gpt-test",
		ModelBaseURL:  "https://api.openai.com/v1",
		ModelAPIKey:   "key",
		Workdir:       filepath.Join(t.TempDir(), "missing"),
		DataDir:       filepath.Join(t.TempDir(), "data"),
		MCPTransport:  "http",
	}
	checks := buildDoctorChecks(cfg)
	if !hasDoctorError(checks) {
		t.Fatalf("expected doctor error: %#v", checks)
	}
	check := doctorCheckByName(checks, "workdir")
	if check.Status != "error" {
		t.Fatalf("workdir status = %q, want error", check.Status)
	}
}

func TestDoctorChecksGatewayEnabledWithoutTokensWarns(t *testing.T) {
	cfg := config.Config{
		ModelProvider:   "openai",
		ModelName:       "gpt-test",
		ModelBaseURL:    "https://api.openai.com/v1",
		ModelAPIKey:     "key",
		Workdir:         t.TempDir(),
		DataDir:         filepath.Join(t.TempDir(), "data"),
		MCPTransport:    "http",
		GatewayEnabled:  true,
		TelegramToken:   "",
		DiscordToken:    "",
		SlackBotToken:   "",
		SlackAppToken:   "",
		TelegramAllowed: "",
	}
	checks := buildDoctorChecks(cfg)
	check := doctorCheckByName(checks, "gateway")
	if check.Status != "warn" {
		t.Fatalf("gateway status = %q, want warn", check.Status)
	}
}

func doctorCheckByName(checks []doctorCheck, name string) doctorCheck {
	for _, check := range checks {
		if check.Name == name {
			return check
		}
	}
	return doctorCheck{Name: name, Status: "missing"}
}

func TestGatewayStatus(t *testing.T) {
	cfg := config.Config{
		GatewayEnabled: true,
		TelegramToken:  "telegram-token",
		DiscordToken:   "",
		SlackBotToken:  "slack-bot",
		SlackAppToken:  "slack-app",
	}
	status := gatewayStatus(cfg)
	if !status.Enabled {
		t.Fatal("gateway should be enabled")
	}
	want := []string{"telegram", "slack"}
	if len(status.ConfiguredPlatforms) != len(want) {
		t.Fatalf("configured platforms = %#v, want %#v", status.ConfiguredPlatforms, want)
	}
	for i := range want {
		if status.ConfiguredPlatforms[i] != want[i] {
			t.Fatalf("configured platforms = %#v, want %#v", status.ConfiguredPlatforms, want)
		}
	}
	if len(status.SupportedPlatforms) != 4 {
		t.Fatalf("supported platforms = %#v", status.SupportedPlatforms)
	}
}
