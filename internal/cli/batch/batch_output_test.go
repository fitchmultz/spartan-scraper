// Package batch contains scenario-focused tests for the batch subcommand.
//
// Purpose:
// - Verify batch output formatting helpers.
//
// Responsibilities:
// - Cover terminal-status detection plus status and list rendering.
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
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	spartanapi "github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

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
		Batch: BatchSummary{
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
			Progress:  spartanapi.BatchProgress{Completed: 5, Remaining: 0, Percent: 100},
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		},
		Jobs: []spartanapi.InspectableJob{},
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
		Batch: BatchSummary{
			ID:       "test-batch-123",
			Kind:     "scrape",
			Status:   "completed",
			JobCount: 1,
			Stats: model.BatchJobStats{
				Succeeded: 1,
			},
			Progress: spartanapi.BatchProgress{Completed: 1, Remaining: 0, Percent: 100},
		},
		Jobs: []spartanapi.InspectableJob{
			{Job: model.Job{ID: "job-1", Kind: "scrape", Status: "succeeded"}},
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
func TestPrintBatchList(t *testing.T) {
	result := &BatchListResponse{
		Batches: []BatchSummary{{
			ID:       "test-batch-123",
			Kind:     "scrape",
			Status:   "processing",
			JobCount: 3,
			Stats: model.BatchJobStats{
				Queued:    1,
				Running:   1,
				Succeeded: 1,
			},
			Progress: spartanapi.BatchProgress{Completed: 1, Remaining: 2, Percent: 33},
		}},
		Total:  1,
		Limit:  100,
		Offset: 0,
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printBatchList(result)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "test-batch-123") {
		t.Errorf("expected output to contain batch ID, got %q", output)
	}
	if !strings.Contains(output, "processing") {
		t.Errorf("expected output to contain batch status, got %q", output)
	}
	if !strings.Contains(output, "showing 1 of 1") {
		t.Errorf("expected output to contain pagination summary, got %q", output)
	}
}
