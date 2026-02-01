// Package batch contains tests for the batch subcommand.
//
// Responsibilities:
// - Testing batch job parsing from CSV/JSON
// - Testing batch command validation (URL limits, required fields)
// - Testing batch status display and watching
//
// Non-goals:
// - Testing actual batch execution or HTTP requests
// - Testing API server behavior
//
// Assumptions:
// - Tests use temporary directories for file operations
// - Tests capture stdout/stderr for validation
// - No external network access required
package batch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
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

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"completed", true},
		{"failed", true},
		{"partial", true},
		{"canceled", true},
		{"pending", false},
		{"processing", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		result := isTerminalStatus(tt.status)
		if result != tt.expected {
			t.Errorf("isTerminalStatus(%q) = %v, expected %v", tt.status, result, tt.expected)
		}
	}
}

func TestPrintBatchStatus(t *testing.T) {
	status := &BatchStatusResponse{
		ID:       "test-batch-123",
		Kind:     "scrape",
		Status:   "completed",
		JobCount: 5,
		Stats: model.BatchJobStats{
			Queued:    0,
			Running:   0,
			Succeeded: 5,
			Failed:    0,
			Canceled:  0,
		},
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printBatchStatus(status)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "test-batch-123") {
		t.Errorf("expected output to contain batch ID, got %q", output)
	}
	if !strings.Contains(output, "scrape") {
		t.Errorf("expected output to contain kind, got %q", output)
	}
	if !strings.Contains(output, "completed") {
		t.Errorf("expected output to contain status, got %q", output)
	}
	if !strings.Contains(output, "5 total") {
		t.Errorf("expected output to contain job count, got %q", output)
	}
}

func TestPrintBatchStatusWithJobs(t *testing.T) {
	status := &BatchStatusResponse{
		ID:       "test-batch-123",
		Kind:     "scrape",
		Status:   "completed",
		JobCount: 1,
		Stats: model.BatchJobStats{
			Succeeded: 1,
		},
		Jobs: []JobInfo{
			{ID: "job-1", Kind: "scrape", Status: "succeeded"},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printBatchStatus(status)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Jobs:") {
		t.Errorf("expected output to contain 'Jobs:' header, got %q", output)
	}
	if !strings.Contains(output, "job-1") {
		t.Errorf("expected output to contain job ID, got %q", output)
	}
}

func TestRunBatch_NoSubcommand(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunBatch(ctx, cfg, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected help output, got %q", output)
	}
}

func TestRunBatch_Help(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunBatch(ctx, cfg, []string{"help"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "submit") {
		t.Errorf("expected help to mention submit subcommand, got %q", output)
	}
}

func TestRunBatch_UnknownSubcommand(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"unknown"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "unknown batch subcommand") {
		t.Errorf("expected error about unknown subcommand, got %q", stderr)
	}
}

func TestRunBatchSubmit_NoKind(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunBatch(ctx, cfg, []string{"submit"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected help output, got %q", output)
	}
}

func TestRunBatchSubmit_UnknownKind(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "unknown"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "unknown batch submit kind") {
		t.Errorf("expected error about unknown kind, got %q", stderr)
	}
}

func TestRunBatchSubmitScrape_NoURLs(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "scrape"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "no URLs provided") {
		t.Errorf("expected error about missing URLs, got %q", stderr)
	}
}

func TestRunBatchSubmitCrawl_NoURLs(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "crawl"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "no URLs provided") {
		t.Errorf("expected error about missing URLs, got %q", stderr)
	}
}

func TestRunBatchSubmitResearch_NoQuery(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "research", "--urls", "https://example.com"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "query is required") {
		t.Errorf("expected error about missing query, got %q", stderr)
	}
}

func TestRunBatchSubmitResearch_NoURLs(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "research", "--query", "test"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "no URLs provided") {
		t.Errorf("expected error about missing URLs, got %q", stderr)
	}
}

func TestRunBatchSubmitScrape_TooManyURLs(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	// Create URLs list with more than maxBatchSize
	urls := make([]string, maxBatchSize+1)
	for i := range urls {
		urls[i] = "https://example" + string(rune('a'+i%26)) + ".com"
	}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "scrape", "--urls", strings.Join(urls, ",")})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "exceeds maximum") {
		t.Errorf("expected error about batch size, got %q", stderr)
	}
}

func TestRunBatchStatus_NoBatchID(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"status"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "batch ID required") {
		t.Errorf("expected error about missing batch ID, got %q", stderr)
	}
}

func TestRunBatchCancel_NoBatchID(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"cancel"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "batch ID required") {
		t.Errorf("expected error about missing batch ID, got %q", stderr)
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
