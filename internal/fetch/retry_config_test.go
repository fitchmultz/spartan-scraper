// Package fetch provides tests for retry configuration.
// Tests cover default configurations, backoff strategy parsing, and retryable codes.
package fetch

import (
	"testing"
	"time"
)

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
