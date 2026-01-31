// Package fetch provides tests for backoff calculation strategies.
// Tests cover exponential, jitter, linear, and fixed backoff strategies.
package fetch

import (
	"testing"
	"time"
)

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
