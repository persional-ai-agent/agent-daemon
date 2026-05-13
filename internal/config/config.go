package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
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
	MCPOAuthAuthURL         string
	MCPOAuthRedirectURL     string
	MCPOAuthClientID        string
	MCPOAuthClientSecret    string
	MCPOAuthScopes          string
	MCPOAuthGrantType       string
	MCPOAuthCallbackPort    int
	MCPTimeoutSeconds       int
	ApprovalTTLSeconds      int
	MaxIterations           int
	MaxContextChars         int
	CompressionTailMessages int
	DataDir                 string
	ListenAddr              string
	Workdir                 string
	GatewayEnabled          bool
	TelegramToken           string
	TelegramAllowed         string
	DiscordToken            string
	DiscordAllowed          string
	SlackBotToken           string
	SlackAppToken           string
	SlackAllowed            string
	YuanbaoToken            string
	YuanbaoAppID            string
	YuanbaoAppSecret        string
	YuanbaoAllowed          string
	ModelCascade            string
	ModelCostAware          bool
	DisabledTools           string
	CronEnabled             bool
	CronTickSeconds         int
	CronMaxConcurrency      int
	EnabledToolsets         string
	UITUIWSBase             string
	UITUIHTTPBase           string
	UITUIWSReadTimeoutSec   int
	UITUITurnTimeoutSec     int
	UITUIReconnectMax       int
	UITUIHistoryMaxLines    int
	UITUIEventMaxItems      int
	UITUIViewMode           string
	UITUIAutoDoctor         *bool
}

type iniValues struct {
	file  *ini.File
	found bool
}

func loadConfigINI() iniValues {
	if p := strings.TrimSpace(os.Getenv("AGENT_CONFIG_FILE")); p != "" {
		if f, err := ini.Load(p); err == nil {
			return iniValues{file: f, found: true}
		}
	}
	for _, p := range []string{"config/config.ini", "config.ini", "../config/config.ini"} {
		f, err := ini.Load(p)
		if err == nil {
			return iniValues{file: f, found: true}
		}
	}
	return iniValues{}
}

func iniStr(iv iniValues, section, key, envVar, def string) string {
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return v
	}
	if iv.found && iv.file != nil {
		if sec := iv.file.Section(section); sec != nil && sec.HasKey(key) {
			return sec.Key(key).String()
		}
	}
	return def
}

func iniBool(iv iniValues, section, key, envVar string) bool {
	if v := os.Getenv(envVar); v != "" {
		return strings.EqualFold(v, "true") || v == "1"
	}
	if iv.found && iv.file != nil {
		if sec := iv.file.Section(section); sec != nil && sec.HasKey(key) {
			v := sec.Key(key).String()
			return strings.EqualFold(v, "true") || v == "1"
		}
	}
	return false
}

func iniInt(iv iniValues, section, key, envVar string, def int) int {
	if v := os.Getenv(envVar); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	if iv.found && iv.file != nil {
		if sec := iv.file.Section(section); sec != nil && sec.HasKey(key) {
			if i, err := sec.Key(key).Int(); err == nil {
				return i
			}
		}
	}
	return def
}

func Load() Config {
	return loadFromINIValues(loadConfigINI())
}

func LoadFile(path string) (Config, error) {
	f, err := LoadConfigFile(path)
	if err != nil {
		return Config{}, err
	}
	return loadFromINIValues(iniValues{file: f, found: true}), nil
}

func loadFromINIValues(iv iniValues) Config {
	home, _ := os.UserHomeDir()
	dataDir := iniStr(iv, "agent", "data_dir", "AGENT_DAEMON_HOME", filepath.Join(home, ".agent-daemon"))

	maxTurns := iniInt(iv, "agent", "max_iterations", "AGENT_MAX_ITERATIONS", 30)
	if maxTurns <= 0 {
		maxTurns = 30
	}
	maxContextChars := iniInt(iv, "agent", "max_context_chars", "AGENT_MAX_CONTEXT_CHARS", 120000)
	if maxContextChars <= 0 {
		maxContextChars = 120000
	}
	tailMessages := iniInt(iv, "agent", "compression_tail_messages", "AGENT_COMPRESSION_TAIL_MESSAGES", 14)
	if tailMessages <= 0 {
		tailMessages = 14
	}

	wd, _ := os.Getwd()

	apiType := iniStr(iv, "api", "type", "AGENT_MODEL_PROVIDER", "openai")
	baseURL := iniStr(iv, "api", "base_url", "OPENAI_BASE_URL", "https://api.openai.com/v1")
	apiKey := iniStr(iv, "api", "api_key", "OPENAI_API_KEY", "")
	model := iniStr(iv, "api", "model", "OPENAI_MODEL", "gpt-4o-mini")

	codexBase := iniStr(iv, "api.codex", "base_url", "CODEX_BASE_URL", baseURL)
	codexKey := iniStr(iv, "api.codex", "api_key", "CODEX_API_KEY", apiKey)
	codexModel := iniStr(iv, "api.codex", "model", "CODEX_MODEL", "gpt-5-codex")

	anthropicBase := iniStr(iv, "api.anthropic", "base_url", "ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1")
	anthropicKey := iniStr(iv, "api.anthropic", "api_key", "ANTHROPIC_API_KEY", "")
	anthropicModel := iniStr(iv, "api.anthropic", "model", "ANTHROPIC_MODEL", "claude-3-5-haiku-latest")

	mcpTimeout := iniInt(iv, "mcp", "timeout_seconds", "AGENT_MCP_TIMEOUT_SECONDS", 30)
	if mcpTimeout <= 0 {
		mcpTimeout = 30
	}
	mcpCallbackPort := iniInt(iv, "mcp", "oauth_callback_port", "AGENT_MCP_OAUTH_CALLBACK_PORT", 9876)
	if mcpCallbackPort <= 0 {
		mcpCallbackPort = 9876
	}

	approvalTTL := iniInt(iv, "agent", "approval_ttl_seconds", "AGENT_APPROVAL_TTL_SECONDS", 300)
	if approvalTTL <= 0 {
		approvalTTL = 300
	}

	circuitThreshold := iniInt(iv, "provider", "circuit_failure_threshold", "AGENT_MODEL_CIRCUIT_FAILURE_THRESHOLD", 3)
	if circuitThreshold <= 0 {
		circuitThreshold = 3
	}
	circuitRecoverySec := iniInt(iv, "provider", "circuit_recovery_seconds", "AGENT_MODEL_CIRCUIT_RECOVERY_TIMEOUT_SECONDS", 60)
	if circuitRecoverySec <= 0 {
		circuitRecoverySec = 60
	}
	circuitHalfOpenMax := iniInt(iv, "provider", "circuit_half_open_max_requests", "AGENT_MODEL_CIRCUIT_HALF_OPEN_MAX_REQUESTS", 1)
	if circuitHalfOpenMax <= 0 {
		circuitHalfOpenMax = 1
	}

	streaming := iniBool(iv, "api", "streaming", "AGENT_MODEL_USE_STREAMING")
	raceEnabled := iniBool(iv, "provider", "race_enabled", "AGENT_MODEL_RACE_ENABLED")
	costAware := iniBool(iv, "provider", "cost_aware", "AGENT_MODEL_COST_AWARE")
	gatewayEnabled := iniBool(iv, "gateway", "enabled", "AGENT_GATEWAY_ENABLED")
	cronEnabled := iniBool(iv, "cron", "enabled", "AGENT_CRON_ENABLED")
	cronTickSeconds := iniInt(iv, "cron", "tick_seconds", "AGENT_CRON_TICK_SECONDS", 5)
	if cronTickSeconds <= 0 {
		cronTickSeconds = 5
	}
	cronMaxConc := iniInt(iv, "cron", "max_concurrency", "AGENT_CRON_MAX_CONCURRENCY", 1)
	if cronMaxConc <= 0 {
		cronMaxConc = 1
	}
	uiWSReadTimeout := iniInt(iv, "ui-tui", "ws_read_timeout_seconds", "AGENT_UI_TUI_WS_READ_TIMEOUT_SECONDS", 45)
	if uiWSReadTimeout <= 0 {
		uiWSReadTimeout = 45
	}
	uiTurnTimeout := iniInt(iv, "ui-tui", "ws_turn_timeout_seconds", "AGENT_UI_TUI_WS_TURN_TIMEOUT_SECONDS", 480)
	if uiTurnTimeout <= 0 {
		uiTurnTimeout = 480
	}
	uiReconnectMax := iniInt(iv, "ui-tui", "ws_reconnect_max", "AGENT_UI_TUI_WS_RECONNECT_MAX", 2)
	if uiReconnectMax < 0 {
		uiReconnectMax = 0
	}
	uiHistoryMaxLines := iniInt(iv, "ui-tui", "history_max_lines", "AGENT_UI_TUI_HISTORY_MAX_LINES", 2000)
	if uiHistoryMaxLines <= 0 {
		uiHistoryMaxLines = 2000
	}
	uiEventMaxItems := iniInt(iv, "ui-tui", "event_max_items", "AGENT_UI_TUI_EVENT_MAX_ITEMS", 2000)
	if uiEventMaxItems <= 0 {
		uiEventMaxItems = 2000
	}
	uiViewMode := iniStr(iv, "ui-tui", "view_mode", "AGENT_UI_TUI_VIEW_MODE", "human")
	uiViewMode = strings.ToLower(strings.TrimSpace(uiViewMode))
	if uiViewMode != "json" && uiViewMode != "human" {
		uiViewMode = "human"
	}
	var autoDoctorPtr *bool
	if v := os.Getenv("AGENT_UI_TUI_AUTO_DOCTOR"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		autoDoctorPtr = &b
	} else if iv.found && iv.file != nil {
		if sec := iv.file.Section("ui-tui"); sec != nil && sec.HasKey("auto_doctor") {
			v := sec.Key("auto_doctor").String()
			b := strings.EqualFold(v, "true") || v == "1"
			autoDoctorPtr = &b
		}
	}

	return Config{
		ModelProvider:           apiType,
		ModelFallbackProvider:   iniStr(iv, "provider", "fallback", "AGENT_MODEL_FALLBACK_PROVIDER", ""),
		ModelUseStreaming:       streaming,
		ModelRaceEnabled:        raceEnabled,
		ModelCircuitThreshold:   circuitThreshold,
		ModelCircuitRecoverySec: circuitRecoverySec,
		ModelCircuitHalfOpenMax: circuitHalfOpenMax,
		ModelBaseURL:            baseURL,
		ModelAPIKey:             apiKey,
		ModelName:               model,
		CodexBaseURL:            codexBase,
		CodexAPIKey:             codexKey,
		CodexModel:              codexModel,
		AnthropicBaseURL:        anthropicBase,
		AnthropicAPIKey:         anthropicKey,
		AnthropicModel:          anthropicModel,
		MCPEndpoint:             iniStr(iv, "mcp", "endpoint", "AGENT_MCP_ENDPOINT", ""),
		MCPTransport:            iniStr(iv, "mcp", "transport", "AGENT_MCP_TRANSPORT", "http"),
		MCPStdioCommand:         iniStr(iv, "mcp", "stdio_command", "AGENT_MCP_STDIO_COMMAND", ""),
		MCPOAuthTokenURL:        iniStr(iv, "mcp", "oauth_token_url", "AGENT_MCP_OAUTH_TOKEN_URL", ""),
		MCPOAuthAuthURL:         iniStr(iv, "mcp", "oauth_auth_url", "AGENT_MCP_OAUTH_AUTH_URL", ""),
		MCPOAuthRedirectURL:     iniStr(iv, "mcp", "oauth_redirect_url", "AGENT_MCP_OAUTH_REDIRECT_URL", ""),
		MCPOAuthClientID:        iniStr(iv, "mcp", "oauth_client_id", "AGENT_MCP_OAUTH_CLIENT_ID", ""),
		MCPOAuthClientSecret:    iniStr(iv, "mcp", "oauth_client_secret", "AGENT_MCP_OAUTH_CLIENT_SECRET", ""),
		MCPOAuthScopes:          iniStr(iv, "mcp", "oauth_scopes", "AGENT_MCP_OAUTH_SCOPES", ""),
		MCPOAuthGrantType:       iniStr(iv, "mcp", "oauth_grant_type", "AGENT_MCP_OAUTH_GRANT_TYPE", ""),
		MCPOAuthCallbackPort:    mcpCallbackPort,
		MCPTimeoutSeconds:       mcpTimeout,
		ApprovalTTLSeconds:      approvalTTL,
		MaxIterations:           maxTurns,
		MaxContextChars:         maxContextChars,
		CompressionTailMessages: tailMessages,
		DataDir:                 dataDir,
		ListenAddr:              iniStr(iv, "agent", "listen_addr", "AGENT_DAEMON_ADDR", ":8080"),
		Workdir:                 iniStr(iv, "agent", "workdir", "AGENT_WORKDIR", wd),
		GatewayEnabled:          gatewayEnabled,
		TelegramToken:           iniStr(iv, "gateway.telegram", "bot_token", "AGENT_TELEGRAM_BOT_TOKEN", ""),
		TelegramAllowed:         iniStr(iv, "gateway.telegram", "allowed_users", "AGENT_TELEGRAM_ALLOWED_USERS", ""),
		DiscordToken:            iniStr(iv, "gateway.discord", "bot_token", "AGENT_DISCORD_BOT_TOKEN", ""),
		DiscordAllowed:          iniStr(iv, "gateway.discord", "allowed_users", "AGENT_DISCORD_ALLOWED_USERS", ""),
		SlackBotToken:           iniStr(iv, "gateway.slack", "bot_token", "AGENT_SLACK_BOT_TOKEN", ""),
		SlackAppToken:           iniStr(iv, "gateway.slack", "app_token", "AGENT_SLACK_APP_TOKEN", ""),
		SlackAllowed:            iniStr(iv, "gateway.slack", "allowed_users", "AGENT_SLACK_ALLOWED_USERS", ""),
		YuanbaoToken:            iniStr(iv, "gateway.yuanbao", "token", "YUANBAO_TOKEN", ""),
		YuanbaoAppID:            iniStr(iv, "gateway.yuanbao", "app_id", "YUANBAO_APP_ID", ""),
		YuanbaoAppSecret:        iniStr(iv, "gateway.yuanbao", "app_secret", "YUANBAO_APP_SECRET", ""),
		YuanbaoAllowed:          iniStr(iv, "gateway.yuanbao", "allowed_users", "AGENT_YUANBAO_ALLOWED_USERS", ""),
		ModelCascade:            iniStr(iv, "provider", "cascade", "AGENT_MODEL_CASCADE", ""),
		ModelCostAware:          costAware,
		DisabledTools:           iniStr(iv, "tools", "disabled", "AGENT_DISABLED_TOOLS", ""),
		EnabledToolsets:         iniStr(iv, "tools", "enabled_toolsets", "AGENT_ENABLED_TOOLSETS", ""),
		CronEnabled:             cronEnabled,
		CronTickSeconds:         cronTickSeconds,
		CronMaxConcurrency:      cronMaxConc,
		UITUIWSBase:             iniStr(iv, "ui-tui", "ws_base", "AGENT_API_BASE", ""),
		UITUIHTTPBase:           iniStr(iv, "ui-tui", "http_base", "AGENT_HTTP_BASE", ""),
		UITUIWSReadTimeoutSec:   uiWSReadTimeout,
		UITUITurnTimeoutSec:     uiTurnTimeout,
		UITUIReconnectMax:       uiReconnectMax,
		UITUIHistoryMaxLines:    uiHistoryMaxLines,
		UITUIEventMaxItems:      uiEventMaxItems,
		UITUIViewMode:           uiViewMode,
		UITUIAutoDoctor:         autoDoctorPtr,
	}
}
