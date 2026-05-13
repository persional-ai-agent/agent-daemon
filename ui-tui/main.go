package main

import (
	"bufio"
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
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type bookmark struct {
	Name      string `json:"name"`
	SessionID string `json:"session_id"`
	WSBase    string `json:"ws_base"`
	HTTPBase  string `json:"http_base"`
}

type runtimeState struct {
	SessionID string `json:"session_id"`
	WSBase    string `json:"ws_base"`
	HTTPBase  string `json:"http_base"`
	UpdatedAt string `json:"updated_at"`
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

	eventLog []map[string]any

	historyPath  string
	bookmarkPath string
	statePath    string

	historyMaxLines int
	eventMaxItems   int
	wsReadTimeout   time.Duration
	wsTurnTimeout   time.Duration
	wsMaxReconnect  int
}

const (
	defaultHistoryMaxLines = 2000
	defaultEventMaxItems   = 2000
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
	wsBase := getenvOr("AGENT_API_BASE", "ws://127.0.0.1:8080/v1/chat/ws")
	httpBase := getenvOr("AGENT_HTTP_BASE", deriveHTTPBase(wsBase))
	session := getenvOr("AGENT_SESSION_ID", uuid.NewString())
	home, _ := os.UserHomeDir()
	if strings.TrimSpace(home) == "" {
		home = "."
	}
	root := filepath.Join(home, ".agent-daemon")
	st := &appState{
		wsBase:          wsBase,
		httpBase:        httpBase,
		session:         session,
		pretty:          true,
		lastStatus:      "ok",
		lastDetail:      "initialized",
		lastCode:        "ok",
		lastShowSession: session,
		lastShowLimit:   20,
		lastSessions:    make([]string, 0),
		eventLog:        make([]map[string]any, 0),
		historyPath:     filepath.Join(root, "ui-tui-history.log"),
		bookmarkPath:    filepath.Join(root, "ui-tui-bookmarks.json"),
		statePath:       filepath.Join(root, "ui-tui-state.json"),
		historyMaxLines: defaultHistoryMaxLines,
		eventMaxItems:   defaultEventMaxItems,
		wsReadTimeout:   45 * time.Second,
		wsTurnTimeout:   8 * time.Minute,
		wsMaxReconnect:  2,
	}
	st.loadRuntimeState()
	return st
}

func (s *appState) saveRuntimeState() error {
	_ = os.MkdirAll(filepath.Dir(s.statePath), 0o755)
	st := runtimeState{
		SessionID: s.session,
		WSBase:    s.wsBase,
		HTTPBase:  s.httpBase,
		UpdatedAt: time.Now().Format(time.RFC3339),
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

func printHelp() {
	fmt.Println("commands:")
	fmt.Println("/help                 show help")
	fmt.Println("/session              show current session id")
	fmt.Println("/session <id>         switch session id")
	fmt.Println("/api                  show websocket endpoint")
	fmt.Println("/api <ws-url>         switch websocket endpoint")
	fmt.Println("/http                 show http api base")
	fmt.Println("/http <http-url>      switch http api base")
	fmt.Println("/tools                list tools")
	fmt.Println("/tool <name>          show tool schema")
	fmt.Println("/sessions [n]         list recent sessions")
	fmt.Println("/pick <index>         switch session from last /sessions result")
	fmt.Println("/show [sid] [o] [l]   show session messages")
	fmt.Println("/next                 show next page (based on last /show)")
	fmt.Println("/prev                 show previous page (based on last /show)")
	fmt.Println("/stats [sid]          show session stats")
	fmt.Println("/gateway status       show gateway status")
	fmt.Println("/gateway enable       enable gateway")
	fmt.Println("/gateway disable      disable gateway")
	fmt.Println("/config get           show config snapshot")
	fmt.Println("/config set k v       set config key/value")
	fmt.Println("/pretty on|off        enable/disable pretty json")
	fmt.Println("/last                 print last json payload")
	fmt.Println("/save <file>          save last json payload")
	fmt.Println("/status               show last command status")
	fmt.Println("/health               check backend health endpoint")
	fmt.Println("/cancel               cancel current session run")
	fmt.Println("/history [n]          show local command history")
	fmt.Println("/rerun <index>        rerun command from history")
	fmt.Println("/events [n]           show recent runtime events")
	fmt.Println("/events save <file>   save runtime events as json")
	fmt.Println("/bookmark add <name>  save current session/api profile")
	fmt.Println("/bookmark list        list bookmarks")
	fmt.Println("/bookmark use <name>  restore session/api profile")
	fmt.Println("/pending              show latest pending approval in session")
	fmt.Println("/approve <id>         approve pending approval id")
	fmt.Println("/deny <id>            deny pending approval id")
	fmt.Println("/quit                 exit")
	fmt.Println("aliases: :q, quit, ls, show, gw, cfg, h")
}

func printEvent(evt map[string]any) {
	evtType := ""
	if v, ok := evt["type"].(string); ok {
		evtType = v
	}
	if evtType == "" {
		if v, ok := evt["Type"].(string); ok {
			evtType = v
		}
	}
	switch evtType {
	case "assistant_message":
		fmt.Printf("[assistant] %v\n", evt["content"])
	case "tool_started", "tool_finished":
		toolName := evt["tool_name"]
		if toolName == nil {
			toolName = evt["ToolName"]
		}
		fmt.Printf("[%s] %v\n", evtType, toolName)
	case "result":
		fmt.Printf("[result] %v\n", evt["final_response"])
	case "error":
		fmt.Printf("[error] %v\n", evt["error"])
	default:
		bs, _ := json.Marshal(evt)
		fmt.Printf("[%s] %s\n", evtType, string(bs))
	}
}

func printJSONMode(v any, pretty bool) {
	var bs []byte
	if pretty {
		bs, _ = json.MarshalIndent(v, "", "  ")
	} else {
		bs, _ = json.Marshal(v)
	}
	fmt.Println(string(bs))
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
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
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(bs)))
	}
	out := map[string]any{}
	if len(bs) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(bs, &out); err != nil {
		return nil, err
	}
	return out, nil
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
	for attempt := 0; attempt <= s.wsMaxReconnect; attempt++ {
		conn, _, dialErr := websocket.DefaultDialer.Dial(u.String(), nil)
		if dialErr != nil {
			if attempt >= s.wsMaxReconnect {
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
			_ = conn.Close()
			if attempt >= s.wsMaxReconnect {
				return err
			}
			fmt.Printf("[ws-reconnect] write failed, retry=%d err=%v\n", attempt+1, err)
			time.Sleep(800 * time.Millisecond)
			continue
		}
		if attempt > 0 {
			fmt.Printf("[ws-reconnect] resumed session=%s turn=%s attempt=%d\n", s.session, turnID, attempt+1)
		}
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
						_ = conn.Close()
						return fmt.Errorf("turn timeout exceeded: %s", s.wsTurnTimeout.String())
					}
					continue
				}
				_ = conn.Close()
				if s.wsTurnTimeout > 0 && time.Since(startedAt) > s.wsTurnTimeout {
					return fmt.Errorf("turn timeout exceeded: %s", s.wsTurnTimeout.String())
				}
				if attempt >= s.wsMaxReconnect {
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
			printEvent(evt)
			if onEvent != nil {
				onEvent(evt)
			}
			evtType, _ := evt["type"].(string)
			if evtType == "" {
				evtType, _ = evt["Type"].(string)
			}
			if evtType == "result" || evtType == "error" || evtType == "cancelled" {
				_ = conn.Close()
				return nil
			}
		}
	}
	return fmt.Errorf("unable to complete turn after reconnect attempts")
}

func main() {
	s := newState()
	fmt.Printf("session: %s\n", s.session)
	fmt.Printf("ws: %s\n", s.wsBase)
	fmt.Printf("http: %s\n", s.httpBase)
	fmt.Println("输入 /help 查看命令")
	reader := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("tui[%s/%s]> ", s.lastStatus, s.lastCode)
		if !reader.Scan() {
			fmt.Println("bye")
			return
		}
		text := strings.TrimSpace(reader.Text())
		if text == "" {
			continue
		}
		fromRerun := false
	REPROCESS:
		text = canonicalInput(text)
		if !fromRerun {
			s.appendHistory(text)
		}

		switch {
		case text == "/quit" || text == "/exit":
			fmt.Println("bye")
			return
		case text == "/help":
			printHelp()
			s.setStatus(true, "ok", "help shown")
		case text == "/status":
			fmt.Printf("status=%s code=%s detail=%s\n", s.lastStatus, s.lastCode, s.lastDetail)
		case text == "/health":
			out, err := httpJSON(http.MethodGet, s.httpBase+"/health", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "health checked")
		case text == "/cancel":
			out, err := httpJSON(http.MethodPost, s.httpBase+"/v1/chat/cancel", map[string]any{"session_id": s.session})
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "cancel requested")
		case strings.HasPrefix(text, "/history"):
			parts := strings.Fields(text)
			limit := 20
			if len(parts) > 1 {
				if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
					limit = v
				}
			}
			if limit > s.historyMaxLines {
				limit = s.historyMaxLines
			}
			items, err := s.readHistory(limit)
			if err != nil {
				fmt.Printf("[history-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			for i, item := range items {
				fmt.Printf("%d. %s\n", i+1, item)
			}
			s.setStatus(true, "ok", "history loaded")
		case strings.HasPrefix(text, "/rerun "):
			idx, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(text, "/rerun ")))
			if err != nil || idx <= 0 {
				fmt.Println("usage: /rerun <index>")
				s.setStatus(false, "invalid_input", "invalid rerun index")
				continue
			}
			items, err := s.readHistory(500)
			if err != nil {
				fmt.Printf("[history-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			if idx > len(items) {
				fmt.Printf("index out of range, max=%d\n", len(items))
				s.setStatus(false, "invalid_input", "rerun index out of range")
				continue
			}
			text = items[idx-1]
			fromRerun = true
			fmt.Printf("rerun: %s\n", text)
			s.setStatus(true, "ok", "rerun selected")
			goto REPROCESS
		case strings.HasPrefix(text, "/events"):
			if strings.HasPrefix(text, "/events save ") {
				path, format, since, until, err := parseEventSaveArgs(text)
				if err != nil {
					fmt.Println(err.Error())
					s.setStatus(false, "invalid_input", "invalid events save args")
					continue
				}
				filtered := filterEventsByTime(s.eventLog, since, until)
				if err := saveEvents(path, format, filtered); err != nil {
					fmt.Printf("[save-error] %v\n", err)
					s.setErrStatus(err)
					continue
				}
				fmt.Printf("saved events: %s (format=%s count=%d)\n", path, format, len(filtered))
				s.setStatus(true, "ok", "events saved")
				continue
			}
			parts := strings.Fields(text)
			limit := 20
			if len(parts) > 1 {
				if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
					limit = v
				}
			}
			if limit > s.eventMaxItems {
				limit = s.eventMaxItems
			}
			start := len(s.eventLog) - limit
			if start < 0 {
				start = 0
			}
			printJSONMode(s.eventLog[start:], s.pretty)
			s.setStatus(true, "ok", "events listed")
		case text == "/pending":
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=200", s.httpBase, url.PathEscape(s.session)), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			msgs, _ := out["messages"].([]any)
			id, payload := findLatestPendingApproval(msgs)
			if id == "" {
				fmt.Println("no pending approval found in recent session messages")
				s.setStatus(false, "not_found", "no pending approval")
				continue
			}
			fmt.Printf("pending approval id: %s\n", id)
			printJSONMode(payload, s.pretty)
			s.setStatus(true, "ok", "pending approval found")
		case strings.HasPrefix(text, "/approve "):
			id := strings.TrimSpace(strings.TrimPrefix(text, "/approve "))
			if id == "" {
				fmt.Println("usage: /approve <approval_id>")
				s.setStatus(false, "invalid_input", "approval id required")
				continue
			}
			out, err := s.confirmApproval(id, true)
			if err != nil {
				fmt.Printf("[approval-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "approval confirmed")
		case strings.HasPrefix(text, "/deny "):
			id := strings.TrimSpace(strings.TrimPrefix(text, "/deny "))
			if id == "" {
				fmt.Println("usage: /deny <approval_id>")
				s.setStatus(false, "invalid_input", "approval id required")
				continue
			}
			out, err := s.confirmApproval(id, false)
			if err != nil {
				fmt.Printf("[approval-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "approval denied")
		case strings.HasPrefix(text, "/bookmark "):
			parts := strings.Fields(text)
			if len(parts) >= 2 && parts[1] == "list" {
				list, err := s.loadBookmarks()
				if err != nil {
					fmt.Printf("[bookmark-error] %v\n", err)
					s.setErrStatus(err)
					continue
				}
				printJSONMode(list, s.pretty)
				s.setStatus(true, "ok", "bookmarks listed")
				continue
			}
			if len(parts) >= 3 && parts[1] == "add" {
				if err := s.addBookmark(parts[2]); err != nil {
					fmt.Printf("[bookmark-error] %v\n", err)
					s.setErrStatus(err)
					continue
				}
				fmt.Printf("bookmark saved: %s\n", parts[2])
				s.setStatus(true, "ok", "bookmark saved")
				continue
			}
			if len(parts) >= 3 && parts[1] == "use" {
				if err := s.useBookmark(parts[2]); err != nil {
					fmt.Printf("[bookmark-error] %v\n", err)
					s.setErrStatus(err)
					continue
				}
				fmt.Printf("bookmark loaded: %s (session=%s)\n", parts[2], s.session)
				_ = s.saveRuntimeState()
				s.setStatus(true, "ok", "bookmark loaded")
				continue
			}
			fmt.Println("usage: /bookmark add <name> | /bookmark list | /bookmark use <name>")
			s.setStatus(false, "invalid_input", "invalid bookmark args")
		case text == "/session":
			fmt.Printf("session: %s\n", s.session)
			s.setStatus(true, "ok", "session shown")
		case strings.HasPrefix(text, "/session "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/session "))
			if next == "" {
				fmt.Println("session id required")
				s.setStatus(false, "invalid_input", "session id required")
				continue
			}
			s.session = next
			_ = s.saveRuntimeState()
			fmt.Printf("session switched: %s\n", s.session)
			s.setStatus(true, "ok", "session switched")
		case text == "/api":
			fmt.Printf("ws: %s\n", s.wsBase)
			s.setStatus(true, "ok", "ws shown")
		case strings.HasPrefix(text, "/api "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/api "))
			if !strings.HasPrefix(next, "ws://") && !strings.HasPrefix(next, "wss://") {
				fmt.Println("api must start with ws:// or wss://")
				s.setStatus(false, "invalid_input", "invalid ws url")
				continue
			}
			s.wsBase = next
			fmt.Printf("ws switched: %s\n", s.wsBase)
			if strings.TrimSpace(os.Getenv("AGENT_HTTP_BASE")) == "" {
				s.httpBase = deriveHTTPBase(s.wsBase)
				fmt.Printf("http auto-updated: %s\n", s.httpBase)
			}
			_ = s.saveRuntimeState()
			s.setStatus(true, "ok", "ws switched")
		case text == "/http":
			fmt.Printf("http: %s\n", s.httpBase)
			s.setStatus(true, "ok", "http shown")
		case strings.HasPrefix(text, "/http "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/http "))
			if !strings.HasPrefix(next, "http://") && !strings.HasPrefix(next, "https://") {
				fmt.Println("http api must start with http:// or https://")
				s.setStatus(false, "invalid_input", "invalid http url")
				continue
			}
			s.httpBase = strings.TrimRight(next, "/")
			_ = s.saveRuntimeState()
			fmt.Printf("http switched: %s\n", s.httpBase)
			s.setStatus(true, "ok", "http switched")
		case text == "/last":
			if s.lastJSON == nil {
				fmt.Println("no last json payload")
				s.setStatus(false, "invalid_input", "no last json")
				continue
			}
			printJSONMode(s.lastJSON, s.pretty)
			s.setStatus(true, "ok", "last json shown")
		case strings.HasPrefix(text, "/save "):
			path := strings.TrimSpace(strings.TrimPrefix(text, "/save "))
			if path == "" {
				fmt.Println("usage: /save <file>")
				s.setStatus(false, "invalid_input", "invalid save args")
				continue
			}
			if s.lastJSON == nil {
				fmt.Println("no last json payload")
				s.setStatus(false, "invalid_input", "no last json")
				continue
			}
			var bs []byte
			if s.pretty {
				bs, _ = json.MarshalIndent(s.lastJSON, "", "  ")
			} else {
				bs, _ = json.Marshal(s.lastJSON)
			}
			if err := os.WriteFile(path, bs, 0o644); err != nil {
				fmt.Printf("[save-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			fmt.Printf("saved: %s\n", path)
			s.setStatus(true, "ok", "json saved")
		case strings.HasPrefix(text, "/pretty "):
			mode := strings.TrimSpace(strings.TrimPrefix(text, "/pretty "))
			switch mode {
			case "on":
				s.pretty = true
				fmt.Println("pretty json: on")
				s.setStatus(true, "ok", "pretty on")
			case "off":
				s.pretty = false
				fmt.Println("pretty json: off")
				s.setStatus(true, "ok", "pretty off")
			default:
				fmt.Println("usage: /pretty on|off")
				s.setStatus(false, "invalid_input", "invalid pretty args")
			}
		case text == "/tools":
			out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "tools listed")
		case strings.HasPrefix(text, "/tool "):
			name := strings.TrimSpace(strings.TrimPrefix(text, "/tool "))
			if name == "" {
				fmt.Println("usage: /tool <name>")
				s.setStatus(false, "invalid_input", "invalid tool args")
				continue
			}
			out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools/"+url.PathEscape(name)+"/schema", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "tool schema loaded")
		case strings.HasPrefix(text, "/sessions"):
			parts := strings.Fields(text)
			limit := 20
			if len(parts) > 1 {
				if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
					limit = v
				}
			}
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions?limit=%d", s.httpBase, limit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastSessions = s.lastSessions[:0]
			if rows, ok := out["sessions"].([]any); ok {
				for _, row := range rows {
					m, ok := row.(map[string]any)
					if !ok {
						continue
					}
					if sid, ok := m["session_id"].(string); ok && strings.TrimSpace(sid) != "" {
						s.lastSessions = append(s.lastSessions, sid)
					}
				}
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "sessions listed")
		case strings.HasPrefix(text, "/pick "):
			idx, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(text, "/pick ")))
			if err != nil || idx <= 0 {
				fmt.Println("usage: /pick <index>")
				s.setStatus(false, "invalid_input", "invalid pick index")
				continue
			}
			if idx > len(s.lastSessions) {
				fmt.Printf("index out of range, max=%d\n", len(s.lastSessions))
				s.setStatus(false, "invalid_input", "pick index out of range")
				continue
			}
			s.session = s.lastSessions[idx-1]
			s.lastShowSession = s.session
			s.lastShowOffset = 0
			_ = s.saveRuntimeState()
			fmt.Printf("session switched: %s\n", s.session)
			s.setStatus(true, "ok", "session switched")
		case strings.HasPrefix(text, "/show"):
			parts := strings.Fields(text)
			sid := s.session
			offset := 0
			limit := 20
			if len(parts) > 1 {
				sid = parts[1]
			}
			if len(parts) > 2 {
				if v, err := strconv.Atoi(parts[2]); err == nil && v >= 0 {
					offset = v
				}
			}
			if len(parts) > 3 {
				if v, err := strconv.Atoi(parts[3]); err == nil && v > 0 {
					limit = v
				}
			}
			s.lastShowSession = sid
			s.lastShowOffset = offset
			s.lastShowLimit = limit
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", s.httpBase, url.PathEscape(sid), offset, limit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "show loaded")
		case text == "/next":
			s.lastShowOffset += s.lastShowLimit
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", s.httpBase, url.PathEscape(s.lastShowSession), s.lastShowOffset, s.lastShowLimit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.lastShowOffset -= s.lastShowLimit
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "next page loaded")
		case text == "/prev":
			s.lastShowOffset -= s.lastShowLimit
			if s.lastShowOffset < 0 {
				s.lastShowOffset = 0
			}
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", s.httpBase, url.PathEscape(s.lastShowSession), s.lastShowOffset, s.lastShowLimit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "prev page loaded")
		case strings.HasPrefix(text, "/stats"):
			parts := strings.Fields(text)
			sid := s.session
			if len(parts) > 1 {
				sid = parts[1]
			}
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=1", s.httpBase, url.PathEscape(sid)), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out["stats"]
			printJSONMode(out["stats"], s.pretty)
			s.setStatus(true, "ok", "stats loaded")
		case strings.HasPrefix(text, "/gateway "):
			parts := strings.Fields(text)
			if len(parts) != 2 {
				fmt.Println("usage: /gateway status|enable|disable")
				s.setStatus(false, "invalid_input", "invalid gateway args")
				continue
			}
			switch parts[1] {
			case "status":
				out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/gateway/status", nil)
				if err != nil {
					fmt.Printf("[http-error] %v\n", err)
					s.setErrStatus(err)
					continue
				}
				s.lastJSON = out
				printJSONMode(out, s.pretty)
				s.setStatus(true, "ok", "gateway status loaded")
			case "enable", "disable":
				out, err := httpJSON(http.MethodPost, s.httpBase+"/v1/ui/gateway/action", map[string]any{"action": parts[1]})
				if err != nil {
					fmt.Printf("[http-error] %v\n", err)
					s.setErrStatus(err)
					continue
				}
				s.lastJSON = out
				printJSONMode(out, s.pretty)
				s.setStatus(true, "ok", "gateway action applied")
			default:
				fmt.Println("usage: /gateway status|enable|disable")
				s.setStatus(false, "invalid_input", "invalid gateway action")
			}
		case text == "/config get":
			out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/config", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "config loaded")
		case strings.HasPrefix(text, "/config set "):
			parts := strings.SplitN(strings.TrimPrefix(text, "/config set "), " ", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
				fmt.Println("usage: /config set <section.key> <value>")
				s.setStatus(false, "invalid_input", "invalid config args")
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := parts[1]
			out, err := httpJSON(http.MethodPost, s.httpBase+"/v1/ui/config/set", map[string]any{"key": key, "value": value})
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setErrStatus(err)
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "ok", "config updated")
		default:
			if err := s.sendTurn(text, s.addEvent); err != nil {
				fmt.Printf("[ws-error] %v\n", err)
				s.setErrStatus(err)
			} else {
				s.setStatus(true, "ok", "chat turn finished")
			}
		}
	}
}
