package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
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
	fmt.Println("/quit                 exit")
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

func main() {
	apiBase := getenvOr("AGENT_API_BASE", "ws://127.0.0.1:8080/v1/chat/ws")
	sessionID := getenvOr("AGENT_SESSION_ID", uuid.NewString())
	fmt.Printf("session: %s\n", sessionID)
	fmt.Printf("ws: %s\n", apiBase)
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
			continue
		default:
			if err := sendTurn(apiBase, sessionID, text); err != nil {
				fmt.Printf("[ws-error] %v\n", err)
			}
		}
	}
}

