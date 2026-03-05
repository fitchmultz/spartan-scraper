// Package extract provides AI-powered intelligent data extraction capabilities.
// This file defines types and interfaces for LLM-based extraction.
package extract

import (
	"context"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// FieldSourceLLM is the source constant for AI-extracted values.
const FieldSourceLLM FieldSource = "llm"

// AIExtractionMode defines how AI extraction operates.
type AIExtractionMode string

const (
	// AIModeNaturalLanguage extracts based on natural language description.
	AIModeNaturalLanguage AIExtractionMode = "natural_language"
	// AIModeSchemaGuided extracts fields matching a provided schema example.
	AIModeSchemaGuided AIExtractionMode = "schema_guided"
)

// AIExtractRequest contains parameters for AI extraction.
type AIExtractRequest struct {
	HTML            string                 `json:"html"`
	URL             string                 `json:"url"`
	Mode            AIExtractionMode       `json:"mode"`
	Prompt          string                 `json:"prompt,omitempty"`            // For natural language mode
	SchemaExample   map[string]interface{} `json:"schema_example,omitempty"`    // For schema-guided mode
	Fields          []string               `json:"fields,omitempty"`            // Specific fields to extract
	MaxContentChars int                    `json:"max_content_chars,omitempty"` // Truncate HTML if needed
}

// AIExtractResult contains the AI extraction output.
type AIExtractResult struct {
	Fields      map[string]FieldValue `json:"fields"`
	Confidence  float64               `json:"confidence"` // 0.0-1.0
	Explanation string                `json:"explanation,omitempty"`
	TokensUsed  int                   `json:"tokens_used,omitempty"`
	Cached      bool                  `json:"cached"`
}

// LLMProvider defines the interface for LLM operations.
type LLMProvider interface {
	Extract(ctx context.Context, req AIExtractRequest) (AIExtractResult, error)
	HealthCheck(ctx context.Context) error
}

// AICache provides caching for AI extraction results.
type AICache interface {
	Get(key string) (*AIExtractResult, bool)
	Set(key string, result *AIExtractResult)
}

// AIExtractOptions configures AI-powered extraction within ExtractOptions.
type AIExtractOptions struct {
	Enabled  bool                   `json:"enabled"`
	Mode     AIExtractionMode       `json:"mode"`
	Prompt   string                 `json:"prompt,omitempty"`
	Schema   map[string]interface{} `json:"schema,omitempty"`
	Fields   []string               `json:"fields,omitempty"`
	UseCache bool                   `json:"use_cache,omitempty"`
}

// IsAIEnabled returns true if AI extraction is configured and enabled.
func IsAIEnabled(cfg config.AIConfig) bool {
	return cfg.Provider != ""
}
