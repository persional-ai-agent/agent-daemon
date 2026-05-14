package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	appconfig "github.com/dingjingmaster/agent-daemon/internal/config"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	BuildVersion = "dev"
	BuildCommit  = "unknown"
	BuildTime    = "unknown"
)

type bookmark struct {
	Name      string `json:"name"`
	SessionID string `json:"session_id"`
	WSBase    string `json:"ws_base"`
	HTTPBase  string `json:"http_base"`
}

type workbenchProfile struct {
	Name             string `json:"name"`
	SessionID        string `json:"session_id"`
	WSBase           string `json:"ws_base"`
	HTTPBase         string `json:"http_base"`
	Fullscreen       bool   `json:"fullscreen"`
	FullscreenPanel  string `json:"fullscreen_panel"`
	PanelAutoRefresh bool   `json:"panel_auto_refresh"`
	PanelRefreshSec  int    `json:"panel_refresh_sec"`
	ViewMode         string `json:"view_mode"`
	UpdatedAt        string `json:"updated_at"`
}

type workflowProfile struct {
	Name      string   `json:"name"`
	Commands  []string `json:"commands"`
	UpdatedAt string   `json:"updated_at"`
}

type runtimeState struct {
	SessionID       string `json:"session_id"`
	WSBase          string `json:"ws_base"`
	HTTPBase        string `json:"http_base"`
	Fullscreen      bool   `json:"fullscreen,omitempty"`
	FullscreenPanel string `json:"fullscreen_panel,omitempty"`
	PanelAuto       bool   `json:"panel_auto,omitempty"`
	PanelInterval   int    `json:"panel_interval_seconds,omitempty"`
	UpdatedAt       string `json:"updated_at"`
}

type doctorItem struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Detail  string `json:"detail"`
	Suggest string `json:"suggest,omitempty"`
	Checked string `json:"checked_at"`
}

type appState struct {
	wsBase   string
	httpBase string
	session  string

	pretty bool

	lastJSON   any
	lastStatus string
	lastDetail string
	lastCode   string

	lastShowSession string
	lastShowOffset  int
	lastShowLimit   int
	lastSessions    []string

	eventLog     []map[string]any
	pendingCache []map[string]any

	historyPath   string
	bookmarkPath  string
	workbenchPath string
	workflowPath  string
	statePath     string
	auditPath     string

	historyMaxLines  int
	eventMaxItems    int
	wsReadTimeout    time.Duration
	wsTurnTimeout    time.Duration
	wsMaxReconnect   int
	viewMode         string
	autoDoctor       bool
	reconnectEnabled bool
	reconnectState   string
	timeoutAction    string
	activeTransport  string
	lastTurnID       string
	reconnectCount   int
	lastErrorCode    string
	lastErrorText    string
	fallbackHint     string
	diagUpdatedAt    string
	fullscreen       bool
	chatLog          []string
	chatMaxLines     int
	fullscreenPanel  string
	panelData        map[string]any
	panelAutoRefresh bool
	panelRefreshSec  int
	lastPanelRefresh time.Time
	commandQueue     []string
}

const (
	defaultHistoryMaxLines = 2000
	defaultEventMaxItems   = 2000
	defaultChatMaxLines    = 2000
	defaultPanelRefreshSec = 8
)

func getenvOr(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func deriveHTTPBase(wsBase string) string {
	u, err := url.Parse(strings.TrimSpace(wsBase))
	if err != nil {
		return "http://127.0.0.1:8080"
	}
	switch u.Scheme {
	case "wss":
		u.Scheme = "https"
	default:
		u.Scheme = "http"
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/")
}

func newState() *appState {
	cfg := appconfig.Load()
	wsBase := strings.TrimSpace(cfg.UITUIWSBase)
	if wsBase == "" {
		wsBase = "ws://127.0.0.1:8080/v1/chat/ws"
	}
	httpBase := strings.TrimSpace(cfg.UITUIHTTPBase)
	if httpBase == "" {
		httpBase = deriveHTTPBase(wsBase)
	}
	session := getenvOr("AGENT_SESSION_ID", uuid.NewString())
	home, _ := os.UserHomeDir()
	if strings.TrimSpace(home) == "" {
		home = "."
	}
	root := filepath.Join(home, ".agent-daemon")
	st := &appState{
		wsBase:           wsBase,
		httpBase:         httpBase,
		session:          session,
		pretty:           true,
		lastStatus:       "ok",
		lastDetail:       "initialized",
		lastCode:         "ok",
		lastShowSession:  session,
		lastShowLimit:    20,
		lastSessions:     make([]string, 0),
		eventLog:         make([]map[string]any, 0),
		historyPath:      filepath.Join(root, "ui-tui-history.log"),
		bookmarkPath:     filepath.Join(root, "ui-tui-bookmarks.json"),
		workbenchPath:    filepath.Join(root, "ui-tui-workbenches.json"),
		workflowPath:     filepath.Join(root, "ui-tui-workflows.json"),
		statePath:        filepath.Join(root, "ui-tui-state.json"),
		auditPath:        filepath.Join(root, "ui-tui-audit.log"),
		historyMaxLines:  cfg.UITUIHistoryMaxLines,
		eventMaxItems:    cfg.UITUIEventMaxItems,
		wsReadTimeout:    time.Duration(cfg.UITUIWSReadTimeoutSec) * time.Second,
		wsTurnTimeout:    time.Duration(cfg.UITUITurnTimeoutSec) * time.Second,
		wsMaxReconnect:   cfg.UITUIReconnectMax,
		viewMode:         "human",
		autoDoctor:       true,
		reconnectEnabled: true,
		reconnectState:   "connecting",
		timeoutAction:    "wait",
		activeTransport:  "ws",
		lastErrorCode:    "ok",
		diagUpdatedAt:    time.Now().Format(time.RFC3339),
		chatLog:          make([]string, 0),
		chatMaxLines:     defaultChatMaxLines,
		fullscreenPanel:  "overview",
		panelData:        make(map[string]any),
		panelAutoRefresh: true,
		panelRefreshSec:  defaultPanelRefreshSec,
		commandQueue:     make([]string, 0),
	}
	st.loadRuntimeState()
	return st
}

func (s *appState) applyConfig(cfg appconfig.Config) {
	if cfg.UITUIHistoryMaxLines > 0 {
		s.historyMaxLines = cfg.UITUIHistoryMaxLines
	}
	if cfg.UITUIEventMaxItems > 0 {
		s.eventMaxItems = cfg.UITUIEventMaxItems
	}
	if cfg.UITUIWSReadTimeoutSec > 0 {
		s.wsReadTimeout = time.Duration(cfg.UITUIWSReadTimeoutSec) * time.Second
	}
	if cfg.UITUITurnTimeoutSec > 0 {
		s.wsTurnTimeout = time.Duration(cfg.UITUITurnTimeoutSec) * time.Second
	}
	if cfg.UITUIReconnectMax >= 0 {
		s.wsMaxReconnect = cfg.UITUIReconnectMax
	}
	if strings.TrimSpace(cfg.UITUIWSBase) != "" {
		s.wsBase = strings.TrimSpace(cfg.UITUIWSBase)
	}
	if strings.TrimSpace(cfg.UITUIHTTPBase) != "" {
		s.httpBase = strings.TrimRight(strings.TrimSpace(cfg.UITUIHTTPBase), "/")
	}
	if strings.TrimSpace(cfg.UITUIViewMode) != "" {
		v := strings.ToLower(strings.TrimSpace(cfg.UITUIViewMode))
		if v == "json" || v == "human" {
			s.viewMode = v
		}
	}
	if cfg.UITUIAutoDoctor != nil {
		s.autoDoctor = *cfg.UITUIAutoDoctor
	}
}

func (s *appState) audit(action, detail string) {
	_ = os.MkdirAll(filepath.Dir(s.auditPath), 0o755)
	f, err := os.OpenFile(s.auditPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(fmt.Sprintf("%s\taction=%s\tsession=%s\tdetail=%s\n", time.Now().Format(time.RFC3339), action, s.session, strings.ReplaceAll(detail, "\n", " ")))
}

func suggestRetry(cmd string) {
	fmt.Printf("retry suggestion: %s\n", cmd)
}

func (s *appState) printData(v any) {
	if s.viewMode == "json" {
		printJSONMode(v, s.pretty)
		return
	}
	switch data := v.(type) {
	case []map[string]any:
		for i, row := range data {
			fmt.Printf("%d. %v\n", i+1, row)
		}
	case map[string]any:
		printJSONMode(data, s.pretty)
	default:
		printJSONMode(v, s.pretty)
	}
}

func (s *appState) saveRuntimeState() error {
	_ = os.MkdirAll(filepath.Dir(s.statePath), 0o755)
	st := runtimeState{
		SessionID:       s.session,
		WSBase:          s.wsBase,
		HTTPBase:        s.httpBase,
		Fullscreen:      s.fullscreen,
		FullscreenPanel: s.fullscreenPanel,
		PanelAuto:       s.panelAutoRefresh,
		PanelInterval:   s.panelRefreshSec,
		UpdatedAt:       time.Now().Format(time.RFC3339),
	}
	bs, _ := json.MarshalIndent(st, "", "  ")
	return os.WriteFile(s.statePath, bs, 0o644)
}

func (s *appState) loadRuntimeState() {
	bs, err := os.ReadFile(s.statePath)
	if err != nil {
		return
	}
	var st runtimeState
	if err := json.Unmarshal(bs, &st); err != nil {
		backup := s.statePath + ".corrupt." + time.Now().Format("20060102T150405")
		_ = os.WriteFile(backup, bs, 0o644)
		_ = os.Remove(s.statePath)
		_ = s.saveRuntimeState()
		return
	}
	if strings.TrimSpace(st.SessionID) != "" {
		s.session = strings.TrimSpace(st.SessionID)
		s.lastShowSession = s.session
	}
	if strings.TrimSpace(st.WSBase) != "" {
		s.wsBase = strings.TrimSpace(st.WSBase)
	}
	if strings.TrimSpace(st.HTTPBase) != "" {
		s.httpBase = strings.TrimRight(strings.TrimSpace(st.HTTPBase), "/")
	}
	s.fullscreen = st.Fullscreen
	if strings.TrimSpace(st.FullscreenPanel) != "" {
		s.fullscreenPanel = strings.TrimSpace(st.FullscreenPanel)
	}
	s.panelAutoRefresh = st.PanelAuto
	if st.PanelInterval > 0 {
		s.panelRefreshSec = st.PanelInterval
	}
}

func (s *appState) setStatus(ok bool, code, detail string) {
	if strings.TrimSpace(code) == "" {
		code = "unknown"
	}
	if ok {
		s.lastStatus = "ok"
	} else {
		s.lastStatus = "err"
	}
	s.lastCode = code
	s.lastDetail = detail
	if ok {
		s.lastErrorCode = "ok"
		s.lastErrorText = ""
	} else {
		s.lastErrorCode = code
		s.lastErrorText = detail
	}
	s.diagUpdatedAt = time.Now().Format(time.RFC3339)
}

func canonicalInput(text string) string {
	text = strings.TrimSpace(text)
	switch text {
	case ":q", "quit":
		return "/quit"
	case "ls":
		return "/tools"
	case "sessions":
		return "/sessions"
	case "show":
		return "/show"
	case "next":
		return "/next"
	case "prev":
		return "/prev"
	case "stats":
		return "/stats"
	case "gateway", "gw":
		return "/gateway status"
	case "config", "cfg":
		return "/config get"
	case "h":
		return "/help"
	}
	if strings.HasPrefix(text, "show ") && !strings.HasPrefix(text, "/show ") {
		return "/show " + strings.TrimSpace(strings.TrimPrefix(text, "show "))
	}
	if strings.HasPrefix(text, "sessions ") && !strings.HasPrefix(text, "/sessions ") {
		return "/sessions " + strings.TrimSpace(strings.TrimPrefix(text, "sessions "))
	}
	if strings.HasPrefix(text, "tool ") && !strings.HasPrefix(text, "/tool ") {
		return "/tool " + strings.TrimSpace(strings.TrimPrefix(text, "tool "))
	}
	if strings.HasPrefix(text, "gw ") && !strings.HasPrefix(text, "/gateway ") {
		return "/gateway " + strings.TrimSpace(strings.TrimPrefix(text, "gw "))
	}
	if strings.HasPrefix(text, "cfg ") && !strings.HasPrefix(text, "/config ") {
		return "/config " + strings.TrimSpace(strings.TrimPrefix(text, "cfg "))
	}
	return text
}

func parseStartupFlags(args []string) (noDoctor bool, fullscreen bool, fullscreenSet bool) {
	for _, a := range args {
		switch strings.TrimSpace(a) {
		case "--no-doctor":
			noDoctor = true
		case "--fullscreen":
			fullscreen = true
			fullscreenSet = true
		}
	}
	if !fullscreenSet {
		if v := strings.TrimSpace(os.Getenv("AGENT_UI_TUI_FULLSCREEN")); v == "1" || strings.EqualFold(v, "true") {
			fullscreen = true
			fullscreenSet = true
		}
	}
	return noDoctor, fullscreen, fullscreenSet
}

func (s *appState) maybeAutoRefreshPanel() {
	if !s.fullscreen || !s.panelAutoRefresh || s.panelRefreshSec <= 0 {
		return
	}
	if !s.lastPanelRefresh.IsZero() && time.Since(s.lastPanelRefresh) < time.Duration(s.panelRefreshSec)*time.Second {
		return
	}
	if err := s.refreshCurrentPanel(); err == nil {
		s.lastPanelRefresh = time.Now()
	}
}

func (s *appState) refreshCurrentPanel() error {
	switch s.fullscreenPanel {
	case "overview":
		out := s.diagnosticsSnapshot()
		s.panelData["overview"] = out
	case "sessions":
		out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions?limit=%d", s.httpBase, 20), nil)
		if err != nil {
			return err
		}
		s.panelData["sessions"] = uiPayload(out, "sessions", "result")
	case "tools":
		out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools", nil)
		if err != nil {
			return err
		}
		s.panelData["tools"] = uiPayload(out, "tools", "result")
	case "approvals":
		out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=200", s.httpBase, url.PathEscape(s.session)), nil)
		if err != nil {
			return err
		}
		msgs, _ := out["messages"].([]any)
		s.panelData["approvals"] = findPendingApprovals(msgs, 50)
	case "gateway":
		out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/gateway/status", nil)
		if err != nil {
			return err
		}
		s.panelData["gateway"] = uiPayload(out, "status", "result")
	case "diag":
		s.panelData["diag"] = s.diagnosticsSnapshot()
	case "dashboard":
		outSessions, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions?limit=%d", s.httpBase, 10), nil)
		if err != nil {
			return err
		}
		outTools, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools", nil)
		if err != nil {
			return err
		}
		outGateway, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/gateway/status", nil)
		if err != nil {
			return err
		}
		outSession, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=200", s.httpBase, url.PathEscape(s.session)), nil)
		if err != nil {
			return err
		}
		msgs, _ := outSession["messages"].([]any)
		s.panelData["dashboard"] = map[string]any{
			"diag":      s.diagnosticsSnapshot(),
			"sessions":  uiPayload(outSessions, "sessions", "result"),
			"tools":     uiPayload(outTools, "tools", "result"),
			"gateway":   uiPayload(outGateway, "status", "result"),
			"approvals": findPendingApprovals(msgs, 20),
		}
	default:
		return fmt.Errorf("unsupported panel: %s", s.fullscreenPanel)
	}
	s.lastPanelRefresh = time.Now()
	return nil
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	out, _ := v.([]any)
	return out
}

func selectSessionIDFromPanelData(v any, idx int) (string, bool) {
	rows := asSlice(v)
	if idx <= 0 || idx > len(rows) {
		return "", false
	}
	row := asMap(rows[idx-1])
	if row == nil {
		return "", false
	}
	sid, _ := row["session_id"].(string)
	return strings.TrimSpace(sid), strings.TrimSpace(sid) != ""
}

func selectToolNameFromPanelData(v any, idx int) (string, bool) {
	rows := asSlice(v)
	if idx <= 0 || idx > len(rows) {
		return "", false
	}
	name, _ := rows[idx-1].(string)
	return strings.TrimSpace(name), strings.TrimSpace(name) != ""
}

func selectApprovalIDFromPanelData(v any, idx int) (string, bool) {
	rows := asSlice(v)
	if idx <= 0 || idx > len(rows) {
		return "", false
	}
	row := asMap(rows[idx-1])
	if row == nil {
		return "", false
	}
	id, _ := row["approval_id"].(string)
	return strings.TrimSpace(id), strings.TrimSpace(id) != ""
}

func findLatestPendingApproval(msgs []any) (string, map[string]any) {
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := asMap(msgs[i])
		if msg == nil {
			continue
		}
		if role, _ := msg["role"].(string); role != "tool" {
			continue
		}
		content, _ := msg["content"].(string)
		if strings.TrimSpace(content) == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			continue
		}
		if status, _ := payload["status"].(string); status == "pending_approval" {
			if id, _ := payload["approval_id"].(string); strings.TrimSpace(id) != "" {
				return id, payload
			}
		}
	}
	return "", nil
}

func findPendingApprovals(msgs []any, limit int) []map[string]any {
	if limit <= 0 {
		limit = 1
	}
	out := make([]map[string]any, 0, limit)
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := asMap(msgs[i])
		if msg == nil {
			continue
		}
		if role, _ := msg["role"].(string); role != "tool" {
			continue
		}
		content, _ := msg["content"].(string)
		if strings.TrimSpace(content) == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			continue
		}
		if status, _ := payload["status"].(string); status != "pending_approval" {
			continue
		}
		id, _ := payload["approval_id"].(string)
		if strings.TrimSpace(id) == "" {
			continue
		}
		item := map[string]any{
			"approval_id": id,
			"status":      "pending_approval",
			"tool_name":   payload["tool_name"],
			"command":     payload["command"],
			"category":    payload["category"],
			"instruction": payload["instruction"],
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (s *appState) confirmApproval(approvalID string, approve bool) (map[string]any, error) {
	body := map[string]any{
		"session_id":  s.session,
		"approval_id": approvalID,
		"approve":     approve,
	}
	return httpJSON(http.MethodPost, s.httpBase+"/v1/ui/approval/confirm", body)
}

func parseEventSaveArgs(text string) (path, format string, since, until time.Time, err error) {
	rest := strings.TrimSpace(strings.TrimPrefix(text, "/events save "))
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("usage: /events save <file> [json|ndjson] [since=<RFC3339>] [until=<RFC3339>]")
	}
	path = parts[0]
	format = "json"
	for _, p := range parts[1:] {
		switch {
		case p == "json" || p == "ndjson":
			format = p
		case strings.HasPrefix(p, "since="):
			v := strings.TrimSpace(strings.TrimPrefix(p, "since="))
			t, parseErr := time.Parse(time.RFC3339, v)
			if parseErr != nil {
				return "", "", time.Time{}, time.Time{}, fmt.Errorf("invalid since: %w", parseErr)
			}
			since = t
		case strings.HasPrefix(p, "until="):
			v := strings.TrimSpace(strings.TrimPrefix(p, "until="))
			t, parseErr := time.Parse(time.RFC3339, v)
			if parseErr != nil {
				return "", "", time.Time{}, time.Time{}, fmt.Errorf("invalid until: %w", parseErr)
			}
			until = t
		default:
			return "", "", time.Time{}, time.Time{}, fmt.Errorf("unknown option: %s", p)
		}
	}
	return path, format, since, until, nil
}

func filterEventsByTime(events []map[string]any, since, until time.Time) []map[string]any {
	if since.IsZero() && until.IsZero() {
		return events
	}
	out := make([]map[string]any, 0, len(events))
	for _, evt := range events {
		raw, _ := evt["_captured_at"].(string)
		if raw == "" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			continue
		}
		if !since.IsZero() && ts.Before(since) {
			continue
		}
		if !until.IsZero() && ts.After(until) {
			continue
		}
		out = append(out, evt)
	}
	return out
}

func saveEvents(path, format string, events []map[string]any) error {
	if format == "ndjson" {
		var b strings.Builder
		for _, evt := range events {
			line, _ := json.Marshal(evt)
			b.Write(line)
			b.WriteByte('\n')
		}
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	bs, _ := json.MarshalIndent(events, "", "  ")
	return os.WriteFile(path, bs, 0o644)
}

func classifyError(err error) (string, string) {
	if err == nil {
		return "ok", ""
	}
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "context deadline exceeded") || strings.Contains(lower, "timeout") {
		return "timeout", msg
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout", msg
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return "timeout", msg
		}
		return "network", msg
	}
	if strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "connection reset by peer") {
		return "network", msg
	}
	if strings.Contains(lower, "http 401") || strings.Contains(lower, "http 403") {
		return "auth", msg
	}
	if strings.Contains(lower, "http 400") || strings.Contains(lower, "http 404") {
		return "request", msg
	}
	if strings.Contains(lower, "http 5") {
		return "server", msg
	}
	if strings.Contains(lower, "websocket") || strings.Contains(lower, "close 1006") {
		return "network", msg
	}
	return "unknown", msg
}

func (s *appState) setErrStatus(err error) {
	code, detail := classifyError(err)
	s.setStatus(false, code, detail)
}

func (s *appState) diagnosticsSnapshot() map[string]any {
	return map[string]any{
		"schema_version":       "diag.v1",
		"source":               "ui-tui",
		"exported_at":          time.Now().Format(time.RFC3339),
		"session_id":           s.session,
		"turn_id":              s.lastTurnID,
		"stream_mode":          true,
		"configured_transport": "ws",
		"active_transport":     s.activeTransport,
		"reconnect_status":     s.reconnectState,
		"reconnect_count":      s.reconnectCount,
		"timeout_action":       s.timeoutAction,
		"read_timeout_sec":     int(s.wsReadTimeout / time.Second),
		"turn_timeout_sec":     int(s.wsTurnTimeout / time.Second),
		"fallback_hint":        s.fallbackHint,
		"last_error_code":      s.lastErrorCode,
		"error_text":           s.lastErrorText,
		"events":               s.eventLog,
		"reconnect_enabled":    s.reconnectEnabled,
		"max_reconnect":        s.wsMaxReconnect,
		"event_count":          len(s.eventLog),
		"updated_at":           s.diagUpdatedAt,
	}
}

func (s *appState) exportDiagnostics(path string) error {
	payload := s.diagnosticsSnapshot()
	bs, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func httpJSON(method, endpoint string, body map[string]any) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		bs, _ := json.Marshal(body)
		reader = bytes.NewReader(bs)
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bs, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(bs) > 0 {
		if err := json.Unmarshal(bs, &out); err != nil {
			if resp.StatusCode >= 400 {
				return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(bs)))
			}
			return nil, err
		}
	}
	if resp.StatusCode >= 400 {
		if errObj, ok := out["error"].(map[string]any); ok {
			code, _ := errObj["code"].(string)
			msg, _ := errObj["message"].(string)
			if strings.TrimSpace(code) != "" || strings.TrimSpace(msg) != "" {
				return nil, fmt.Errorf("http %d [%s]: %s", resp.StatusCode, code, msg)
			}
		}
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(bs)))
	}
	if okVal, exists := out["ok"]; exists {
		if okBool, ok := okVal.(bool); ok && !okBool {
			if errObj, ok := out["error"].(map[string]any); ok {
				code, _ := errObj["code"].(string)
				msg, _ := errObj["message"].(string)
				return nil, fmt.Errorf("[%s]: %s", code, msg)
			}
			return nil, fmt.Errorf("ui api returned ok=false")
		}
	}
	return out, nil
}

func uiPayload(out map[string]any, keys ...string) any {
	if out == nil {
		return nil
	}
	for _, k := range keys {
		if v, ok := out[k]; ok {
			return v
		}
	}
	if v, ok := out["result"]; ok {
		return v
	}
	return out
}

func httpStatus(method, endpoint string, body map[string]any) (int, string, error) {
	var reader io.Reader
	if body != nil {
		bs, _ := json.Marshal(body)
		reader = bytes.NewReader(bs)
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return 0, "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	bs, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(bs)), nil
}

func (s *appState) runDoctor() ([]doctorItem, bool) {
	now := time.Now().Format(time.RFC3339)
	items := make([]doctorItem, 0, 5)
	allOK := true
	add := func(name, status, detail, suggest string) {
		if status != "ok" {
			allOK = false
		}
		items = append(items, doctorItem{Name: name, Status: status, Detail: detail, Suggest: suggest, Checked: now})
	}

	if out, err := httpJSON(http.MethodGet, s.httpBase+"/health", nil); err != nil {
		add("health", "fail", err.Error(), "check agentd serve and AGENT_HTTP_BASE")
	} else {
		add("health", "ok", fmt.Sprintf("status=%v", out["status"]), "")
	}

	if _, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=1", s.httpBase, url.PathEscape(s.session)), nil); err != nil {
		add("sessions_detail", "fail", err.Error(), "verify /v1/ui/sessions endpoint and database health")
	} else {
		add("sessions_detail", "ok", "session detail endpoint reachable", "")
	}

	code, body, err := httpStatus(http.MethodPost, s.httpBase+"/v1/ui/approval/confirm", map[string]any{
		"session_id":  s.session,
		"approval_id": "doctor-probe",
		"approve":     true,
	})
	if err != nil {
		add("approval_confirm", "fail", err.Error(), "upgrade backend to include /v1/ui/approval/confirm")
	} else {
		switch code {
		case http.StatusNotFound:
			add("approval_confirm", "fail", "endpoint missing: upgrade backend to support /v1/ui/approval/confirm", "deploy latest backend")
		case http.StatusBadRequest, http.StatusOK:
			add("approval_confirm", "ok", fmt.Sprintf("endpoint reachable (http %d)", code), "")
		default:
			add("approval_confirm", "warn", fmt.Sprintf("unexpected status=%d body=%s", code, body), "check backend logs and API compatibility")
		}
	}

	add("config_effective", "ok", fmt.Sprintf("ws=%s http=%s reconnect=%d read_timeout=%s turn_timeout=%s history_max=%d event_max=%d", s.wsBase, s.httpBase, s.wsMaxReconnect, s.wsReadTimeout, s.wsTurnTimeout, s.historyMaxLines, s.eventMaxItems), "")

	u, err := url.Parse(s.wsBase)
	if err != nil {
		add("ws_reachable", "fail", err.Error(), "fix ws_base URL format")
	} else {
		d := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
		conn, _, dialErr := d.Dial(u.String(), nil)
		if dialErr != nil {
			add("ws_reachable", "warn", dialErr.Error(), "ensure /v1/chat/ws is reachable from current host")
		} else {
			_ = conn.Close()
			add("ws_reachable", "ok", "websocket handshake ok", "")
		}
	}
	return items, allOK
}

func (s *appState) appendHistory(cmd string) {
	if strings.TrimSpace(cmd) == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.historyPath), 0o755)
	f, err := os.OpenFile(s.historyPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(fmt.Sprintf("%s\t%s\n", time.Now().Format(time.RFC3339), cmd))
	_ = s.pruneHistoryFile()
}

func (s *appState) pruneHistoryFile() error {
	if s.historyMaxLines <= 0 {
		return nil
	}
	bs, err := os.ReadFile(s.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	raw := strings.Split(strings.TrimSpace(string(bs)), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) <= s.historyMaxLines {
		return nil
	}
	lines = lines[len(lines)-s.historyMaxLines:]
	data := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(s.historyPath, []byte(data), 0o644)
}

func (s *appState) readHistory(limit int) ([]string, error) {
	bs, err := os.ReadFile(s.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(bs)), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			out = append(out, parts[1])
		} else {
			out = append(out, line)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (s *appState) loadBookmarks() ([]bookmark, error) {
	bs, err := os.ReadFile(s.bookmarkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []bookmark{}, nil
		}
		return nil, err
	}
	var out []bookmark
	if err := json.Unmarshal(bs, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *appState) saveBookmarks(list []bookmark) error {
	_ = os.MkdirAll(filepath.Dir(s.bookmarkPath), 0o755)
	bs, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(s.bookmarkPath, bs, 0o644)
}

func (s *appState) addBookmark(name string) error {
	list, err := s.loadBookmarks()
	if err != nil {
		return err
	}
	next := bookmark{Name: name, SessionID: s.session, WSBase: s.wsBase, HTTPBase: s.httpBase}
	replaced := false
	for i := range list {
		if list[i].Name == name {
			list[i] = next
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, next)
	}
	return s.saveBookmarks(list)
}

func (s *appState) useBookmark(name string) error {
	list, err := s.loadBookmarks()
	if err != nil {
		return err
	}
	for _, b := range list {
		if b.Name == name {
			s.session = b.SessionID
			s.wsBase = b.WSBase
			s.httpBase = b.HTTPBase
			s.lastShowSession = b.SessionID
			return nil
		}
	}
	return fmt.Errorf("bookmark not found: %s", name)
}

func (s *appState) loadWorkbenchProfiles() ([]workbenchProfile, error) {
	bs, err := os.ReadFile(s.workbenchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []workbenchProfile{}, nil
		}
		return nil, err
	}
	var out []workbenchProfile
	if err := json.Unmarshal(bs, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *appState) saveWorkbenchProfiles(list []workbenchProfile) error {
	_ = os.MkdirAll(filepath.Dir(s.workbenchPath), 0o755)
	bs, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(s.workbenchPath, bs, 0o644)
}

func (s *appState) saveWorkbench(name string) error {
	list, err := s.loadWorkbenchProfiles()
	if err != nil {
		return err
	}
	next := workbenchProfile{
		Name:             name,
		SessionID:        s.session,
		WSBase:           s.wsBase,
		HTTPBase:         s.httpBase,
		Fullscreen:       s.fullscreen,
		FullscreenPanel:  s.fullscreenPanel,
		PanelAutoRefresh: s.panelAutoRefresh,
		PanelRefreshSec:  s.panelRefreshSec,
		ViewMode:         s.viewMode,
		UpdatedAt:        time.Now().Format(time.RFC3339),
	}
	replaced := false
	for i := range list {
		if strings.EqualFold(strings.TrimSpace(list[i].Name), strings.TrimSpace(name)) {
			list[i] = next
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, next)
	}
	return s.saveWorkbenchProfiles(list)
}

func (s *appState) loadWorkbench(name string) error {
	list, err := s.loadWorkbenchProfiles()
	if err != nil {
		return err
	}
	for _, wb := range list {
		if strings.EqualFold(strings.TrimSpace(wb.Name), strings.TrimSpace(name)) {
			if strings.TrimSpace(wb.SessionID) != "" {
				s.session = wb.SessionID
				s.lastShowSession = wb.SessionID
			}
			if strings.TrimSpace(wb.WSBase) != "" {
				s.wsBase = wb.WSBase
			}
			if strings.TrimSpace(wb.HTTPBase) != "" {
				s.httpBase = wb.HTTPBase
			}
			s.fullscreen = wb.Fullscreen
			if strings.TrimSpace(wb.FullscreenPanel) != "" {
				s.fullscreenPanel = wb.FullscreenPanel
			}
			s.panelAutoRefresh = wb.PanelAutoRefresh
			if wb.PanelRefreshSec > 0 {
				s.panelRefreshSec = wb.PanelRefreshSec
			}
			if wb.ViewMode == "json" || wb.ViewMode == "human" {
				s.viewMode = wb.ViewMode
			}
			return nil
		}
	}
	return fmt.Errorf("workbench not found: %s", name)
}

func (s *appState) deleteWorkbench(name string) error {
	list, err := s.loadWorkbenchProfiles()
	if err != nil {
		return err
	}
	next := make([]workbenchProfile, 0, len(list))
	removed := false
	for _, wb := range list {
		if strings.EqualFold(strings.TrimSpace(wb.Name), strings.TrimSpace(name)) {
			removed = true
			continue
		}
		next = append(next, wb)
	}
	if !removed {
		return fmt.Errorf("workbench not found: %s", name)
	}
	return s.saveWorkbenchProfiles(next)
}

func (s *appState) loadWorkflows() ([]workflowProfile, error) {
	bs, err := os.ReadFile(s.workflowPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []workflowProfile{}, nil
		}
		return nil, err
	}
	var out []workflowProfile
	if err := json.Unmarshal(bs, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *appState) saveWorkflows(list []workflowProfile) error {
	_ = os.MkdirAll(filepath.Dir(s.workflowPath), 0o755)
	bs, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(s.workflowPath, bs, 0o644)
}

func parseWorkflowCommands(raw string) []string {
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		if !strings.HasPrefix(v, "/") {
			v = "/" + v
		}
		out = append(out, v)
	}
	return out
}

func (s *appState) saveWorkflow(name string, commands []string) error {
	list, err := s.loadWorkflows()
	if err != nil {
		return err
	}
	next := workflowProfile{Name: name, Commands: append([]string(nil), commands...), UpdatedAt: time.Now().Format(time.RFC3339)}
	replaced := false
	for i := range list {
		if strings.EqualFold(strings.TrimSpace(list[i].Name), strings.TrimSpace(name)) {
			list[i] = next
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, next)
	}
	return s.saveWorkflows(list)
}

func (s *appState) getWorkflow(name string) (workflowProfile, bool, error) {
	list, err := s.loadWorkflows()
	if err != nil {
		return workflowProfile{}, false, err
	}
	for _, wf := range list {
		if strings.EqualFold(strings.TrimSpace(wf.Name), strings.TrimSpace(name)) {
			return wf, true, nil
		}
	}
	return workflowProfile{}, false, nil
}

func (s *appState) deleteWorkflow(name string) error {
	list, err := s.loadWorkflows()
	if err != nil {
		return err
	}
	next := make([]workflowProfile, 0, len(list))
	removed := false
	for _, wf := range list {
		if strings.EqualFold(strings.TrimSpace(wf.Name), strings.TrimSpace(name)) {
			removed = true
			continue
		}
		next = append(next, wf)
	}
	if !removed {
		return fmt.Errorf("workflow not found: %s", name)
	}
	return s.saveWorkflows(next)
}

func (s *appState) addEvent(evt map[string]any) {
	if evt != nil {
		if _, ok := evt["_captured_at"]; !ok {
			evt["_captured_at"] = time.Now().Format(time.RFC3339)
		}
	}
	if s.eventMaxItems > 0 && len(s.eventLog) >= s.eventMaxItems {
		trim := s.eventMaxItems / 2
		if trim < 1 {
			trim = 1
		}
		s.eventLog = s.eventLog[len(s.eventLog)-trim:]
	}
	s.eventLog = append(s.eventLog, evt)
}

func (s *appState) sendTurn(message string, onEvent func(map[string]any)) error {
	u, err := url.Parse(s.wsBase)
	if err != nil {
		return err
	}
	turnID := uuid.NewString()
	startedAt := time.Now()
	s.reconnectState = "connecting"
	s.activeTransport = "ws"
	s.lastTurnID = turnID
	s.reconnectCount = 0
	s.fallbackHint = ""
	s.diagUpdatedAt = time.Now().Format(time.RFC3339)
	seenPayload := map[string]struct{}{}
	dedupeEnabled := false
	maxReconnect := s.wsMaxReconnect
	if !s.reconnectEnabled {
		maxReconnect = 0
	}
	reconnectReason := ""
	for attempt := 0; attempt <= maxReconnect; attempt++ {
		if attempt > 0 {
			dedupeEnabled = true
			s.reconnectState = "degraded"
			s.reconnectCount++
			if strings.TrimSpace(reconnectReason) != "" {
				s.fallbackHint = fmt.Sprintf("ws reconnect attempt=%d reason=%s", attempt, reconnectReason)
			}
			s.diagUpdatedAt = time.Now().Format(time.RFC3339)
		}
		conn, _, dialErr := websocket.DefaultDialer.Dial(u.String(), nil)
		if dialErr != nil {
			reconnectReason = dialErr.Error()
			if attempt >= maxReconnect {
				s.reconnectState = "failed"
				return dialErr
			}
			fmt.Printf("[ws-reconnect] dial failed, retry=%d err=%v\n", attempt+1, dialErr)
			time.Sleep(800 * time.Millisecond)
			continue
		}
		req := map[string]any{
			"session_id": s.session,
			"message":    message,
			"turn_id":    turnID,
			"resume":     attempt > 0,
		}
		if err := conn.WriteJSON(req); err != nil {
			reconnectReason = err.Error()
			_ = conn.Close()
			if attempt >= maxReconnect {
				s.reconnectState = "failed"
				return err
			}
			fmt.Printf("[ws-reconnect] write failed, retry=%d err=%v\n", attempt+1, err)
			time.Sleep(800 * time.Millisecond)
			continue
		}
		if attempt > 0 {
			fmt.Printf("[ws-reconnect] resumed session=%s turn=%s attempt=%d\n", s.session, turnID, attempt+1)
		}
		forceReconnect := false
		for {
			if s.wsReadTimeout > 0 {
				_ = conn.SetReadDeadline(time.Now().Add(s.wsReadTimeout))
			}
			_, payload, err := conn.ReadMessage()
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					fmt.Printf("[ws-timeout] %s 内未收到事件，等待服务端响应中\n", s.wsReadTimeout.String())
					if s.wsTurnTimeout > 0 && time.Since(startedAt) > s.wsTurnTimeout {
						switch s.timeoutAction {
						case "wait":
							startedAt = time.Now()
							fmt.Printf("[ws-timeout] timeout action=wait, continue waiting\n")
						case "reconnect":
							_ = conn.Close()
							fmt.Printf("[ws-timeout] timeout action=reconnect, forcing reconnect\n")
							s.reconnectState = "degraded"
							reconnectReason = "read timeout"
							time.Sleep(300 * time.Millisecond)
							forceReconnect = true
						case "cancel":
							_ = conn.Close()
							_, _ = httpJSON(http.MethodPost, s.httpBase+"/v1/chat/cancel", map[string]any{"session_id": s.session})
							s.reconnectState = "failed"
							return fmt.Errorf("turn timeout exceeded and cancelled: %s", s.wsTurnTimeout.String())
						default:
							_ = conn.Close()
							s.reconnectState = "failed"
							return fmt.Errorf("turn timeout exceeded: %s", s.wsTurnTimeout.String())
						}
					}
					if forceReconnect {
						break
					}
					continue
				}
				_ = conn.Close()
				reconnectReason = err.Error()
				if s.wsTurnTimeout > 0 && time.Since(startedAt) > s.wsTurnTimeout {
					s.reconnectState = "failed"
					return fmt.Errorf("turn timeout exceeded: %s", s.wsTurnTimeout.String())
				}
				if attempt >= maxReconnect {
					s.reconnectState = "failed"
					return err
				}
				fmt.Printf("[ws-reconnect] stream dropped, retry=%d err=%v\n", attempt+1, err)
				time.Sleep(800 * time.Millisecond)
				break
			}
			var evt map[string]any
			if err := json.Unmarshal(payload, &evt); err != nil {
				fmt.Printf("[decode-error] %v\n", err)
				continue
			}
			evtType, _ := evt["type"].(string)
			if evtType == "" {
				evtType, _ = evt["Type"].(string)
			}
			if evtType == "resumed" {
				dedupeEnabled = true
				s.reconnectState = "resumed"
				s.diagUpdatedAt = time.Now().Format(time.RFC3339)
			}
			key := string(payload)
			if _, ok := seenPayload[key]; ok {
				if dedupeEnabled {
					continue
				}
			}
			seenPayload[key] = struct{}{}
			if line := printEvent(evt, false); strings.TrimSpace(line) != "" {
				s.addChatLine(line)
			}
			if onEvent != nil {
				onEvent(evt)
			}
			if evtType == "result" || evtType == "error" || evtType == "cancelled" {
				_ = conn.Close()
				if evtType == "result" {
					if s.reconnectState != "resumed" && s.reconnectState != "degraded" {
						s.reconnectState = "connecting"
					}
				}
				if evtType == "error" || evtType == "cancelled" {
					s.reconnectState = "failed"
					if code, ok := evt["error_code"].(string); ok && strings.TrimSpace(code) != "" {
						s.lastErrorCode = code
					}
					if em, ok := evt["error"].(string); ok && strings.TrimSpace(em) != "" {
						s.lastErrorText = em
					}
				}
				s.diagUpdatedAt = time.Now().Format(time.RFC3339)
				return nil
			}
		}
		if forceReconnect {
			continue
		}
	}
	s.reconnectState = "failed"
	return fmt.Errorf("unable to complete turn after reconnect attempts")
}

func main() {
	if strings.TrimSpace(os.Getenv("COLORFGBG")) == "" {
		_ = os.Setenv("COLORFGBG", "15;0")
	}
	s := newState()
	noDoctor, fullscreen, fullscreenSet := parseStartupFlags(os.Args[1:])
	if fullscreenSet {
		s.fullscreen = fullscreen
	}
	if err := runBubbleTeaUI(s, noDoctor); err != nil {
		fmt.Printf("[tui-error] %v\n", err)
		os.Exit(1)
	}
}
