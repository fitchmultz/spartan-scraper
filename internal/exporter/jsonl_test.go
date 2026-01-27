// Tests for JSONL export functionality in the exporter package.
//
// This file tests the Export() and ExportStream() functions with
// "jsonl" format, covering:
// - Scrape, Crawl, and Research job kinds
// - JSONL output remains unchanged (pass-through)
// - Proper newline-delimited JSON validation
// - Error handling for writer failures
//
// The JSONL export simply passes through JSONL input with a trailing
// newline added. It does NOT convert to JSON objects or arrays.
package exporter

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExportJSONLRemainsUnchanged(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  []byte
	}{
		{
			name: "Scrape job JSONL",
			kind: model.KindScrape,
			raw:  []byte(sampleScrapeResultJSONL()),
		},
		{
			name: "Crawl job JSONL",
			kind: model.KindCrawl,
			raw:  []byte(sampleCrawlResultJSONL(2)),
		},
		{
			name: "Research job JSONL",
			kind: model.KindResearch,
			raw:  []byte(sampleResearchResultJSONL()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}
			original := string(tt.raw)

			result, err := Export(job, tt.raw, "jsonl")
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			normalizedResult := strings.TrimRight(result, "\n")
			normalizedOriginal := strings.TrimRight(original, "\n")

			if normalizedResult != normalizedOriginal {
				t.Errorf("JSONL output content should be unchanged from input\nExpected: %q\nGot: %q", normalizedOriginal, normalizedResult)
			}

			lines := strings.Split(strings.TrimSpace(result), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					var v interface{}
					if err := json.Unmarshal([]byte(line), &v); err != nil {
						t.Errorf("Line is not valid JSON: %q", line)
					}
				}
			}
		})
	}
}

func TestExportStreamJSONL(t *testing.T) {
	raw := sampleScrapeResultJSONL()
	job := model.Job{Kind: model.KindScrape}
	var buf bytes.Buffer

	err := ExportStream(job, strings.NewReader(raw), "jsonl", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed: %v", err)
	}

	result := buf.String()
	if result != raw+"\n" {
		t.Errorf("JSONL output mismatch\nExpected: %q\nGot: %q", raw+"\n", result)
	}
}

func TestExportStreamErrorHandlingWriterError(t *testing.T) {
	job := model.Job{Kind: model.KindScrape}
	raw := sampleScrapeResultJSONL()

	errWriter := &errorWriter{err: io.ErrClosedPipe}
	err := ExportStream(job, strings.NewReader(raw), "jsonl", errWriter)
	if err == nil {
		t.Error("Expected error from errorWriter, got nil")
	}
}

func TestExportStreamMatchesExportJSONL(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  string
	}{
		{"Scrape", model.KindScrape, sampleScrapeResultJSONL()},
		{"Crawl", model.KindCrawl, sampleCrawlResultJSONL(3)},
		{"Research", model.KindResearch, sampleResearchResultJSONL()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}

			resultOld, err := Export(job, []byte(tt.raw), "jsonl")
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			var buf bytes.Buffer
			err = ExportStream(job, strings.NewReader(tt.raw), "jsonl", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}
			resultNew := buf.String()

			resultOld = strings.TrimRight(resultOld, "\n")
			resultNew = strings.TrimRight(resultNew, "\n")

			if resultOld != resultNew {
				t.Errorf("Export and ExportStream produced different output\nOld (len=%d): %q\nNew (len=%d): %q",
					len(resultOld), resultOld, len(resultNew), resultNew)
			}
		})
	}
}
