package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/api"
	"github.com/dingjingmaster/agent-daemon/internal/cli"
	"github.com/dingjingmaster/agent-daemon/internal/config"
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
		eng := mustBuildEngine(cfg)
		for _, name := range eng.Registry.Names() {
			fmt.Println(name)
		}
	case "config":
		runConfig(os.Args[2:])
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

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agentd config list [-file path] [-show-secrets]")
	fmt.Fprintln(os.Stderr, "  agentd config get [-file path] section.key")
	fmt.Fprintln(os.Stderr, "  agentd config set [-file path] section.key value")
}

func runChat(cfg config.Config, first string, sessionID ...string) {
	eng := mustBuildEngine(cfg)
	id := uuid.NewString()
	skills := ""
	if len(sessionID) > 0 && sessionID[0] != "" {
		id = sessionID[0]
	}
	if len(sessionID) > 1 && sessionID[1] != "" {
		skills = sessionID[1]
	}
	if err := cli.RunChat(context.Background(), eng, id, first, skills); err != nil {
		log.Fatal(err)
	}
}

func runServe(cfg config.Config) {
	if cfg.GatewayEnabled {
		cfg.ModelUseStreaming = true
	}
	eng := mustBuildEngine(cfg)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: (&api.Server{Engine: eng}).Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("agent-daemon listening on %s", cfg.ListenAddr)

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

func mustBuildEngine(cfg config.Config) *agent.Engine {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatal(err)
	}
	sessionStore, err := store.NewSessionStore(filepath.Join(cfg.DataDir, "sessions.db"))
	if err != nil {
		log.Fatal(err)
	}
	memoryStore, err := memory.NewStore(cfg.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	registry := tools.NewRegistry()
	proc := tools.NewProcessRegistry(filepath.Join(cfg.DataDir, "processes"))
	tools.RegisterBuiltins(registry, proc)
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
	}
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
