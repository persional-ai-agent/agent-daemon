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
	eng.EventSink = func(evt core.AgentEvent) {
		switch evt.Type {
		case "turn_started":
			fmt.Printf("[%-20s] turn=%d\n", evt.Type, evt.Turn)
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
		case "completed", "cancelled", "error", "max_iterations_reached":
			fmt.Printf("[%-20s] %s\n", evt.Type, time.Now().Format(time.RFC3339))
		}
	}
	return RunChat(ctx, eng, sessionID, firstMessage, preloadSkills)
}

