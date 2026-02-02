// Package webhook provides tests for dispatcher configuration and initialization.
//
// Tests cover:
// - Default configuration values
// - Custom configuration values
// - MaxConcurrentDispatches validation (zero, negative, positive values)
//
// Does NOT test:
// - Actual dispatch behavior (see dispatcher_success_test.go, dispatcher_retry_test.go)
// - Concurrency behavior under load (see dispatcher_concurrency_test.go)
//
// Assumes:
// - NewDispatcher uses sensible defaults for zero-valued Config fields
// - Negative or zero MaxConcurrentDispatches falls back to default (100)
package webhook

import (
	"testing"
	"time"
)

func TestNewDispatcher_Defaults(t *testing.T) {
	d := NewDispatcher(Config{})

	if d.maxRetries != 3 {
		t.Errorf("expected maxRetries=3, got %d", d.maxRetries)
	}
	if d.baseDelay != 1*time.Second {
		t.Errorf("expected baseDelay=1s, got %v", d.baseDelay)
	}
	if d.maxDelay != 30*time.Second {
		t.Errorf("expected maxDelay=30s, got %v", d.maxDelay)
	}
	if d.timeout != 30*time.Second {
		t.Errorf("expected timeout=30s, got %v", d.timeout)
	}
	if d.allowInternal {
		t.Error("expected allowInternal to be false by default")
	}
}

func TestNewDispatcher_CustomValues(t *testing.T) {
	cfg := Config{
		Secret:        "test-secret",
		MaxRetries:    5,
		BaseDelay:     500 * time.Millisecond,
		MaxDelay:      60 * time.Second,
		Timeout:       10 * time.Second,
		AllowInternal: true,
	}
	d := NewDispatcher(cfg)

	if d.maxRetries != 5 {
		t.Errorf("expected maxRetries=5, got %d", d.maxRetries)
	}
	if d.baseDelay != 500*time.Millisecond {
		t.Errorf("expected baseDelay=500ms, got %v", d.baseDelay)
	}
	if d.maxDelay != 60*time.Second {
		t.Errorf("expected maxDelay=60s, got %v", d.maxDelay)
	}
	if d.timeout != 10*time.Second {
		t.Errorf("expected timeout=10s, got %v", d.timeout)
	}
	if d.secret != "test-secret" {
		t.Errorf("expected secret='test-secret', got %q", d.secret)
	}
	if !d.allowInternal {
		t.Error("expected allowInternal to be true")
	}
}

func TestNewDispatcher_DefaultMaxConcurrent(t *testing.T) {
	d := NewDispatcher(Config{})

	// Default should be 100
	if cap(d.sem) != 100 {
		t.Errorf("expected default maxConcurrent=100, got %d", cap(d.sem))
	}
}

func TestNewDispatcher_CustomMaxConcurrent(t *testing.T) {
	d := NewDispatcher(Config{MaxConcurrentDispatches: 50})

	if cap(d.sem) != 50 {
		t.Errorf("expected maxConcurrent=50, got %d", cap(d.sem))
	}
}

func TestNewDispatcher_ZeroMaxConcurrentUsesDefault(t *testing.T) {
	d := NewDispatcher(Config{MaxConcurrentDispatches: 0})

	// Zero should use default of 100
	if cap(d.sem) != 100 {
		t.Errorf("expected default maxConcurrent=100 when 0, got %d", cap(d.sem))
	}
}

func TestNewDispatcher_NegativeMaxConcurrentUsesDefault(t *testing.T) {
	d := NewDispatcher(Config{MaxConcurrentDispatches: -5})

	// Negative should use default of 100
	if cap(d.sem) != 100 {
		t.Errorf("expected default maxConcurrent=100 when negative, got %d", cap(d.sem))
	}
}
