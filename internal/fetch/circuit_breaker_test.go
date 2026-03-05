// Package fetch provides tests for circuit breaker functionality.
// Tests cover state transitions, request blocking, and concurrent safety.
package fetch

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreakerState_String(t *testing.T) {
	tests := []struct {
		state CircuitBreakerState
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitBreakerState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("State.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.FailureThreshold != 5 {
		t.Errorf("Expected FailureThreshold = 5, got %d", cfg.FailureThreshold)
	}
	if cfg.SuccessThreshold != 3 {
		t.Errorf("Expected SuccessThreshold = 3, got %d", cfg.SuccessThreshold)
	}
	if cfg.ResetTimeout != 30*time.Second {
		t.Errorf("Expected ResetTimeout = 30s, got %v", cfg.ResetTimeout)
	}
	if cfg.HalfOpenMaxRequests != 3 {
		t.Errorf("Expected HalfOpenMaxRequests = 3, got %d", cfg.HalfOpenMaxRequests)
	}
}

func TestNewCircuitBreaker_Defaults(t *testing.T) {
	// Test with zero values - should apply defaults
	cfg := CircuitBreakerConfig{Enabled: true}
	cb := NewCircuitBreaker(cfg)

	if cb.config.FailureThreshold != 5 {
		t.Errorf("Expected default FailureThreshold = 5, got %d", cb.config.FailureThreshold)
	}
	if cb.config.SuccessThreshold != 3 {
		t.Errorf("Expected default SuccessThreshold = 3, got %d", cb.config.SuccessThreshold)
	}
	if cb.config.ResetTimeout != 30*time.Second {
		t.Errorf("Expected default ResetTimeout = 30s, got %v", cb.config.ResetTimeout)
	}
	if cb.config.HalfOpenMaxRequests != 3 {
		t.Errorf("Expected default HalfOpenMaxRequests = 3, got %d", cb.config.HalfOpenMaxRequests)
	}
}

func TestCircuitBreaker_StateTransitions_ClosedToOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	// Initial state should be Closed
	if state := cb.GetState("test"); state != StateClosed {
		t.Errorf("Initial state = %v, want StateClosed", state)
	}

	// Record failures up to threshold
	cb.RecordFailure("test")
	if state := cb.GetState("test"); state != StateClosed {
		t.Errorf("After 1 failure, state = %v, want StateClosed", state)
	}

	cb.RecordFailure("test")
	if state := cb.GetState("test"); state != StateClosed {
		t.Errorf("After 2 failures, state = %v, want StateClosed", state)
	}

	// Third failure should open the circuit
	cb.RecordFailure("test")
	if state := cb.GetState("test"); state != StateOpen {
		t.Errorf("After 3 failures, state = %v, want StateOpen", state)
	}
}

func TestCircuitBreaker_Allow_BlocksWhenOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		SuccessThreshold: 1,
		ResetTimeout:     1 * time.Hour, // Long timeout to stay open
	}
	cb := NewCircuitBreaker(cfg)

	// Initially should allow
	if !cb.Allow("test") {
		t.Error("Expected Allow() = true for new host")
	}

	// Open the circuit
	cb.RecordFailure("test")
	cb.RecordFailure("test")

	// Should block when open
	if cb.Allow("test") {
		t.Error("Expected Allow() = false when circuit is open")
	}
	if cb.Allow("test") {
		t.Error("Expected Allow() = false when circuit is open (second check)")
	}
}

func TestCircuitBreaker_StateTransitions_OpenToHalfOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:             true,
		FailureThreshold:    2,
		SuccessThreshold:    2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxRequests: 2,
	}
	cb := NewCircuitBreaker(cfg)

	// Open the circuit
	cb.RecordFailure("test")
	cb.RecordFailure("test")

	if state := cb.GetState("test"); state != StateOpen {
		t.Errorf("Expected StateOpen, got %v", state)
	}

	// Wait for reset timeout
	time.Sleep(75 * time.Millisecond)

	// Allow() should now transition to half-open and allow the request
	if !cb.Allow("test") {
		t.Error("Expected Allow() = true for first half-open request")
	}

	// GetState should now return HalfOpen
	if state := cb.GetState("test"); state != StateHalfOpen {
		t.Errorf("After Allow() in half-open, expected StateHalfOpen, got %v", state)
	}

	// Second request should also be allowed
	if !cb.Allow("test") {
		t.Error("Expected Allow() = true for second half-open request")
	}

	// Third request should be blocked (exceeds HalfOpenMaxRequests)
	if cb.Allow("test") {
		t.Error("Expected Allow() = false when half-open max requests reached")
	}
}

func TestCircuitBreaker_StateTransitions_HalfOpenToClosed(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:             true,
		FailureThreshold:    2,
		SuccessThreshold:    2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxRequests: 3,
	}
	cb := NewCircuitBreaker(cfg)

	// Open the circuit
	cb.RecordFailure("test")
	cb.RecordFailure("test")

	// Wait for reset timeout to enter half-open
	time.Sleep(75 * time.Millisecond)

	// Record successes to close the circuit
	cb.RecordSuccess("test")
	if state := cb.GetState("test"); state != StateHalfOpen {
		t.Errorf("After 1 success in half-open, expected StateHalfOpen, got %v", state)
	}

	cb.RecordSuccess("test")
	if state := cb.GetState("test"); state != StateClosed {
		t.Errorf("After 2 successes in half-open, expected StateClosed, got %v", state)
	}

	// Should allow all requests when closed
	if !cb.Allow("test") {
		t.Error("Expected Allow() = true when circuit is closed")
	}
}

func TestCircuitBreaker_StateTransitions_HalfOpenToOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:             true,
		FailureThreshold:    2,
		SuccessThreshold:    3,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxRequests: 3,
	}
	cb := NewCircuitBreaker(cfg)

	// Open the circuit
	cb.RecordFailure("test")
	cb.RecordFailure("test")

	// Wait for reset timeout
	time.Sleep(75 * time.Millisecond)

	// Record a failure in half-open - should go back to open
	cb.RecordFailure("test")
	if state := cb.GetState("test"); state != StateOpen {
		t.Errorf("After failure in half-open, expected StateOpen, got %v", state)
	}

	// Should block again
	if cb.Allow("test") {
		t.Error("Expected Allow() = false after half-open failure")
	}
}

func TestCircuitBreaker_RecordSuccess_ResetsFailureCount(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
	}
	cb := NewCircuitBreaker(cfg)

	// Record some failures but not enough to open
	cb.RecordFailure("test")
	cb.RecordFailure("test")

	// Record a success - should reset failure count
	cb.RecordSuccess("test")

	// Need 3 more failures to open now
	cb.RecordFailure("test")
	cb.RecordFailure("test")
	if state := cb.GetState("test"); state != StateClosed {
		t.Errorf("Expected StateClosed after 2 more failures, got %v", state)
	}

	// Third failure should open
	cb.RecordFailure("test")
	if state := cb.GetState("test"); state != StateOpen {
		t.Errorf("Expected StateOpen after 3rd failure, got %v", state)
	}
}

func TestCircuitBreaker_Disabled(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          false,
		FailureThreshold: 1,
	}
	cb := NewCircuitBreaker(cfg)

	// Should always allow when disabled
	cb.RecordFailure("test")
	cb.RecordFailure("test")
	cb.RecordFailure("test")

	if !cb.Allow("test") {
		t.Error("Expected Allow() = true when circuit breaker is disabled")
	}

	if state := cb.GetState("test"); state != StateClosed {
		t.Errorf("Expected StateClosed when disabled, got %v", state)
	}
}

func TestCircuitBreaker_Nil(t *testing.T) {
	var cb *CircuitBreaker

	// Should not panic and should allow
	if !cb.Allow("test") {
		t.Error("Expected Allow() = true for nil circuit breaker")
	}

	// Should not panic
	cb.RecordSuccess("test")
	cb.RecordFailure("test")

	if state := cb.GetState("test"); state != StateClosed {
		t.Errorf("Expected StateClosed for nil CB, got %v", state)
	}

	if status := cb.GetHostStatus(); status != nil {
		t.Error("Expected GetHostStatus() = nil for nil CB")
	}

	if cb.IsEnabled() {
		t.Error("Expected IsEnabled() = false for nil CB")
	}
}

func TestCircuitBreaker_MultipleHosts(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
	}
	cb := NewCircuitBreaker(cfg)

	// Open circuit for host1
	cb.RecordFailure("host1")
	cb.RecordFailure("host1")

	// host1 should be blocked
	if cb.Allow("host1") {
		t.Error("Expected Allow() = false for host1")
	}

	// host2 should still be allowed
	if !cb.Allow("host2") {
		t.Error("Expected Allow() = true for host2")
	}

	// Record failures for host2 too
	cb.RecordFailure("host2")
	cb.RecordFailure("host2")

	if cb.Allow("host2") {
		t.Error("Expected Allow() = false for host2 after failures")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:             true,
		FailureThreshold:    100,
		SuccessThreshold:    50,
		ResetTimeout:        10 * time.Millisecond,
		HalfOpenMaxRequests: 10,
	}
	cb := NewCircuitBreaker(cfg)

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent Allow calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cb.Allow("concurrent-test")
			}
		}()
	}

	// Concurrent RecordSuccess calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cb.RecordSuccess("concurrent-test")
			}
		}()
	}

	// Concurrent RecordFailure calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cb.RecordFailure("concurrent-test")
			}
		}()
	}

	// Concurrent GetState calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cb.GetState("concurrent-test")
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no panic or deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for concurrent operations")
	}
}

func TestCircuitBreaker_GetHostStatus(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		ResetTimeout:     1 * time.Hour,
	}
	cb := NewCircuitBreaker(cfg)

	// No hosts yet
	status := cb.GetHostStatus()
	if len(status) != 0 {
		t.Errorf("Expected 0 hosts initially, got %d", len(status))
	}

	// Create some state
	cb.RecordFailure("host1")
	cb.RecordFailure("host1")
	cb.RecordSuccess("host2")

	status = cb.GetHostStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(status))
	}

	// Find host1 and host2 in status
	var host1Found, host2Found bool
	for _, s := range status {
		if s.Host == "host1" {
			host1Found = true
			if s.FailureCount != 2 {
				t.Errorf("Expected host1 FailureCount = 2, got %d", s.FailureCount)
			}
		}
		if s.Host == "host2" {
			host2Found = true
			if s.SuccessCount != 1 {
				t.Errorf("Expected host2 SuccessCount = 1, got %d", s.SuccessCount)
			}
		}
	}

	if !host1Found {
		t.Error("host1 not found in status")
	}
	if !host2Found {
		t.Error("host2 not found in status")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
	}
	cb := NewCircuitBreaker(cfg)

	// Open circuits for multiple hosts
	cb.RecordFailure("host1")
	cb.RecordFailure("host1")
	cb.RecordFailure("host2")
	cb.RecordFailure("host2")

	// Verify they're open
	if cb.Allow("host1") {
		t.Error("Expected host1 to be blocked")
	}
	if cb.Allow("host2") {
		t.Error("Expected host2 to be blocked")
	}

	// Reset specific host
	cb.Reset("host1")

	// host1 should be allowed now
	if !cb.Allow("host1") {
		t.Error("Expected host1 to be allowed after reset")
	}

	// host2 should still be blocked
	if cb.Allow("host2") {
		t.Error("Expected host2 to still be blocked")
	}

	// Reset all hosts
	cb.Reset("")

	// host2 should now be allowed
	if !cb.Allow("host2") {
		t.Error("Expected host2 to be allowed after reset all")
	}
}

func TestCircuitBreaker_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cb       *CircuitBreaker
		expected bool
	}{
		{
			name:     "nil circuit breaker",
			cb:       nil,
			expected: false,
		},
		{
			name:     "disabled circuit breaker",
			cb:       NewCircuitBreaker(CircuitBreakerConfig{Enabled: false}),
			expected: false,
		},
		{
			name:     "enabled circuit breaker",
			cb:       NewCircuitBreaker(CircuitBreakerConfig{Enabled: true}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cb.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCircuitBreakerHostStatus_String(t *testing.T) {
	status := CircuitBreakerHostStatus{
		Host:             "example.com",
		State:            "closed",
		FailureCount:     5,
		SuccessCount:     3,
		LastFailureTime:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		HalfOpenRequests: 1,
	}

	got := status.String()
	expected := "example.com: state=closed failures=5 successes=3 last_fail=2024-01-15T10:30:00Z"
	if got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestCircuitBreaker_GetConfig(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 10,
		SuccessThreshold: 5,
		ResetTimeout:     60 * time.Second,
	}
	cb := NewCircuitBreaker(cfg)

	got := cb.GetConfig()

	if got.Enabled != cfg.Enabled {
		t.Errorf("GetConfig().Enabled = %v, want %v", got.Enabled, cfg.Enabled)
	}
	if got.FailureThreshold != cfg.FailureThreshold {
		t.Errorf("GetConfig().FailureThreshold = %d, want %d", got.FailureThreshold, cfg.FailureThreshold)
	}
	if got.SuccessThreshold != cfg.SuccessThreshold {
		t.Errorf("GetConfig().SuccessThreshold = %d, want %d", got.SuccessThreshold, cfg.SuccessThreshold)
	}
	if got.ResetTimeout != cfg.ResetTimeout {
		t.Errorf("GetConfig().ResetTimeout = %v, want %v", got.ResetTimeout, cfg.ResetTimeout)
	}

	// Verify it's a copy - modifying returned config shouldn't affect original
	got.Enabled = false
	if !cb.config.Enabled {
		t.Error("Modifying returned config affected original")
	}
}

func TestCircuitBreaker_Allow_CreatesHostEntry(t *testing.T) {
	cfg := CircuitBreakerConfig{Enabled: true}
	cb := NewCircuitBreaker(cfg)

	// Allow should create host entry for new hosts
	if !cb.Allow("new-host") {
		t.Error("Expected Allow() = true for new host")
	}

	// Check that host was created
	if state := cb.GetState("new-host"); state != StateClosed {
		t.Errorf("Expected new host to be in StateClosed, got %v", state)
	}
}

func TestCircuitBreaker_RecordFailure_FirstFailure(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
	}
	cb := NewCircuitBreaker(cfg)

	// Record failure for new host should create entry
	cb.RecordFailure("new-host")

	status := cb.GetHostStatus()
	if len(status) != 1 {
		t.Fatalf("Expected 1 host in status, got %d", len(status))
	}

	if status[0].FailureCount != 1 {
		t.Errorf("Expected FailureCount = 1, got %d", status[0].FailureCount)
	}
}

func TestCircuitBreaker_OpenState_UpdatesLastFailureTime(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		ResetTimeout:     1 * time.Hour,
	}
	cb := NewCircuitBreaker(cfg)

	// Open the circuit
	cb.RecordFailure("test")
	cb.RecordFailure("test")

	// Get initial last failure time
	cb.mu.RLock()
	initialLastFail := cb.hosts["test"].lastFailureTime
	cb.mu.RUnlock()

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Record another failure while open - should update lastFailureTime
	cb.RecordFailure("test")

	cb.mu.RLock()
	newLastFail := cb.hosts["test"].lastFailureTime
	cb.mu.RUnlock()

	if !newLastFail.After(initialLastFail) {
		t.Error("Expected lastFailureTime to be updated while in open state")
	}
}
