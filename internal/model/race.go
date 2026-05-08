package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type RaceClient struct {
	Primary         Client
	PrimaryName     string
	Fallback        Client
	FallbackName    string
	PrimaryCircuit  *ProviderCircuit
	FallbackCircuit *ProviderCircuit
}

func NewRaceClient(primary Client, primaryName string, fallback Client, fallbackName string, circuitThreshold int, circuitRecoveryTimeout time.Duration, circuitHalfOpenMaxReqs int) *RaceClient {
	return &RaceClient{
		Primary:         primary,
		PrimaryName:     strings.TrimSpace(primaryName),
		Fallback:        fallback,
		FallbackName:    strings.TrimSpace(fallbackName),
		PrimaryCircuit:  NewProviderCircuit(circuitThreshold, circuitRecoveryTimeout, circuitHalfOpenMaxReqs),
		FallbackCircuit: NewProviderCircuit(circuitThreshold, circuitRecoveryTimeout, circuitHalfOpenMaxReqs),
	}
}

func (c *RaceClient) ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error) {
	return c.ChatCompletionWithEvents(ctx, messages, tools, nil)
}

func (c *RaceClient) ChatCompletionWithEvents(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	if c == nil || c.Primary == nil {
		return core.Message{}, fmt.Errorf("primary model client unavailable")
	}

	type providerCall struct {
		name    string
		client  Client
		circuit *ProviderCircuit
	}

	var candidates []providerCall

	primaryAllowed := c.PrimaryCircuit == nil || c.PrimaryCircuit.AllowRequest()
	fallbackAllowed := c.Fallback != nil && (c.FallbackCircuit == nil || c.FallbackCircuit.AllowRequest())

	if primaryAllowed {
		candidates = append(candidates, providerCall{c.PrimaryName, c.Primary, c.PrimaryCircuit})
	}
	if fallbackAllowed {
		candidates = append(candidates, providerCall{c.FallbackName, c.Fallback, c.FallbackCircuit})
	}

	if len(candidates) == 0 {
		return core.Message{}, fmt.Errorf("all provider circuit breakers open")
	}

	if len(candidates) == 1 {
		return c.callWithCircuit(ctx, candidates[0].client, candidates[0].name, candidates[0].circuit, messages, tools, sink)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		msg     core.Message
		err     error
		name    string
		circuit *ProviderCircuit
	}

	resultCh := make(chan result, len(candidates))

	for _, pc := range candidates {
		go func(pc providerCall) {
			if pc.circuit != nil && pc.circuit.State() == CircuitHalfOpen {
				pc.circuit.IncrementHalfOpenRequests()
			}
			msg, err := CompleteWithEvents(ctx, pc.client, messages, tools, sink)
			if pc.circuit != nil {
				if err == nil {
					pc.circuit.RecordSuccess()
				} else {
					pc.circuit.RecordFailure()
				}
			}
			resultCh <- result{msg, err, pc.name, pc.circuit}
		}(pc)
	}

	var firstErr error
	for i := 0; i < len(candidates); i++ {
		r := <-resultCh
		if r.err == nil {
			cancel()
			return r.msg, nil
		}
		if firstErr == nil {
			firstErr = r.err
		}
	}

	return core.Message{}, fmt.Errorf("all providers failed: %v", firstErr)
}

func (c *RaceClient) callWithCircuit(ctx context.Context, client Client, name string, circuit *ProviderCircuit, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
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
