// Purpose: Load and validate circuit-breaker startup configuration independently from other config domains.
// Responsibilities:
// - Parse CIRCUIT_BREAKER_* environment variables into Config.
// - Enforce circuit-breaker invariants with operator-visible startup notices.
// - Keep resilience-policy configuration isolated from unrelated startup parsing.
// Scope:
// - Circuit-breaker configuration only.
// Usage:
// - Call loadCircuitBreakerConfig during Load(), then run validateAndFixCircuitBreakerConfig.
// Invariants/Assumptions:
// - Disabled circuit breakers skip threshold validation.
// - Validation corrects invalid values in-place rather than failing startup.
package config

func loadCircuitBreakerConfig(cfg Config) Config {
	cfg.CircuitBreakerEnabled = getenvBool("CIRCUIT_BREAKER_ENABLED", true)
	cfg.CircuitBreakerFailureThreshold = getenvInt("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5)
	cfg.CircuitBreakerSuccessThreshold = getenvInt("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3)
	cfg.CircuitBreakerResetTimeoutSecs = getenvInt("CIRCUIT_BREAKER_RESET_TIMEOUT_SECONDS", 30)
	cfg.CircuitBreakerHalfOpenMaxRequests = getenvInt("CIRCUIT_BREAKER_HALF_OPEN_MAX_REQUESTS", 3)
	return cfg
}

// validateAndFixCircuitBreakerConfig ensures circuit breaker configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixCircuitBreakerConfig(cfg Config) Config {
	if !cfg.CircuitBreakerEnabled {
		return cfg
	}

	if cfg.CircuitBreakerFailureThreshold <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "circuit-breaker-failure-threshold-invalid",
			Severity: "warning",
			Title:    "Circuit-breaker failure threshold was reset",
			Message:  "CIRCUIT_BREAKER_FAILURE_THRESHOLD must be positive, so Spartan is using 5 for this session.",
		})
		cfg.CircuitBreakerFailureThreshold = 5
	}
	if cfg.CircuitBreakerSuccessThreshold <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "circuit-breaker-success-threshold-invalid",
			Severity: "warning",
			Title:    "Circuit-breaker success threshold was reset",
			Message:  "CIRCUIT_BREAKER_SUCCESS_THRESHOLD must be positive, so Spartan is using 3 for this session.",
		})
		cfg.CircuitBreakerSuccessThreshold = 3
	}
	if cfg.CircuitBreakerResetTimeoutSecs <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "circuit-breaker-reset-timeout-invalid",
			Severity: "warning",
			Title:    "Circuit-breaker reset timeout was reset",
			Message:  "CIRCUIT_BREAKER_RESET_TIMEOUT_SECONDS must be positive, so Spartan is using 30 seconds for this session.",
		})
		cfg.CircuitBreakerResetTimeoutSecs = 30
	}
	if cfg.CircuitBreakerHalfOpenMaxRequests <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "circuit-breaker-half-open-invalid",
			Severity: "warning",
			Title:    "Circuit-breaker half-open limit was reset",
			Message:  "CIRCUIT_BREAKER_HALF_OPEN_MAX_REQUESTS must be positive, so Spartan is using 3 for this session.",
		})
		cfg.CircuitBreakerHalfOpenMaxRequests = 3
	}

	return cfg
}
