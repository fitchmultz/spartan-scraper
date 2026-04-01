// Package batch contains scenario-focused tests for the batch subcommand.
//
// Purpose:
// - Verify batch CSV parsing helpers.
//
// Responsibilities:
// - Cover CSV headers, empty-line handling, quoted values, content-type variations, and missing-column validation.
//
// Scope:
// - Batch CLI behavior only.
//
// Usage:
// - Run with `go test ./internal/cli/batch`.
//
// Invariants/Assumptions:
// - Tests use temp files and local stores only.
package batch

import (
	"os"
	"strings"
	"testing"
)

func TestParseBatchJobs_FromCSVFileWithHeaders(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	csvData := `url,method,body,contentType
https://example.com,GET,,
https://example.org,POST,test body,application/json`
	if _, err := tempFile.WriteString(csvData); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	parsed, err := parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(parsed))
	}
	if parsed[0].Method != "GET" {
		t.Errorf("expected method GET for first job, got %s", parsed[0].Method)
	}
	if parsed[1].Method != "POST" {
		t.Errorf("expected method POST for second job, got %s", parsed[1].Method)
	}
	if parsed[1].Body != "test body" {
		t.Errorf("expected body 'test body', got %s", parsed[1].Body)
	}
}
func TestParseBatchJobs_FromCSVFileWithoutHeaders(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	csvData := `https://example.com,GET
https://example.org,POST,body,content`
	if _, err := tempFile.WriteString(csvData); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	parsed, err := parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(parsed))
	}
	if parsed[0].URL != "https://example.com" {
		t.Errorf("expected URL https://example.com, got %s", parsed[0].URL)
	}
}
func TestParseBatchJobs_FromCSVFileEmptyLines(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	csvData := `url
https://example.com

https://example.org

`
	if _, err := tempFile.WriteString(csvData); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	parsed, err := parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 jobs (empty lines filtered), got %d", len(parsed))
	}
}
func TestParseBatchJobs_CSVWithQuotedValues(t *testing.T) {
	csvData := `url,body
https://example.com,"test, with, commas"
https://example.org,"another ""quoted"" value"`

	jobs, err := parseBatchJobsFromCSV([]byte(csvData))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}
func TestParseBatchJobs_CSVCaseInsensitiveHeaders(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	csvData := `URL,Method,BODY,ContentType
https://example.com,get,test,application/json`
	if _, err := tempFile.WriteString(csvData); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	parsed, err := parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 job, got %d", len(parsed))
	}
	if parsed[0].Method != "GET" {
		t.Errorf("expected method uppercase GET, got %s", parsed[0].Method)
	}
}
func TestParseBatchJobs_CSVContentTypeVariations(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	csvData := `url,content-type
https://example.com,application/json
https://example.org,text/html`
	if _, err := tempFile.WriteString(csvData); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	parsed, err := parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(parsed))
	}
	if parsed[0].ContentType != "application/json" {
		t.Errorf("expected content-type application/json, got %s", parsed[0].ContentType)
	}
}
func TestParseBatchJobsFromCSV_MissingURLColumn(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	csvData := `method,body
GET,test`
	if _, err := tempFile.WriteString(csvData); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = parseBatchJobsFromCSV([]byte(csvData))
	if err == nil {
		t.Fatal("expected error for missing URL column")
	}
	if !strings.Contains(err.Error(), "must have a 'url' column") {
		t.Errorf("expected 'url column' error, got %v", err)
	}
}
