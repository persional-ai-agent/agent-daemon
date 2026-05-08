package model

import (
	"testing"
	"time"
)

func TestCircuitInitialState(t *testing.T) {
	c := NewProviderCircuit(3, 60*time.Second, 1)
	if c.State() != CircuitClosed {
		t.Fatalf("expected closed, got %v", c.State())
	}
	if !c.AllowRequest() {
		t.Fatal("expected to allow request in closed state")
	}
}

func TestCircuitOpensAfterThreshold(t *testing.T) {
	c := NewProviderCircuit(3, 60*time.Second, 1)
	for i := 0; i < 3; i++ {
		c.RecordFailure()
	}
	if c.State() != CircuitOpen {
		t.Fatalf("expected open after 3 failures, got %v", c.State())
	}
	if c.AllowRequest() {
		t.Fatal("expected to reject request in open state")
	}
}

func TestCircuitHalfOpenAfterTimeout(t *testing.T) {
	c := NewProviderCircuit(3, 100*time.Millisecond, 1)
	for i := 0; i < 3; i++ {
		c.RecordFailure()
	}
	time.Sleep(150 * time.Millisecond)
	if c.State() != CircuitHalfOpen {
		t.Fatalf("expected half-open after timeout, got %v", c.State())
	}
	if !c.AllowRequest() {
		t.Fatal("expected to allow request in half-open state")
	}
}

func TestCircuitClosesOnSuccessInHalfOpen(t *testing.T) {
	c := NewProviderCircuit(3, 100*time.Millisecond, 1)
	for i := 0; i < 3; i++ {
		c.RecordFailure()
	}
	time.Sleep(150 * time.Millisecond)
	c.AllowRequest()
	c.RecordSuccess()
	if c.State() != CircuitClosed {
		t.Fatalf("expected closed after success in half-open, got %v", c.State())
	}
	if c.FailureCount() != 0 {
		t.Fatal("expected failure count reset")
	}
}

func TestCircuitStaysOpenOnFailureInHalfOpen(t *testing.T) {
	c := NewProviderCircuit(3, 100*time.Millisecond, 1)
	for i := 0; i < 3; i++ {
		c.RecordFailure()
	}
	time.Sleep(150 * time.Millisecond)
	c.AllowRequest()
	c.RecordFailure()
	if c.State() != CircuitOpen {
		t.Fatalf("expected open after failure in half-open, got %v", c.State())
	}
}

func TestCircuitResetsFailureCountOnSuccess(t *testing.T) {
	c := NewProviderCircuit(3, 60*time.Second, 1)
	c.RecordFailure()
	c.RecordFailure()
	c.RecordSuccess()
	if c.FailureCount() != 0 {
		t.Fatal("expected failure count reset on success")
	}
}

func TestCircuitHalfOpenMaxRequests(t *testing.T) {
	c := NewProviderCircuit(3, 100*time.Millisecond, 2)
	for i := 0; i < 3; i++ {
		c.RecordFailure()
	}
	time.Sleep(150 * time.Millisecond)
	c.AllowRequest()
	c.IncrementHalfOpenRequests()
	c.IncrementHalfOpenRequests()
	if c.AllowRequest() {
		t.Fatal("expected to reject after max half-open requests")
	}
}
