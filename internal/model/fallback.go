package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/core"
)

type FallbackClient struct {
	Primary            Client
	PrimaryName        string
	Fallback           Client
	FallbackName       string
	IncludeStatusCodes []int
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

func (c *FallbackClient) ChatCompletion(ctx context.Context, messages []core.Message, tools []core.ToolSchema) (core.Message, error) {
	return c.ChatCompletionWithEvents(ctx, messages, tools, nil)
}

func (c *FallbackClient) ChatCompletionWithEvents(ctx context.Context, messages []core.Message, tools []core.ToolSchema, sink StreamEventSink) (core.Message, error) {
	if c == nil || c.Primary == nil {
		return core.Message{}, fmt.Errorf("primary model client unavailable")
	}
	msg, err := CompleteWithEvents(ctx, c.Primary, messages, tools, sink)
	if err == nil {
		return msg, nil
	}
	if c.Fallback == nil || !shouldFallbackOnError(err, c.IncludeStatusCodes) {
		return core.Message{}, err
	}
	fallbackMsg, fallbackErr := CompleteWithEvents(ctx, c.Fallback, messages, tools, sink)
	if fallbackErr != nil {
		return core.Message{}, fmt.Errorf("primary (%s) failed: %v; fallback (%s) failed: %v", c.PrimaryName, err, c.FallbackName, fallbackErr)
	}
	return fallbackMsg, nil
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
