// Package exporter renders stable direct-export artifacts and delivery metadata.
//
// Purpose:
//   - Centralize buffered result-export rendering for workflows that need the exact
//     rendered bytes plus metadata such as filename, content type, size, and
//     record count.
//
// Responsibilities:
// - Validate and normalize the shared direct-export contract.
// - Render exported bytes for the requested format/shape/transform.
// - Derive stable export metadata used by webhook delivery and API handlers.
//
// Scope:
// - Buffered export rendering only.
//
// Usage:
// - Used by direct export handlers and scheduled webhook exports.
//
// Invariants/Assumptions:
//   - Returned content bytes match the shared direct-export contract exactly.
//   - RecordCount reflects the number of logical result records after any
//     configured transform and before format-specific serialization.
package exporter

import (
	"bytes"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// RenderedResultExport is a fully rendered export artifact plus stable metadata.
type RenderedResultExport struct {
	Format      string
	Filename    string
	ContentType string
	Content     []byte
	RecordCount int
	Size        int64
}

// RenderResultExport renders a result export into bytes together with delivery metadata.
func RenderResultExport(job model.Job, raw []byte, config ResultExportConfig) (RenderedResultExport, error) {
	config = NormalizeResultExportConfig(config)
	if err := ValidateResultExportConfig(config); err != nil {
		return RenderedResultExport{}, err
	}

	recordCount, err := resultExportRecordCount(job, raw, config)
	if err != nil {
		return RenderedResultExport{}, err
	}

	content, err := ExportResult(job, raw, config)
	if err != nil {
		return RenderedResultExport{}, err
	}

	return RenderedResultExport{
		Format:      config.Format,
		Filename:    ResultExportFilename(job, config),
		ContentType: ResultExportContentType(config.Format),
		Content:     content,
		RecordCount: recordCount,
		Size:        int64(len(content)),
	}, nil
}

func resultExportRecordCount(job model.Job, raw []byte, config ResultExportConfig) (int, error) {
	results, err := loadResultsForTransform(job, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	if HasMeaningfulTransform(config.Transform) {
		transformed, err := ApplyTransformConfig(results, config.Transform)
		if err != nil {
			return 0, err
		}
		return len(transformed), nil
	}
	return len(results), nil
}
