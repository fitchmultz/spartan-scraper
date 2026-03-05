// Tests for the Export API-level behavior in the exporter package.
//
// This file tests API-level behavior that applies across all export formats:
// - Unsupported format error handling
//
// Format-specific tests (JSON, JSONL, Markdown, CSV) are in their
// respective test files.
package exporter

import (
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExportUnsupportedFormat(t *testing.T) {
	raw := []byte(sampleScrapeResultJSONL())
	job := model.Job{Kind: model.KindScrape}

	_, err := Export(job, raw, "xml")
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected 'unsupported format' error, got: %v", err)
	}
}
