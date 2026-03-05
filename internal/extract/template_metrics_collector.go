// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// This file implements metrics collection for template extraction performance.
//
// The collector tracks per-extraction metrics and aggregates them hourly for
// storage efficiency. It provides thread-safe operations for concurrent
// extraction pipelines.
package extract

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/google/uuid"
)

// TemplateMetricsCollector handles recording of template extraction metrics.
// It maintains an in-memory cache of hourly aggregates and periodically
// flushes to persistent storage.
type TemplateMetricsCollector struct {
	store *store.Store
	mu    sync.RWMutex
	cache map[string]*TemplateMetrics // key: "hour:templateName"
}

// NewTemplateMetricsCollector creates a new metrics collector.
func NewTemplateMetricsCollector(store *store.Store) *TemplateMetricsCollector {
	return &TemplateMetricsCollector{
		store: store,
		cache: make(map[string]*TemplateMetrics),
	}
}

// RecordExtraction records a single extraction result.
// This method is thread-safe and can be called concurrently.
func (c *TemplateMetricsCollector) RecordExtraction(
	ctx context.Context,
	templateName string,
	targetURL string,
	success bool,
	fieldCoverage float64,
	extractionTimeMs int64,
	validationErrors []string,
	extractedFields []string,
	testID *string,
) error {
	hour := time.Now().UTC().Truncate(time.Hour)
	cacheKey := hour.Format(time.RFC3339) + ":" + templateName

	// Update in-memory cache
	c.mu.Lock()
	metrics, exists := c.cache[cacheKey]
	if !exists {
		metrics = &TemplateMetrics{
			TemplateName: templateName,
			Hour:         hour,
		}
		c.cache[cacheKey] = metrics
	}

	metrics.ExtractionsTotal++
	if success {
		metrics.ExtractionsSuccess++
	}
	metrics.FieldCoverageSum += fieldCoverage
	metrics.FieldCoverageCount++
	metrics.TotalExtractionTimeMs += extractionTimeMs
	c.mu.Unlock()

	// Store per-extraction record if part of an A/B test
	if testID != nil {
		record := &store.TemplateExtractionRecord{
			ID:               uuid.New().String(),
			TestID:           testID,
			TemplateName:     templateName,
			TargetURL:        targetURL,
			Success:          success,
			FieldCoverage:    fieldCoverage,
			ExtractionTimeMs: extractionTimeMs,
			Timestamp:        time.Now().UTC(),
		}

		if len(validationErrors) > 0 {
			errorsJSON, _ := json.Marshal(validationErrors)
			record.ValidationErrors = string(errorsJSON)
		}
		if len(extractedFields) > 0 {
			fieldsJSON, _ := json.Marshal(extractedFields)
			record.ExtractedFields = string(fieldsJSON)
		}

		if err := c.store.RecordExtraction(ctx, record); err != nil {
			// Log but don't fail - extraction succeeded, metrics are secondary
			// In a production system, you might want to log this error
			_ = err
		}
	}

	return nil
}

// FlushCache writes all cached metrics to persistent storage.
// This should be called periodically (e.g., every 5 minutes) to ensure
// metrics are persisted before the process exits.
func (c *TemplateMetricsCollector) FlushCache(ctx context.Context) error {
	c.mu.Lock()
	cacheCopy := make(map[string]*TemplateMetrics, len(c.cache))
	for k, v := range c.cache {
		// Only flush hours that have passed (not the current hour)
		if v.Hour.Before(time.Now().UTC().Truncate(time.Hour)) {
			cacheCopy[k] = v
			delete(c.cache, k)
		}
	}
	c.mu.Unlock()

	for _, metrics := range cacheCopy {
		analyticsMetrics := &store.AnalyticsTemplateMetrics{
			Hour:                metrics.Hour,
			TemplateName:        metrics.TemplateName,
			ExtractionsTotal:    metrics.ExtractionsTotal,
			ExtractionsSuccess:  metrics.ExtractionsSuccess,
			FieldCoverageAvg:    metrics.FieldCoverage(),
			AvgExtractionTimeMs: float64(metrics.AvgExtractionTimeMs()),
		}

		if err := c.store.RecordTemplateMetrics(ctx, analyticsMetrics); err != nil {
			// Put back in cache to retry later
			c.mu.Lock()
			c.cache[metrics.Hour.Format(time.RFC3339)+":"+metrics.TemplateName] = metrics
			c.mu.Unlock()
			return err
		}
	}

	return nil
}

// GetCachedMetrics returns a copy of the current cached metrics.
// Useful for debugging and real-time monitoring.
func (c *TemplateMetricsCollector) GetCachedMetrics() []TemplateMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]TemplateMetrics, 0, len(c.cache))
	for _, m := range c.cache {
		result = append(result, *m)
	}
	return result
}

// CalculateFieldCoverage calculates the field coverage for a normalized document.
// It compares the extracted fields against the template's required fields.
func CalculateFieldCoverage(doc *NormalizedDocument, template *Template) float64 {
	if template.Schema == nil || len(template.Schema.Required) == 0 {
		// No required fields defined, consider it 100% coverage
		return 1.0
	}

	required := template.Schema.Required
	if len(required) == 0 {
		return 1.0
	}

	found := 0
	for _, field := range required {
		if _, ok := doc.Fields[field]; ok {
			found++
		}
	}

	return float64(found) / float64(len(required))
}

// ExtractedFieldNames returns the list of field names that were extracted.
func ExtractedFieldNames(doc *NormalizedDocument) []string {
	names := make([]string, 0, len(doc.Fields))
	for name := range doc.Fields {
		names = append(names, name)
	}
	return names
}
