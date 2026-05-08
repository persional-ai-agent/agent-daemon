package agent

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

const promptFileSizeLimit = 32 * 1024

func buildRuntimeSystemPrompt(basePrompt, workdir string, memoryStore tools.MemoryStore, registry *tools.Registry) string {
	parts := make([]string, 0, 4)
	basePrompt = strings.TrimSpace(basePrompt)
	if basePrompt != "" {
		parts = append(parts, basePrompt)
	}
	if memoryBlock := buildMemoryPromptBlock(memoryStore); memoryBlock != "" {
		parts = append(parts, memoryBlock)
	}
	if skillsBlock := buildSkillsIndexBlock(workdir, registry); skillsBlock != "" {
		parts = append(parts, skillsBlock)
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

type skillFrontmatter struct {
	RequiresTools []string `yaml:"requires_tools"`
	FallbackTools []string `yaml:"fallback_for_tools"`
}

func buildSkillsIndexBlock(workdir string, registry *tools.Registry) string {
	root := filepath.Join(workdir, "skills")
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}

	availableTools := buildToolNameSet(registry)

	var lines []string
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if count >= 50 {
			break
		}
		name := entry.Name()
		skillMD := filepath.Join(root, name, "SKILL.md")
		bs, err := os.ReadFile(skillMD)
		if err != nil {
			continue
		}

		fm := parseSkillFrontmatter(bs)
		if !skillShouldShow(fm, availableTools) {
			continue
		}

		desc := readSkillDescriptionFromBytes(bs)
		if desc == "" {
			desc = "(no description)"
		}
		lines = append(lines, "- "+name+": "+desc)
		count++
	}
	if len(lines) == 0 {
		return ""
	}

	header := "## Available Skills\nBefore each task, scan the skills below. If a skill is relevant, load it with skill_view(name) and follow its instructions.\n"
	return header + strings.Join(lines, "\n")
}

func parseSkillFrontmatter(bs []byte) skillFrontmatter {
	text := string(bs)
	if !strings.HasPrefix(strings.TrimSpace(text), "---") {
		return skillFrontmatter{}
	}
	parts := strings.SplitN(text[3:], "---", 2)
	if len(parts) < 2 {
		return skillFrontmatter{}
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(parts[0]), &fm); err != nil {
		return skillFrontmatter{}
	}
	return fm
}

func buildToolNameSet(registry *tools.Registry) map[string]struct{} {
	tools := make(map[string]struct{})
	if registry == nil {
		return tools
	}
	for _, name := range registry.Names() {
		tools[name] = struct{}{}
	}
	return tools
}

func skillShouldShow(fm skillFrontmatter, availableTools map[string]struct{}) bool {
	if len(fm.RequiresTools) > 0 {
		for _, req := range fm.RequiresTools {
			if _, ok := availableTools[req]; !ok {
				return false
			}
		}
	}
	if len(fm.FallbackTools) > 0 {
		for _, fb := range fm.FallbackTools {
			if _, ok := availableTools[fb]; ok {
				return false
			}
		}
	}
	return true
}

func readSkillDescriptionFromBytes(bs []byte) string {
	lines := strings.Split(string(bs), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if line != "" && !strings.HasPrefix(line, "---") {
			if len(line) > 120 {
				line = line[:120] + "..."
			}
			return line
		}
	}
	return ""
}
