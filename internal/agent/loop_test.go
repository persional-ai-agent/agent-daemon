package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	"github.com/dingjingmaster/agent-daemon/internal/memory"
	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type testTool struct {
	name string
	call func(context.Context, map[string]any, tools.ToolContext) (map[string]any, error)
}

func (t testTool) Name() string { return t.name }

func (t testTool) Schema() core.ToolSchema {
	return core.ToolSchema{
		Type: "function",
		Function: core.ToolSchemaDetail{
			Name:        t.name,
			Description: "test tool",
			Parameters:  map[string]any{"type": "object"},
		},
	}
}

func (t testTool) Call(ctx context.Context, args map[string]any, tc tools.ToolContext) (map[string]any, error) {
	return t.call(ctx, args, tc)
}

type scriptedClient struct {
	mu        sync.Mutex
	responses []core.Message
}

func (c *scriptedClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.responses) == 0 {
		return core.Message{}, nil
	}
	msg := c.responses[0]
	c.responses = c.responses[1:]
	return msg, nil
}

type waitOnContextClient struct{}

func (waitOnContextClient) ChatCompletion(ctx context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	<-ctx.Done()
	return core.Message{}, ctx.Err()
}

type recordingClient struct {
	mu       sync.Mutex
	messages [][]core.Message
	response core.Message
}

func (c *recordingClient) ChatCompletion(_ context.Context, messages []core.Message, _ []core.ToolSchema) (core.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, core.CloneMessages(messages))
	return c.response, nil
}

func TestRunEmitsDelegateEvents(t *testing.T) {
	args, err := json.Marshal(map[string]any{
		"goal":           "child-task",
		"context":        "child-context",
		"max_iterations": 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	client := &scriptedClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "delegate_task",
						Arguments: string(args),
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "child completed",
			},
			{
				Role:    "assistant",
				Content: "parent completed",
			},
		},
	}

	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, nil)

	events := make([]core.AgentEvent, 0)
	eng := &Engine{
		Client:       client,
		Registry:     registry,
		SystemPrompt: DefaultSystemPrompt(),
		EventSink: func(evt core.AgentEvent) {
			events = append(events, evt)
		},
	}

	res, err := eng.Run(context.Background(), "parent-session", "do work", eng.SystemPrompt, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalResponse != "parent completed" {
		t.Fatalf("unexpected final response: %+v", res)
	}

	foundStarted := false
	foundFinished := false
	for _, evt := range events {
		if evt.Type == "delegate_started" {
			foundStarted = true
			if evt.Content != "child-task" {
				t.Fatalf("unexpected delegate_started event: %+v", evt)
			}
			if evt.Data["status"] != "running" || evt.Data["goal"] != "child-task" {
				t.Fatalf("expected delegate_started data, got %+v", evt)
			}
		}
		if evt.Type == "delegate_finished" {
			foundFinished = true
			if evt.Content != "child completed" {
				t.Fatalf("unexpected delegate_finished event: %+v", evt)
			}
			if evt.Data["status"] != "completed" || evt.Data["success"] != true {
				t.Fatalf("expected delegate_finished status data, got %+v", evt)
			}
			result, ok := evt.Data["result"].(map[string]any)
			if !ok || result["final_response"] != "child completed" {
				t.Fatalf("expected delegate_finished result payload, got %+v", evt)
			}
		}
	}
	if !foundStarted || !foundFinished {
		t.Fatalf("expected delegate events, got %+v", events)
	}
}

func TestRunEmitsStructuredToolFinishedEvent(t *testing.T) {
	args, err := json.Marshal(map[string]any{"value": "ping"})
	if err != nil {
		t.Fatal(err)
	}

	client := &scriptedClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "echo_tool",
						Arguments: string(args),
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "done",
			},
		},
	}

	registry := tools.NewRegistry()
	registry.Register(testTool{
		name: "echo_tool",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"echo": args["value"]}, nil
		},
	})

	events := make([]core.AgentEvent, 0)
	eng := &Engine{
		Client:       client,
		Registry:     registry,
		SystemPrompt: DefaultSystemPrompt(),
		EventSink: func(evt core.AgentEvent) {
			events = append(events, evt)
		},
	}

	_, err = eng.Run(context.Background(), "tool-session", "use tool", eng.SystemPrompt, nil)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, evt := range events {
		if evt.Type != "tool_finished" {
			continue
		}
		found = true
		if evt.ToolName != "echo_tool" || evt.Data["status"] != "completed" || evt.Data["success"] != true {
			t.Fatalf("unexpected tool_finished event: %+v", evt)
		}
		result, ok := evt.Data["result"].(map[string]any)
		if !ok || result["echo"] != "ping" {
			t.Fatalf("expected structured tool result, got %+v", evt)
		}
	}
	if !found {
		t.Fatalf("expected tool_finished event, got %+v", events)
	}
}

func TestRunEmitsStructuredToolStartedEvent(t *testing.T) {
	args, err := json.Marshal(map[string]any{"value": "ping"})
	if err != nil {
		t.Fatal(err)
	}

	client := &scriptedClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-42",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "echo_tool",
						Arguments: string(args),
					},
				}},
			},
			{
				Role:    "assistant",
				Content: "done",
			},
		},
	}

	registry := tools.NewRegistry()
	registry.Register(testTool{
		name: "echo_tool",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"echo": args["value"]}, nil
		},
	})

	events := make([]core.AgentEvent, 0)
	eng := &Engine{
		Client:       client,
		Registry:     registry,
		SystemPrompt: DefaultSystemPrompt(),
		EventSink: func(evt core.AgentEvent) {
			events = append(events, evt)
		},
	}

	_, err = eng.Run(context.Background(), "tool-session", "use tool", eng.SystemPrompt, nil)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, evt := range events {
		if evt.Type != "tool_started" {
			continue
		}
		found = true
		if evt.ToolName != "echo_tool" || evt.Data["status"] != "running" || evt.Data["tool_call_id"] != "call-42" {
			t.Fatalf("unexpected tool_started event: %+v", evt)
		}
		if evt.Data["tool_name"] != "echo_tool" {
			t.Fatalf("expected tool_name in tool_started data, got %+v", evt)
		}
		arguments, ok := evt.Data["arguments"].(map[string]any)
		if !ok || arguments["value"] != "ping" {
			t.Fatalf("expected structured tool arguments, got %+v", evt)
		}
	}
	if !found {
		t.Fatalf("expected tool_started event, got %+v", events)
	}
}

func TestRunEmitsStructuredAssistantAndCompletedEvents(t *testing.T) {
	client := &scriptedClient{
		responses: []core.Message{
			{
				Role:    "assistant",
				Content: "plain answer",
			},
		},
	}

	registry := tools.NewRegistry()
	events := make([]core.AgentEvent, 0)
	eng := &Engine{
		Client:       client,
		Registry:     registry,
		SystemPrompt: DefaultSystemPrompt(),
		EventSink: func(evt core.AgentEvent) {
			events = append(events, evt)
		},
	}

	res, err := eng.Run(context.Background(), "assistant-session", "say hi", eng.SystemPrompt, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalResponse != "plain answer" {
		t.Fatalf("unexpected final response: %+v", res)
	}

	foundAssistant := false
	foundCompleted := false
	for _, evt := range events {
		if evt.Type == "assistant_message" {
			foundAssistant = true
			if evt.Data["status"] != "completed" || evt.Data["message_role"] != "assistant" {
				t.Fatalf("unexpected assistant_message data: %+v", evt)
			}
			if evt.Data["tool_call_count"] != 0 || evt.Data["has_tool_calls"] != false {
				t.Fatalf("unexpected assistant_message tool metadata: %+v", evt)
			}
		}
		if evt.Type == "completed" {
			foundCompleted = true
			if evt.Data["status"] != "completed" || evt.Data["finished_naturally"] != true {
				t.Fatalf("unexpected completed data: %+v", evt)
			}
			if evt.Data["content_length"] != len("plain answer") {
				t.Fatalf("unexpected completed content length: %+v", evt)
			}
		}
	}
	if !foundAssistant || !foundCompleted {
		t.Fatalf("expected assistant_message and completed events, got %+v", events)
	}
}

func TestRunEmitsStructuredCancelledEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	events := make([]core.AgentEvent, 0)
	eng := &Engine{
		Client:       waitOnContextClient{},
		Registry:     tools.NewRegistry(),
		SystemPrompt: DefaultSystemPrompt(),
		EventSink: func(evt core.AgentEvent) {
			events = append(events, evt)
		},
	}

	_, err := eng.Run(ctx, "cancel-session", "cancel me", eng.SystemPrompt, nil)
	if err == nil {
		t.Fatal("expected cancellation error")
	}

	found := false
	for _, evt := range events {
		if evt.Type != "cancelled" {
			continue
		}
		found = true
		if evt.Data["status"] != "cancelled" || evt.Data["turn"] != 1 || evt.Data["error"] == "" {
			t.Fatalf("unexpected cancelled event: %+v", evt)
		}
	}
	if !found {
		t.Fatalf("expected cancelled event, got %+v", events)
	}
}

func TestRunEmitsStructuredErrorEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	events := make([]core.AgentEvent, 0)
	eng := &Engine{
		Client:       waitOnContextClient{},
		Registry:     tools.NewRegistry(),
		SystemPrompt: DefaultSystemPrompt(),
		EventSink: func(evt core.AgentEvent) {
			events = append(events, evt)
		},
	}

	_, err := eng.Run(ctx, "error-session", "timeout me", eng.SystemPrompt, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	found := false
	for _, evt := range events {
		if evt.Type != "error" {
			continue
		}
		found = true
		if evt.Data["status"] != "error" || evt.Data["turn"] != 1 || evt.Data["error"] == "" {
			t.Fatalf("unexpected error event: %+v", evt)
		}
	}
	if !found {
		t.Fatalf("expected error event, got %+v", events)
	}
}

func TestRunEmitsStructuredMaxIterationsEvent(t *testing.T) {
	args, err := json.Marshal(map[string]any{"value": "ping"})
	if err != nil {
		t.Fatal(err)
	}

	client := &scriptedClient{
		responses: []core.Message{
			{
				Role: "assistant",
				ToolCalls: []core.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: core.ToolFunction{
						Name:      "echo_tool",
						Arguments: string(args),
					},
				}},
			},
		},
	}

	registry := tools.NewRegistry()
	registry.Register(testTool{
		name: "echo_tool",
		call: func(_ context.Context, args map[string]any, _ tools.ToolContext) (map[string]any, error) {
			return map[string]any{"echo": args["value"]}, nil
		},
	})

	events := make([]core.AgentEvent, 0)
	eng := &Engine{
		Client:        client,
		Registry:      registry,
		SystemPrompt:  DefaultSystemPrompt(),
		MaxIterations: 1,
		EventSink: func(evt core.AgentEvent) {
			events = append(events, evt)
		},
	}

	res, err := eng.Run(context.Background(), "max-session", "loop", eng.SystemPrompt, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.FinishedNaturally {
		t.Fatalf("expected unfinished result, got %+v", res)
	}

	found := false
	for _, evt := range events {
		if evt.Type != "max_iterations_reached" {
			continue
		}
		found = true
		if evt.Data["status"] != "max_iterations_reached" || evt.Data["max_iterations"] != 1 || evt.Data["finished"] != false {
			t.Fatalf("unexpected max_iterations event: %+v", evt)
		}
	}
	if !found {
		t.Fatalf("expected max_iterations_reached event, got %+v", events)
	}
}

func TestRunInjectsRuntimeSystemPromptWithMemoryAndWorkspaceRules(t *testing.T) {
	workdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workdir, "AGENTS.md"), []byte("project rule: keep tests focused"), 0o644); err != nil {
		t.Fatal(err)
	}
	memStore, err := memory.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := memStore.Manage("add", "memory", "user prefers concise output", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := memStore.Manage("add", "user", "workspace uses Go", ""); err != nil {
		t.Fatal(err)
	}

	client := &recordingClient{response: core.Message{Role: "assistant", Content: "done"}}
	eng := &Engine{
		Client:       client,
		Registry:     tools.NewRegistry(),
		MemoryStore:  memStore,
		Workdir:      workdir,
		SystemPrompt: "base system prompt",
	}

	_, err = eng.Run(context.Background(), "prompt-session", "hello", "ignored override", []core.Message{{Role: "user", Content: "older turn"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(client.messages) != 1 || len(client.messages[0]) == 0 {
		t.Fatalf("expected recorded model call, got %+v", client.messages)
	}
	system := client.messages[0][0]
	if system.Role != "system" {
		t.Fatalf("expected first message to be system, got %+v", client.messages[0])
	}
	for _, want := range []string{
		"ignored override",
		"user prefers concise output",
		"workspace uses Go",
		"project rule: keep tests focused",
	} {
		if !strings.Contains(system.Content, want) {
			t.Fatalf("expected system prompt to contain %q, got %q", want, system.Content)
		}
	}
}

func TestRunReplacesExistingSystemPromptInsteadOfDuplicating(t *testing.T) {
	client := &recordingClient{response: core.Message{Role: "assistant", Content: "done"}}
	eng := &Engine{
		Client:       client,
		Registry:     tools.NewRegistry(),
		SystemPrompt: "fresh system prompt",
		Workdir:      t.TempDir(),
	}

	existing := []core.Message{
		{Role: "system", Content: "stale system prompt"},
		{Role: "user", Content: "previous"},
	}
	_, err := eng.Run(context.Background(), "replace-system", "next", "", existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(client.messages) != 1 {
		t.Fatalf("expected one model call, got %+v", client.messages)
	}
	msgs := client.messages[0]
	if msgs[0].Role != "system" || !strings.Contains(msgs[0].Content, "fresh system prompt") {
		t.Fatalf("expected fresh system prompt, got %+v", msgs[0])
	}
	systemCount := 0
	for _, msg := range msgs {
		if msg.Role == "system" {
			systemCount++
		}
	}
	if systemCount != 1 {
		t.Fatalf("expected exactly one system message, got %+v", msgs)
	}
}

func TestRunEmitsContextCompactedEvent(t *testing.T) {
	client := &recordingClient{response: core.Message{Role: "assistant", Content: "done"}}
	events := make([]core.AgentEvent, 0)
	eng := &Engine{
		Client:                  client,
		Registry:                tools.NewRegistry(),
		SystemPrompt:            "base prompt",
		Workdir:                 t.TempDir(),
		MaxContextChars:         600,
		CompressionTailMessages: 2,
		EventSink: func(evt core.AgentEvent) {
			events = append(events, evt)
		},
	}
	existing := []core.Message{
		{Role: "user", Content: strings.Repeat("u1 ", 200)},
		{Role: "assistant", Content: strings.Repeat("a1 ", 200)},
		{Role: "user", Content: strings.Repeat("u2 ", 200)},
		{Role: "assistant", Content: strings.Repeat("a2 ", 200)},
	}
	_, err := eng.Run(context.Background(), "compact-session", "latest", "", existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(client.messages) != 1 {
		t.Fatalf("expected single model call, got %+v", client.messages)
	}
	foundSummary := false
	for _, msg := range client.messages[0] {
		if msg.Role == "assistant" && strings.Contains(msg.Content, contextSummaryPrefix) {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Fatalf("expected compacted summary message in model input, got %+v", client.messages[0])
	}
	foundEvent := false
	for _, evt := range events {
		if evt.Type == "context_compacted" {
			foundEvent = true
			if evt.Data["before_chars"] == nil || evt.Data["after_chars"] == nil {
				t.Fatalf("expected compaction event metadata, got %+v", evt)
			}
		}
	}
	if !foundEvent {
		t.Fatalf("expected context_compacted event, got %+v", events)
	}
}
