package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/api"
	"github.com/dingjingmaster/agent-daemon/internal/cli"
	"github.com/dingjingmaster/agent-daemon/internal/config"
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
		_ = fs.Parse(os.Args[2:])
		runChat(cfg, *message, *sessionID)
	case "serve":
		runServe(cfg)
	case "tools":
		eng := mustBuildEngine(cfg)
		for _, name := range eng.Registry.Names() {
			fmt.Println(name)
		}
	default:
		runChat(cfg, "", uuid.NewString())
	}
}

func runChat(cfg config.Config, first string, sessionID ...string) {
	eng := mustBuildEngine(cfg)
	id := uuid.NewString()
	if len(sessionID) > 0 && sessionID[0] != "" {
		id = sessionID[0]
	}
	if err := cli.RunChat(context.Background(), eng, id, first); err != nil {
		log.Fatal(err)
	}
}

func runServe(cfg config.Config) {
	eng := mustBuildEngine(cfg)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: (&api.Server{Engine: eng}).Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("agent-daemon listening on %s", cfg.ListenAddr)
	log.Fatal(srv.ListenAndServe())
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
	approvalStore := tools.NewApprovalStore(time.Duration(cfg.ApprovalTTLSeconds) * time.Second)
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
			if strings.TrimSpace(cfg.MCPOAuthTokenURL) != "" {
				mcpClient.ConfigureOAuthClientCredentials(tools.MCPOAuthConfig{
					TokenURL:     cfg.MCPOAuthTokenURL,
					ClientID:     cfg.MCPOAuthClientID,
					ClientSecret: cfg.MCPOAuthClientSecret,
					Scopes:       cfg.MCPOAuthScopes,
				})
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
