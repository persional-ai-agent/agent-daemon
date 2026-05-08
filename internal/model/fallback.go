package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type FallbackClient struct {
	Primary            Client
	PrimaryName        string
	Fallback           Client
	FallbackName       string
	IncludeStatusCodes []int
	PrimaryCircuit     *ProviderCircuit
	FallbackCircuit    *ProviderCircuit
}

func NewFallbackClient(primary Client, primaryName string, fallback Client, fallbackName string) *FallbackClient {
	return &FallbackClient{
		Primary:            primary,
		PrimaryName:        strings.TrimSpace(primaryName),
		Fallback:           fallback,
		FallbackName:       strings.TrimSpace(fallbackName),
		IncludeStatusCodes: []int{408, 429, 500, 502, 503, 504},
	}
}

func NewFallbackClientWithCircuit(primary Client, primaryName string, fallback Client, fallbackName string, circuitThreshold int, circuitRecoveryTimeout time.Duration, circuitHalfOpenMaxReqs int) *FallbackClient {
	return &FallbackClient{
		Primary:            primary,
		PrimaryName:        strings.TrimSpace(primaryName),
		Fallback:           fallback,
		FallbackName:       strings.TrimSpace(fallbackName),
		IncludeStatusCodes: []int{408, 429, 500, 502, 503, 504},
		PrimaryCircuit:     NewProviderCircuit(circuitThreshold, circuitRecoveryTimeout, circuitHalfOpenMaxReqs),
		FallbackCircuit:    NewProviderCircuit(circuitThreshold, circuitRecoveryTimeout, circuitHalfOpenMaxReqs),
	}
}

func (c *FallbackClient) ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error) {
	return c.ChatCompletionWithEvents(ctx, messages, tools, nil)
}

func (c *FallbackClient) ChatCompletionWithEvents(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	if c == nil || c.Primary == nil {
		return core.Message{}, fmt.Errorf("primary model client unavailable")
	}

	primaryAllowed := c.PrimaryCircuit == nil || c.PrimaryCircuit.AllowRequest()
	if !primaryAllowed {
		if c.Fallback == nil {
			return core.Message{}, fmt.Errorf("primary (%s) circuit breaker open, no fallback available", c.PrimaryName)
		}
		fallbackAllowed := c.FallbackCircuit == nil || c.FallbackCircuit.AllowRequest()
		if !fallbackAllowed {
			return core.Message{}, fmt.Errorf("primary (%s) and fallback (%s) circuit breakers both open", c.PrimaryName, c.FallbackName)
		}
		return c.callWithCircuit(ctx, c.Fallback, c.FallbackName, c.FallbackCircuit, messages, tools, sink)
	}

	msg, err := c.callWithCircuit(ctx, c.Primary, c.PrimaryName, c.PrimaryCircuit, messages, tools, sink)
	if err == nil {
		return msg, nil
	}

	if c.Fallback == nil || !shouldFallbackOnError(err, c.IncludeStatusCodes) {
		return core.Message{}, err
	}

	fallbackAllowed := c.FallbackCircuit == nil || c.FallbackCircuit.AllowRequest()
	if !fallbackAllowed {
		return core.Message{}, fmt.Errorf("primary (%s) failed: %v; fallback (%s) circuit breaker open", c.PrimaryName, err, c.FallbackName)
	}

	return c.callWithCircuit(ctx, c.Fallback, c.FallbackName, c.FallbackCircuit, messages, tools, sink)
}

func (c *FallbackClient) callWithCircuit(ctx context.Context, client Client, name string, circuit *ProviderCircuit, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	if circuit != nil && circuit.State() == CircuitHalfOpen {
		circuit.IncrementHalfOpenRequests()
	}

	msg, err := CompleteWithEvents(ctx, client, messages, tools, sink)

	if circuit != nil {
		if err == nil {
			circuit.RecordSuccess()
		} else {
			circuit.RecordFailure()
		}
	}

	return msg, err
}

func shouldFallbackOnError(err error, includeStatusCodes []int) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	for _, code := range includeStatusCodes {
		if strings.Contains(text, fmt.Sprintf("(%d)", code)) {
			return true
		}
	}
	if strings.Contains(text, "timeout") || strings.Contains(text, "context deadline exceeded") {
		return true
	}
	if strings.Contains(text, "connection reset") || strings.Contains(text, "connection refused") {
		return true
	}
	return false
}
