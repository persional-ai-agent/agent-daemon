package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	ModelProvider           string
	ModelFallbackProvider   string
	ModelUseStreaming       bool
	ModelRaceEnabled        bool
	ModelCircuitThreshold   int
	ModelCircuitRecoverySec int
	ModelCircuitHalfOpenMax int
	ModelBaseURL            string
	ModelAPIKey             string
	ModelName               string
	CodexBaseURL            string
	CodexAPIKey             string
	CodexModel              string
	AnthropicBaseURL        string
	AnthropicAPIKey         string
	AnthropicModel          string
	MCPEndpoint             string
	MCPTransport            string
	MCPStdioCommand         string
	MCPOAuthTokenURL        string
	MCPOAuthClientID        string
	MCPOAuthClientSecret    string
	MCPOAuthScopes          string
	MCPTimeoutSeconds       int
	ApprovalTTLSeconds      int
	MaxIterations           int
	MaxContextChars         int
	CompressionTailMessages int
	DataDir                 string
	ListenAddr              string
	Workdir                 string
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
	maxContextChars := 120000
	if v := os.Getenv("AGENT_MAX_CONTEXT_CHARS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			maxContextChars = i
		}
	}
	tailMessages := 14
	if v := os.Getenv("AGENT_COMPRESSION_TAIL_MESSAGES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			tailMessages = i
		}
	}
	mcpTimeout := 30
	if v := os.Getenv("AGENT_MCP_TIMEOUT_SECONDS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			mcpTimeout = i
		}
	}
	approvalTTL := 300
	if v := os.Getenv("AGENT_APPROVAL_TTL_SECONDS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			approvalTTL = i
		}
	}
	raceEnabled := strings.EqualFold(os.Getenv("AGENT_MODEL_RACE_ENABLED"), "true") || os.Getenv("AGENT_MODEL_RACE_ENABLED") == "1"
	circuitThreshold := 3
	if v := os.Getenv("AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			circuitThreshold = i
		}
	}
	circuitRecoverySec := 60
	if v := os.Getenv("AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			circuitRecoverySec = i
		}
	}
	circuitHalfOpenMax := 1
	if v := os.Getenv("AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			circuitHalfOpenMax = i
		}
	}
	wd, _ := os.Getwd()
	return Config{
		ModelProvider:           getenv("AGENT_MODEL_PROVIDER", "openai"),
		ModelFallbackProvider:   strings.TrimSpace(os.Getenv("AGENT_MODEL_FALLBACK_PROVIDER")),
		ModelUseStreaming:       getenv("AGENT_MODEL_USE_STREAMING", "") == "1" || strings.EqualFold(getenv("AGENT_MODEL_USE_STREAMING", ""), "true"),
		ModelRaceEnabled:        raceEnabled,
		ModelCircuitThreshold:   circuitThreshold,
		ModelCircuitRecoverySec: circuitRecoverySec,
		ModelCircuitHalfOpenMax: circuitHalfOpenMax,
		ModelBaseURL:            getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		ModelAPIKey:             os.Getenv("OPENAI_API_KEY"),
		ModelName:               getenv("OPENAI_MODEL", "gpt-4o-mini"),
		CodexBaseURL:            getenv("CODEX_BASE_URL", getenv("OPENAI_BASE_URL", "https://api.openai.com/v1")),
		CodexAPIKey:             getenv("CODEX_API_KEY", os.Getenv("OPENAI_API_KEY")),
		CodexModel:              getenv("CODEX_MODEL", "gpt-5-codex"),
		AnthropicBaseURL:        getenv("ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1"),
		AnthropicAPIKey:         os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:          getenv("ANTHROPIC_MODEL", "claude-3-5-haiku-latest"),
		MCPEndpoint:             strings.TrimSpace(os.Getenv("AGENT_MCP_ENDPOINT")),
		MCPTransport:            getenv("AGENT_MCP_TRANSPORT", "http"),
		MCPStdioCommand:         strings.TrimSpace(os.Getenv("AGENT_MCP_STDIO_COMMAND")),
		MCPOAuthTokenURL:        strings.TrimSpace(os.Getenv("AGENT_MCP_OAUTH_TOKEN_URL")),
		MCPOAuthClientID:        strings.TrimSpace(os.Getenv("AGENT_MCP_OAUTH_CLIENT_ID")),
		MCPOAuthClientSecret:    os.Getenv("AGENT_MCP_OAUTH_CLIENT_SECRET"),
		MCPOAuthScopes:          strings.TrimSpace(os.Getenv("AGENT_MCP_OAUTH_SCOPES")),
		MCPTimeoutSeconds:       mcpTimeout,
		ApprovalTTLSeconds:      approvalTTL,
		MaxIterations:           maxTurns,
		MaxContextChars:         maxContextChars,
		CompressionTailMessages: tailMessages,
		DataDir:                 dataDir,
		ListenAddr:              getenv("AGENT_DAEMON_ADDR", ":8080"),
		Workdir:                 getenv("AGENT_WORKDIR", wd),
	}
}

func getenv(k, d string) string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return v
}
