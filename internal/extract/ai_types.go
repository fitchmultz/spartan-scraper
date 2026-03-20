// Package extract defines the AI-provider contracts used by Spartan extraction flows.
//
// Purpose:
// - Hold shared request, result, cache, and provider types for AI-backed extraction.
//
// Responsibilities:
// - Define transport-safe AI extract and template payloads.
// - Expose the provider interface consumed by extract, research, and API layers.
// - Map product-facing extraction modes to configured AI routing capabilities.
//
// Scope:
// - Shared AI extraction types only; orchestration and bridge implementations live elsewhere.
//
// Usage:
// - Imported by provider implementations, extract orchestration, and tests.
//
// Invariants/Assumptions:
// - Provider implementations return normalized field values and health snapshots.
// - AI capability routing stays aligned with config capability constants.
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
	Prompt          string                 `json:"prompt,omitempty"`
	SchemaExample   map[string]interface{} `json:"schema_example,omitempty"`
	Fields          []string               `json:"fields,omitempty"`
	Images          []AIImageInput         `json:"images,omitempty"`
	MaxContentChars int                    `json:"max_content_chars,omitempty"`
}

// AIExtractResult contains the AI extraction output.
type AIExtractResult struct {
	Fields      map[string]FieldValue `json:"fields"`
	Confidence  float64               `json:"confidence"`
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
	HealthStatus(ctx context.Context) (AIHealthSnapshot, error)
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
