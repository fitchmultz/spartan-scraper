// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// BackoffStrategy defines the backoff calculation strategy.
type BackoffStrategy int

const (
	// BackoffStrategyExponential uses exponential backoff: base * 2^attempt
	BackoffStrategyExponential BackoffStrategy = iota
	// BackoffStrategyExponentialJitter adds random jitter to exponential backoff
	BackoffStrategyExponentialJitter
	// BackoffStrategyLinear uses linear backoff: base * (attempt + 1)
	BackoffStrategyLinear
	// BackoffStrategyFixed uses a fixed delay regardless of attempt
	BackoffStrategyFixed
)

// String returns the string representation of the backoff strategy.
func (s BackoffStrategy) String() string {
	switch s {
	case BackoffStrategyExponential:
		return "exponential"
	case BackoffStrategyExponentialJitter:
		return "exponential_jitter"
	case BackoffStrategyLinear:
		return "linear"
	case BackoffStrategyFixed:
		return "fixed"
	default:
		return "unknown"
	}
}

// ParseBackoffStrategy parses a backoff strategy string.
func ParseBackoffStrategy(s string) BackoffStrategy {
	switch strings.ToLower(s) {
	case "exponential":
		return BackoffStrategyExponential
	case "exponential_jitter", "exponential-jitter", "exponentialjitter":
		return BackoffStrategyExponentialJitter
	case "linear":
		return BackoffStrategyLinear
	case "fixed":
		return BackoffStrategyFixed
	default:
		return BackoffStrategyExponentialJitter // Default to jitter
	}
}

// RetryConfig configures retry behavior with per-status-code policies and backoff strategies.
type RetryConfig struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration   // Cap on delay (default: 60s)
	Strategy        BackoffStrategy // Backoff calculation strategy
	RetryableCodes  map[int]bool    // Status codes that trigger retry (nil = use defaults)
	RetryableErrors []error         // Error types that trigger retry (empty = use defaults)
}

// DefaultRetryConfig returns a RetryConfig with sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 2,
		BaseDelay:  400 * time.Millisecond,
		MaxDelay:   60 * time.Second,
		Strategy:   BackoffStrategyExponentialJitter,
		RetryableCodes: map[int]bool{
			429: true,
			500: true,
			502: true,
			503: true,
			504: true,
		},
	}
}

// DefaultRetryableCodes returns the default set of HTTP status codes that should trigger a retry.
func DefaultRetryableCodes() map[int]bool {
	return map[int]bool{
		429: true, // Too Many Requests
		500: true, // Internal Server Error
		502: true, // Bad Gateway
		503: true, // Service Unavailable
		504: true, // Gateway Timeout
	}
}

// randSource is a thread-safe random source for jitter calculations.
var randSource = &lockedRand{r: rand.New(rand.NewSource(time.Now().UnixNano()))}

// lockedRand wraps rand.Rand with a mutex for thread safety.
type lockedRand struct {
	r  *rand.Rand
	mu sync.Mutex
}

// Float64 returns a random float64 in [0.0, 1.0).
func (lr *lockedRand) Float64() float64 {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	return lr.r.Float64()
}

func shouldRetry(err error, status int) bool {
	if status == 429 {
		return true
	}
	if status >= 500 && status < 600 {
		return true
	}

	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidURLScheme) {
			return false
		}
		if errors.Is(err, apperrors.ErrInvalidURLHost) {
			return false
		}

		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			if dnsErr.IsNotFound {
				return false
			}
			if dnsErr.IsTimeout {
				return true
			}
			return false
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return true
		}

		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return true
			}
		}

		if errors.Is(err, net.ErrClosed) {
			return true
		}

		if strings.Contains(err.Error(), "timeout") {
			return true
		}

		return false
	}

	return false
}

func backoff(base time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return base
	}
	multiplier := math.Pow(2, float64(attempt))
	return time.Duration(float64(base) * multiplier)
}

// jitterBackoff adds random jitter to exponential backoff to prevent thundering herd.
// The jitter is 0-50% of the calculated delay, capped at maxDelay.
func jitterBackoff(base time.Duration, attempt int, maxDelay time.Duration) time.Duration {
	expDelay := backoff(base, attempt)
	if expDelay > maxDelay && maxDelay > 0 {
		expDelay = maxDelay
	}
	// Add 0-50% jitter
	jitter := time.Duration(float64(expDelay) * 0.5 * randSource.Float64())
	delay := expDelay + jitter
	// Ensure final delay respects maxDelay
	if delay > maxDelay && maxDelay > 0 {
		delay = maxDelay
	}
	return delay
}

// linearBackoff returns linear increasing delay: base * (attempt + 1).
func linearBackoff(base time.Duration, attempt int, maxDelay time.Duration) time.Duration {
	delay := base * time.Duration(attempt+1)
	if delay > maxDelay && maxDelay > 0 {
		delay = maxDelay
	}
	return delay
}

// fixedBackoff returns a fixed delay regardless of attempt.
func fixedBackoff(base time.Duration, _ int, maxDelay time.Duration) time.Duration {
	if base > maxDelay && maxDelay > 0 {
		return maxDelay
	}
	return base
}

// CalculateBackoff returns backoff duration based on the configured strategy.
// This is the main entry point for computing retry delays.
func CalculateBackoff(cfg RetryConfig, attempt int) time.Duration {
	// Use defaults for zero values
	baseDelay := cfg.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 400 * time.Millisecond
	}
	maxDelay := cfg.MaxDelay
	if maxDelay <= 0 {
		maxDelay = 60 * time.Second
	}

	switch cfg.Strategy {
	case BackoffStrategyExponentialJitter:
		return jitterBackoff(baseDelay, attempt, maxDelay)
	case BackoffStrategyLinear:
		return linearBackoff(baseDelay, attempt, maxDelay)
	case BackoffStrategyFixed:
		return fixedBackoff(baseDelay, attempt, maxDelay)
	default: // BackoffStrategyExponential
		delay := backoff(baseDelay, attempt)
		if delay > maxDelay {
			delay = maxDelay
		}
		return delay
	}
}

// ShouldRetryWithConfig checks if retry should occur using configurable rules.
// It first checks the configured status codes, then falls back to default logic.
func ShouldRetryWithConfig(err error, status int, cfg RetryConfig) bool {
	// Check configured status codes first
	if cfg.RetryableCodes != nil {
		if shouldRetry, ok := cfg.RetryableCodes[status]; ok {
			return shouldRetry
		}
		// When explicit config is provided, only configured codes are retryable
		return false
	}

	// Fall back to default logic
	return shouldRetry(err, status)
}

// IsStatusCodeRetryable checks if a status code is in the retryable set.
func IsStatusCodeRetryable(status int, retryableCodes map[int]bool) bool {
	if retryableCodes == nil {
		retryableCodes = DefaultRetryableCodes()
	}
	return retryableCodes[status]
}

func clampRetry(count int) int {
	if count < 0 {
		return 0
	}
	if count > 10 {
		return 10
	}
	return count
}

func readRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	value := resp.Header.Get("Retry-After")
	if value == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(value + "s"); err == nil {
		return seconds
	}
	if t, err := http.ParseTime(value); err == nil {
		return time.Until(t)
	}
	return 0
}
