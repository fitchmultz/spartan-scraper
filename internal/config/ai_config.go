// Purpose: Load and validate AI bridge startup configuration independently from the rest of the environment loader.
// Responsibilities:
// - Parse PI_* environment variables into AIConfig.
// - Enforce AI timeout and routing-file invariants with operator-visible startup notices.
// - Reject legacy AI_* environment variables and load optional routing config files.
// Scope:
// - AI extraction bridge configuration only.
// Usage:
// - Call loadAIConfig during Load(), then run validateAndFixAIConfig and validateNoLegacyAIConfig.
// Invariants/Assumptions:
// - AI routing defaults come from DefaultAIRoutingConfig unless explicitly overridden.
// - Invalid optional PI_CONFIG_PATH disables AI for the session instead of aborting startup.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func loadAIConfig() AIConfig {
	return AIConfig{
		Enabled:            getenvBool("PI_ENABLED", false),
		ConfigPath:         getenv("PI_CONFIG_PATH", ""),
		Mode:               getenv("PI_MODE", DefaultPIMode),
		NodeBin:            getenv("PI_NODE_BIN", DefaultPINodeBin),
		BridgeScript:       getenv("PI_BRIDGE_SCRIPT", DefaultPIBridgeScript),
		StartupTimeoutSecs: getenvInt("PI_STARTUP_TIMEOUT_SECONDS", DefaultPIStartupTimeoutSecs),
		RequestTimeoutSecs: getenvInt("PI_REQUEST_TIMEOUT_SECONDS", DefaultPIRequestTimeoutSecs),
		Routing:            DefaultAIRoutingConfig(),
	}
}

// validateAndFixAIConfig ensures AI configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixAIConfig(cfg Config) Config {
	ai := cfg.AI

	if ai.StartupTimeoutSecs < 1 {
		recordStartupNotice(StartupNotice{
			ID:       "pi-startup-timeout-too-low",
			Severity: "warning",
			Title:    "AI startup timeout was raised",
			Message:  fmt.Sprintf("PI_STARTUP_TIMEOUT_SECONDS was %d, so Spartan raised it to the minimum 1 second.", ai.StartupTimeoutSecs),
		})
		ai.StartupTimeoutSecs = 1
	}
	if ai.StartupTimeoutSecs > 60 {
		recordStartupNotice(StartupNotice{
			ID:       "pi-startup-timeout-too-high",
			Severity: "warning",
			Title:    "AI startup timeout was capped",
			Message:  fmt.Sprintf("PI_STARTUP_TIMEOUT_SECONDS was %d, so Spartan capped it at 60 seconds.", ai.StartupTimeoutSecs),
		})
		ai.StartupTimeoutSecs = 60
	}

	if ai.RequestTimeoutSecs < 5 {
		recordStartupNotice(StartupNotice{
			ID:       "pi-request-timeout-too-low",
			Severity: "warning",
			Title:    "AI request timeout was raised",
			Message:  fmt.Sprintf("PI_REQUEST_TIMEOUT_SECONDS was %d, so Spartan raised it to the minimum 5 seconds.", ai.RequestTimeoutSecs),
		})
		ai.RequestTimeoutSecs = 5
	}
	if ai.RequestTimeoutSecs > 300 {
		recordStartupNotice(StartupNotice{
			ID:       "pi-request-timeout-too-high",
			Severity: "warning",
			Title:    "AI request timeout was capped",
			Message:  fmt.Sprintf("PI_REQUEST_TIMEOUT_SECONDS was %d, so Spartan capped it at 300 seconds.", ai.RequestTimeoutSecs),
		})
		ai.RequestTimeoutSecs = 300
	}

	if ai.Mode == "" {
		ai.Mode = DefaultPIMode
	}
	if ai.NodeBin == "" {
		ai.NodeBin = DefaultPINodeBin
	}
	if ai.BridgeScript == "" {
		ai.BridgeScript = DefaultPIBridgeScript
	}

	if ai.Enabled && ai.ConfigPath != "" {
		loaded, err := loadAIRoutingConfig(ai.ConfigPath)
		if err != nil {
			recordStartupNotice(StartupNotice{
				ID:       "pi-config-path-invalid",
				Severity: "warning",
				Title:    "AI routing config could not be loaded",
				Message:  fmt.Sprintf("PI_CONFIG_PATH %q could not be loaded (%v), so AI features were disabled for this session.", ai.ConfigPath, err),
			})
			ai.Enabled = false
		} else {
			if loaded.Mode != "" {
				ai.Mode = loaded.Mode
			}
			if len(loaded.Routes.Routes) > 0 {
				ai.Routing = loaded.Routes
			}
		}
	}

	cfg.AI = ai
	return cfg
}

func validateNoLegacyAIConfig() error {
	legacyKeys := []string{
		"AI_PROVIDER",
		"AI_API_KEY",
		"AI_MODEL",
		"AI_TIMEOUT_SECONDS",
		"AI_MAX_TOKENS",
		"AI_TEMPERATURE",
		"OLLAMA_URL",
	}

	used := make([]string, 0)
	for _, key := range legacyKeys {
		if strings.TrimSpace(getenv(key, "")) != "" {
			used = append(used, key)
		}
	}
	if len(used) == 0 {
		return nil
	}

	return apperrors.Validation(
		"legacy AI configuration is no longer supported: " + strings.Join(used, ", ") + ". Use PI_ENABLED and related PI_* bridge settings instead.",
	)
}

type aiBridgeConfigFile struct {
	Mode   string              `json:"mode"`
	Routes map[string][]string `json:"routes"`
}

type loadedAIRoutingConfig struct {
	Mode   string
	Routes AIRoutingConfig
}

func loadAIRoutingConfig(path string) (loadedAIRoutingConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return loadedAIRoutingConfig{}, err
	}

	var file aiBridgeConfigFile
	if err := json.Unmarshal(data, &file); err != nil {
		return loadedAIRoutingConfig{}, err
	}

	routing := DefaultAIRoutingConfig()
	for capability, routes := range file.Routes {
		routing.Routes[capability] = normalizeAIRouteList(routes)
	}

	return loadedAIRoutingConfig{
		Mode:   strings.TrimSpace(file.Mode),
		Routes: routing,
	}, nil
}
