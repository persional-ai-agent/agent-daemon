package model

import (
	"sync"
	"time"
)

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type ProviderCircuit struct {
	mu              sync.RWMutex
	state           CircuitState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	lastStateChange time.Time

	threshold       int
	recoveryTimeout time.Duration
	halfOpenMaxReqs int
	halfOpenReqs    int
}

func NewProviderCircuit(threshold int, recoveryTimeout time.Duration, halfOpenMaxReqs int) *ProviderCircuit {
	if threshold <= 0 {
		threshold = 3
	}
	if recoveryTimeout <= 0 {
		recoveryTimeout = 60 * time.Second
	}
	if halfOpenMaxReqs <= 0 {
		halfOpenMaxReqs = 1
	}
	return &ProviderCircuit{
		state:           CircuitClosed,
		threshold:       threshold,
		recoveryTimeout: recoveryTimeout,
		halfOpenMaxReqs: halfOpenMaxReqs,
		lastStateChange: time.Now(),
	}
}

func (c *ProviderCircuit) AllowRequest() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(c.lastStateChange) >= c.recoveryTimeout {
			c.state = CircuitHalfOpen
			c.halfOpenReqs = 0
			c.lastStateChange = time.Now()
			return true
		}
		return false
	case CircuitHalfOpen:
		return c.halfOpenReqs < c.halfOpenMaxReqs
	default:
		return true
	}
}

func (c *ProviderCircuit) RecordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.successCount++
	switch c.state {
	case CircuitHalfOpen:
		c.state = CircuitClosed
		c.failureCount = 0
		c.halfOpenReqs = 0
		c.lastStateChange = time.Now()
	case CircuitClosed:
		c.failureCount = 0
	}
}

func (c *ProviderCircuit) RecordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failureCount++
	c.lastFailureTime = time.Now()

	switch c.state {
	case CircuitHalfOpen:
		c.state = CircuitOpen
		c.halfOpenReqs = 0
		c.lastStateChange = time.Now()
	case CircuitClosed:
		if c.failureCount >= c.threshold {
			c.state = CircuitOpen
			c.lastStateChange = time.Now()
		}
	}
}

func (c *ProviderCircuit) State() CircuitState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.state == CircuitOpen && time.Since(c.lastStateChange) >= c.recoveryTimeout {
		return CircuitHalfOpen
	}
	return c.state
}

func (c *ProviderCircuit) FailureCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.failureCount
}

func (c *ProviderCircuit) TransitionToHalfOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == CircuitOpen && time.Since(c.lastStateChange) >= c.recoveryTimeout {
		c.state = CircuitHalfOpen
		c.halfOpenReqs = 0
		c.lastStateChange = time.Now()
		return true
	}
	return false
}

func (c *ProviderCircuit) IncrementHalfOpenRequests() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == CircuitHalfOpen {
		c.halfOpenReqs++
	}
}
