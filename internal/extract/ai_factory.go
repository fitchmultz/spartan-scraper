// Package extract validates and constructs AI providers from runtime configuration.
//
// Purpose:
// - Create the configured AI provider and reject only truly invalid bridge prerequisites.
//
// Responsibilities:
// - Gate provider creation on required bridge/node settings.
// - Allow per-capability route disables without treating them as fatal config errors.
// - Preserve hard validation for impossible timeout settings.
//
// Scope:
// - Provider construction only; request-time capability failures remain downstream.
//
// Usage:
// - Called by NewAIExtractor during runtime startup.
//
// Invariants/Assumptions:
// - AI may be fully or partially disabled via routing config while still remaining a valid config state.
// - Request-time calls should fail for disabled capabilities rather than preventing unrelated capabilities from starting.
package extract

import (
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// CreateLLMProvider creates the pi-backed provider.
func CreateLLMProvider(cfg config.AIConfig) (LLMProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("pi bridge is disabled")
	}
	return NewPIProvider(cfg), nil
}

// ValidateProvider checks if pi bridge configuration is valid.
func ValidateProvider(cfg config.AIConfig) error {
	if !cfg.Enabled {
		return fmt.Errorf("PI_ENABLED must be true")
	}
	if strings.TrimSpace(cfg.NodeBin) == "" {
		return fmt.Errorf("PI_NODE_BIN is required")
	}
	if strings.TrimSpace(cfg.BridgeScript) == "" {
		return fmt.Errorf("PI_BRIDGE_SCRIPT is required")
	}
	if cfg.StartupTimeoutSecs < 1 || cfg.StartupTimeoutSecs > 60 {
		return fmt.Errorf("PI_STARTUP_TIMEOUT_SECONDS must be between 1 and 60 seconds")
	}
	if cfg.RequestTimeoutSecs < 5 || cfg.RequestTimeoutSecs > 300 {
		return fmt.Errorf("PI_REQUEST_TIMEOUT_SECONDS must be between 5 and 300 seconds")
	}
	return nil
}
