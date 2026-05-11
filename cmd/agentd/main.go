package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/api"
	"github.com/dingjingmaster/agent-daemon/internal/cli"
	"github.com/dingjingmaster/agent-daemon/internal/config"
	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/cronrunner"
	"github.com/dingjingmaster/agent-daemon/internal/gateway"
	"github.com/dingjingmaster/agent-daemon/internal/gateway/platforms"
	"github.com/dingjingmaster/agent-daemon/internal/memory"
	"github.com/dingjingmaster/agent-daemon/internal/model"
	"github.com/dingjingmaster/agent-daemon/internal/store"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

func main() {
	cfg := config.Load()
	if len(os.Args) < 2 {
		runChat(cfg, "")
		return
	}
	switch os.Args[1] {
	case "chat":
		fs := flag.NewFlagSet("chat", flag.ExitOnError)
		message := fs.String("message", "", "first message to send")
		sessionID := fs.String("session", uuid.NewString(), "session id")
		skills := fs.String("skills", "", "comma-separated skill names to preload")
		_ = fs.Parse(os.Args[2:])
		runChat(cfg, *message, *sessionID, *skills)
	case "serve":
		runServe(cfg)
	case "tools":
		runTools(cfg, os.Args[2:])
	case "toolsets":
		runToolsets(os.Args[2:])
	case "config":
		runConfig(os.Args[2:])
	case "model":
		runModel(cfg, os.Args[2:])
	case "doctor":
		runDoctor(cfg, os.Args[2:])
	case "gateway":
		runGateway(cfg, os.Args[2:])
	case "sessions":
		runSessions(cfg, os.Args[2:])
	default:
		runChat(cfg, "", uuid.NewString())
	}
}

func runConfig(args []string) {
	if len(args) == 0 {
		printConfigUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("config list", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		showSecrets := fs.Bool("show-secrets", false, "show secret values")
		_ = fs.Parse(args[1:])
		entries, err := config.ListConfigValues(*path)
		if err != nil {
			log.Fatal(err)
		}
		for _, entry := range entries {
			value := entry.Value
			if !*showSecrets {
				value = config.RedactConfigValue(entry.Key, value)
			}
			fmt.Printf("%s=%s\n", entry.Key, value)
		}
	case "get":
		fs := flag.NewFlagSet("config get", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd config get [-file path] section.key")
		}
		value, ok, err := config.ReadConfigValue(*path, fs.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			os.Exit(1)
		}
		fmt.Println(value)
	case "set":
		fs := flag.NewFlagSet("config set", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 2 {
			log.Fatal("usage: agentd config set [-file path] section.key value")
		}
		if err := config.SaveConfigValue(*path, fs.Arg(0), fs.Arg(1)); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("updated %s in %s\n", fs.Arg(0), config.ConfigFilePath(*path))
	default:
		printConfigUsage()
		os.Exit(2)
	}
}

func runToolsets(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintln(os.Stderr, "  agentd toolsets list")
		fmt.Fprintln(os.Stderr, "  agentd toolsets resolve name[,name...]")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		printJSON(tools.ListToolsets())
	case "resolve":
		fs := flag.NewFlagSet("toolsets resolve", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd toolsets resolve name[,name...]")
		}
		names := parseNameList(fs.Arg(0))
		allowed, err := tools.ResolveToolset(names)
		if err != nil {
			log.Fatal(err)
		}
		out := make([]string, 0, len(allowed))
		for name := range allowed {
			out = append(out, name)
		}
		sort.Strings(out)
		printToolNames(out)
	default:
		log.Fatal("usage: agentd toolsets list | agentd toolsets resolve name[,name...]")
	}
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd config list [-file path] [-show-secrets]")
	fmt.Fprintln(os.Stderr, "  agentd config get [-file path] section.key")
	fmt.Fprintln(os.Stderr, "  agentd config set [-file path] section.key value")
}

func runModel(cfg config.Config, args []string) {
	if len(args) == 0 {
		printModelUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "show", "current":
		fs := flag.NewFlagSet("model show", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd model show [-file path]")
		}
		showCfg := cfg
		if strings.TrimSpace(*path) != "" {
			var err error
			showCfg, err = config.LoadFile(*path)
			if err != nil {
				log.Fatal(err)
			}
		}
		provider := strings.ToLower(strings.TrimSpace(showCfg.ModelProvider))
		if provider == "" {
			provider = "openai"
		}
		modelName, baseURL := currentModelConfig(showCfg, provider)
		fmt.Printf("provider=%s\n", provider)
		fmt.Printf("model=%s\n", modelName)
		fmt.Printf("base_url=%s\n", baseURL)
	case "providers":
		for _, provider := range supportedModelProviders() {
			fmt.Println(provider)
		}
	case "set":
		fs := flag.NewFlagSet("model set", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		baseURL := fs.String("base-url", "", "provider base URL")
		_ = fs.Parse(args[1:])
		provider, modelName, err := parseModelSetArgs(fs.Args())
		if err != nil {
			log.Fatal(err)
		}
		if err := saveModelSelection(*path, provider, modelName, *baseURL); err != nil {
			log.Fatal(err)
		}
		if strings.TrimSpace(*baseURL) == "" {
			fmt.Printf("updated model to %s:%s in %s\n", provider, modelName, config.ConfigFilePath(*path))
		} else {
			fmt.Printf("updated model to %s:%s (%s) in %s\n", provider, modelName, *baseURL, config.ConfigFilePath(*path))
		}
	default:
		printModelUsage()
		os.Exit(2)
	}
}

func printModelUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd model show [-file path]")
	fmt.Fprintln(os.Stderr, "  agentd model providers")
	fmt.Fprintln(os.Stderr, "  agentd model set [-file path] [-base-url url] provider model")
	fmt.Fprintln(os.Stderr, "  agentd model set [-file path] [-base-url url] provider:model")
}

func runTools(cfg config.Config, args []string) {
	if len(args) == 0 {
		eng, _ := mustBuildEngine(cfg)
		printToolNames(eng.Registry.Names())
		return
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("tools list", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd tools list")
		}
		eng, _ := mustBuildEngine(cfg)
		printToolNames(eng.Registry.Names())
	case "show":
		fs := flag.NewFlagSet("tools show", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd tools show tool_name")
		}
		eng, _ := mustBuildEngine(cfg)
		schema, ok := findToolSchema(eng.Registry.Schemas(), fs.Arg(0))
		if !ok {
			log.Fatalf("unknown tool: %s", fs.Arg(0))
		}
		printJSON(schema)
	case "schemas":
		fs := flag.NewFlagSet("tools schemas", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd tools schemas")
		}
		eng, _ := mustBuildEngine(cfg)
		printJSON(eng.Registry.Schemas())
	case "disabled":
		fs := flag.NewFlagSet("tools disabled", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd tools disabled [-file path]")
		}
		disabled := parseNameList(cfg.DisabledTools)
		if strings.TrimSpace(*path) != "" {
			var err error
			disabled, err = readDisabledToolsConfig(*path)
			if err != nil {
				log.Fatal(err)
			}
		}
		printToolNames(disabled)
	case "disable":
		path, toolName := parseToolToggleArgs(args[1:], "tools disable")
		disabled, err := readDisabledToolsConfig(path)
		if err != nil {
			log.Fatal(err)
		}
		disabled = addName(disabled, toolName)
		if err := config.SaveConfigValue(path, "tools.disabled", strings.Join(disabled, ",")); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("disabled tool %s in %s\n", toolName, config.ConfigFilePath(path))
	case "enable":
		path, toolName := parseToolToggleArgs(args[1:], "tools enable")
		disabled, err := readDisabledToolsConfig(path)
		if err != nil {
			log.Fatal(err)
		}
		disabled = removeName(disabled, toolName)
		if err := config.SaveConfigValue(path, "tools.disabled", strings.Join(disabled, ",")); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("enabled tool %s in %s\n", toolName, config.ConfigFilePath(path))
	default:
		printToolsUsage()
		os.Exit(2)
	}
}

func printToolNames(names []string) {
	for _, name := range names {
		fmt.Println(name)
	}
}

func printToolsUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd tools")
	fmt.Fprintln(os.Stderr, "  agentd tools list")
	fmt.Fprintln(os.Stderr, "  agentd tools show tool_name")
	fmt.Fprintln(os.Stderr, "  agentd tools schemas")
	fmt.Fprintln(os.Stderr, "  agentd tools disabled [-file path]")
	fmt.Fprintln(os.Stderr, "  agentd tools disable [-file path] tool_name")
	fmt.Fprintln(os.Stderr, "  agentd tools enable [-file path] tool_name")
}

func findToolSchema(schemas []core.ToolSchema, name string) (core.ToolSchema, bool) {
	name = strings.TrimSpace(name)
	for _, schema := range schemas {
		if schema.Function.Name == name {
			return schema, true
		}
	}
	return core.ToolSchema{}, false
}

func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

func parseToolToggleArgs(args []string, name string) (string, string) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	_ = fs.Parse(args)
	if fs.NArg() != 1 {
		log.Fatalf("usage: agentd %s [-file path] tool_name", name)
	}
	toolName := strings.TrimSpace(fs.Arg(0))
	if toolName == "" {
		log.Fatal("tool_name is required")
	}
	return *path, toolName
}

func readDisabledToolsConfig(path string) ([]string, error) {
	value, ok, err := config.ReadConfigValue(path, "tools.disabled")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return parseNameList(value), nil
}

func parseNameList(value string) []string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func addName(names []string, name string) []string {
	names = append(names, name)
	return parseNameList(strings.Join(names, ","))
}

func removeName(names []string, name string) []string {
	name = strings.TrimSpace(name)
	var out []string
	for _, item := range names {
		if item != name {
			out = append(out, item)
		}
	}
	return parseNameList(strings.Join(out, ","))
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func runDoctor(cfg config.Config, args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatal("usage: agentd doctor [-json]")
	}
	checks := buildDoctorChecks(cfg)
	if *jsonOutput {
		printJSON(checks)
	} else {
		for _, check := range checks {
			fmt.Printf("[%s] %s: %s\n", check.Status, check.Name, check.Detail)
		}
	}
	if hasDoctorError(checks) {
		os.Exit(1)
	}
}

func buildDoctorChecks(cfg config.Config) []doctorCheck {
	checks := []doctorCheck{
		{Name: "config_file", Status: "ok", Detail: "using " + config.ConfigFilePath("") + " when present; environment variables take precedence"},
		checkDirectory("workdir", cfg.Workdir, false),
		checkDirectory("data_dir", cfg.DataDir, true),
		checkModelConfig(cfg),
		checkProviderCredentials(cfg),
		checkMCPConfig(cfg),
		checkGatewayConfig(cfg),
		checkRegisteredTools(),
	}
	return checks
}

func checkDirectory(name, path string, create bool) doctorCheck {
	path = strings.TrimSpace(path)
	if path == "" {
		return doctorCheck{Name: name, Status: "error", Detail: "path is empty"}
	}
	if create {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return doctorCheck{Name: name, Status: "error", Detail: err.Error()}
		}
		f, err := os.CreateTemp(path, ".agentd-doctor-*")
		if err != nil {
			return doctorCheck{Name: name, Status: "error", Detail: "not writable: " + err.Error()}
		}
		tmpName := f.Name()
		_ = f.Close()
		_ = os.Remove(tmpName)
		return doctorCheck{Name: name, Status: "ok", Detail: path + " is writable"}
	}
	info, err := os.Stat(path)
	if err != nil {
		return doctorCheck{Name: name, Status: "error", Detail: err.Error()}
	}
	if !info.IsDir() {
		return doctorCheck{Name: name, Status: "error", Detail: "not a directory: " + path}
	}
	return doctorCheck{Name: name, Status: "ok", Detail: path}
}

func checkModelConfig(cfg config.Config) doctorCheck {
	provider := strings.ToLower(strings.TrimSpace(cfg.ModelProvider))
	if provider == "" {
		provider = "openai"
	}
	if _, _, err := normalizeModelSelection(provider, selectedModelName(cfg, provider)); err != nil {
		return doctorCheck{Name: "model", Status: "error", Detail: err.Error()}
	}
	modelName, baseURL := currentModelConfig(cfg, provider)
	return doctorCheck{Name: "model", Status: "ok", Detail: fmt.Sprintf("%s:%s (%s)", provider, modelName, baseURL)}
}

func checkProviderCredentials(cfg config.Config) doctorCheck {
	provider := strings.ToLower(strings.TrimSpace(cfg.ModelProvider))
	if provider == "" {
		provider = "openai"
	}
	keyName, value := selectedProviderKey(cfg, provider)
	if strings.TrimSpace(value) == "" {
		return doctorCheck{Name: "provider_credentials", Status: "warn", Detail: keyName + " is empty"}
	}
	return doctorCheck{Name: "provider_credentials", Status: "ok", Detail: keyName + " is set"}
}

func selectedProviderKey(cfg config.Config, provider string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return "ANTHROPIC_API_KEY", cfg.AnthropicAPIKey
	case "codex":
		return "CODEX_API_KEY", cfg.CodexAPIKey
	default:
		return "OPENAI_API_KEY", cfg.ModelAPIKey
	}
}

func selectedModelName(cfg config.Config, provider string) string {
	modelName, _ := currentModelConfig(cfg, provider)
	return modelName
}

func checkMCPConfig(cfg config.Config) doctorCheck {
	transport := strings.ToLower(strings.TrimSpace(cfg.MCPTransport))
	if transport == "" {
		transport = "http"
	}
	switch transport {
	case "http":
		if strings.TrimSpace(cfg.MCPEndpoint) == "" {
			return doctorCheck{Name: "mcp", Status: "ok", Detail: "disabled"}
		}
		return doctorCheck{Name: "mcp", Status: "ok", Detail: "http endpoint configured"}
	case "stdio":
		if strings.TrimSpace(cfg.MCPStdioCommand) == "" {
			return doctorCheck{Name: "mcp", Status: "warn", Detail: "stdio transport selected but command is empty"}
		}
		return doctorCheck{Name: "mcp", Status: "ok", Detail: "stdio command configured"}
	default:
		return doctorCheck{Name: "mcp", Status: "error", Detail: "unsupported transport: " + transport}
	}
}

func checkGatewayConfig(cfg config.Config) doctorCheck {
	if !cfg.GatewayEnabled {
		return doctorCheck{Name: "gateway", Status: "ok", Detail: "disabled"}
	}
	configured := make([]string, 0, 3)
	if strings.TrimSpace(cfg.TelegramToken) != "" {
		configured = append(configured, "telegram")
	}
	if strings.TrimSpace(cfg.DiscordToken) != "" {
		configured = append(configured, "discord")
	}
	if strings.TrimSpace(cfg.SlackBotToken) != "" && strings.TrimSpace(cfg.SlackAppToken) != "" {
		configured = append(configured, "slack")
	}
	if len(configured) == 0 {
		return doctorCheck{Name: "gateway", Status: "warn", Detail: "enabled but no platform tokens are configured"}
	}
	return doctorCheck{Name: "gateway", Status: "ok", Detail: "configured platforms: " + strings.Join(configured, ",")}
}

func checkRegisteredTools() doctorCheck {
	registry := tools.NewRegistry()
	procDir, err := os.MkdirTemp("", "agentd-doctor-tools-*")
	if err != nil {
		return doctorCheck{Name: "tools", Status: "error", Detail: err.Error()}
	}
	defer os.RemoveAll(procDir)
	tools.RegisterBuiltins(registry, tools.NewProcessRegistry(procDir))
	names := registry.Names()
	if len(names) == 0 {
		return doctorCheck{Name: "tools", Status: "error", Detail: "no tools registered"}
	}
	return doctorCheck{Name: "tools", Status: "ok", Detail: fmt.Sprintf("%d builtin tools registered", len(names))}
}

func hasDoctorError(checks []doctorCheck) bool {
	for _, check := range checks {
		if check.Status == "error" {
			return true
		}
	}
	return false
}

func runGateway(cfg config.Config, args []string) {
	if len(args) == 0 {
		printGatewayUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "status":
		fs := flag.NewFlagSet("gateway status", flag.ExitOnError)
		path := fs.String("file", "", "config file path")
		jsonOutput := fs.Bool("json", false, "output JSON")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd gateway status [-file path] [-json]")
		}
		statusCfg := cfg
		if strings.TrimSpace(*path) != "" {
			var err error
			statusCfg, err = config.LoadFile(*path)
			if err != nil {
				log.Fatal(err)
			}
		}
		status := gatewayStatus(statusCfg)
		if *jsonOutput {
			printJSON(status)
			return
		}
		fmt.Printf("enabled=%t\n", status.Enabled)
		if len(status.ConfiguredPlatforms) == 0 {
			fmt.Println("configured_platforms=")
		} else {
			fmt.Println("configured_platforms=" + strings.Join(status.ConfiguredPlatforms, ","))
		}
		fmt.Println("supported_platforms=" + strings.Join(status.SupportedPlatforms, ","))
	case "platforms":
		for _, platform := range supportedGatewayPlatforms() {
			fmt.Println(platform)
		}
	case "enable":
		path := parseGatewayConfigPath(args[1:], "gateway enable")
		if err := config.SaveConfigValue(path, "gateway.enabled", "true"); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("enabled gateway in %s\n", config.ConfigFilePath(path))
	case "disable":
		path := parseGatewayConfigPath(args[1:], "gateway disable")
		if err := config.SaveConfigValue(path, "gateway.enabled", "false"); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("disabled gateway in %s\n", config.ConfigFilePath(path))
	default:
		printGatewayUsage()
		os.Exit(2)
	}
}

func parseGatewayConfigPath(args []string, name string) string {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	path := fs.String("file", "", "config file path")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		log.Fatalf("usage: agentd %s [-file path]", name)
	}
	return *path
}

func printGatewayUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd gateway status [-file path] [-json]")
	fmt.Fprintln(os.Stderr, "  agentd gateway platforms")
	fmt.Fprintln(os.Stderr, "  agentd gateway enable [-file path]")
	fmt.Fprintln(os.Stderr, "  agentd gateway disable [-file path]")
}

type gatewayStatusInfo struct {
	Enabled             bool     `json:"enabled"`
	ConfiguredPlatforms []string `json:"configured_platforms"`
	SupportedPlatforms  []string `json:"supported_platforms"`
}

func gatewayStatus(cfg config.Config) gatewayStatusInfo {
	return gatewayStatusInfo{
		Enabled:             cfg.GatewayEnabled,
		ConfiguredPlatforms: configuredGatewayPlatforms(cfg),
		SupportedPlatforms:  supportedGatewayPlatforms(),
	}
}

func supportedGatewayPlatforms() []string {
	return []string{"telegram", "discord", "slack"}
}

func configuredGatewayPlatforms(cfg config.Config) []string {
	out := make([]string, 0, 3)
	if strings.TrimSpace(cfg.TelegramToken) != "" {
		out = append(out, "telegram")
	}
	if strings.TrimSpace(cfg.DiscordToken) != "" {
		out = append(out, "discord")
	}
	if strings.TrimSpace(cfg.SlackBotToken) != "" && strings.TrimSpace(cfg.SlackAppToken) != "" {
		out = append(out, "slack")
	}
	return out
}

func runSessions(cfg config.Config, args []string) {
	if len(args) == 0 {
		printSessionsUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("sessions list", flag.ExitOnError)
		dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir (contains sessions.db)")
		limit := fs.Int("limit", 20, "result limit")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 0 {
			log.Fatal("usage: agentd sessions list [-data-dir dir] [-limit N]")
		}
		ss, err := store.NewSessionStore(filepath.Join(strings.TrimSpace(*dataDir), "sessions.db"))
		if err != nil {
			log.Fatal(err)
		}
		defer ss.Close()
		rows, err := ss.ListRecentSessions(*limit)
		if err != nil {
			log.Fatal(err)
		}
		printJSON(rows)
	case "search":
		fs := flag.NewFlagSet("sessions search", flag.ExitOnError)
		dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir (contains sessions.db)")
		limit := fs.Int("limit", 20, "result limit")
		exclude := fs.String("exclude", "", "exclude session_id")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd sessions search [-data-dir dir] [-limit N] [-exclude session_id] query")
		}
		query := fs.Arg(0)
		ss, err := store.NewSessionStore(filepath.Join(strings.TrimSpace(*dataDir), "sessions.db"))
		if err != nil {
			log.Fatal(err)
		}
		defer ss.Close()
		rows, err := ss.Search(query, *limit, *exclude)
		if err != nil {
			log.Fatal(err)
		}
		printJSON(rows)
	case "show":
		fs := flag.NewFlagSet("sessions show", flag.ExitOnError)
		dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir (contains sessions.db)")
		offset := fs.Int("offset", 0, "message offset (0-based)")
		limit := fs.Int("limit", 200, "message limit")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd sessions show [-data-dir dir] [-offset N] [-limit N] session_id")
		}
		sessionID := fs.Arg(0)
		ss, err := store.NewSessionStore(filepath.Join(strings.TrimSpace(*dataDir), "sessions.db"))
		if err != nil {
			log.Fatal(err)
		}
		defer ss.Close()
		msgs, err := ss.LoadMessagesPage(sessionID, *offset, *limit)
		if err != nil {
			log.Fatal(err)
		}
		payload := map[string]any{
			"session_id": sessionID,
			"offset":     *offset,
			"limit":      *limit,
			"messages":   msgs,
		}
		printJSON(payload)
	case "stats":
		fs := flag.NewFlagSet("sessions stats", flag.ExitOnError)
		dataDir := fs.String("data-dir", cfg.DataDir, "agent data dir (contains sessions.db)")
		_ = fs.Parse(args[1:])
		if fs.NArg() != 1 {
			log.Fatal("usage: agentd sessions stats [-data-dir dir] session_id")
		}
		sessionID := fs.Arg(0)
		ss, err := store.NewSessionStore(filepath.Join(strings.TrimSpace(*dataDir), "sessions.db"))
		if err != nil {
			log.Fatal(err)
		}
		defer ss.Close()
		stats, err := ss.SessionStats(sessionID)
		if err != nil {
			log.Fatal(err)
		}
		printJSON(stats)
	default:
		printSessionsUsage()
		os.Exit(2)
	}
}

func printSessionsUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd sessions list [-data-dir dir] [-limit N]")
	fmt.Fprintln(os.Stderr, "  agentd sessions search [-data-dir dir] [-limit N] [-exclude session_id] query")
	fmt.Fprintln(os.Stderr, "  agentd sessions show [-data-dir dir] [-offset N] [-limit N] session_id")
	fmt.Fprintln(os.Stderr, "  agentd sessions stats [-data-dir dir] session_id")
}

func supportedModelProviders() []string {
	return []string{"openai", "anthropic", "codex"}
}

func currentModelConfig(cfg config.Config, provider string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return cfg.AnthropicModel, cfg.AnthropicBaseURL
	case "codex":
		return cfg.CodexModel, cfg.CodexBaseURL
	default:
		return cfg.ModelName, cfg.ModelBaseURL
	}
}

func parseModelSetArgs(args []string) (string, string, error) {
	if len(args) == 1 {
		parts := strings.SplitN(args[0], ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("usage: agentd model set provider model or provider:model")
		}
		return normalizeModelSelection(parts[0], parts[1])
	}
	if len(args) == 2 {
		return normalizeModelSelection(args[0], args[1])
	}
	return "", "", fmt.Errorf("usage: agentd model set provider model or provider:model")
}

func normalizeModelSelection(provider, modelName string) (string, string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return "", "", fmt.Errorf("model is required")
	}
	for _, supported := range supportedModelProviders() {
		if provider == supported {
			return provider, modelName, nil
		}
	}
	return "", "", fmt.Errorf("unsupported provider %q (supported: openai, anthropic, codex)", provider)
}

func saveModelSelection(path, provider, modelName, baseURL string) error {
	provider, modelName, err := normalizeModelSelection(provider, modelName)
	if err != nil {
		return err
	}
	if err := config.SaveConfigValue(path, "api.type", provider); err != nil {
		return err
	}
	modelKey, baseURLKey := modelConfigKeys(provider)
	if err := config.SaveConfigValue(path, modelKey, modelName); err != nil {
		return err
	}
	if strings.TrimSpace(baseURL) != "" {
		if err := config.SaveConfigValue(path, baseURLKey, strings.TrimSpace(baseURL)); err != nil {
			return err
		}
	}
	return nil
}

func modelConfigKeys(provider string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return "api.anthropic.model", "api.anthropic.base_url"
	case "codex":
		return "api.codex.model", "api.codex.base_url"
	default:
		return "api.model", "api.base_url"
	}
}

func runChat(cfg config.Config, first string, sessionID ...string) {
	eng, cronStore := mustBuildEngine(cfg)
	id := uuid.NewString()
	skills := ""
	if len(sessionID) > 0 && sessionID[0] != "" {
		id = sessionID[0]
	}
	if len(sessionID) > 1 && sessionID[1] != "" {
		skills = sessionID[1]
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cfg.CronEnabled && cronStore != nil {
		s := &cronrunner.Scheduler{
			Store:          cronStore,
			Engine:         eng,
			Tick:           time.Duration(cfg.CronTickSeconds) * time.Second,
			MaxConcurrency: cfg.CronMaxConcurrency,
		}
		if err := s.Start(ctx); err != nil {
			log.Printf("cron scheduler start failed: %v", err)
		} else {
			log.Printf("cron scheduler enabled (tick=%ds max_concurrency=%d)", cfg.CronTickSeconds, cfg.CronMaxConcurrency)
		}
	}
	if err := cli.RunChat(ctx, eng, id, first, skills); err != nil {
		log.Fatal(err)
	}
}

func runServe(cfg config.Config) {
	if cfg.GatewayEnabled {
		cfg.ModelUseStreaming = true
	}
	eng, cronStore := mustBuildEngine(cfg)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: (&api.Server{Engine: eng}).Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("agent-daemon listening on %s", cfg.ListenAddr)

	cronCtx, cronCancel := context.WithCancel(context.Background())
	defer cronCancel()
	if cfg.CronEnabled && cronStore != nil {
		s := &cronrunner.Scheduler{
			Store:          cronStore,
			Engine:         eng,
			Tick:           time.Duration(cfg.CronTickSeconds) * time.Second,
			MaxConcurrency: cfg.CronMaxConcurrency,
		}
		if err := s.Start(cronCtx); err != nil {
			log.Printf("cron scheduler start failed: %v", err)
		} else {
			log.Printf("cron scheduler enabled (tick=%ds max_concurrency=%d)", cfg.CronTickSeconds, cfg.CronMaxConcurrency)
		}
	}

	if cfg.GatewayEnabled {
		log.Printf("gateway enabled")
		gatewayCtx, gatewayCancel := context.WithCancel(context.Background())
		defer gatewayCancel()

		adapters := buildGatewayAdapters(cfg)
		if len(adapters) > 0 {
			runner := gateway.NewRunner(adapters, eng, func(platform string) string {
				switch platform {
				case "telegram":
					return cfg.TelegramAllowed
				case "discord":
					return cfg.DiscordAllowed
				case "slack":
					return cfg.SlackAllowed
				}
				return ""
			})
			if err := runner.Start(gatewayCtx); err != nil {
				log.Printf("gateway start failed: %v", err)
			}
		} else {
			log.Printf("gateway enabled but no platform adapters configured")
		}

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			log.Printf("shutting down...")
			gatewayCancel()
			cronCancel()
			_ = srv.Shutdown(context.Background())
		}()
	}

	log.Fatal(srv.ListenAndServe())
}

func buildGatewayAdapters(cfg config.Config) []gateway.PlatformAdapter {
	var adapters []gateway.PlatformAdapter
	if strings.TrimSpace(cfg.TelegramToken) != "" {
		ta, err := platforms.NewTelegramAdapter(cfg.TelegramToken)
		if err != nil {
			log.Printf("telegram adapter: %v", err)
		} else {
			adapters = append(adapters, ta)
			log.Printf("telegram adapter configured")
		}
	}
	if strings.TrimSpace(cfg.DiscordToken) != "" {
		da, err := platforms.NewDiscordAdapter(cfg.DiscordToken)
		if err != nil {
			log.Printf("discord adapter: %v", err)
		} else {
			adapters = append(adapters, da)
			log.Printf("discord adapter configured")
		}
	}
	if strings.TrimSpace(cfg.SlackBotToken) != "" && strings.TrimSpace(cfg.SlackAppToken) != "" {
		sa, err := platforms.NewSlackAdapter(cfg.SlackBotToken, cfg.SlackAppToken)
		if err != nil {
			log.Printf("slack adapter: %v", err)
		} else {
			adapters = append(adapters, sa)
			log.Printf("slack adapter configured")
		}
	}
	return adapters
}

func applyDisabledTools(registry *tools.Registry, disabled string) {
	registry.Disable(parseNameList(disabled)...)
}

func applyEnabledToolsets(registry *tools.Registry, enabled string) error {
	names := parseNameList(enabled)
	if len(names) == 0 {
		return nil
	}
	allowed, err := tools.ResolveToolset(names)
	if err != nil {
		return err
	}
	all := registry.Names()
	disable := make([]string, 0, len(all))
	for _, toolName := range all {
		if _, ok := allowed[toolName]; !ok {
			disable = append(disable, toolName)
		}
	}
	registry.Disable(disable...)
	return nil
}

func mustBuildEngine(cfg config.Config) (*agent.Engine, *store.CronStore) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatal(err)
	}
	sessionStore, err := store.NewSessionStore(filepath.Join(cfg.DataDir, "sessions.db"))
	if err != nil {
		log.Fatal(err)
	}
	cronStore, err := store.NewCronStore(sessionStore.DB())
	if err != nil {
		log.Printf("cron store init failed: %v", err)
		cronStore = nil
	}
	memoryStore, err := memory.NewStore(cfg.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	registry := tools.NewRegistry()
	proc := tools.NewProcessRegistry(filepath.Join(cfg.DataDir, "processes"))
	tools.RegisterBuiltins(registry, proc)
	if cronStore != nil {
		registry.Register(tools.NewCronJobTool(cronStore))
	}
	registry.Register(tools.NewSendMessageTool())
	approvalStore := tools.NewPersistentApprovalStore(time.Duration(cfg.ApprovalTTLSeconds)*time.Second, sessionStore)
	switch strings.ToLower(strings.TrimSpace(cfg.MCPTransport)) {
	case "stdio":
		if strings.TrimSpace(cfg.MCPStdioCommand) != "" {
			mcpClient := tools.NewMCPStdioClient(cfg.MCPStdioCommand, time.Duration(cfg.MCPTimeoutSeconds)*time.Second)
			if names, err := tools.RegisterMCPTools(context.Background(), registry, mcpClient); err != nil {
				log.Printf("mcp stdio discovery failed: %v", err)
			} else if len(names) > 0 {
				log.Printf("registered %d mcp tools via stdio command", len(names))
			}
		}
	default:
		if strings.TrimSpace(cfg.MCPEndpoint) != "" {
			mcpClient := tools.NewMCPClient(cfg.MCPEndpoint, time.Duration(cfg.MCPTimeoutSeconds)*time.Second)
			mcpClient.TokenStore = sessionStore
			if strings.TrimSpace(cfg.MCPOAuthTokenURL) != "" {
				grantType := strings.ToLower(strings.TrimSpace(cfg.MCPOAuthGrantType))
				if grantType == "authorization_code" {
					mcpClient.ConfigureOAuthAuthCode(tools.MCPOAuthConfig{
						TokenURL:     cfg.MCPOAuthTokenURL,
						AuthURL:      cfg.MCPOAuthAuthURL,
						RedirectURL:  cfg.MCPOAuthRedirectURL,
						ClientID:     cfg.MCPOAuthClientID,
						ClientSecret: cfg.MCPOAuthClientSecret,
						Scopes:       cfg.MCPOAuthScopes,
					})
					done := make(chan string, 1)
					if err := mcpClient.StartOAuthCallbackServer(cfg.MCPOAuthCallbackPort, done); err != nil {
						log.Printf("mcp oauth callback server failed: %v", err)
					} else {
						authURL := mcpClient.BuildAuthURL("mcp-auth")
						log.Printf("mcp oauth: open this URL to authorize: %s", authURL)
						select {
						case <-done:
							log.Printf("mcp oauth: authorization successful")
						case <-time.After(5 * time.Minute):
							log.Printf("mcp oauth: authorization timed out")
						}
					}
				} else {
					mcpClient.ConfigureOAuthClientCredentials(tools.MCPOAuthConfig{
						TokenURL:     cfg.MCPOAuthTokenURL,
						ClientID:     cfg.MCPOAuthClientID,
						ClientSecret: cfg.MCPOAuthClientSecret,
						Scopes:       cfg.MCPOAuthScopes,
					})
				}
			}
			if names, err := tools.RegisterMCPTools(context.Background(), registry, mcpClient); err != nil {
				log.Printf("mcp discovery failed: %v", err)
			} else if len(names) > 0 {
				log.Printf("registered %d mcp tools from %s", len(names), cfg.MCPEndpoint)
			}
		}
	}
	if err := applyEnabledToolsets(registry, cfg.EnabledToolsets); err != nil {
		log.Printf("enabled_toolsets ignored: %v", err)
	}
	applyDisabledTools(registry, cfg.DisabledTools)
	client := buildModelClient(cfg)
	return &agent.Engine{
		Client:                  client,
		Registry:                registry,
		SessionStore:            sessionStore,
		SearchStore:             sessionStore,
		MemoryStore:             memoryStore,
		TodoStore:               tools.NewTodoStore(),
		ApprovalStore:           approvalStore,
		Workdir:                 cfg.Workdir,
		SystemPrompt:            agent.DefaultSystemPrompt(),
		MaxIterations:           cfg.MaxIterations,
		MaxContextChars:         cfg.MaxContextChars,
		CompressionTailMessages: cfg.CompressionTailMessages,
	}, cronStore
}

func buildModelClient(cfg config.Config) model.Client {
	if strings.TrimSpace(cfg.ModelCascade) != "" {
		return buildCascadeClient(cfg)
	}

	primaryProvider := strings.ToLower(strings.TrimSpace(cfg.ModelProvider))
	primary := buildProviderClient(cfg, primaryProvider)
	fallbackProvider := strings.ToLower(strings.TrimSpace(cfg.ModelFallbackProvider))
	if fallbackProvider == "" || fallbackProvider == primaryProvider {
		return primary
	}
	fallback := buildProviderClient(cfg, fallbackProvider)
	if fallback == nil {
		return primary
	}

	circuitThreshold := cfg.ModelCircuitThreshold
	circuitRecovery := time.Duration(cfg.ModelCircuitRecoverySec) * time.Second
	circuitHalfOpenMax := cfg.ModelCircuitHalfOpenMax

	if cfg.ModelRaceEnabled {
		log.Printf("model race enabled: primary=%s fallback=%s", primaryProvider, fallbackProvider)
		return model.NewRaceClient(primary, primaryProvider, fallback, fallbackProvider, circuitThreshold, circuitRecovery, circuitHalfOpenMax)
	}

	log.Printf("model fallback enabled: primary=%s fallback=%s", primaryProvider, fallbackProvider)
	return model.NewFallbackClientWithCircuit(primary, primaryProvider, fallback, fallbackProvider, circuitThreshold, circuitRecovery, circuitHalfOpenMax)
}

func buildProviderClient(cfg config.Config, provider string) model.Client {
	switch provider {
	case "anthropic":
		client := model.NewAnthropicClient(cfg.AnthropicBaseURL, cfg.AnthropicAPIKey, cfg.AnthropicModel)
		client.UseStreaming = cfg.ModelUseStreaming
		return client
	case "codex":
		client := model.NewCodexClient(cfg.CodexBaseURL, cfg.CodexAPIKey, cfg.CodexModel)
		client.UseStreaming = cfg.ModelUseStreaming
		return client
	case "openai", "":
		client := model.NewOpenAIClient(cfg.ModelBaseURL, cfg.ModelAPIKey, cfg.ModelName)
		client.UseStreaming = cfg.ModelUseStreaming
		return client
	default:
		client := model.NewOpenAIClient(cfg.ModelBaseURL, cfg.ModelAPIKey, cfg.ModelName)
		client.UseStreaming = cfg.ModelUseStreaming
		return client
	}
}

func buildCascadeClient(cfg config.Config) model.Client {
	builder := func(name string) (model.Client, string, error) {
		client := buildProviderClient(cfg, name)
		if client == nil {
			return nil, "", fmt.Errorf("unknown provider: %s", name)
		}
		return client, name, nil
	}
	entries, err := model.ParseCascadeProviders(cfg.ModelCascade, builder)
	if err != nil {
		log.Printf("cascade parse error: %v, falling back to single provider", err)
		return buildProviderClient(cfg, cfg.ModelProvider)
	}
	circuitThreshold := cfg.ModelCircuitThreshold
	circuitRecovery := time.Duration(cfg.ModelCircuitRecoverySec) * time.Second
	circuitHalfOpenMax := cfg.ModelCircuitHalfOpenMax

	if cfg.ModelCostAware {
		log.Printf("model cascade (cost-aware): %d providers", len(entries))
	} else {
		log.Printf("model cascade (ordered): %d providers", len(entries))
	}
	return model.NewCascadeClientWithCircuit(entries, cfg.ModelCostAware, circuitThreshold, circuitRecovery, circuitHalfOpenMax)
}
