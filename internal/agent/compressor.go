package agent

import (
	"fmt"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

const (
	defaultMaxContextChars       = 120000
	defaultCompressionTailTurns  = 14
	compressionSummaryCharBudget = 6000
	contextSummaryPrefix         = "[CONTEXT COMPACTION] Earlier conversation was compacted. Use this as reference and continue from the latest user request."
)

func compressMessages(messages []core.Message, maxChars, tailMessages int) ([]core.Message, map[string]any) {
	if maxChars <= 0 {
		maxChars = defaultMaxContextChars
	}
	if tailMessages <= 0 {
		tailMessages = defaultCompressionTailTurns
	}
	before := estimateMessagesChars(messages)
	if before <= maxChars || len(messages) <= 2 {
		return messages, nil
	}

	start := 0
	head := make([]core.Message, 0, 1)
	if len(messages) > 0 && messages[0].Role == "system" {
		head = append(head, messages[0])
		start = 1
	}
	if len(messages)-start <= tailMessages {
		return messages, nil
	}

	tailStart := len(messages) - tailMessages
	for tailStart > start && messages[tailStart].Role == "tool" {
		tailStart--
	}
	if tailStart <= start {
		return messages, nil
	}

	mid := messages[start:tailStart]
	tail := messages[tailStart:]
	summary := summarizeMessages(mid, compressionSummaryCharBudget)
	if strings.TrimSpace(summary) == "" {
		return messages, nil
	}
	summaryMsg := core.Message{
		Role:    "assistant",
		Content: contextSummaryPrefix + "\n\n" + summary,
	}
	out := make([]core.Message, 0, len(head)+1+len(tail))
	out = append(out, head...)
	out = append(out, summaryMsg)
	out = append(out, tail...)
	after := estimateMessagesChars(out)
	meta := map[string]any{
		"before_chars":      before,
		"after_chars":       after,
		"max_context_chars": maxChars,
		"dropped_messages":  len(mid),
	}
	return out, meta
}

func summarizeMessages(messages []core.Message, budget int) string {
	if budget <= 0 {
		budget = compressionSummaryCharBudget
	}
	lines := make([]string, 0, len(messages))
	used := 0
	appendLine := func(line string) bool {
		line = strings.TrimSpace(line)
		if line == "" {
			return true
		}
		if used+len(line)+1 > budget {
			remain := budget - used - 12
			if remain <= 0 {
				return false
			}
			line = line[:remain] + "...[truncated]"
		}
		lines = append(lines, line)
		used += len(line) + 1
		return used < budget
	}

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			if !appendLine("- user: " + msg.Content) {
				return strings.Join(lines, "\n")
			}
		case "assistant":
			text := msg.Content
			if len(msg.ToolCalls) > 0 {
				toolNames := make([]string, 0, len(msg.ToolCalls))
				for _, tc := range msg.ToolCalls {
					toolNames = append(toolNames, tc.Function.Name)
				}
				if strings.TrimSpace(text) == "" {
					text = fmt.Sprintf("requested tools: %s", strings.Join(toolNames, ", "))
				} else {
					text = text + fmt.Sprintf(" (requested tools: %s)", strings.Join(toolNames, ", "))
				}
			}
			if !appendLine("- assistant: " + text) {
				return strings.Join(lines, "\n")
			}
		case "tool":
			text := msg.Content
			if len(text) > 300 {
				text = text[:300] + "...[truncated]"
			}
			if !appendLine(fmt.Sprintf("- tool `%s`: %s", msg.Name, text)) {
				return strings.Join(lines, "\n")
			}
		}
	}
	return strings.Join(lines, "\n")
}

func estimateMessagesChars(messages []core.Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Role) + len(msg.Content) + len(msg.Name) + len(msg.ToolCallID) + 16
		for _, tc := range msg.ToolCalls {
			total += len(tc.ID) + len(tc.Type) + len(tc.Function.Name) + len(tc.Function.Arguments) + 8
		}
	}
	return total
}
