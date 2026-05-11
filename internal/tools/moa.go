package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

func (b *BuiltinTools) mixtureOfAgents(ctx context.Context, args map[string]any, tc ToolContext) (map[string]any, error) {
	userPrompt := strings.TrimSpace(strArg(args, "user_prompt"))
	if userPrompt == "" {
		return nil, errors.New("user_prompt required")
	}
	if tc.DelegateRunner == nil {
		return map[string]any{"success": false, "error": "delegate runner unavailable", "available": false}, nil
	}

	maxIterations := intArg(args, "max_iterations", 12)
	timeoutSeconds := intArg(args, "timeout_seconds", 180)
	numRefs := intArg(args, "reference_agents", 3)
	if numRefs <= 0 {
		numRefs = 3
	}
	if numRefs > 6 {
		numRefs = 6
	}

	roles := []string{
		"Propose a solution with clear steps and edge cases.",
		"Find flaws, missing cases, and suggest improvements.",
		"Give a concise, production-ready final answer.",
		"Consider security/safety implications and guardrails.",
		"Consider performance and simplicity tradeoffs.",
		"Write a minimal implementation plan/checklist.",
	}
	if numRefs < len(roles) {
		roles = roles[:numRefs]
	}

	type refOut struct {
		index int
		role  string
		res   map[string]any
		err   error
	}
	outCh := make(chan refOut, len(roles))
	var wg sync.WaitGroup
	wg.Add(len(roles))
	for i, role := range roles {
		go func(idx int, r string) {
			defer wg.Done()
			goal := userPrompt
			taskCtx := fmt.Sprintf("You are a reference agent in a mixture-of-agents workflow.\nRole: %s\n\nUser prompt:\n%s\n", r, userPrompt)
			res, err := runDelegateSubtask(ctx, tc.DelegateRunner, tc.SessionID, goal, taskCtx, maxIterations, timeoutSeconds)
			outCh <- refOut{index: idx, role: r, res: res, err: err}
		}(i, role)
	}
	wg.Wait()
	close(outCh)

	references := make([]map[string]any, 0, len(roles))
	refTexts := make([]string, len(roles))
	for item := range outCh {
		entry := map[string]any{"index": item.index, "role": item.role}
		if item.err != nil {
			entry["success"] = false
			entry["error"] = item.err.Error()
		} else {
			entry["success"] = true
			entry["result"] = item.res
			if s, ok := item.res["final_response"].(string); ok {
				refTexts[item.index] = strings.TrimSpace(s)
			}
		}
		references = append(references, entry)
	}

	// Aggregator subtask: summarize and synthesize reference outputs.
	aggCtx := strings.Builder{}
	aggCtx.WriteString("You are the aggregator in a mixture-of-agents workflow. Synthesize the best answer from the references. Be critical; references may be wrong.\n\nUser prompt:\n")
	aggCtx.WriteString(userPrompt)
	aggCtx.WriteString("\n\nReference responses:\n")
	for i, txt := range refTexts {
		if strings.TrimSpace(txt) == "" {
			continue
		}
		aggCtx.WriteString(fmt.Sprintf("\n[%d]\n%s\n", i+1, txt))
	}
	aggRes, aggErr := runDelegateSubtask(ctx, tc.DelegateRunner, tc.SessionID, userPrompt, aggCtx.String(), maxIterations, timeoutSeconds)
	if aggErr != nil {
		return map[string]any{
			"success":    false,
			"error":      aggErr.Error(),
			"references": references,
		}, nil
	}

	return map[string]any{
		"success":     true,
		"user_prompt": userPrompt,
		"references":  references,
		"aggregated":  aggRes,
	}, nil
}

