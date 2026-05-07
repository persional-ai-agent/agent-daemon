package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	ModelBaseURL  string
	ModelAPIKey   string
	ModelName     string
	MaxIterations int
	DataDir       string
	ListenAddr    string
	Workdir       string
}

func Load() Config {
	home, _ := os.UserHomeDir()
	dataDir := getenv("AGENT_DAEMON_HOME", filepath.Join(home, ".agent-daemon"))
	maxTurns := 30
	if v := os.Getenv("AGENT_MAX_ITERATIONS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			maxTurns = i
		}
	}
	wd, _ := os.Getwd()
	return Config{
		ModelBaseURL:  getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		ModelAPIKey:   os.Getenv("OPENAI_API_KEY"),
		ModelName:     getenv("OPENAI_MODEL", "gpt-4o-mini"),
		MaxIterations: maxTurns,
		DataDir:       dataDir,
		ListenAddr:    getenv("AGENT_DAEMON_ADDR", ":8080"),
		Workdir:       getenv("AGENT_WORKDIR", wd),
	}
}

func getenv(k, d string) string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return v
}
