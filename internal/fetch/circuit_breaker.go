// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"fmt"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState int

const (
	// StateClosed is the normal operating state where requests are allowed.
	StateClosed CircuitBreakerState = iota
	// StateOpen means the failure threshold was reached; requests are blocked.
	StateOpen
	// StateHalfOpen is a testing state to check if the service has recovered.
	StateHalfOpen
)

// String returns the string representation of the circuit breaker state.
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	Enabled             bool          // Whether circuit breaker is enabled
	FailureThreshold    int           // Failures before opening circuit (default: 5)
	SuccessThreshold    int           // Successes in half-open to close (default: 3)
	ResetTimeout        time.Duration // Time before attempting half-open (default: 30s)
	HalfOpenMaxRequests int           // Max requests in half-open state (default: 3)
}

// DefaultCircuitBreakerConfig returns a CircuitBreakerConfig with sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:             true,
		FailureThreshold:    5,
		SuccessThreshold:    3,
		ResetTimeout:        30 * time.Second,
		HalfOpenMaxRequests: 3,
	}
}

// CircuitBreakerHostStatus represents the current state of a circuit breaker for a host.
type CircuitBreakerHostStatus struct {
	Host             string
	State            string
	FailureCount     int
	SuccessCount     int
	LastFailureTime  time.Time
	HalfOpenRequests int
}

// circuitBreakerHost tracks the state for a single host.
type circuitBreakerHost struct {
	state            CircuitBreakerState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	halfOpenRequests int
	lastStateChange  time.Time
}

// CircuitBreaker tracks failure state per host and implements the circuit breaker pattern.
// It is safe for concurrent use by multiple goroutines.
type CircuitBreaker struct {
	config CircuitBreakerConfig
	mu     sync.RWMutex
	hosts  map[string]*circuitBreakerHost
}

// ErrCircuitBreakerOpen is returned when the circuit breaker is open and requests are blocked.
// This maps to HTTP 503 Service Unavailable.
var ErrCircuitBreakerOpen = apperrors.New(apperrors.KindInternal, "circuit breaker is open")

// NewCircuitBreaker creates a new CircuitBreaker with the given configuration.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	// Apply defaults for zero values
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 3
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = 30 * time.Second
	}
	if cfg.HalfOpenMaxRequests <= 0 {
		cfg.HalfOpenMaxRequests = 3
	}

	return &CircuitBreaker{
		config: cfg,
		hosts:  make(map[string]*circuitBreakerHost),
	}
}

// Allow checks if a request to the given host should be allowed.
// Returns true if the request can proceed, false if it should be blocked.
func (cb *CircuitBreaker) Allow(host string) bool {
	if cb == nil || !cb.config.Enabled {
		return true
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	h, exists := cb.hosts[host]
	if !exists {
		// New host, create entry and allow
		cb.hosts[host] = &circuitBreakerHost{
			state:           StateClosed,
			lastStateChange: time.Now(),
		}
		return true
	}

	switch h.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if enough time has passed to transition to half-open
		if time.Since(h.lastFailureTime) >= cb.config.ResetTimeout {
			h.state = StateHalfOpen
			h.successCount = 0
			h.halfOpenRequests = 1
			h.lastStateChange = time.Now()
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited number of requests in half-open state
		// halfOpenRequests was set to 1 when transitioning from Open
		if h.halfOpenRequests < cb.config.HalfOpenMaxRequests {
			h.halfOpenRequests++
			return true
		}
		return false

	default:
		return true
	}
}

// RecordSuccess records a successful request to the given host.
// This may transition the circuit breaker from half-open to closed.
func (cb *CircuitBreaker) RecordSuccess(host string) {
	if cb == nil || !cb.config.Enabled {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	h, exists := cb.hosts[host]
	if !exists {
		// Create entry for new host with success state
		cb.hosts[host] = &circuitBreakerHost{
			state:           StateClosed,
			successCount:    1,
			lastStateChange: time.Now(),
		}
		return
	}

	switch h.state {
	case StateClosed:
		// Reset failure count on success in closed state
		h.failureCount = 0

	case StateHalfOpen:
		h.successCount++
		// Transition to closed if success threshold reached
		if h.successCount >= cb.config.SuccessThreshold {
			h.state = StateClosed
			h.failureCount = 0
			h.successCount = 0
			h.halfOpenRequests = 0
			h.lastStateChange = time.Now()
		}

	case StateOpen:
		// Check if timeout has elapsed - if so, treat as half-open
		if time.Since(h.lastFailureTime) >= cb.config.ResetTimeout {
			h.state = StateHalfOpen
			h.successCount = 1
			h.halfOpenRequests = 0
			// Check if we should immediately transition to closed
			if h.successCount >= cb.config.SuccessThreshold {
				h.state = StateClosed
				h.failureCount = 0
				h.successCount = 0
				h.halfOpenRequests = 0
			}
			h.lastStateChange = time.Now()
		}
		// Otherwise, stay in open state and don't record success
	}
}

// RecordFailure records a failed request to the given host.
// This may transition the circuit breaker from closed to open, or half-open to open.
func (cb *CircuitBreaker) RecordFailure(host string) {
	if cb == nil || !cb.config.Enabled {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	h, exists := cb.hosts[host]
	if !exists {
		// First failure for this host
		cb.hosts[host] = &circuitBreakerHost{
			state:           StateClosed,
			failureCount:    1,
			lastFailureTime: time.Now(),
			lastStateChange: time.Now(),
		}
		return
	}

	switch h.state {
	case StateClosed:
		h.failureCount++
		h.lastFailureTime = time.Now()
		// Transition to open if failure threshold reached
		if h.failureCount >= cb.config.FailureThreshold {
			h.state = StateOpen
			h.lastStateChange = time.Now()
		}

	case StateHalfOpen:
		// Any failure in half-open goes back to open
		h.state = StateOpen
		h.failureCount++
		h.lastFailureTime = time.Now()
		h.successCount = 0
		h.halfOpenRequests = 0
		h.lastStateChange = time.Now()

	case StateOpen:
		// Update last failure time to extend the reset timeout
		h.failureCount++
		h.lastFailureTime = time.Now()
	}
}

// GetState returns the current circuit breaker state for the given host.
func (cb *CircuitBreaker) GetState(host string) CircuitBreakerState {
	if cb == nil || !cb.config.Enabled {
		return StateClosed
	}

	cb.mu.RLock()
	defer cb.mu.RUnlock()

	h, exists := cb.hosts[host]
	if !exists {
		return StateClosed
	}

	// Check if we need to transition from Open to Half-Open
	if h.state == StateOpen && time.Since(h.lastFailureTime) >= cb.config.ResetTimeout {
		// Don't modify state here, let Allow() handle the transition
		return StateHalfOpen
	}

	return h.state
}

// GetHostStatus returns circuit breaker status for all known hosts.
func (cb *CircuitBreaker) GetHostStatus() []CircuitBreakerHostStatus {
	if cb == nil {
		return nil
	}

	cb.mu.RLock()
	defer cb.mu.RUnlock()

	result := make([]CircuitBreakerHostStatus, 0, len(cb.hosts))
	for host, h := range cb.hosts {
		state := h.state
		// Check if we should report as half-open (timeout elapsed)
		if state == StateOpen && time.Since(h.lastFailureTime) >= cb.config.ResetTimeout {
			state = StateHalfOpen
		}

		result = append(result, CircuitBreakerHostStatus{
			Host:             host,
			State:            state.String(),
			FailureCount:     h.failureCount,
			SuccessCount:     h.successCount,
			LastFailureTime:  h.lastFailureTime,
			HalfOpenRequests: h.halfOpenRequests,
		})
	}

	return result
}

// IsEnabled returns true if the circuit breaker is enabled.
func (cb *CircuitBreaker) IsEnabled() bool {
	return cb != nil && cb.config.Enabled
}

// GetConfig returns a copy of the circuit breaker configuration.
func (cb *CircuitBreaker) GetConfig() CircuitBreakerConfig {
	if cb == nil {
		return CircuitBreakerConfig{Enabled: false}
	}
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.config
}

// Reset resets the circuit breaker state for a specific host or all hosts if host is empty.
func (cb *CircuitBreaker) Reset(host string) {
	if cb == nil {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if host == "" {
		// Reset all hosts
		cb.hosts = make(map[string]*circuitBreakerHost)
	} else {
		// Reset specific host
		delete(cb.hosts, host)
	}
}

// String returns a human-readable description of the circuit breaker state.
func (cbs CircuitBreakerHostStatus) String() string {
	return fmt.Sprintf("%s: state=%s failures=%d successes=%d last_fail=%s",
		cbs.Host,
		cbs.State,
		cbs.FailureCount,
		cbs.SuccessCount,
		cbs.LastFailureTime.Format(time.RFC3339),
	)
}
