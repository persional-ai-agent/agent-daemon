package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/config"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/gateway/platforms"
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

	provider, modelName, err = parseModelSetArgs([]string{"unknown", "model"})
	if err != nil {
		t.Fatal(err)
	}
	if provider != "unknown" || modelName != "model" {
		t.Fatalf("parsed %s %s", provider, modelName)
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

func TestCheckStubToolsNoToolsetFilter(t *testing.T) {
	check := checkStubTools(config.Config{})
	if check.Status != "ok" {
		t.Fatalf("stub_tools status = %q, want ok", check.Status)
	}
	if check.Detail != "none enabled" {
		t.Fatalf("stub_tools detail = %q, want none enabled", check.Detail)
	}
}

func TestCheckStubToolsWithToolsetFilter(t *testing.T) {
	check := checkStubTools(config.Config{EnabledToolsets: "core"})
	if check.Status != "ok" {
		t.Fatalf("stub_tools status = %q, want ok", check.Status)
	}
	if check.Detail != "none enabled" {
		t.Fatalf("stub_tools detail = %q, want none enabled", check.Detail)
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
	if len(status.SupportedPlatforms) != 13 {
		t.Fatalf("supported platforms = %#v", status.SupportedPlatforms)
	}
}

func TestGatewayStatusLockFields(t *testing.T) {
	workdir := t.TempDir()
	cfg := config.Config{
		Workdir:       workdir,
		TelegramToken: "telegram-token-" + strings.ReplaceAll(workdir, string(os.PathSeparator), "_"),
	}
	lockPath := gatewayLockPath(cfg)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte("999999"), 0o644); err != nil {
		t.Fatal(err)
	}
	tokenLockPath := gatewayTokenLockPath(cfg)
	if strings.TrimSpace(tokenLockPath) == "" {
		t.Fatal("token lock path should not be empty when token configured")
	}
	if err := os.MkdirAll(filepath.Dir(tokenLockPath), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tokenLockPath) })
	if err := os.WriteFile(tokenLockPath, []byte("999999"), 0o644); err != nil {
		t.Fatal(err)
	}
	status := gatewayStatus(cfg)
	if status.LockPID != 999999 || status.TokenLockPID != 999999 {
		t.Fatalf("unexpected lock pid values: lock=%d token=%d", status.LockPID, status.TokenLockPID)
	}
	if status.Locked || status.TokenLocked {
		t.Fatalf("stale lock should not be marked as locked: %#v", status)
	}
}

func TestGatewayStatusTokenLockAlive(t *testing.T) {
	workdir := t.TempDir()
	cfg := config.Config{
		Workdir:       workdir,
		TelegramToken: "telegram-token-" + strings.ReplaceAll(workdir, string(os.PathSeparator), "_"),
	}
	tokenLockPath := gatewayTokenLockPath(cfg)
	if strings.TrimSpace(tokenLockPath) == "" {
		t.Fatal("token lock path should not be empty when token configured")
	}
	if err := os.MkdirAll(filepath.Dir(tokenLockPath), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tokenLockPath) })
	if err := os.WriteFile(tokenLockPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	status := gatewayStatus(cfg)
	if !status.TokenLocked {
		t.Fatalf("expected token lock alive, got %#v", status)
	}
}

func TestBuildYuanbaoManifestExportContainsApprovalQuickReplies(t *testing.T) {
	out := buildYuanbaoManifestExport()
	raw, ok := out["quick_replies"].([]map[string]string)
	if !ok {
		t.Fatalf("quick_replies type invalid: %T", out["quick_replies"])
	}
	routeByText := map[string]string{}
	for _, item := range raw {
		routeByText[item["text"]] = item["route"]
	}
	if routeByText["批准"] != "/approve" || routeByText["拒绝"] != "/deny" {
		t.Fatalf("approval quick replies mismatch: %#v", routeByText)
	}
}

func TestGatewayManifestCoreCommandsParity(t *testing.T) {
	core := []string{"/pair", "/unpair", "/cancel", "/queue", "/status", "/pending", "/approvals", "/grant", "/revoke", "/approve", "/deny", "/help"}
	telegram := map[string]bool{}
	for _, c := range platforms.TelegramCommands() {
		telegram["/"+c.Command] = true
	}
	discord := map[string]bool{}
	for _, c := range platforms.DiscordApplicationCommands() {
		discord["/"+c.Name] = true
	}
	slackManifest := buildSlackManifestExport("/agent")
	slack := map[string]bool{}
	for _, c := range slackManifest.Commands {
		slack[strings.TrimSpace(c["command"])] = true
	}
	yb := map[string]bool{}
	raw, _ := buildYuanbaoManifestExport()["commands"].([]map[string]string)
	for _, c := range raw {
		cmd := strings.TrimSpace(c["command"])
		if cmd == "" {
			continue
		}
		yb[strings.Fields(cmd)[0]] = true
	}
	for _, name := range core {
		if !telegram[name] || !discord[name] || !slack[name] || !yb[name] {
			t.Fatalf("core command missing %s: telegram=%v discord=%v slack=%v yuanbao=%v", name, telegram[name], discord[name], slack[name], yb[name])
		}
	}
}

func TestReadGatewayLockStateAndCleanup(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "gateway.lock")
	if err := os.WriteFile(lockPath, []byte("999999"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := readGatewayLockState(lockPath)
	if st.PID != 999999 || !st.Stale || st.Alive {
		t.Fatalf("unexpected state: %#v", st)
	}
	if !cleanupStaleGatewayLock(lockPath) {
		t.Fatal("expected stale lock cleanup")
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("stale lock file should be removed, err=%v", err)
	}
}

func TestLoadConfiguredPluginsFiltersDisabled(t *testing.T) {
	workdir := t.TempDir()
	dir := filepath.Join(workdir, "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"demo_tool","type":"tool","tool":{"command":"./plugin.sh","schema":{"type":"object"}}}`
	if err := os.WriteFile(filepath.Join(dir, "demo.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Workdir: workdir, DisabledPlugins: "demo_tool"}
	items, err := loadConfiguredPlugins(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("disabled plugin should be filtered: %#v", items)
	}
}

func TestCheckPluginsConfig(t *testing.T) {
	workdir := t.TempDir()
	dir := filepath.Join(workdir, "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"demo_tool","type":"tool","tool":{"command":"./plugin.sh","schema":{"type":"object"}}}`
	if err := os.WriteFile(filepath.Join(dir, "demo.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	check := checkPluginsConfig(config.Config{Workdir: workdir})
	if check.Status != "ok" {
		t.Fatalf("plugins check status=%q detail=%q", check.Status, check.Detail)
	}
}

func TestAvailableModelProvidersIncludesPluginProvider(t *testing.T) {
	workdir := t.TempDir()
	dir := filepath.Join(workdir, "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"demo_provider","type":"provider","provider":{"command":"./provider.sh"}}`
	if err := os.WriteFile(filepath.Join(dir, "provider.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	names := availableModelProviders(config.Config{Workdir: workdir})
	if !containsName(names, "demo_provider") {
		t.Fatalf("providers=%#v", names)
	}
}

func TestBuildProviderClientFromPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell provider plugin test not supported on windows")
	}
	workdir := t.TempDir()
	dir := filepath.Join(workdir, "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "provider.sh")
	body := "#!/usr/bin/env bash\ncat <<'EOF'\n{\"message\":{\"role\":\"assistant\",\"content\":\"plugin-provider-ok\"}}\nEOF\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"demo_provider","type":"provider","provider":{"command":"./provider.sh"}}`
	if err := os.WriteFile(filepath.Join(dir, "provider.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Workdir: workdir, ModelName: "x-model"}
	client := buildProviderClient(cfg, "demo_provider")
	if client == nil {
		t.Fatal("expected plugin provider client")
	}
	msg, err := client.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "plugin-provider-ok" {
		t.Fatalf("content=%q", msg.Content)
	}
}
