package agent

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

const promptFileSizeLimit = 32 * 1024

func buildRuntimeSystemPrompt(basePrompt, workdir string, memoryStore tools.MemoryStore) string {
	parts := make([]string, 0, 4)
	basePrompt = strings.TrimSpace(basePrompt)
	if basePrompt != "" {
		parts = append(parts, basePrompt)
	}
	if memoryBlock := buildMemoryPromptBlock(memoryStore); memoryBlock != "" {
		parts = append(parts, memoryBlock)
	}
	if rulesBlock := buildWorkspaceRulesPromptBlock(workdir); rulesBlock != "" {
		parts = append(parts, rulesBlock)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func withSystemPrompt(messages []core.Message, prompt string) []core.Message {
	if strings.TrimSpace(prompt) == "" {
		return messages
	}
	if len(messages) > 0 && messages[0].Role == "system" {
		out := core.CloneMessages(messages)
		out[0].Content = prompt
		return out
	}
	return append([]core.Message{{Role: "system", Content: prompt}}, messages...)
}

func buildMemoryPromptBlock(memoryStore tools.MemoryStore) string {
	if memoryStore == nil {
		return ""
	}
	snapshot, err := memoryStore.Snapshot()
	if err != nil {
		return ""
	}
	memoryText := strings.TrimSpace(snapshot["memory"])
	userText := strings.TrimSpace(snapshot["user"])
	if memoryText == "" && userText == "" {
		return ""
	}
	parts := []string{"## Persistent Memory"}
	if memoryText != "" {
		parts = append(parts, "### MEMORY.md\n"+memoryText)
	}
	if userText != "" {
		parts = append(parts, "### USER.md\n"+userText)
	}
	return strings.Join(parts, "\n\n")
}

func buildWorkspaceRulesPromptBlock(workdir string) string {
	path, content := loadWorkspaceRules(workdir)
	if content == "" {
		return ""
	}
	return "## Workspace Rules\nFollow the repository instructions from `" + path + "`.\n\n" + content
}

func loadWorkspaceRules(workdir string) (string, string) {
	base := strings.TrimSpace(workdir)
	if base == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", ""
		}
		base = wd
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return "", ""
	}
	for dir := base; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, "AGENTS.md")
		content, err := readPromptFile(path)
		if err == nil && strings.TrimSpace(content) != "" {
			return path, content
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", ""
}

func readPromptFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	size := info.Size()
	if size > promptFileSizeLimit {
		size = promptFileSizeLimit
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, size)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(string(buf[:n])), nil
}
