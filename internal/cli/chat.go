package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	printChatWelcome(sessionID)
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
		if strings.HasPrefix(line, "/") {
			nextHistory, nextPrompt, handled, err := handleSlashCommand(line, sessionID, systemPrompt, history, eng)
			if err != nil {
				return err
			}
			if handled {
				history = nextHistory
				systemPrompt = nextPrompt
				continue
			}
		}
		res, err := eng.Run(ctx, sessionID, line, systemPrompt, history)
		if err != nil {
			return err
		}
		history = append([]core.Message(nil), res.Messages...)
		fmt.Println(res.FinalResponse)
	}
}

func printChatWelcome(sessionID string) {
	fmt.Printf("session: %s\n", sessionID)
	fmt.Println("输入 /help 查看可用命令，/quit 退出。")
}

func handleSlashCommand(line, sessionID, systemPrompt string, history []core.Message, eng *agent.Engine) ([]core.Message, string, bool, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return history, systemPrompt, true, nil
	}
	switch fields[0] {
	case "/help":
		printSlashHelp()
		return history, systemPrompt, true, nil
	case "/session":
		fmt.Printf("当前 session: %s\n", sessionID)
		return history, systemPrompt, true, nil
	case "/tools":
		names := eng.Registry.Names()
		fmt.Printf("可用工具 (%d): %s\n", len(names), strings.Join(names, ", "))
		return history, systemPrompt, true, nil
	case "/history":
		limit := 10
		if len(fields) > 1 {
			v, err := strconv.Atoi(fields[1])
			if err != nil || v <= 0 {
				fmt.Println("用法: /history [n]  (n 必须是正整数)")
				return history, systemPrompt, true, nil
			}
			limit = v
		}
		printHistoryPreview(history, limit)
		return history, systemPrompt, true, nil
	case "/reload":
		loaded, err := eng.SessionStore.LoadMessages(sessionID, 500)
		if err != nil {
			return history, systemPrompt, true, err
		}
		fmt.Printf("已从存储重载历史消息: %d 条\n", len(loaded))
		return loaded, systemPrompt, true, nil
	case "/clear":
		// 仅清空当前进程内上下文；不会删除持久化会话。
		fmt.Println("已清空当前进程内历史上下文（持久化会话未删除）。")
		return nil, systemPrompt, true, nil
	case "/tui":
		printTUIStatus(eng)
		return history, systemPrompt, true, nil
	default:
		fmt.Printf("未知命令: %s（输入 /help 查看命令）\n", fields[0])
		return history, systemPrompt, true, nil
	}
}

func printSlashHelp() {
	lines := []string{
		"/help                 显示命令帮助",
		"/session              显示当前会话 ID",
		"/tools                列出当前启用工具",
		"/history [n]          预览最近 n 条历史消息（默认 10）",
		"/reload               从存储重载历史消息",
		"/clear                清空当前进程内上下文",
		"/tui                  显示 CLI/TUI 能力状态",
		"/quit 或 /exit         退出会话",
	}
	fmt.Println(strings.Join(lines, "\n"))
}

func printHistoryPreview(history []core.Message, limit int) {
	if len(history) == 0 {
		fmt.Println("历史为空。")
		return
	}
	if limit > len(history) {
		limit = len(history)
	}
	start := len(history) - limit
	for i := start; i < len(history); i++ {
		msg := history[i]
		content := strings.TrimSpace(msg.Content)
		if len(content) > 120 {
			content = content[:120] + "..."
		}
		if content == "" {
			content = "(empty)"
		}
		fmt.Printf("%d. [%s] %s\n", i+1, msg.Role, content)
	}
}

func printTUIStatus(eng *agent.Engine) {
	tools := eng.Registry.Names()
	sort.Strings(tools)
	fmt.Println("CLI/TUI 状态")
	fmt.Printf("- 运行模式: 交互式 CLI（slash 命令已启用）\n")
	fmt.Printf("- 当前工具数量: %d\n", len(tools))
	fmt.Printf("- 功能入口: /help, /tools, /history, /reload\n")
	fmt.Printf("- Web 面板: 参见 docs/dev/207-* 文档与 web/ 工程\n")
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
