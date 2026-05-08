package model

import (
	"context"
	"fmt"
	"testing"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type fakeClient struct {
	resp core.Message
	err  error
}

func (f fakeClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	if f.err != nil {
		return core.Message{}, f.err
	}
	return f.resp, nil
}

func TestFallbackClientUsesPrimaryOnSuccess(t *testing.T) {
	primary := fakeClient{resp: core.Message{Role: "assistant", Content: "primary"}}
	fallback := fakeClient{resp: core.Message{Role: "assistant", Content: "fallback"}}
	c := NewFallbackClient(primary, "openai", fallback, "anthropic")
	got, err := c.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "primary" {
		t.Fatalf("expected primary response, got %+v", got)
	}
}

func TestFallbackClientUsesFallbackOnRetriableError(t *testing.T) {
	primary := fakeClient{err: fmt.Errorf("openai api error (503): overloaded")}
	fallback := fakeClient{resp: core.Message{Role: "assistant", Content: "fallback"}}
	c := NewFallbackClient(primary, "openai", fallback, "anthropic")
	got, err := c.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "fallback" {
		t.Fatalf("expected fallback response, got %+v", got)
	}
}

func TestFallbackClientSkipsFallbackOnNonRetriableError(t *testing.T) {
	primaryErr := fmt.Errorf("openai api error (400): bad request")
	primary := fakeClient{err: primaryErr}
	fallback := fakeClient{resp: core.Message{Role: "assistant", Content: "fallback"}}
	c := NewFallbackClient(primary, "openai", fallback, "anthropic")
	_, err := c.ChatCompletion(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != primaryErr.Error() {
		t.Fatalf("expected primary error returned, got %v", err)
	}
}
