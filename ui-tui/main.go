package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

type appState struct {
	wsBase   string
	httpBase string
	session  string

	pretty bool

	lastJSON   any
	lastStatus string
	lastDetail string

	lastShowSession string
	lastShowOffset  int
	lastShowLimit   int
	lastSessions    []string

	eventLog []map[string]any

	historyPath  string
	bookmarkPath string
}

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
	return &appState{
		wsBase:          wsBase,
		httpBase:        httpBase,
		session:         session,
		pretty:          true,
		lastStatus:      "ok",
		lastDetail:      "initialized",
		lastShowSession: session,
		lastShowLimit:   20,
		lastSessions:    make([]string, 0),
		eventLog:        make([]map[string]any, 0),
		historyPath:     filepath.Join(root, "ui-tui-history.log"),
		bookmarkPath:    filepath.Join(root, "ui-tui-bookmarks.json"),
	}
}

func (s *appState) setStatus(ok bool, detail string) {
	if ok {
		s.lastStatus = "ok"
	} else {
		s.lastStatus = "err"
	}
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
	if len(s.eventLog) > 500 {
		s.eventLog = s.eventLog[len(s.eventLog)-300:]
	}
	s.eventLog = append(s.eventLog, evt)
}

func sendTurn(wsBase, sessionID, message string, onEvent func(map[string]any)) error {
	u, err := url.Parse(wsBase)
	if err != nil {
		return err
	}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	req := map[string]any{"session_id": sessionID, "message": message}
	if err := conn.WriteJSON(req); err != nil {
		return err
	}
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return nil
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
			return nil
		}
	}
}

func main() {
	s := newState()
	fmt.Printf("session: %s\n", s.session)
	fmt.Printf("ws: %s\n", s.wsBase)
	fmt.Printf("http: %s\n", s.httpBase)
	fmt.Println("输入 /help 查看命令")
	reader := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("tui[%s]> ", s.lastStatus)
		if !reader.Scan() {
			fmt.Println("bye")
			return
		}
		text := strings.TrimSpace(reader.Text())
		if text == "" {
			continue
		}
		text = canonicalInput(text)
		s.appendHistory(text)

		switch {
		case text == "/quit" || text == "/exit":
			fmt.Println("bye")
			return
		case text == "/help":
			printHelp()
			s.setStatus(true, "help shown")
		case text == "/status":
			fmt.Printf("status=%s detail=%s\n", s.lastStatus, s.lastDetail)
		case text == "/health":
			out, err := httpJSON(http.MethodGet, s.httpBase+"/health", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "health checked")
		case text == "/cancel":
			out, err := httpJSON(http.MethodPost, s.httpBase+"/v1/chat/cancel", map[string]any{"session_id": s.session})
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "cancel requested")
		case strings.HasPrefix(text, "/history"):
			parts := strings.Fields(text)
			limit := 20
			if len(parts) > 1 {
				if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
					limit = v
				}
			}
			items, err := s.readHistory(limit)
			if err != nil {
				fmt.Printf("[history-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			for i, item := range items {
				fmt.Printf("%d. %s\n", i+1, item)
			}
			s.setStatus(true, "history loaded")
		case strings.HasPrefix(text, "/rerun "):
			idx, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(text, "/rerun ")))
			if err != nil || idx <= 0 {
				fmt.Println("usage: /rerun <index>")
				s.setStatus(false, "invalid rerun index")
				continue
			}
			items, err := s.readHistory(500)
			if err != nil {
				fmt.Printf("[history-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			if idx > len(items) {
				fmt.Printf("index out of range, max=%d\n", len(items))
				s.setStatus(false, "rerun index out of range")
				continue
			}
			fmt.Printf("rerun: %s\n", items[idx-1])
			s.setStatus(true, "rerun selected")
		case strings.HasPrefix(text, "/events"):
			if strings.HasPrefix(text, "/events save ") {
				path := strings.TrimSpace(strings.TrimPrefix(text, "/events save "))
				if path == "" {
					fmt.Println("usage: /events save <file>")
					s.setStatus(false, "invalid events save args")
					continue
				}
				bs, _ := json.MarshalIndent(s.eventLog, "", "  ")
				if err := os.WriteFile(path, bs, 0o644); err != nil {
					fmt.Printf("[save-error] %v\n", err)
					s.setStatus(false, err.Error())
					continue
				}
				fmt.Printf("saved events: %s\n", path)
				s.setStatus(true, "events saved")
				continue
			}
			parts := strings.Fields(text)
			limit := 20
			if len(parts) > 1 {
				if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
					limit = v
				}
			}
			start := len(s.eventLog) - limit
			if start < 0 {
				start = 0
			}
			printJSONMode(s.eventLog[start:], s.pretty)
			s.setStatus(true, "events listed")
		case strings.HasPrefix(text, "/bookmark "):
			parts := strings.Fields(text)
			if len(parts) >= 2 && parts[1] == "list" {
				list, err := s.loadBookmarks()
				if err != nil {
					fmt.Printf("[bookmark-error] %v\n", err)
					s.setStatus(false, err.Error())
					continue
				}
				printJSONMode(list, s.pretty)
				s.setStatus(true, "bookmarks listed")
				continue
			}
			if len(parts) >= 3 && parts[1] == "add" {
				if err := s.addBookmark(parts[2]); err != nil {
					fmt.Printf("[bookmark-error] %v\n", err)
					s.setStatus(false, err.Error())
					continue
				}
				fmt.Printf("bookmark saved: %s\n", parts[2])
				s.setStatus(true, "bookmark saved")
				continue
			}
			if len(parts) >= 3 && parts[1] == "use" {
				if err := s.useBookmark(parts[2]); err != nil {
					fmt.Printf("[bookmark-error] %v\n", err)
					s.setStatus(false, err.Error())
					continue
				}
				fmt.Printf("bookmark loaded: %s (session=%s)\n", parts[2], s.session)
				s.setStatus(true, "bookmark loaded")
				continue
			}
			fmt.Println("usage: /bookmark add <name> | /bookmark list | /bookmark use <name>")
			s.setStatus(false, "invalid bookmark args")
		case text == "/session":
			fmt.Printf("session: %s\n", s.session)
			s.setStatus(true, "session shown")
		case strings.HasPrefix(text, "/session "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/session "))
			if next == "" {
				fmt.Println("session id required")
				s.setStatus(false, "session id required")
				continue
			}
			s.session = next
			fmt.Printf("session switched: %s\n", s.session)
			s.setStatus(true, "session switched")
		case text == "/api":
			fmt.Printf("ws: %s\n", s.wsBase)
			s.setStatus(true, "ws shown")
		case strings.HasPrefix(text, "/api "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/api "))
			if !strings.HasPrefix(next, "ws://") && !strings.HasPrefix(next, "wss://") {
				fmt.Println("api must start with ws:// or wss://")
				s.setStatus(false, "invalid ws url")
				continue
			}
			s.wsBase = next
			fmt.Printf("ws switched: %s\n", s.wsBase)
			if strings.TrimSpace(os.Getenv("AGENT_HTTP_BASE")) == "" {
				s.httpBase = deriveHTTPBase(s.wsBase)
				fmt.Printf("http auto-updated: %s\n", s.httpBase)
			}
			s.setStatus(true, "ws switched")
		case text == "/http":
			fmt.Printf("http: %s\n", s.httpBase)
			s.setStatus(true, "http shown")
		case strings.HasPrefix(text, "/http "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/http "))
			if !strings.HasPrefix(next, "http://") && !strings.HasPrefix(next, "https://") {
				fmt.Println("http api must start with http:// or https://")
				s.setStatus(false, "invalid http url")
				continue
			}
			s.httpBase = strings.TrimRight(next, "/")
			fmt.Printf("http switched: %s\n", s.httpBase)
			s.setStatus(true, "http switched")
		case text == "/last":
			if s.lastJSON == nil {
				fmt.Println("no last json payload")
				s.setStatus(false, "no last json")
				continue
			}
			printJSONMode(s.lastJSON, s.pretty)
			s.setStatus(true, "last json shown")
		case strings.HasPrefix(text, "/save "):
			path := strings.TrimSpace(strings.TrimPrefix(text, "/save "))
			if path == "" {
				fmt.Println("usage: /save <file>")
				s.setStatus(false, "invalid save args")
				continue
			}
			if s.lastJSON == nil {
				fmt.Println("no last json payload")
				s.setStatus(false, "no last json")
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
				s.setStatus(false, err.Error())
				continue
			}
			fmt.Printf("saved: %s\n", path)
			s.setStatus(true, "json saved")
		case strings.HasPrefix(text, "/pretty "):
			mode := strings.TrimSpace(strings.TrimPrefix(text, "/pretty "))
			switch mode {
			case "on":
				s.pretty = true
				fmt.Println("pretty json: on")
				s.setStatus(true, "pretty on")
			case "off":
				s.pretty = false
				fmt.Println("pretty json: off")
				s.setStatus(true, "pretty off")
			default:
				fmt.Println("usage: /pretty on|off")
				s.setStatus(false, "invalid pretty args")
			}
		case text == "/tools":
			out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "tools listed")
		case strings.HasPrefix(text, "/tool "):
			name := strings.TrimSpace(strings.TrimPrefix(text, "/tool "))
			if name == "" {
				fmt.Println("usage: /tool <name>")
				s.setStatus(false, "invalid tool args")
				continue
			}
			out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/tools/"+url.PathEscape(name)+"/schema", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "tool schema loaded")
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
				s.setStatus(false, err.Error())
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
			s.setStatus(true, "sessions listed")
		case strings.HasPrefix(text, "/pick "):
			idx, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(text, "/pick ")))
			if err != nil || idx <= 0 {
				fmt.Println("usage: /pick <index>")
				s.setStatus(false, "invalid pick index")
				continue
			}
			if idx > len(s.lastSessions) {
				fmt.Printf("index out of range, max=%d\n", len(s.lastSessions))
				s.setStatus(false, "pick index out of range")
				continue
			}
			s.session = s.lastSessions[idx-1]
			s.lastShowSession = s.session
			s.lastShowOffset = 0
			fmt.Printf("session switched: %s\n", s.session)
			s.setStatus(true, "session switched")
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
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "show loaded")
		case text == "/next":
			s.lastShowOffset += s.lastShowLimit
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", s.httpBase, url.PathEscape(s.lastShowSession), s.lastShowOffset, s.lastShowLimit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.lastShowOffset -= s.lastShowLimit
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "next page loaded")
		case text == "/prev":
			s.lastShowOffset -= s.lastShowLimit
			if s.lastShowOffset < 0 {
				s.lastShowOffset = 0
			}
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", s.httpBase, url.PathEscape(s.lastShowSession), s.lastShowOffset, s.lastShowLimit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "prev page loaded")
		case strings.HasPrefix(text, "/stats"):
			parts := strings.Fields(text)
			sid := s.session
			if len(parts) > 1 {
				sid = parts[1]
			}
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=1", s.httpBase, url.PathEscape(sid)), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out["stats"]
			printJSONMode(out["stats"], s.pretty)
			s.setStatus(true, "stats loaded")
		case strings.HasPrefix(text, "/gateway "):
			parts := strings.Fields(text)
			if len(parts) != 2 {
				fmt.Println("usage: /gateway status|enable|disable")
				s.setStatus(false, "invalid gateway args")
				continue
			}
			switch parts[1] {
			case "status":
				out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/gateway/status", nil)
				if err != nil {
					fmt.Printf("[http-error] %v\n", err)
					s.setStatus(false, err.Error())
					continue
				}
				s.lastJSON = out
				printJSONMode(out, s.pretty)
				s.setStatus(true, "gateway status loaded")
			case "enable", "disable":
				out, err := httpJSON(http.MethodPost, s.httpBase+"/v1/ui/gateway/action", map[string]any{"action": parts[1]})
				if err != nil {
					fmt.Printf("[http-error] %v\n", err)
					s.setStatus(false, err.Error())
					continue
				}
				s.lastJSON = out
				printJSONMode(out, s.pretty)
				s.setStatus(true, "gateway action applied")
			default:
				fmt.Println("usage: /gateway status|enable|disable")
				s.setStatus(false, "invalid gateway action")
			}
		case text == "/config get":
			out, err := httpJSON(http.MethodGet, s.httpBase+"/v1/ui/config", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "config loaded")
		case strings.HasPrefix(text, "/config set "):
			parts := strings.SplitN(strings.TrimPrefix(text, "/config set "), " ", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
				fmt.Println("usage: /config set <section.key> <value>")
				s.setStatus(false, "invalid config args")
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := parts[1]
			out, err := httpJSON(http.MethodPost, s.httpBase+"/v1/ui/config/set", map[string]any{"key": key, "value": value})
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				s.setStatus(false, err.Error())
				continue
			}
			s.lastJSON = out
			printJSONMode(out, s.pretty)
			s.setStatus(true, "config updated")
		default:
			if err := sendTurn(s.wsBase, s.session, text, s.addEvent); err != nil {
				fmt.Printf("[ws-error] %v\n", err)
				s.setStatus(false, err.Error())
			} else {
				s.setStatus(true, "chat turn finished")
			}
		}
	}
}

