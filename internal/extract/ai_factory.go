// Package extract provides factory functions for creating LLM providers.
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
	for _, capability := range []string{
		config.AICapabilityExtractNatural,
		config.AICapabilityExtractSchema,
		config.AICapabilityTemplateGeneration,
	} {
		if len(cfg.Routing.RoutesFor(capability)) == 0 {
			return fmt.Errorf("no routes configured for capability %s", capability)
		}
	}
	if cfg.StartupTimeoutSecs < 1 || cfg.StartupTimeoutSecs > 60 {
		return fmt.Errorf("PI_STARTUP_TIMEOUT_SECONDS must be between 1 and 60 seconds")
	}
	if cfg.RequestTimeoutSecs < 5 || cfg.RequestTimeoutSecs > 300 {
		return fmt.Errorf("PI_REQUEST_TIMEOUT_SECONDS must be between 5 and 300 seconds")
	}
	return nil
}
