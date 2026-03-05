// Package-level test helpers for the exporter package.
//
// Provides shared test data generators and error types used across
// all exporter test files. This file exists to reduce duplication
// and keep test utilities in one place.
//
// This file does NOT contain any test functions itself - only helper
// functions and types used by other test files.
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
