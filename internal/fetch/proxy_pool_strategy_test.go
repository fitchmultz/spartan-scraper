// Package fetch provides HTTP and headless browser content fetching capabilities.
package fetch

import (
	"testing"
)

func TestRotationStrategy_String(t *testing.T) {
	tests := []struct {
		strategy RotationStrategy
		want     string
	}{
		{RotationRoundRobin, "round_robin"},
		{RotationRandom, "random"},
		{RotationLeastUsed, "least_used"},
		{RotationWeighted, "weighted"},
		{RotationLeastLatency, "least_latency"},
		{RotationStrategy(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.strategy.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRotationStrategy(t *testing.T) {
	tests := []struct {
		input string
		want  RotationStrategy
	}{
		{"round_robin", RotationRoundRobin},
		{"random", RotationRandom},
		{"least_used", RotationLeastUsed},
		{"weighted", RotationWeighted},
		{"least_latency", RotationLeastLatency},
		{"unknown", RotationRoundRobin},
		{"", RotationRoundRobin},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseRotationStrategy(tt.input)
			if got != tt.want {
				t.Errorf("ParseRotationStrategy(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultHealthCheckConfig(t *testing.T) {
	cfg := DefaultHealthCheckConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.IntervalSeconds != 60 {
		t.Errorf("Expected IntervalSeconds = 60, got %d", cfg.IntervalSeconds)
	}
	if cfg.TimeoutSeconds != 10 {
		t.Errorf("Expected TimeoutSeconds = 10, got %d", cfg.TimeoutSeconds)
	}
	if cfg.MaxConsecutiveFails != 3 {
		t.Errorf("Expected MaxConsecutiveFails = 3, got %d", cfg.MaxConsecutiveFails)
	}
	if cfg.RecoveryAfterSeconds != 300 {
		t.Errorf("Expected RecoveryAfterSeconds = 300, got %d", cfg.RecoveryAfterSeconds)
	}
	if cfg.TestURL != "http://httpbin.org/ip" {
		t.Errorf("Expected TestURL = http://httpbin.org/ip, got %q", cfg.TestURL)
	}
}
