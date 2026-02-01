// Package extract provides factory functions for creating LLM providers.
package extract

import (
	"fmt"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// CreateLLMProvider creates the appropriate provider based on config.
func CreateLLMProvider(cfg config.AIConfig) (LLMProvider, error) {
	switch cfg.Provider {
	case config.AIProviderOpenAI:
		return NewOpenAIProvider(cfg), nil
	case config.AIProviderAnthropic:
		return NewAnthropicProvider(cfg), nil
	case config.AIProviderOllama:
		return NewOllamaProvider(cfg), nil
	case "":
		return nil, fmt.Errorf("AI provider not configured")
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.Provider)
	}
}

// GetDefaultModel returns the default model for a provider.
func GetDefaultModel(provider config.AIProvider) string {
	switch provider {
	case config.AIProviderOpenAI:
		return "gpt-4o-mini"
	case config.AIProviderAnthropic:
		return "claude-3-haiku-20240307"
	case config.AIProviderOllama:
		return "llama3.1"
	default:
		return ""
	}
}

// ValidateProvider checks if provider configuration is valid.
func ValidateProvider(cfg config.AIConfig) error {
	if cfg.Provider == "" {
		return fmt.Errorf("AI provider not specified")
	}

	validProviders := map[config.AIProvider]bool{
		config.AIProviderOpenAI:    true,
		config.AIProviderAnthropic: true,
		config.AIProviderOllama:    true,
	}
	if !validProviders[cfg.Provider] {
		return fmt.Errorf("invalid AI provider: %s", cfg.Provider)
	}

	// Cloud providers require API key
	if cfg.Provider == config.AIProviderOpenAI || cfg.Provider == config.AIProviderAnthropic {
		if cfg.APIKey == "" {
			return fmt.Errorf("API key required for %s provider", cfg.Provider)
		}
	}

	// Validate timeout
	if cfg.TimeoutSecs < 5 || cfg.TimeoutSecs > 300 {
		return fmt.Errorf("AI timeout must be between 5 and 300 seconds")
	}

	// Validate temperature
	if cfg.Temperature < 0 || cfg.Temperature > 1.0 {
		return fmt.Errorf("AI temperature must be between 0.0 and 1.0")
	}

	return nil
}
