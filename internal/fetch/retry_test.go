// Package fetch provides tests for retry logic and backoff strategies.
// Tests cover retry eligibility for errors and status codes, exponential backoff, jitter, and circuit breaker patterns.
package fetch

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   bool
	}{
		{
			name:   "network error with ErrClosed retries",
			err:    net.ErrClosed,
			status: 0,
			want:   true,
		},
		{
			name:   "timeout error retries",
			err:    errors.New("connection timeout"),
			status: 0,
			want:   true,
		},
		{
			name:   "other errors do not retry",
			err:    errors.New("some error"),
			status: 0,
			want:   false,
		},
		{
			name:   "success status does not retry",
			err:    nil,
			status: 200,
			want:   false,
		},
		{
			name:   "403 does not retry",
			err:    nil,
			status: 403,
			want:   false,
		},
		{
			name:   "401 does not retry",
			err:    nil,
			status: 401,
			want:   false,
		},
		{
			name:   "429 rate limit retries",
			err:    nil,
			status: 429,
			want:   true,
		},
		{
			name:   "500 server error retries",
			err:    nil,
			status: 500,
			want:   true,
		},
		{
			name:   "502 bad gateway retries",
			err:    nil,
			status: 502,
			want:   true,
		},
		{
			name:   "503 service unavailable retries",
			err:    nil,
			status: 503,
			want:   true,
		},
		{
			name:   "504 gateway timeout retries",
			err:    nil,
			status: 504,
			want:   true,
		},
		{
			name:   "400 bad request does not retry",
			err:    nil,
			status: 400,
			want:   false,
		},
		{
			name:   "404 not found does not retry",
			err:    nil,
			status: 404,
			want:   false,
		},
		{
			name:   "context deadline exceeded retries",
			err:    context.DeadlineExceeded,
			status: 0,
			want:   true,
		},
		{
			name: "DNS NXDOMAIN does not retry",
			err: &net.DNSError{
				Err:        "no such host",
				Name:       "nonexistent.example.com",
				IsNotFound: true,
			},
			status: 0,
			want:   false,
		},
		{
			name: "DNS timeout retries",
			err: &net.DNSError{
				Err:       "lookup nonexistent.example.com on 127.0.0.53:53: read udp 127.0.0.1:12345->127.0.0.53:53: i/o timeout",
				Name:      "nonexistent.example.com",
				IsTimeout: true,
			},
			status: 0,
			want:   true,
		},
		{
			name:   "invalid URL scheme does not retry",
			err:    apperrors.ErrInvalidURLScheme,
			status: 0,
			want:   false,
		},
		{
			name:   "invalid URL host does not retry",
			err:    apperrors.ErrInvalidURLHost,
			status: 0,
			want:   false,
		},
		{
			name: "connection refused does not retry",
			err: &net.OpError{
				Err: errors.New("connect: connection refused"),
				Op:  "dial",
			},
			status: 0,
			want:   false,
		},
		{
			name: "no such host (DNS lookup failed) does not retry",
			err: &net.OpError{
				Err: &net.DNSError{
					Err:        "no such host",
					Name:       "invalid-host-name",
					IsNotFound: true,
				},
				Op: "dial",
			},
			status: 0,
			want:   false,
		},
		{
			name: "net.Error with Timeout flag retries",
			err: &net.OpError{
				Err: errors.New("i/o timeout"),
				Op:  "read",
			},
			status: 0,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetry(tt.err, tt.status); got != tt.want {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		name    string
		base    time.Duration
		attempt int
		want    time.Duration
	}{
		{
			name:    "attempt 0 returns base",
			base:    100 * time.Millisecond,
			attempt: 0,
			want:    100 * time.Millisecond,
		},
		{
			name:    "attempt 1 returns base * 2",
			base:    100 * time.Millisecond,
			attempt: 1,
			want:    200 * time.Millisecond,
		},
		{
			name:    "attempt 2 returns base * 4",
			base:    100 * time.Millisecond,
			attempt: 2,
			want:    400 * time.Millisecond,
		},
		{
			name:    "attempt 3 returns base * 8",
			base:    100 * time.Millisecond,
			attempt: 3,
			want:    800 * time.Millisecond,
		},
		{
			name:    "different base 1s",
			base:    1 * time.Second,
			attempt: 1,
			want:    2 * time.Second,
		},
		{
			name:    "exponential growth continues",
			base:    100 * time.Millisecond,
			attempt: 4,
			want:    1600 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backoff(tt.base, tt.attempt); got != tt.want {
				t.Errorf("backoff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClampRetry(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  int
	}{
		{
			name:  "negative count returns 0",
			count: -5,
			want:  0,
		},
		{
			name:  "zero returns 0",
			count: 0,
			want:  0,
		},
		{
			name:  "small count returns as is",
			count: 3,
			want:  3,
		},
		{
			name:  "exactly 10 returns 10",
			count: 10,
			want:  10,
		},
		{
			name:  "15 is clamped to 10",
			count: 15,
			want:  10,
		},
		{
			name:  "100 is clamped to 10",
			count: 100,
			want:  10,
		},
		{
			name:  "just above limit",
			count: 11,
			want:  10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampRetry(tt.count); got != tt.want {
				t.Errorf("clampRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		want     time.Duration
		validate func(t *testing.T, got time.Duration)
	}{
		{
			name: "nil response returns 0",
			resp: nil,
			want: 0,
		},
		{
			name: "empty header returns 0",
			resp: &http.Response{Header: http.Header{}},
			want: 0,
		},
		{
			name: "seconds format 60",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"60"}},
			},
			want: 60 * time.Second,
		},
		{
			name: "seconds format 120",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"120"}},
			},
			want: 120 * time.Second,
		},
		{
			name: "future date returns positive duration",
			resp: func() *http.Response {
				futureTime := time.Now().Add(5 * time.Minute).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
				return &http.Response{
					Header: http.Header{"Retry-After": []string{futureTime}},
				}
			}(),
			validate: func(t *testing.T, got time.Duration) {
				if got < 4*time.Minute || got > 6*time.Minute {
					t.Errorf("readRetryAfter() = %v, want approximately 5m0s", got)
				}
			},
		},
		{
			name: "past date returns <= 0",
			resp: func() *http.Response {
				pastTime := time.Now().Add(-5 * time.Minute).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
				return &http.Response{
					Header: http.Header{"Retry-After": []string{pastTime}},
				}
			}(),
			validate: func(t *testing.T, got time.Duration) {
				if got > 0 {
					t.Errorf("readRetryAfter() = %v, want <= 0s", got)
				}
			},
		},
		{
			name: "invalid format returns 0",
			resp: &http.Response{
				Header: http.Header{"Retry-After": []string{"invalid"}},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readRetryAfter(tt.resp)
			if tt.validate != nil {
				tt.validate(t, got)
			} else if got != tt.want {
				t.Errorf("readRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Backoff Strategy Tests
// ============================================================================

func TestBackoffStrategy_String(t *testing.T) {
	tests := []struct {
		strategy BackoffStrategy
		want     string
	}{
		{BackoffStrategyExponential, "exponential"},
		{BackoffStrategyExponentialJitter, "exponential_jitter"},
		{BackoffStrategyLinear, "linear"},
		{BackoffStrategyFixed, "fixed"},
		{BackoffStrategy(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBackoffStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected BackoffStrategy
	}{
		{"exponential", BackoffStrategyExponential},
		{"EXPONENTIAL", BackoffStrategyExponential},
		{"Exponential", BackoffStrategyExponential},
		{"exponential_jitter", BackoffStrategyExponentialJitter},
		{"exponential-jitter", BackoffStrategyExponentialJitter},
		{"exponentialjitter", BackoffStrategyExponentialJitter},
		{"EXPONENTIAL_JITTER", BackoffStrategyExponentialJitter},
		{"linear", BackoffStrategyLinear},
		{"LINEAR", BackoffStrategyLinear},
		{"fixed", BackoffStrategyFixed},
		{"FIXED", BackoffStrategyFixed},
		{"", BackoffStrategyExponentialJitter},        // Default
		{"unknown", BackoffStrategyExponentialJitter}, // Default
		{"random", BackoffStrategyExponentialJitter},  // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseBackoffStrategy(tt.input)
			if got != tt.expected {
				t.Errorf("ParseBackoffStrategy(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != 2 {
		t.Errorf("Expected MaxRetries = 2, got %d", cfg.MaxRetries)
	}
	if cfg.BaseDelay != 400*time.Millisecond {
		t.Errorf("Expected BaseDelay = 400ms, got %v", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 60*time.Second {
		t.Errorf("Expected MaxDelay = 60s, got %v", cfg.MaxDelay)
	}
	if cfg.Strategy != BackoffStrategyExponentialJitter {
		t.Errorf("Expected Strategy = exponential_jitter, got %v", cfg.Strategy)
	}
	if cfg.RetryableCodes == nil {
		t.Error("Expected RetryableCodes to be set")
	}
	if !cfg.RetryableCodes[429] || !cfg.RetryableCodes[500] {
		t.Error("Expected 429 and 500 to be retryable")
	}
}

func TestDefaultRetryableCodes(t *testing.T) {
	codes := DefaultRetryableCodes()

	expectedCodes := []int{429, 500, 502, 503, 504}
	for _, code := range expectedCodes {
		if !codes[code] {
			t.Errorf("Expected status code %d to be retryable", code)
		}
	}

	// These should NOT be retryable
	unexpectedCodes := []int{200, 400, 401, 403, 404}
	for _, code := range unexpectedCodes {
		if codes[code] {
			t.Errorf("Expected status code %d to NOT be retryable", code)
		}
	}
}

func TestJitterBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	maxDelay := 5 * time.Second

	// Test that jitter adds some randomness
	var allSame bool = true
	var firstDelay time.Duration

	for i := 0; i < 10; i++ {
		delay := jitterBackoff(base, 1, maxDelay)
		if i == 0 {
			firstDelay = delay
		} else if delay != firstDelay {
			allSame = false
		}

		// Delay should be >= expected exponential (base * 2)
		expectedExp := base * 2
		if delay < expectedExp {
			t.Errorf("jitterBackoff delay %v < expected exponential %v", delay, expectedExp)
		}

		// Delay should be <= 1.5x expected (base * 2 * 1.5)
		maxExpected := time.Duration(float64(expectedExp) * 1.5)
		if delay > maxExpected {
			t.Errorf("jitterBackoff delay %v > max expected with jitter %v", delay, maxExpected)
		}
	}

	if allSame {
		t.Error("jitterBackoff should produce varied delays")
	}
}

func TestJitterBackoff_RespectsMaxDelay(t *testing.T) {
	base := 1 * time.Second
	maxDelay := 2 * time.Second

	// At attempt 2, exponential would be 4s, but maxDelay is 2s
	delay := jitterBackoff(base, 2, maxDelay)

	if delay > maxDelay {
		t.Errorf("jitterBackoff delay %v > maxDelay %v", delay, maxDelay)
	}

	// Should be at least maxDelay (since jitter adds on top of capped value)
	// Actually jitter adds to the capped value, so delay could be up to maxDelay * 1.5
	maxWithJitter := time.Duration(float64(maxDelay) * 1.5)
	if delay > maxWithJitter {
		t.Errorf("jitterBackoff delay %v > max with jitter %v", delay, maxWithJitter)
	}
}

func TestLinearBackoff(t *testing.T) {
	tests := []struct {
		name    string
		base    time.Duration
		attempt int
		max     time.Duration
		want    time.Duration
	}{
		{"attempt 0", 100 * time.Millisecond, 0, 10 * time.Second, 100 * time.Millisecond},
		{"attempt 1", 100 * time.Millisecond, 1, 10 * time.Second, 200 * time.Millisecond},
		{"attempt 2", 100 * time.Millisecond, 2, 10 * time.Second, 300 * time.Millisecond},
		{"attempt 5", 100 * time.Millisecond, 5, 10 * time.Second, 600 * time.Millisecond},
		{"with max limit", 1 * time.Second, 10, 5 * time.Second, 5 * time.Second},
		{"zero max ignored", 100 * time.Millisecond, 1, 0, 200 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := linearBackoff(tt.base, tt.attempt, tt.max)
			if got != tt.want {
				t.Errorf("linearBackoff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFixedBackoff(t *testing.T) {
	tests := []struct {
		name    string
		base    time.Duration
		attempt int
		max     time.Duration
		want    time.Duration
	}{
		{"attempt 0", 100 * time.Millisecond, 0, 10 * time.Second, 100 * time.Millisecond},
		{"attempt 5", 100 * time.Millisecond, 5, 10 * time.Second, 100 * time.Millisecond},
		{"attempt 10", 1 * time.Second, 10, 5 * time.Second, 1 * time.Second},
		{"with max limit", 10 * time.Second, 0, 5 * time.Second, 5 * time.Second},
		{"zero max ignored", 100 * time.Millisecond, 5, 0, 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fixedBackoff(tt.base, tt.attempt, tt.max)
			if got != tt.want {
				t.Errorf("fixedBackoff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name    string
		cfg     RetryConfig
		attempt int
		minWant time.Duration // Minimum expected (for jitter)
		maxWant time.Duration // Maximum expected
	}{
		{
			name: "exponential strategy",
			cfg: RetryConfig{
				Strategy:  BackoffStrategyExponential,
				BaseDelay: 100 * time.Millisecond,
				MaxDelay:  10 * time.Second,
			},
			attempt: 2,
			minWant: 400 * time.Millisecond,
			maxWant: 400 * time.Millisecond,
		},
		{
			name: "linear strategy",
			cfg: RetryConfig{
				Strategy:  BackoffStrategyLinear,
				BaseDelay: 100 * time.Millisecond,
				MaxDelay:  10 * time.Second,
			},
			attempt: 2,
			minWant: 300 * time.Millisecond,
			maxWant: 300 * time.Millisecond,
		},
		{
			name: "fixed strategy",
			cfg: RetryConfig{
				Strategy:  BackoffStrategyFixed,
				BaseDelay: 100 * time.Millisecond,
				MaxDelay:  10 * time.Second,
			},
			attempt: 5,
			minWant: 100 * time.Millisecond,
			maxWant: 100 * time.Millisecond,
		},
		{
			name: "exponential with max cap",
			cfg: RetryConfig{
				Strategy:  BackoffStrategyExponential,
				BaseDelay: 1 * time.Second,
				MaxDelay:  2 * time.Second,
			},
			attempt: 5, // Would be 32s without cap
			minWant: 2 * time.Second,
			maxWant: 2 * time.Second,
		},
		{
			name:    "zero values use defaults",
			cfg:     RetryConfig{},
			attempt: 1,
			minWant: 400 * time.Millisecond, // Default base
			maxWant: 800 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBackoff(tt.cfg, tt.attempt)
			if got < tt.minWant || got > tt.maxWant {
				t.Errorf("CalculateBackoff() = %v, want between %v and %v", got, tt.minWant, tt.maxWant)
			}
		})
	}
}

func TestCalculateBackoff_ExponentialJitter(t *testing.T) {
	cfg := RetryConfig{
		Strategy:  BackoffStrategyExponentialJitter,
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  10 * time.Second,
	}

	// Run multiple times to account for randomness
	for i := 0; i < 20; i++ {
		delay := CalculateBackoff(cfg, 1)

		// Base exponential would be 200ms
		// With jitter, delay should be between 200ms and 300ms
		if delay < 200*time.Millisecond {
			t.Errorf("CalculateBackoff() = %v, want >= 200ms", delay)
		}
		if delay > 300*time.Millisecond {
			t.Errorf("CalculateBackoff() = %v, want <= 300ms", delay)
		}
	}
}

func TestShouldRetryWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		cfg    RetryConfig
		want   bool
	}{
		{
			name:   "configured 429 retryable",
			status: 429,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{429: true}},
			want:   true,
		},
		{
			name:   "configured 429 not retryable",
			status: 429,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{429: false}},
			want:   false,
		},
		{
			name:   "configured 503 retryable",
			status: 503,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{503: true, 504: true}},
			want:   true,
		},
		{
			name:   "status not in configured list",
			status: 500,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{503: true}},
			want:   false,
		},
		{
			name:   "nil RetryableCodes uses defaults - 500",
			status: 500,
			cfg:    RetryConfig{},
			want:   true,
		},
		{
			name:   "nil RetryableCodes uses defaults - 429",
			status: 429,
			cfg:    RetryConfig{},
			want:   true,
		},
		{
			name:   "nil RetryableCodes uses defaults - 400 not retryable",
			status: 400,
			cfg:    RetryConfig{},
			want:   false,
		},
		{
			name:   "5xx uses defaults if not explicitly configured",
			status: 501,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{429: true}},
			want:   false, // 501 is not in the configured list
		},
		{
			name:   "error triggers retry",
			err:    context.DeadlineExceeded,
			status: 0,
			cfg:    RetryConfig{},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRetryWithConfig(tt.err, tt.status, tt.cfg)
			if got != tt.want {
				t.Errorf("ShouldRetryWithConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStatusCodeRetryable(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		retryableCodes map[int]bool
		want           bool
	}{
		{
			name:           "retryable in custom set",
			status:         418, // I'm a teapot
			retryableCodes: map[int]bool{418: true},
			want:           true,
		},
		{
			name:           "not retryable in custom set",
			status:         500,
			retryableCodes: map[int]bool{418: true},
			want:           false,
		},
		{
			name:           "nil uses defaults - 429 retryable",
			status:         429,
			retryableCodes: nil,
			want:           true,
		},
		{
			name:           "nil uses defaults - 404 not retryable",
			status:         404,
			retryableCodes: nil,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsStatusCodeRetryable(tt.status, tt.retryableCodes)
			if got != tt.want {
				t.Errorf("IsStatusCodeRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLockedRand_ConcurrentAccess(t *testing.T) {
	// Ensure the random source is safe for concurrent use
	var wg sync.WaitGroup
	numGoroutines := 10
	numIterations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				_ = randSource.Float64()
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
		// Success - no panic
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for concurrent random access")
	}
}

// TestJitterBackoff_Distribution checks that jitter provides reasonable distribution
func TestJitterBackoff_Distribution(t *testing.T) {
	base := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	attempt := 2

	expectedBase := backoff(base, attempt) // 400ms
	minExpected := expectedBase
	maxExpected := time.Duration(float64(expectedBase) * 1.5) // 600ms

	// Collect samples
	numSamples := 100
	var minObserved, maxObserved time.Duration
	minObserved = maxExpected
	maxObserved = 0

	for i := 0; i < numSamples; i++ {
		delay := jitterBackoff(base, attempt, maxDelay)
		if delay < minObserved {
			minObserved = delay
		}
		if delay > maxObserved {
			maxObserved = delay
		}
	}

	// Check distribution is within expected bounds
	if minObserved < minExpected {
		t.Errorf("Minimum observed delay %v < expected minimum %v", minObserved, minExpected)
	}
	if maxObserved > maxExpected {
		t.Errorf("Maximum observed delay %v > expected maximum %v", maxObserved, maxExpected)
	}

	// Check we have some variation (jitter is working)
	variation := maxObserved - minObserved
	if variation < 5*time.Millisecond {
		t.Errorf("Expected jitter variation, but only got %v difference", variation)
	}
}

// TestCalculateBackoff_ZeroMaxDelay tests behavior when maxDelay is zero (disabled)
func TestCalculateBackoff_ZeroMaxDelay(t *testing.T) {
	cfg := RetryConfig{
		Strategy:  BackoffStrategyExponential,
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  0, // Disabled
	}

	delay := CalculateBackoff(cfg, 5)
	// At attempt 5: 100ms * 2^5 = 3.2s
	expected := 3200 * time.Millisecond
	if delay != expected {
		t.Errorf("CalculateBackoff() with zero MaxDelay = %v, want %v", delay, expected)
	}
}

// TestCalculateBackoff_DefaultValues verifies defaults are applied correctly
func TestCalculateBackoff_DefaultValues(t *testing.T) {
	cfg := RetryConfig{} // All zero values

	delay := CalculateBackoff(cfg, 0)
	if delay != 400*time.Millisecond {
		t.Errorf("CalculateBackoff() with defaults, attempt 0 = %v, want 400ms", delay)
	}

	delay = CalculateBackoff(cfg, 1)
	// Should be ~800ms with jitter (0.5*0.5 to 0.5*1.0 added)
	if delay < 800*time.Millisecond || delay > 1200*time.Millisecond {
		t.Errorf("CalculateBackoff() with defaults, attempt 1 = %v, want between 800ms and 1200ms", delay)
	}
}

// TestJitterBackoff_PreventsThunderingHerd demonstrates that jitter spreads retry times
func TestJitterBackoff_PreventsThunderingHerd(t *testing.T) {
	base := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	attempt := 1

	// Simulate 100 concurrent retries
	delays := make([]time.Duration, 100)
	for i := range delays {
		delays[i] = jitterBackoff(base, attempt, maxDelay)
	}

	// Count how many have the exact same delay (would happen without jitter)
	exactMatches := 0
	firstDelay := delays[0]
	for _, d := range delays {
		if d == firstDelay {
			exactMatches++
		}
	}

	// With jitter, we should have very few exact matches
	// (statistically unlikely to have more than ~5% collision)
	collisionRate := float64(exactMatches) / float64(len(delays))
	if collisionRate > 0.1 {
		t.Errorf("Too many exact delay matches (%.1f%%), jitter may not be working", collisionRate*100)
	}
}

// TestBackoffStrategy_String_AllValues ensures all strategies have string representations
func TestBackoffStrategy_String_AllValues(t *testing.T) {
	strategies := []BackoffStrategy{
		BackoffStrategyExponential,
		BackoffStrategyExponentialJitter,
		BackoffStrategyLinear,
		BackoffStrategyFixed,
	}

	seen := make(map[string]bool)
	for _, s := range strategies {
		str := s.String()
		if str == "" || str == "unknown" {
			t.Errorf("Strategy %d has empty or unknown string representation", s)
		}
		if seen[str] {
			t.Errorf("Duplicate string representation: %s", str)
		}
		seen[str] = true
	}
}

// TestParseBackoffStrategy_AllParsable ensures all strategy strings are parseable
func TestParseBackoffStrategy_AllParsable(t *testing.T) {
	// Each strategy should be parseable from its own string representation
	tests := []struct {
		strategy BackoffStrategy
	}{
		{BackoffStrategyExponential},
		{BackoffStrategyExponentialJitter},
		{BackoffStrategyLinear},
		{BackoffStrategyFixed},
	}

	for _, tt := range tests {
		t.Run(tt.strategy.String(), func(t *testing.T) {
			parsed := ParseBackoffStrategy(tt.strategy.String())
			if parsed != tt.strategy {
				t.Errorf("ParseBackoffStrategy(%q) = %v, want %v", tt.strategy.String(), parsed, tt.strategy)
			}
		})
	}
}

// TestShouldRetryWithConfig_5xxDefaultBehavior tests 5xx handling when configured
func TestShouldRetryWithConfig_5xxDefaultBehavior(t *testing.T) {
	// When RetryableCodes is set, 5xx codes NOT in the map should NOT retry
	// (unless they are in the default list and the code chooses to fall back)
	cfg := RetryConfig{
		RetryableCodes: map[int]bool{
			429: true,
			// 500, 502, 503, 504 NOT included
		},
	}

	// 429 is configured - should retry
	if !ShouldRetryWithConfig(nil, 429, cfg) {
		t.Error("Expected 429 to be retryable when configured")
	}

	// 500 is not in config but is in defaults - behavior depends on implementation
	// Current implementation: 5xx not in config returns false, unless using nil fallback
	result500 := ShouldRetryWithConfig(nil, 500, cfg)
	// With explicit config, 500 should NOT retry
	if result500 {
		t.Error("Expected 500 to NOT be retryable when using explicit config without 500")
	}
}

// TestRetryConfig_CustomRetryableCodes verifies custom retryable codes work
func TestRetryConfig_CustomRetryableCodes(t *testing.T) {
	// Create config with only specific codes
	cfg := RetryConfig{
		RetryableCodes: map[int]bool{
			418: true, // I'm a teapot (custom)
			420: true, // Enhance Your Calm (custom)
		},
	}

	tests := []struct {
		status int
		want   bool
	}{
		{418, true},
		{420, true},
		{429, false}, // Not in custom list
		{500, false}, // Not in custom list
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(http.StatusText(tt.status)), func(t *testing.T) {
			got := ShouldRetryWithConfig(nil, tt.status, cfg)
			if got != tt.want {
				t.Errorf("ShouldRetryWithConfig(status=%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
