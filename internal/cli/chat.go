package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/agent"
	"github.com/dingjingmaster/agent-daemon/internal/core"
)

func RunChat(ctx context.Context, eng *agent.Engine, sessionID, firstMessage, preloadSkills string) error {
	reader := bufio.NewReader(os.Stdin)
	history, _ := eng.SessionStore.LoadMessages(sessionID, 500)
	systemPrompt := agent.DefaultSystemPrompt()

	if strings.TrimSpace(preloadSkills) != "" {
		block := buildPreloadedSkillsBlock(eng.Workdir, preloadSkills)
		if block != "" {
			systemPrompt = systemPrompt + "\n\n" + block
		}
	}

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

func buildPreloadedSkillsBlock(workdir, skillsCSV string) string {
	names := strings.Split(skillsCSV, ",")
	var parts []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path := filepath.Join(workdir, "skills", name, "SKILL.md")
		bs, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parts = append(parts, "## Preloaded Skill: "+name+"\n"+string(bs))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[IMPORTANT: The user has preloaded the following skill(s) for this session. Follow their instructions carefully.]\n\n" + strings.Join(parts, "\n\n")
}
