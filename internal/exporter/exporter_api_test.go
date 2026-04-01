// Package exporter provides exporter functionality for Spartan Scraper.
//
// Purpose:
// - Verify exporter api test behavior for package exporter.
//
// Responsibilities:
// - Define focused Go test coverage, fixtures, and assertions for the package behavior exercised here.
//
// Scope:
// - Automated test coverage only; production behavior stays in non-test package files.
//
// Usage:
// - Run with `go test` for package `exporter` or through `make test-ci`/`make ci`.
//
// Invariants/Assumptions:
// - Tests should remain deterministic and describe the package contract they protect.

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
