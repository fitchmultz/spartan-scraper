// Purpose: Load and validate rate-limit, adaptive throttling, and retry-related startup configuration.
// Responsibilities:
// - Parse RATE_LIMIT_*, ADAPTIVE_*, and RETRY_* environment variables into Config.
// - Enforce adaptive throttling and retry invariants with operator-visible startup notices.
// - Keep rate-control policy separate from unrelated startup config domains.
// Scope:
// - Request-rate and retry configuration only.
// Usage:
// - Call loadRateLimitConfig during Load(), then run validateAndFixAdaptiveConfig and validateAndFixRetryConfig.
// Invariants/Assumptions:
// - RATE_LIMIT_QPS is loaded before ADAPTIVE_MAX_QPS fallback calculation.
// - Validation mutates only the relevant Config fields and records notices instead of failing startup.
package config

import (
	"fmt"
	"strings"
)

func loadRateLimitConfig(cfg Config) Config {
	cfg.RateLimitQPS = getenvInt("RATE_LIMIT_QPS", 2)
	cfg.RateLimitBurst = getenvInt("RATE_LIMIT_BURST", 4)
	cfg.MaxRetries = getenvInt("MAX_RETRIES", 2)
	cfg.RetryBaseMs = getenvInt("RETRY_BASE_MS", 400)
	cfg.AdaptiveRateLimit = getenvBool("ADAPTIVE_RATE_LIMIT", false)
	cfg.AdaptiveMinQPS = getenvFloat64("ADAPTIVE_MIN_QPS", 0.1)
	cfg.AdaptiveMaxQPS = getenvFloat64("ADAPTIVE_MAX_QPS", float64(cfg.RateLimitQPS))
	cfg.AdaptiveIncreaseQPS = getenvFloat64("ADAPTIVE_INCREASE_QPS", 0.5)
	cfg.AdaptiveDecreaseFactor = getenvFloat64("ADAPTIVE_DECREASE_FACTOR", 0.5)
	cfg.AdaptiveSuccessThreshold = getenvInt("ADAPTIVE_SUCCESS_THRESHOLD", 5)
	cfg.AdaptiveCooldownMs = getenvInt("ADAPTIVE_COOLDOWN_MS", 1000)
	cfg.RetryMaxDelaySecs = getenvInt("RETRY_MAX_DELAY_SECONDS", 60)
	cfg.RetryBackoffStrategy = getenv("RETRY_BACKOFF_STRATEGY", "exponential_jitter")
	cfg.RetryStatusCodes = getenv("RETRY_STATUS_CODES", "429,500,502,503,504")
	return cfg
}

// validateAndFixAdaptiveConfig ensures adaptive rate limiting configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixAdaptiveConfig(cfg Config) Config {
	if !cfg.AdaptiveRateLimit {
		return cfg
	}

	if cfg.AdaptiveMinQPS > cfg.AdaptiveMaxQPS {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-min-max-swapped",
			Severity: "warning",
			Title:    "Adaptive rate-limit bounds were corrected",
			Message:  fmt.Sprintf("ADAPTIVE_MIN_QPS (%.2f) exceeded ADAPTIVE_MAX_QPS (%.2f), so Spartan swapped them for this session.", cfg.AdaptiveMinQPS, cfg.AdaptiveMaxQPS),
		})
		cfg.AdaptiveMinQPS, cfg.AdaptiveMaxQPS = cfg.AdaptiveMaxQPS, cfg.AdaptiveMinQPS
	}

	if cfg.AdaptiveMinQPS <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-min-invalid",
			Severity: "warning",
			Title:    "Adaptive minimum QPS was reset",
			Message:  "ADAPTIVE_MIN_QPS must be positive, so Spartan is using 0.1 for this session.",
		})
		cfg.AdaptiveMinQPS = 0.1
	}

	if cfg.AdaptiveMaxQPS <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-max-invalid",
			Severity: "warning",
			Title:    "Adaptive maximum QPS was reset",
			Message:  "ADAPTIVE_MAX_QPS must be positive, so Spartan is using RATE_LIMIT_QPS for this session.",
		})
		cfg.AdaptiveMaxQPS = float64(cfg.RateLimitQPS)
	}

	if cfg.AdaptiveDecreaseFactor <= 0 || cfg.AdaptiveDecreaseFactor >= 1 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-decrease-invalid",
			Severity: "warning",
			Title:    "Adaptive decrease factor was reset",
			Message:  "ADAPTIVE_DECREASE_FACTOR must be between 0 and 1, so Spartan is using 0.5 for this session.",
		})
		cfg.AdaptiveDecreaseFactor = 0.5
	}

	if cfg.AdaptiveIncreaseQPS <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-increase-invalid",
			Severity: "warning",
			Title:    "Adaptive increase QPS was reset",
			Message:  "ADAPTIVE_INCREASE_QPS must be positive, so Spartan is using 0.5 for this session.",
		})
		cfg.AdaptiveIncreaseQPS = 0.5
	}

	if cfg.AdaptiveSuccessThreshold <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-success-threshold-invalid",
			Severity: "warning",
			Title:    "Adaptive success threshold was reset",
			Message:  "ADAPTIVE_SUCCESS_THRESHOLD must be positive, so Spartan is using 5 for this session.",
		})
		cfg.AdaptiveSuccessThreshold = 5
	}

	if cfg.AdaptiveCooldownMs < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-cooldown-invalid",
			Severity: "warning",
			Title:    "Adaptive cooldown was reset",
			Message:  "ADAPTIVE_COOLDOWN_MS must be non-negative, so Spartan is using 1000ms for this session.",
		})
		cfg.AdaptiveCooldownMs = 1000
	}

	return cfg
}

// validateAndFixRetryConfig ensures retry configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixRetryConfig(cfg Config) Config {
	if cfg.RetryMaxDelaySecs < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retry-max-delay-invalid",
			Severity: "warning",
			Title:    "Retry max delay was reset",
			Message:  "RETRY_MAX_DELAY_SECONDS must be non-negative, so Spartan is using 60 seconds for this session.",
		})
		cfg.RetryMaxDelaySecs = 60
	}

	validStrategies := map[string]bool{
		"exponential":        true,
		"exponential_jitter": true,
		"exponential-jitter": true,
		"exponentialjitter":  true,
		"linear":             true,
		"fixed":              true,
	}

	strategyLower := strings.ToLower(cfg.RetryBackoffStrategy)
	if cfg.RetryBackoffStrategy != "" && !validStrategies[strategyLower] {
		recordStartupNotice(StartupNotice{
			ID:       "retry-backoff-strategy-invalid",
			Severity: "warning",
			Title:    "Retry backoff strategy was reset",
			Message:  fmt.Sprintf("RETRY_BACKOFF_STRATEGY %q is unsupported, so Spartan is using exponential_jitter for this session.", cfg.RetryBackoffStrategy),
		})
		cfg.RetryBackoffStrategy = "exponential_jitter"
	}

	if strategyLower == "exponential-jitter" || strategyLower == "exponentialjitter" {
		cfg.RetryBackoffStrategy = "exponential_jitter"
	}

	return cfg
}
