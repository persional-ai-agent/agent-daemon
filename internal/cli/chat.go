package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func RunChat(ctx context.Context, eng *agent.Engine, sessionID, firstMessage string) error {
	reader := bufio.NewReader(os.Stdin)
	history, _ := eng.SessionStore.LoadMessages(sessionID, 500)

	systemPrompt := defaultSystemPrompt()
	if strings.TrimSpace(firstMessage) != "" {
		res, err := eng.Run(ctx, sessionID, firstMessage, systemPrompt, history)
		if err != nil {
			return err
		}
		history = res.Messages
		fmt.Println(res.FinalResponse)
	}

	for {
		fmt.Print("agent> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "/exit" || line == "/quit" {
			return nil
		}
		res, err := eng.Run(ctx, sessionID, line, systemPrompt, history)
		if err != nil {
			return err
		}
		history = append([]core.Message(nil), res.Messages...)
		fmt.Println(res.FinalResponse)
	}
}

func defaultSystemPrompt() string {
	return `You are AgentDaemon, a tool-using coding agent. Use tools when actions are required.
- For filesystem operations, use read_file/write_file/search_files.
- For shell tasks, use terminal.
- Use todo to keep progress transparent.
- Keep responses concise and actionable.`
}
