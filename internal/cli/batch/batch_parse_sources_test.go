// Package batch contains scenario-focused tests for the batch subcommand.
//
// Purpose:
// - Verify batch URL and JSON source parsing helpers.
//
// Responsibilities:
// - Cover inline URL parsing, JSON-file parsing, format detection, and generic source errors.
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
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseBatchJobs_FromURLs(t *testing.T) {
	jobs, err := parseBatchJobs("", "https://a.com,https://b.com", "GET", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].URL != "https://a.com" {
		t.Errorf("expected first URL https://a.com, got %s", jobs[0].URL)
	}
	if jobs[0].Method != "GET" {
		t.Errorf("expected method GET, got %s", jobs[0].Method)
	}
}
func TestParseBatchJobs_FromURLsWithSpaces(t *testing.T) {
	jobs, err := parseBatchJobs("", "  https://a.com  ,  https://b.com  ", "POST", "body", "application/json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].URL != "https://a.com" {
		t.Errorf("expected trimmed URL, got %s", jobs[0].URL)
	}
}
func TestParseBatchJobs_FromJSONFile(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	jobs := []BatchJobRequest{
		{URL: "https://example.com", Method: "GET"},
		{URL: "https://example.org", Method: "POST", Body: "test", ContentType: "text/plain"},
	}
	data, _ := json.Marshal(jobs)
	if _, err := tempFile.Write(data); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	parsed, err := parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(parsed))
	}
	if parsed[1].ContentType != "text/plain" {
		t.Errorf("expected content type text/plain, got %s", parsed[1].ContentType)
	}
}
func TestParseBatchJobs_UnsupportedFormat(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	if _, err := tempFile.WriteString("not json or csv"); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err == nil {
		t.Fatal("expected error for unsupported file format")
	}
	if !strings.Contains(err.Error(), "unsupported file format") {
		t.Errorf("expected 'unsupported file format' error, got %v", err)
	}
}
func TestParseBatchJobs_FileNotFound(t *testing.T) {
	_, err := parseBatchJobs("/nonexistent/path/file.json", "", "", "", "")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}
func TestParseBatchJobs_InvalidJSON(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	if _, err := tempFile.WriteString("{invalid json}"); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
func TestParseBatchJobs_EmptyFile(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "batch-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	// Write only header
	if _, err := tempFile.WriteString("url\n"); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	parsed, err := parseBatchJobs(tempFile.Name(), "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("expected 0 jobs for empty data, got %d", len(parsed))
	}
}
func TestLooksLikeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`[]`, true},
		{`{}`, true},
		{`[{"url":"test"}]`, true},
		{`{"url":"test"}`, true},
		{`  ["test"]`, true},
		{`not json`, false},
		{``, false},
		{`   `, false},
		{`url,method`, false},
	}

	for _, tt := range tests {
		result := looksLikeJSON([]byte(tt.input))
		if result != tt.expected {
			t.Errorf("looksLikeJSON(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}
