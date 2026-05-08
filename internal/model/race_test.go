package model

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type slowClient struct {
	delay  time.Duration
	resp   core.Message
	err    error
	called bool
	mu     sync.Mutex
}

func (f *slowClient) ChatCompletion(_ context.Context, _ []core.Message, _ []core.ToolSchema) (core.Message, error) {
	f.mu.Lock()
	f.called = true
	f.mu.Unlock()
	time.Sleep(f.delay)
	if f.err != nil {
		return core.Message{}, f.err
	}
	return f.resp, nil
}

func TestRaceClientPicksFastest(t *testing.T) {
	slow := &slowClient{delay: 200 * time.Millisecond, resp: core.Message{Content: "slow"}}
	fast := &slowClient{delay: 50 * time.Millisecond, resp: core.Message{Content: "fast"}}
	c := NewRaceClient(slow, "slow", fast, "fast", 3, 60*time.Second, 1)
	got, err := c.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "fast" {
		t.Fatalf("expected fast response, got %s", got.Content)
	}
}

func TestRaceClientFallsBackWhenPrimaryOpen(t *testing.T) {
	primary := &slowClient{err: fmt.Errorf("api error (500): internal error")}
	fallback := &slowClient{delay: 50 * time.Millisecond, resp: core.Message{Content: "fallback"}}
	c := NewRaceClient(primary, "primary", fallback, "fallback", 2, 60*time.Second, 1)
	c.PrimaryCircuit.RecordFailure()
	c.PrimaryCircuit.RecordFailure()
	got, err := c.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "fallback" {
		t.Fatalf("expected fallback response, got %s", got.Content)
	}
}

func TestRaceClientAllOpenReturnsError(t *testing.T) {
	primary := &slowClient{err: fmt.Errorf("error")}
	fallback := &slowClient{err: fmt.Errorf("error")}
	c := NewRaceClient(primary, "primary", fallback, "fallback", 2, 60*time.Second, 1)
	c.PrimaryCircuit.RecordFailure()
	c.PrimaryCircuit.RecordFailure()
	c.FallbackCircuit.RecordFailure()
	c.FallbackCircuit.RecordFailure()
	_, err := c.ChatCompletion(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error when all circuits open")
	}
}

func TestFallbackClientCircuitIntegration(t *testing.T) {
	primary := fakeClient{err: fmt.Errorf("api error (503): overloaded")}
	fallback := fakeClient{resp: core.Message{Content: "fallback"}}
	c := NewFallbackClientWithCircuit(primary, "primary", fallback, "fallback", 3, 60*time.Second, 1)
	c.PrimaryCircuit.RecordFailure()
	c.PrimaryCircuit.RecordFailure()
	c.PrimaryCircuit.RecordFailure()
	got, err := c.ChatCompletion(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "fallback" {
		t.Fatalf("expected fallback, got %s", got.Content)
	}
}
