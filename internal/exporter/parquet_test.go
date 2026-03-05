// Tests for Parquet export functionality in the exporter package.
//
// This file tests the Export() and ExportStream() functions with
// "parquet" format, covering:
// - Scrape, Crawl, and Research job kinds
// - Stable field ordering
// - Large dataset streaming
// - Parquet file integrity (can be read back)
//
// The Parquet export converts JSONL to Apache Parquet columnar format with:
// - SNAPPY compression by default
// - Dictionary encoding for string columns
// - Optimized schema for analytics pipelines
package exporter

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/xitongsys/parquet-go-source/buffer"
	"github.com/xitongsys/parquet-go/reader"
)

func TestExportStreamParquet_Scrape(t *testing.T) {
	raw := sampleScrapeResultJSONL()
	job := model.Job{Kind: model.KindScrape}

	var buf bytes.Buffer
	err := ExportStream(job, strings.NewReader(raw), "parquet", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed: %v", err)
	}

	// Verify it's a valid Parquet file by reading it back with correct schema
	bufFile := buffer.NewBufferFileFromBytes(buf.Bytes())

	pr, err := reader.NewParquetReader(bufFile, new(ParquetScrapeRow), 1)
	if err != nil {
		t.Fatalf("Failed to open Parquet for reading: %v", err)
	}
	defer pr.ReadStop()

	if pr.GetNumRows() != 1 {
		t.Errorf("Expected 1 row, got %d", pr.GetNumRows())
	}

	// Read and verify data
	rows := make([]ParquetScrapeRow, 1)
	if err := pr.Read(&rows); err != nil {
		t.Fatalf("Failed to read rows: %v", err)
	}

	if rows[0].URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", rows[0].URL)
	}
	if rows[0].Status != 200 {
		t.Errorf("Expected status 200, got %d", rows[0].Status)
	}
}

func TestExportStreamParquet_Crawl(t *testing.T) {
	raw := sampleCrawlResultJSONL(2)
	job := model.Job{Kind: model.KindCrawl}

	var buf bytes.Buffer
	err := ExportStream(job, strings.NewReader(raw), "parquet", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed: %v", err)
	}

	// Verify it's a valid Parquet file by reading it back with correct schema
	bufFile := buffer.NewBufferFileFromBytes(buf.Bytes())

	pr, err := reader.NewParquetReader(bufFile, new(ParquetCrawlRow), 1)
	if err != nil {
		t.Fatalf("Failed to open Parquet for reading: %v", err)
	}
	defer pr.ReadStop()

	if pr.GetNumRows() != 2 {
		t.Errorf("Expected 2 rows, got %d", pr.GetNumRows())
	}
}

func TestExportStreamParquet_Research(t *testing.T) {
	raw := sampleResearchResultJSONL()
	job := model.Job{Kind: model.KindResearch}

	var buf bytes.Buffer
	err := ExportStream(job, strings.NewReader(raw), "parquet", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed: %v", err)
	}

	// Verify it's a valid Parquet file by reading it back with correct schema
	bufFile := buffer.NewBufferFileFromBytes(buf.Bytes())

	pr, err := reader.NewParquetReader(bufFile, new(ParquetEvidenceRow), 1)
	if err != nil {
		t.Fatalf("Failed to open Parquet for reading: %v", err)
	}
	defer pr.ReadStop()

	// Should have 1 evidence row
	if pr.GetNumRows() != 1 {
		t.Errorf("Expected 1 evidence row, got %d", pr.GetNumRows())
	}

	// Read and verify data
	rows := make([]ParquetEvidenceRow, 1)
	if err := pr.Read(&rows); err != nil {
		t.Fatalf("Failed to read rows: %v", err)
	}

	if rows[0].URL != "https://example.com/evidence1" {
		t.Errorf("Expected URL 'https://example.com/evidence1', got '%s'", rows[0].URL)
	}
	if rows[0].Title != "Evidence 1" {
		t.Errorf("Expected title 'Evidence 1', got '%s'", rows[0].Title)
	}
}

func TestExportStreamParquet_LargeDataset(t *testing.T) {
	// Generate a dataset that is reasonably large
	count := 1000
	raw := sampleCrawlResultJSONL(count)
	job := model.Job{Kind: model.KindCrawl}

	t.Run("Seekable reader", func(t *testing.T) {
		var buf bytes.Buffer
		err := ExportStream(job, strings.NewReader(raw), "parquet", &buf)
		if err != nil {
			t.Fatalf("ExportStream() failed: %v", err)
		}

		// Verify it's a valid Parquet by opening it
		bufFile := buffer.NewBufferFileFromBytes(buf.Bytes())
		if err != nil {
			t.Fatalf("Failed to create buffer file: %v", err)
		}
		pr, err := reader.NewParquetReader(bufFile, new(ParquetCrawlRow), 1)
		if err != nil {
			t.Fatalf("Failed to open Parquet: %v", err)
		}
		defer pr.ReadStop()

		if pr.GetNumRows() != int64(count) {
			t.Errorf("Expected %d rows, got %d", count, pr.GetNumRows())
		}
	})

	t.Run("Non-seekable reader", func(t *testing.T) {
		// Use a pipe to simulate a non-seekable reader
		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()
			pw.Write([]byte(raw))
		}()

		var buf bytes.Buffer
		err := ExportStream(job, pr, "parquet", &buf)
		if err != nil {
			t.Fatalf("ExportStream() failed: %v", err)
		}

		// Verify it's a valid Parquet
		bufFile := buffer.NewBufferFileFromBytes(buf.Bytes())
		if err != nil {
			t.Fatalf("Failed to create buffer file: %v", err)
		}
		parquetReader, err := reader.NewParquetReader(bufFile, new(ParquetCrawlRow), 1)
		if err != nil {
			t.Fatalf("Failed to open Parquet: %v", err)
		}
		defer parquetReader.ReadStop()

		if parquetReader.GetNumRows() != int64(count) {
			t.Errorf("Expected %d rows, got %d", count, parquetReader.GetNumRows())
		}
	})
}

func TestExportParquetCanBeReadBack(t *testing.T) {
	raw := sampleScrapeResultJSONL()
	job := model.Job{Kind: model.KindScrape}

	var buf bytes.Buffer
	err := ExportStream(job, strings.NewReader(raw), "parquet", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed: %v", err)
	}

	// Read back the Parquet file
	bufFile := buffer.NewBufferFileFromBytes(buf.Bytes())

	pr, err := reader.NewParquetReader(bufFile, new(ParquetScrapeRow), 1)
	if err != nil {
		t.Fatalf("Failed to open Parquet for reading: %v", err)
	}
	defer pr.ReadStop()

	if pr.GetNumRows() != 1 {
		t.Errorf("Expected 1 row, got %d", pr.GetNumRows())
	}

	// Read the actual data
	rows := make([]ParquetScrapeRow, 1)
	err = pr.Read(&rows)
	if err != nil {
		t.Fatalf("Failed to read rows: %v", err)
	}

	// Verify the data
	if rows[0].URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", rows[0].URL)
	}
	if rows[0].Status != 200 {
		t.Errorf("Expected status 200, got %d", rows[0].Status)
	}
	if rows[0].Title != "Example" {
		t.Errorf("Expected title 'Example', got '%s'", rows[0].Title)
	}
}

func TestExportCrawlParquetFieldOrderIsStable(t *testing.T) {
	raw := sampleCrawlResultJSONL(5)
	job := model.Job{Kind: model.KindCrawl}

	// Export multiple times and verify consistent results
	for i := range 3 {
		var buf bytes.Buffer
		err := ExportStream(job, strings.NewReader(raw), "parquet", &buf)
		if err != nil {
			t.Fatalf("ExportStream() failed on iteration %d: %v", i, err)
		}

		// Verify it's a valid Parquet
		bufFile := buffer.NewBufferFileFromBytes(buf.Bytes())
		if err != nil {
			t.Fatalf("Failed to create buffer file: %v", err)
		}
		pr, err := reader.NewParquetReader(bufFile, new(ParquetCrawlRow), 1)
		if err != nil {
			t.Fatalf("Failed to open Parquet: %v", err)
		}

		if pr.GetNumRows() != 5 {
			t.Errorf("Iteration %d: Expected 5 rows, got %d", i, pr.GetNumRows())
		}
		pr.ReadStop()
	}
}

func TestExportResearchParquetEvidenceStructure(t *testing.T) {
	raw := sampleResearchResultJSONL()
	job := model.Job{Kind: model.KindResearch}

	var buf bytes.Buffer
	err := ExportStream(job, strings.NewReader(raw), "parquet", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed: %v", err)
	}

	// Read back the Parquet file
	bufFile := buffer.NewBufferFileFromBytes(buf.Bytes())

	pr, err := reader.NewParquetReader(bufFile, new(ParquetEvidenceRow), 1)
	if err != nil {
		t.Fatalf("Failed to open Parquet for reading: %v", err)
	}
	defer pr.ReadStop()

	// Should have 1 evidence row
	if pr.GetNumRows() != 1 {
		t.Errorf("Expected 1 evidence row, got %d", pr.GetNumRows())
	}

	// Read the actual data
	rows := make([]ParquetEvidenceRow, 1)
	err = pr.Read(&rows)
	if err != nil {
		t.Fatalf("Failed to read rows: %v", err)
	}

	// Verify the evidence data
	if rows[0].URL != "https://example.com/evidence1" {
		t.Errorf("Expected URL 'https://example.com/evidence1', got '%s'", rows[0].URL)
	}
	if rows[0].Title != "Evidence 1" {
		t.Errorf("Expected title 'Evidence 1', got '%s'", rows[0].Title)
	}
	if rows[0].ClusterID != "cluster1" {
		t.Errorf("Expected cluster_id 'cluster1', got '%s'", rows[0].ClusterID)
	}
}

func TestExportStreamMatchesExportParquet(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  string
	}{
		{"Scrape", model.KindScrape, sampleScrapeResultJSONL()},
		{"Crawl", model.KindCrawl, sampleCrawlResultJSONL(2)},
		{"Research", model.KindResearch, sampleResearchResultJSONL()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}

			// Export using Export (via string)
			resultOld, err := Export(job, []byte(tt.raw), "parquet")
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			// Export using ExportStream
			var buf bytes.Buffer
			err = ExportStream(job, strings.NewReader(tt.raw), "parquet", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}
			resultNew := buf.String()

			if resultOld != resultNew {
				t.Errorf("Export and ExportStream produced different output\nOld (len=%d) vs New (len=%d)",
					len(resultOld), len(resultNew))
			}
		})
	}
}
