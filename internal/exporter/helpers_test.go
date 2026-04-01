// Package exporter provides exporter functionality for Spartan Scraper.
//
// Purpose:
// - Verify helpers test behavior for package exporter.
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
	"fmt"
	"strings"
)

func sampleScrapeResultJSONL() string {
	return `{"url":"https://example.com","status":200,"title":"Example Page","text":"Content here","metadata":{"description":"A test page"},"normalized":{"title":"Example","description":"Test description","text":"Normalized text","fields":{}}}`
}

func sampleCrawlResultJSONL(count int) string {
	var lines []string
	for i := 0; i < count; i++ {
		lines = append(lines, fmt.Sprintf(`{"url":"https://example.com/page%d","status":200,"title":"Page %d","text":"Content %d","normalized":{"title":"Page %d","text":"Text %d","fields":{}}}`, i+1, i+1, i+1, i+1, i+1))
	}
	return strings.Join(lines, "\n")
}

func sampleResearchResultJSONL() string {
	return `{"query":"test query","summary":"Test summary","confidence":0.95,"evidence":[{"url":"https://example.com/evidence1","title":"Evidence 1","snippet":"Test snippet","score":0.9,"simhash":1234567890,"clusterId":"cluster1","confidence":0.9,"citationUrl":"https://example.com/cite1"}],"clusters":[{"id":"cluster1","label":"Test Cluster","confidence":0.9,"evidence":[]}],"citations":[{"url":"https://example.com/cite1","anchor":"section1","canonical":"https://example.com/canonical1"}]}`
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}
