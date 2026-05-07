package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	ctx := context.Background()
	if err := cli.RunChat(ctx, eng, id, first); err != nil {
		log.Fatal(err)
	}
}

func runServe(cfg config.Config) {
	eng := mustBuildEngine(cfg)
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           (&api.Server{Engine: eng}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
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
	client := model.NewOpenAIClient(cfg.ModelBaseURL, cfg.ModelAPIKey, cfg.ModelName)
	return &agent.Engine{
		Client:        client,
		Registry:      registry,
		SessionStore:  sessionStore,
		SearchStore:   sessionStore,
		MemoryStore:   memoryStore,
		TodoStore:     tools.NewTodoStore(),
		Workdir:       cfg.Workdir,
		MaxIterations: cfg.MaxIterations,
	}
}
