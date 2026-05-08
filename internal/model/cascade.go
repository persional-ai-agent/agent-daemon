package model

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type ProviderEntry struct {
	Client  Client
	Name    string
	Cost    float64
	Circuit *ProviderCircuit
}

type CascadeClient struct {
	Providers   []ProviderEntry
	CostAware   bool
	IncludeCodes []int
}

func NewCascadeClient(providers []ProviderEntry, costAware bool) *CascadeClient {
	return &CascadeClient{
		Providers:    providers,
		CostAware:    costAware,
		IncludeCodes: []int{408, 429, 500, 502, 503, 504},
	}
}

func NewCascadeClientWithCircuit(providers []ProviderEntry, costAware bool, circuitThreshold int, circuitRecovery time.Duration, circuitHalfOpenMax int) *CascadeClient {
	for i := range providers {
		if providers[i].Circuit == nil {
			providers[i].Circuit = NewProviderCircuit(circuitThreshold, circuitRecovery, circuitHalfOpenMax)
		}
	}
	return NewCascadeClient(providers, costAware)
}

func (c *CascadeClient) ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error) {
	return c.ChatCompletionWithEvents(ctx, messages, tools, nil)
}

func (c *CascadeClient) ChatCompletionWithEvents(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	ordered := c.buildOrder()

	var lastErr error
	for _, pe := range ordered {
		if pe.Circuit != nil && !pe.Circuit.AllowRequest() {
			c.emitCircuitEvent(sink, pe.Name, pe.Circuit.State().String())
			if lastErr == nil {
				lastErr = fmt.Errorf("provider %s circuit breaker open", pe.Name)
			}
			continue
		}

		if pe.Circuit != nil && pe.Circuit.State() == CircuitHalfOpen {
			pe.Circuit.IncrementHalfOpenRequests()
		}

		msg, err := CompleteWithEvents(ctx, pe.Client, messages, tools, sink)
		if pe.Circuit != nil {
			if err == nil {
				pe.Circuit.RecordSuccess()
			} else {
				pe.Circuit.RecordFailure()
			}
		}

		if err == nil {
			return msg, nil
		}

		lastErr = err
		if !shouldFallbackOnError(err, c.IncludeCodes) {
			return core.Message{}, err
		}
	}

	if lastErr != nil {
		return core.Message{}, lastErr
	}
	return core.Message{}, fmt.Errorf("no providers available")
}

func (c *CascadeClient) buildOrder() []ProviderEntry {
	if !c.CostAware {
		return c.Providers
	}
	ordered := make([]ProviderEntry, len(c.Providers))
	copy(ordered, c.Providers)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Cost < ordered[j].Cost
	})
	return ordered
}

func (c *CascadeClient) emitCircuitEvent(sink StreamEventSink, providerName, state string) {
	if sink == nil {
		return
	}
	t := time.Now().UTC().Format(time.RFC3339)
	sink(StreamEvent{
		Provider: providerName,
		Type:     "circuit_state",
		Data: map[string]any{
			"provider": providerName,
			"state":    state,
			"reason":   "circuit breaker " + state,
			"time":     t,
		},
	})
}

func ParseCascadeProviders(spec string, builder func(string) (Client, string, error)) ([]ProviderEntry, error) {
	parts := strings.Split(spec, ",")
	var entries []ProviderEntry
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		providerName := part
		cost := 1.0
		if i := strings.Index(part, ":"); i >= 0 {
			providerName = strings.TrimSpace(part[:i])
			costStr := strings.TrimSpace(part[i+1:])
			fmt.Sscanf(costStr, "%f", &cost)
		}
		client, name, err := builder(providerName)
		if err != nil {
			return nil, fmt.Errorf("provider %q: %w", providerName, err)
		}
		entries = append(entries, ProviderEntry{
			Client: client,
			Name:   name,
			Cost:   cost,
		})
	}
	return entries, nil
}
