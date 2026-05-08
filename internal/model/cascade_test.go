package model

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type mockCascadeClient struct {
	name     string
	response string
	fail     bool
}

func (m *mockCascadeClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	if m.fail {
		return core.Message{}, fmt.Errorf("service unavailable (503)")
	}
	return core.Message{Role: "assistant", Content: m.response}, nil
}

func (m *mockCascadeClient) ChatCompletionWithEvents(_ context.Context, _ []core.Message, _ []core.ToolSchema, _ StreamEventSink) (core.Message, error) {
	return m.ChatCompletion(nil, nil, nil)
}

func TestCascadeClientFirstProviderSucceeds(t *testing.T) {
	c := NewCascadeClient([]ProviderEntry{
		{Client: &mockCascadeClient{name: "a", response: "from a"}, Name: "a", Cost: 1.0},
		{Client: &mockCascadeClient{name: "b", response: "from b"}, Name: "b", Cost: 2.0},
	}, false)

	msg, err := c.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "from a" {
		t.Fatalf("expected 'from a', got %q", msg.Content)
	}
}

func TestCascadeClientFallbackOnFailure(t *testing.T) {
	c := NewCascadeClient([]ProviderEntry{
		{Client: &mockCascadeClient{name: "a", fail: true}, Name: "a", Cost: 1.0},
		{Client: &mockCascadeClient{name: "b", response: "from b"}, Name: "b", Cost: 2.0},
	}, false)

	msg, err := c.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "from b" {
		t.Fatalf("expected 'from b', got %q", msg.Content)
	}
}

func TestCascadeClientCostAware(t *testing.T) {
	c := NewCascadeClient([]ProviderEntry{
		{Client: &mockCascadeClient{name: "expensive", fail: true}, Name: "expensive", Cost: 3.0},
		{Client: &mockCascadeClient{name: "cheap", response: "from cheap"}, Name: "cheap", Cost: 0.5},
	}, true)

	msg, err := c.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "from cheap" {
		t.Fatalf("expected cheapest first: 'from cheap', got %q", msg.Content)
	}
}

func TestCascadeClientAllFail(t *testing.T) {
	c := NewCascadeClient([]ProviderEntry{
		{Client: &mockCascadeClient{name: "a", fail: true}, Name: "a", Cost: 1.0},
		{Client: &mockCascadeClient{name: "b", fail: true}, Name: "b", Cost: 2.0},
	}, false)

	_, err := c.ChatCompletion(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCascadeProviders(t *testing.T) {
	builder := func(name string) (Client, string, error) {
		if name == "unknown" {
			return nil, "", fmt.Errorf("unknown")
		}
		return &mockCascadeClient{name: name}, name, nil
	}

	entries, err := ParseCascadeProviders("openai:1.0, anthropic:2.5, codex", builder)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3, got %d", len(entries))
	}
	if entries[0].Cost != 1.0 || entries[1].Cost != 2.5 || entries[2].Cost != 1.0 {
		t.Fatalf("unexpected costs: %+v", entries)
	}
}

func TestParseCascadeProvidersUnknown(t *testing.T) {
	builder := func(name string) (Client, string, error) {
		if name == "unknown" {
			return nil, "", fmt.Errorf("unknown")
		}
		return &mockCascadeClient{name: name}, name, nil
	}

	_, err := ParseCascadeProviders("unknown", builder)
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected unknown error, got %v", err)
	}
}
