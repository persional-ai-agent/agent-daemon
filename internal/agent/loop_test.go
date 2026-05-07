package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
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
