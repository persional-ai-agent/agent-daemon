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
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func getenvOr(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
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
	fmt.Println("/quit                 exit")
	fmt.Println("aliases: :q, quit, ls, show, gw, cfg")
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

func sendTurn(apiBase, sessionID, message string) error {
	u, err := url.Parse(apiBase)
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
		evtType, _ := evt["type"].(string)
		if evtType == "" {
			evtType, _ = evt["Type"].(string)
		}
		if evtType == "result" || evtType == "error" || evtType == "cancelled" {
			return nil
		}
	}
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

func printJSON(v any) {
	bs, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(bs))
}

func printJSONMode(v any, pretty bool) {
	if pretty {
		printJSON(v)
		return
	}
	bs, _ := json.Marshal(v)
	fmt.Println(string(bs))
}

func main() {
	apiBase := getenvOr("AGENT_API_BASE", "ws://127.0.0.1:8080/v1/chat/ws")
	httpBase := getenvOr("AGENT_HTTP_BASE", deriveHTTPBase(apiBase))
	sessionID := getenvOr("AGENT_SESSION_ID", uuid.NewString())
	pretty := true
	var lastJSON any
	lastShowSession := sessionID
	lastShowOffset := 0
	lastShowLimit := 20
	lastSessions := make([]string, 0)
	fmt.Printf("session: %s\n", sessionID)
	fmt.Printf("ws: %s\n", apiBase)
	fmt.Printf("http: %s\n", httpBase)
	fmt.Println("输入 /help 查看命令")
	reader := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("tui> ")
		if !reader.Scan() {
			fmt.Println("bye")
			return
		}
		text := strings.TrimSpace(reader.Text())
		if text == "" {
			continue
		}
		switch text {
		case ":q", "quit":
			text = "/quit"
		case "ls":
			text = "/tools"
		case "sessions":
			text = "/sessions"
		case "show":
			text = "/show"
		case "next":
			text = "/next"
		case "prev":
			text = "/prev"
		case "stats":
			text = "/stats"
		case "gateway", "gw":
			text = "/gateway status"
		case "config", "cfg":
			text = "/config get"
		}
		if strings.HasPrefix(text, "show ") && !strings.HasPrefix(text, "/show ") {
			text = "/show " + strings.TrimSpace(strings.TrimPrefix(text, "show "))
		}
		if strings.HasPrefix(text, "sessions ") && !strings.HasPrefix(text, "/sessions ") {
			text = "/sessions " + strings.TrimSpace(strings.TrimPrefix(text, "sessions "))
		}
		if strings.HasPrefix(text, "tool ") && !strings.HasPrefix(text, "/tool ") {
			text = "/tool " + strings.TrimSpace(strings.TrimPrefix(text, "tool "))
		}
		if strings.HasPrefix(text, "gw ") && !strings.HasPrefix(text, "/gateway ") {
			text = "/gateway " + strings.TrimSpace(strings.TrimPrefix(text, "gw "))
		}
		if strings.HasPrefix(text, "cfg ") && !strings.HasPrefix(text, "/config ") {
			text = "/config " + strings.TrimSpace(strings.TrimPrefix(text, "cfg "))
		}
		switch {
		case text == "/quit" || text == "/exit":
			fmt.Println("bye")
			return
		case text == "/help":
			printHelp()
			continue
		case text == "/session":
			fmt.Printf("session: %s\n", sessionID)
			continue
		case strings.HasPrefix(text, "/session "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/session "))
			if next == "" {
				fmt.Println("session id required")
				continue
			}
			sessionID = next
			fmt.Printf("session switched: %s\n", sessionID)
			continue
		case text == "/api":
			fmt.Printf("ws: %s\n", apiBase)
			continue
		case strings.HasPrefix(text, "/api "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/api "))
			if !strings.HasPrefix(next, "ws://") && !strings.HasPrefix(next, "wss://") {
				fmt.Println("api must start with ws:// or wss://")
				continue
			}
			apiBase = next
			fmt.Printf("ws switched: %s\n", apiBase)
			if strings.TrimSpace(os.Getenv("AGENT_HTTP_BASE")) == "" {
				httpBase = deriveHTTPBase(apiBase)
				fmt.Printf("http auto-updated: %s\n", httpBase)
			}
			continue
		case text == "/http":
			fmt.Printf("http: %s\n", httpBase)
			continue
		case strings.HasPrefix(text, "/http "):
			next := strings.TrimSpace(strings.TrimPrefix(text, "/http "))
			if !strings.HasPrefix(next, "http://") && !strings.HasPrefix(next, "https://") {
				fmt.Println("http api must start with http:// or https://")
				continue
			}
			httpBase = strings.TrimRight(next, "/")
			fmt.Printf("http switched: %s\n", httpBase)
			continue
		case text == "/last":
			if lastJSON == nil {
				fmt.Println("no last json payload")
				continue
			}
			printJSONMode(lastJSON, pretty)
			continue
		case strings.HasPrefix(text, "/save "):
			path := strings.TrimSpace(strings.TrimPrefix(text, "/save "))
			if path == "" {
				fmt.Println("usage: /save <file>")
				continue
			}
			if lastJSON == nil {
				fmt.Println("no last json payload")
				continue
			}
			var bs []byte
			if pretty {
				bs, _ = json.MarshalIndent(lastJSON, "", "  ")
			} else {
				bs, _ = json.Marshal(lastJSON)
			}
			if err := os.WriteFile(path, bs, 0o644); err != nil {
				fmt.Printf("[save-error] %v\n", err)
				continue
			}
			fmt.Printf("saved: %s\n", path)
			continue
		case strings.HasPrefix(text, "/pretty "):
			mode := strings.TrimSpace(strings.TrimPrefix(text, "/pretty "))
			switch mode {
			case "on":
				pretty = true
				fmt.Println("pretty json: on")
			case "off":
				pretty = false
				fmt.Println("pretty json: off")
			default:
				fmt.Println("usage: /pretty on|off")
			}
			continue
		case text == "/tools":
			out, err := httpJSON(http.MethodGet, httpBase+"/v1/ui/tools", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				continue
			}
			lastJSON = out
			printJSONMode(out, pretty)
			continue
		case strings.HasPrefix(text, "/tool "):
			name := strings.TrimSpace(strings.TrimPrefix(text, "/tool "))
			if name == "" {
				fmt.Println("usage: /tool <name>")
				continue
			}
			out, err := httpJSON(http.MethodGet, httpBase+"/v1/ui/tools/"+url.PathEscape(name)+"/schema", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				continue
			}
			lastJSON = out
			printJSONMode(out, pretty)
			continue
		case strings.HasPrefix(text, "/sessions"):
			parts := strings.Fields(text)
			limit := 20
			if len(parts) > 1 {
				if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
					limit = v
				}
			}
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions?limit=%d", httpBase, limit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				continue
			}
			lastSessions = lastSessions[:0]
			if rows, ok := out["sessions"].([]any); ok {
				for _, row := range rows {
					m, ok := row.(map[string]any)
					if !ok {
						continue
					}
					if sid, ok := m["session_id"].(string); ok && strings.TrimSpace(sid) != "" {
						lastSessions = append(lastSessions, sid)
					}
				}
			}
			lastJSON = out
			printJSONMode(out, pretty)
			continue
		case strings.HasPrefix(text, "/pick "):
			arg := strings.TrimSpace(strings.TrimPrefix(text, "/pick "))
			idx, err := strconv.Atoi(arg)
			if err != nil || idx <= 0 {
				fmt.Println("usage: /pick <index>")
				continue
			}
			if idx > len(lastSessions) {
				fmt.Printf("index out of range, max=%d\n", len(lastSessions))
				continue
			}
			sessionID = lastSessions[idx-1]
			lastShowSession = sessionID
			lastShowOffset = 0
			fmt.Printf("session switched: %s\n", sessionID)
			continue
		case strings.HasPrefix(text, "/show"):
			parts := strings.Fields(text)
			sid := sessionID
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
			lastShowSession = sid
			lastShowOffset = offset
			lastShowLimit = limit
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", httpBase, url.PathEscape(sid), offset, limit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				continue
			}
			lastJSON = out
			printJSONMode(out, pretty)
			continue
		case text == "/next":
			lastShowOffset += lastShowLimit
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", httpBase, url.PathEscape(lastShowSession), lastShowOffset, lastShowLimit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				lastShowOffset -= lastShowLimit
				continue
			}
			lastJSON = out
			printJSONMode(out, pretty)
			continue
		case text == "/prev":
			lastShowOffset -= lastShowLimit
			if lastShowOffset < 0 {
				lastShowOffset = 0
			}
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=%d&limit=%d", httpBase, url.PathEscape(lastShowSession), lastShowOffset, lastShowLimit), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				continue
			}
			lastJSON = out
			printJSONMode(out, pretty)
			continue
		case strings.HasPrefix(text, "/stats"):
			parts := strings.Fields(text)
			sid := sessionID
			if len(parts) > 1 {
				sid = parts[1]
			}
			out, err := httpJSON(http.MethodGet, fmt.Sprintf("%s/v1/ui/sessions/%s?offset=0&limit=1", httpBase, url.PathEscape(sid)), nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				continue
			}
			lastJSON = out["stats"]
			printJSONMode(out["stats"], pretty)
			continue
		case strings.HasPrefix(text, "/gateway "):
			parts := strings.Fields(text)
			if len(parts) != 2 {
				fmt.Println("usage: /gateway status|enable|disable")
				continue
			}
			switch parts[1] {
			case "status":
				out, err := httpJSON(http.MethodGet, httpBase+"/v1/ui/gateway/status", nil)
				if err != nil {
					fmt.Printf("[http-error] %v\n", err)
					continue
				}
				lastJSON = out
				printJSONMode(out, pretty)
			case "enable", "disable":
				out, err := httpJSON(http.MethodPost, httpBase+"/v1/ui/gateway/action", map[string]any{"action": parts[1]})
				if err != nil {
					fmt.Printf("[http-error] %v\n", err)
					continue
				}
				lastJSON = out
				printJSONMode(out, pretty)
			default:
				fmt.Println("usage: /gateway status|enable|disable")
			}
			continue
		case text == "/config get":
			out, err := httpJSON(http.MethodGet, httpBase+"/v1/ui/config", nil)
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				continue
			}
			lastJSON = out
			printJSONMode(out, pretty)
			continue
		case strings.HasPrefix(text, "/config set "):
			parts := strings.SplitN(strings.TrimPrefix(text, "/config set "), " ", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
				fmt.Println("usage: /config set <section.key> <value>")
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := parts[1]
			out, err := httpJSON(http.MethodPost, httpBase+"/v1/ui/config/set", map[string]any{"key": key, "value": value})
			if err != nil {
				fmt.Printf("[http-error] %v\n", err)
				continue
			}
			lastJSON = out
			printJSONMode(out, pretty)
			continue
		default:
			if err := sendTurn(apiBase, sessionID, text); err != nil {
				fmt.Printf("[ws-error] %v\n", err)
			}
		}
	}
}
