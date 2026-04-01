// Package exporter provides exporter functionality for Spartan Scraper.
//
// Purpose:
// - Implement shape support for package exporter.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `exporter` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package exporter

import (
	"fmt"
	"sort"
	"strings"
)

// ShapeConfig defines optional export-shaping directives for human-readable and
// tabular exports. The keys are deterministic field references derived from job
// results:
//   - top-level scrape/crawl fields: url, status, title, description, text
//   - top-level research fields: query, summary, confidence, agentic.status,
//     agentic.summary, agentic.objective, agentic.error
//   - normalized extraction fields: field.<name>
//   - research evidence fields: evidence.url, evidence.title, evidence.score,
//     evidence.confidence, evidence.clusterId, evidence.citationUrl,
//     evidence.snippet
//
// SummaryFields may reference any top-level or field.* key and are rendered as a
// deterministic summary list in Markdown exports.
type ShapeConfig struct {
	TopLevelFields   []string              `json:"topLevelFields,omitempty"`
	NormalizedFields []string              `json:"normalizedFields,omitempty"`
	EvidenceFields   []string              `json:"evidenceFields,omitempty"`
	SummaryFields    []string              `json:"summaryFields,omitempty"`
	FieldLabels      map[string]string     `json:"fieldLabels,omitempty"`
	Formatting       ExportFormattingHints `json:"formatting,omitempty"`
}

type ExportFormattingHints struct {
	EmptyValue     string `json:"emptyValue,omitempty"`
	MultiValueJoin string `json:"multiValueJoin,omitempty"`
	MarkdownTitle  string `json:"markdownTitle,omitempty"`
}

func NormalizeShapeConfig(shape ShapeConfig) ShapeConfig {
	shape.TopLevelFields = normalizeFieldRefs(shape.TopLevelFields)
	shape.NormalizedFields = normalizeFieldRefs(shape.NormalizedFields)
	shape.EvidenceFields = normalizeFieldRefs(shape.EvidenceFields)
	shape.SummaryFields = normalizeFieldRefs(shape.SummaryFields)
	shape.FieldLabels = normalizeFieldLabels(shape.FieldLabels)
	shape.Formatting.EmptyValue = strings.TrimSpace(shape.Formatting.EmptyValue)
	shape.Formatting.MultiValueJoin = strings.TrimSpace(shape.Formatting.MultiValueJoin)
	shape.Formatting.MarkdownTitle = strings.TrimSpace(shape.Formatting.MarkdownTitle)
	return shape
}

func HasMeaningfulShape(shape ShapeConfig) bool {
	shape = NormalizeShapeConfig(shape)
	return len(shape.TopLevelFields) > 0 ||
		len(shape.NormalizedFields) > 0 ||
		len(shape.EvidenceFields) > 0 ||
		len(shape.SummaryFields) > 0 ||
		len(shape.FieldLabels) > 0 ||
		shape.Formatting.EmptyValue != "" ||
		shape.Formatting.MultiValueJoin != "" ||
		shape.Formatting.MarkdownTitle != ""
}

func SupportsShapeFormat(format string) bool {
	switch strings.TrimSpace(format) {
	case "md", "csv", "xlsx":
		return true
	default:
		return false
	}
}

func normalizeFieldRefs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeFieldLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range labels {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		out[trimmedKey] = trimmedValue
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func shapeMultiValueJoin(shape ShapeConfig) string {
	if trimmed := strings.TrimSpace(shape.Formatting.MultiValueJoin); trimmed != "" {
		return trimmed
	}
	return "; "
}

func shapeEmptyValue(shape ShapeConfig) string {
	return strings.TrimSpace(shape.Formatting.EmptyValue)
}

func shapeMarkdownTitle(shape ShapeConfig) string {
	return strings.TrimSpace(shape.Formatting.MarkdownTitle)
}

func labelForShapeField(shape ShapeConfig, key string) string {
	if label, ok := NormalizeShapeConfig(shape).FieldLabels[key]; ok && strings.TrimSpace(label) != "" {
		return strings.TrimSpace(label)
	}
	return defaultShapeFieldLabel(key)
}

func defaultShapeFieldLabel(key string) string {
	trimmed := strings.TrimSpace(key)
	trimmed = strings.TrimPrefix(trimmed, "field.")
	trimmed = strings.TrimPrefix(trimmed, "evidence.")
	trimmed = strings.ReplaceAll(trimmed, ".", " ")
	trimmed = strings.ReplaceAll(trimmed, "_", " ")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return key
	}
	parts := strings.Fields(trimmed)
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func sortedNormalizedFieldRefs(fields map[string]fieldValueLike) []string {
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for key := range fields {
		out = append(out, "field."+key)
	}
	sort.Strings(out)
	return out
}

type fieldValueLike struct {
	Values []string
}

func selectFields(configured []string, defaults []string) []string {
	if len(configured) > 0 {
		return configured
	}
	return defaults
}

func joinValues(values []string, shape ShapeConfig) string {
	if len(values) == 0 {
		return shapeEmptyValue(shape)
	}
	return strings.Join(values, shapeMultiValueJoin(shape))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func formatInt(value int) string {
	return fmt.Sprintf("%d", value)
}

func nonEmptyFields(fields []string, resolve func(string) string) []string {
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(resolve(field)) != "" {
			out = append(out, field)
		}
	}
	return out
}
