// Package extract provides the main AI extraction orchestrator.
package extract

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// DefaultMaxContentChars is the default maximum HTML content to send to LLM.
const DefaultMaxContentChars = 100000 // ~25k tokens

// AIExtractor orchestrates LLM-based extraction with caching.
type AIExtractor struct {
	provider LLMProvider
	cache    AICache
	config   config.AIConfig
}

// NewAIExtractor creates an AI extractor with the given provider and config.
// Returns nil if AI is not configured.
func NewAIExtractor(cfg config.AIConfig) (*AIExtractor, error) {
	return NewAIExtractorWithDataDir(cfg, ".data")
}

// NewAIExtractorWithDataDir creates an AI extractor with the given config and data directory.
func NewAIExtractorWithDataDir(cfg config.AIConfig, dataDir string) (*AIExtractor, error) {
	if !IsAIEnabled(cfg) {
		return nil, nil
	}

	if err := ValidateProvider(cfg); err != nil {
		return nil, fmt.Errorf("invalid AI configuration: %w", err)
	}

	provider, err := CreateLLMProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Create cache with default TTL
	cache := NewFileAICache(dataDir, DefaultAICacheTTL)

	return &AIExtractor{
		provider: provider,
		cache:    cache,
		config:   cfg,
	}, nil
}

// NewAIExtractorWithProvider creates an AI extractor around a caller-supplied provider.
func NewAIExtractorWithProvider(cfg config.AIConfig, dataDir string, provider LLMProvider) *AIExtractor {
	return &AIExtractor{
		provider: provider,
		cache:    NewFileAICache(dataDir, DefaultAICacheTTL),
		config:   cfg,
	}
}

// Extract performs AI-powered extraction with caching.
func (a *AIExtractor) Extract(ctx context.Context, req AIExtractRequest) (AIExtractResult, error) {
	if a == nil || a.provider == nil {
		return AIExtractResult{}, fmt.Errorf("AI extractor not initialized")
	}

	// Set default max content chars if not specified
	if req.MaxContentChars <= 0 {
		req.MaxContentChars = DefaultMaxContentChars
	}

	// Pre-clean HTML to reduce token usage
	req.HTML = cleanHTMLForExtraction(req.HTML)

	// Generate cache key
	cacheKey := GenerateCacheKey(req, a.provider.RouteFingerprint(CapabilityForExtractMode(req.Mode)))

	// Check cache
	if cached, ok := a.cache.Get(cacheKey); ok {
		slog.Debug("AI extraction cache hit", "key", cacheKey[:16])
		return *cached, nil
	}

	// Call provider
	slog.Debug("AI extraction cache miss, calling pi bridge", "capability", CapabilityForExtractMode(req.Mode))
	result, err := a.provider.Extract(ctx, req)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Store in cache
	result.Cached = false
	a.cache.Set(cacheKey, &result)

	return result, nil
}

// GenerateTemplate performs AI-powered template generation without caching.
func (a *AIExtractor) GenerateTemplate(ctx context.Context, req AITemplateGenerateRequest) (AITemplateGenerateResult, error) {
	if a == nil || a.provider == nil {
		return AITemplateGenerateResult{}, fmt.Errorf("AI extractor not initialized")
	}

	req.HTML = cleanHTMLForExtraction(req.HTML)

	result, err := a.provider.GenerateTemplate(ctx, req)
	if err != nil {
		return AITemplateGenerateResult{}, fmt.Errorf("template generation failed: %w", err)
	}

	return result, nil
}

// HealthCheck checks if the AI provider is healthy.
func (a *AIExtractor) HealthCheck(ctx context.Context) error {
	if a == nil || a.provider == nil {
		return fmt.Errorf("AI extractor not initialized")
	}
	return a.provider.HealthCheck(ctx)
}

// cleanHTMLForExtraction removes unnecessary elements to reduce token usage.
func cleanHTMLForExtraction(html string) string {
	if len(html) == 0 {
		return html
	}

	// Remove script tags and content
	scriptRe := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")

	// Remove style tags and content
	styleRe := regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")

	// Remove noscript tags
	noscriptRe := regexp.MustCompile(`(?s)<noscript[^>]*>.*?</noscript>`)
	html = noscriptRe.ReplaceAllString(html, "")

	// Remove comments
	commentRe := regexp.MustCompile(`(?s)<!--.*?-->`)
	html = commentRe.ReplaceAllString(html, "")

	// Remove inline event handlers (onclick, onload, etc.)
	eventRe := regexp.MustCompile(`\son\w+="[^"]*"`)
	html = eventRe.ReplaceAllString(html, "")

	// Remove data attributes (often large)
	dataAttrRe := regexp.MustCompile(`\sdata-[\w-]+="[^"]*"`)
	html = dataAttrRe.ReplaceAllString(html, "")

	// Collapse multiple whitespace
	wsRe := regexp.MustCompile(`\s+`)
	html = wsRe.ReplaceAllString(html, " ")

	return strings.TrimSpace(html)
}

// truncateHTML limits HTML content to avoid token limits.
func TruncateHTML(html string, maxChars int) string {
	if len(html) <= maxChars {
		return html
	}

	// Try to truncate at a word boundary
	truncated := html[:maxChars]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > int(float64(maxChars)*0.8) { // Only truncate at word if not losing too much
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// GetCacheStats returns cache statistics.
func (a *AIExtractor) GetCacheStats() (total int, expired int) {
	if a == nil || a.cache == nil {
		return 0, 0
	}
	if fc, ok := a.cache.(*FileAICache); ok {
		return fc.Stats()
	}
	return 0, 0
}

// ClearCache clears the AI extraction cache.
func (a *AIExtractor) ClearCache() {
	if a == nil || a.cache == nil {
		return
	}
	if fc, ok := a.cache.(*FileAICache); ok {
		fc.Clear()
	}
}
