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

// AIImageInput contains optional screenshot/image context for multimodal AI requests.
type AIImageInput struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
}

// AIExtractRequest contains parameters for AI extraction.
type AIExtractRequest struct {
	HTML            string                 `json:"html"`
	URL             string                 `json:"url"`
	Mode            AIExtractionMode       `json:"mode"`
	Prompt          string                 `json:"prompt,omitempty"`         // For natural language mode
	SchemaExample   map[string]interface{} `json:"schema_example,omitempty"` // For schema-guided mode
	Fields          []string               `json:"fields,omitempty"`         // Specific fields to extract
	Images          []AIImageInput         `json:"images,omitempty"`
	MaxContentChars int                    `json:"max_content_chars,omitempty"` // Truncate HTML if needed
}

// AIExtractResult contains the AI extraction output.
type AIExtractResult struct {
	Fields      map[string]FieldValue `json:"fields"`
	Confidence  float64               `json:"confidence"` // 0.0-1.0
	Explanation string                `json:"explanation,omitempty"`
	TokensUsed  int                   `json:"tokens_used,omitempty"`
	RouteID     string                `json:"route_id,omitempty"`
	Provider    string                `json:"provider,omitempty"`
	Model       string                `json:"model,omitempty"`
	Cached      bool                  `json:"cached"`
}

// AITemplateGenerateRequest contains parameters for AI-powered template generation.
type AITemplateGenerateRequest struct {
	HTML         string         `json:"html"`
	URL          string         `json:"url"`
	Description  string         `json:"description"`
	SampleFields []string       `json:"sample_fields,omitempty"`
	Feedback     string         `json:"feedback,omitempty"`
	Images       []AIImageInput `json:"images,omitempty"`
}

// AITemplateGenerateResult contains the generated template and model metadata.
type AITemplateGenerateResult struct {
	Template    Template `json:"template"`
	Explanation string   `json:"explanation,omitempty"`
	RouteID     string   `json:"route_id,omitempty"`
	Provider    string   `json:"provider,omitempty"`
	Model       string   `json:"model,omitempty"`
}

// LLMProvider defines the interface for LLM operations.
type LLMProvider interface {
	Extract(ctx context.Context, req AIExtractRequest) (AIExtractResult, error)
	GenerateTemplate(ctx context.Context, req AITemplateGenerateRequest) (AITemplateGenerateResult, error)
	HealthCheck(ctx context.Context) error
	RouteFingerprint(capability string) string
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
	return cfg.Enabled
}

// CapabilityForExtractMode maps extraction modes to pi routing capabilities.
func CapabilityForExtractMode(mode AIExtractionMode) string {
	switch mode {
	case AIModeSchemaGuided:
		return config.AICapabilityExtractSchema
	default:
		return config.AICapabilityExtractNatural
	}
}
