// Package exporter provides exporter functionality for Spartan Scraper.
//
// Purpose:
// - Implement result export support for package exporter.
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
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// ResultExportConfig is the canonical direct-export contract shared by API,
// CLI, MCP, and Web-triggered saved-result exports.
type ResultExportConfig struct {
	Format    string          `json:"format,omitempty"`
	Shape     ShapeConfig     `json:"shape,omitempty"`
	Transform TransformConfig `json:"transform,omitempty"`
}

func NormalizeResultExportConfig(config ResultExportConfig) ResultExportConfig {
	config.Format = strings.TrimSpace(config.Format)
	if config.Format == "" {
		config.Format = "jsonl"
	}
	config.Shape = NormalizeShapeConfig(config.Shape)
	config.Transform = NormalizeTransformConfig(config.Transform)
	return config
}

func ValidateResultExportConfig(config ResultExportConfig) error {
	config = NormalizeResultExportConfig(config)
	if !isSupportedResultExportFormat(config.Format) {
		return apperrors.Validation("export format must be jsonl, json, md, csv, or xlsx")
	}
	if HasMeaningfulShape(config.Shape) && !SupportsShapeFormat(config.Format) {
		return apperrors.Validation("export shape is supported only for md, csv, and xlsx formats")
	}
	if HasMeaningfulTransform(config.Transform) {
		if err := ValidateTransformConfig(config.Transform); err != nil {
			return err
		}
	}
	if HasMeaningfulShape(config.Shape) && HasMeaningfulTransform(config.Transform) {
		return apperrors.Validation("export shape and transform cannot be combined")
	}
	return nil
}

func ResultExportContentType(format string) string {
	switch NormalizeResultExportConfig(ResultExportConfig{Format: format}).Format {
	case "json":
		return "application/json"
	case "jsonl":
		return "application/x-ndjson"
	case "md":
		return "text/markdown; charset=utf-8"
	case "csv":
		return "text/csv; charset=utf-8"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return "application/octet-stream"
	}
}

func ResultExportFilename(job model.Job, config ResultExportConfig) string {
	config = NormalizeResultExportConfig(config)
	return fmt.Sprintf("%s.%s", job.ID, config.Format)
}

func ResultExportIsBinary(format string) bool {
	return NormalizeResultExportConfig(ResultExportConfig{Format: format}).Format == "xlsx"
}

func ExportResult(job model.Job, raw []byte, config ResultExportConfig) ([]byte, error) {
	var buf bytes.Buffer
	if err := ExportResultStream(job, bytes.NewReader(raw), config, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ExportResultStream(job model.Job, r io.Reader, config ResultExportConfig, w io.Writer) error {
	config = NormalizeResultExportConfig(config)
	if err := ValidateResultExportConfig(config); err != nil {
		return err
	}
	return ExportStreamWithShapeAndTransform(job, r, config.Format, config.Shape, config.Transform, w)
}

func isSupportedResultExportFormat(format string) bool {
	switch strings.TrimSpace(format) {
	case "json", "jsonl", "md", "csv", "xlsx":
		return true
	default:
		return false
	}
}
