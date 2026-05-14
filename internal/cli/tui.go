package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
)

// RunTUI starts chat mode with live event traces.
// This is a lightweight TUI step-up over plain chat.
func RunTUI(ctx context.Context, eng *agent.Engine, sessionID, firstMessage, preloadSkills string) error {
	previousSink := eng.EventSink
	eng.EventSink = func(evt core.AgentEvent) {
		if previousSink != nil {
			previousSink(evt)
		}
		switch evt.Type {
		case "user_message":
			fmt.Printf("[%-20s] session=%s chars=%d\n", evt.Type, evt.SessionID, len(evt.Content))
		case "turn_started":
			fmt.Printf("[%-20s] turn=%d\n", evt.Type, evt.Turn)
		case "model_stream_event":
			if evt.Data != nil {
				if typ, _ := evt.Data["event_type"].(string); strings.TrimSpace(typ) != "" {
					fmt.Printf("[%-20s] %s\n", evt.Type, typ)
				}
			}
		case "assistant_message":
			if evt.Content != "" {
				toolCount := any(0)
				if evt.Data != nil {
					toolCount = evt.Data["tool_call_count"]
				}
				fmt.Printf("[%-20s] chars=%d tools=%v\n", evt.Type, len(evt.Content), toolCount)
			}
		case "tool_started":
			fmt.Printf("[%-20s] %s\n", evt.Type, evt.ToolName)
		case "tool_finished":
			status := "completed"
			if evt.Data != nil {
				if v, ok := evt.Data["status"].(string); ok && strings.TrimSpace(v) != "" {
					status = v
				}
			}
			fmt.Printf("[%-20s] %s status=%s\n", evt.Type, evt.ToolName, status)
		case "mcp_stream_event":
			if evt.Data != nil {
				fmt.Printf("[%-20s] %s/%v\n", evt.Type, evt.ToolName, evt.Data["event_type"])
			}
		case "delegate_started", "delegate_finished", "delegate_failed":
			fmt.Printf("[%-20s] session=%s %s\n", evt.Type, evt.SessionID, strings.TrimSpace(evt.Content))
		case "context_compacted":
			fmt.Printf("[%-20s] %v\n", evt.Type, evt.Data)
		case "completed", "cancelled", "error", "max_iterations_reached":
			fmt.Printf("[%-20s] %s\n", evt.Type, time.Now().Format(time.RFC3339))
		}
	}
	return RunChat(ctx, eng, sessionID, firstMessage, preloadSkills)
}
