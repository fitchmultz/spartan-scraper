// Package api provides tests for traffic replay utility functions.
//
// This file contains tests for utility functions including body diff preview,
// string truncation, and loading intercepted entries from files.
package api

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestGenerateBodyDiffPreview(t *testing.T) {
	tests := []struct {
		name     string
		original string
		replayed string
		want     string
	}{
		{
			name:     "identical bodies",
			original: "hello world",
			replayed: "hello world",
			want:     "Bodies are identical",
		},
		{
			name:     "different length",
			original: "short",
			replayed: "short!",
			want:     "Length differs: 5 vs 6",
		},
		{
			name:     "different content",
			original: "hello world this is a test string",
			replayed: "hello world that is a test string",
			want:     "Diff at position",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateBodyDiffPreview(tt.original, tt.replayed)
			if !strings.Contains(got, tt.want) {
				t.Errorf("generateBodyDiffPreview() = %q, want to contain %q", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"test", 4, "test"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestLoadInterceptedEntries(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create result file with multiple lines (JSONL format) in server's data directory
	jobID := "test-job-entries"
	jobDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	os.MkdirAll(jobDir, 0755)
	resultPath := filepath.Join(jobDir, "results.jsonl")

	entries := []struct {
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		{
			InterceptedData: []fetch.InterceptedEntry{
				{
					Request: fetch.InterceptedRequest{
						RequestID: "req-1",
						URL:       "https://example.com/api/1",
						Method:    "GET",
					},
				},
			},
		},
		{
			InterceptedData: []fetch.InterceptedEntry{
				{
					Request: fetch.InterceptedRequest{
						RequestID: "req-2",
						URL:       "https://example.com/api/2",
						Method:    "POST",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	for _, e := range entries {
		data, _ := json.Marshal(e)
		buf.Write(data)
		buf.WriteByte('\n')
	}
	os.WriteFile(resultPath, buf.Bytes(), 0644)

	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
	}

	loaded, err := srv.loadInterceptedEntries(job)
	if err != nil {
		t.Fatalf("failed to load entries: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 entries, got %d", len(loaded))
	}
}

func TestLoadInterceptedEntriesMissingFile(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Use a valid path within data directory but for a file that doesn't exist
	jobID := "test-job-missing"
	jobDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	os.MkdirAll(jobDir, 0755)
	resultPath := filepath.Join(jobDir, "results.jsonl")

	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
	}

	_, err := srv.loadInterceptedEntries(job)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadInterceptedEntriesNoResultPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	job := model.Job{
		ID:     "test-job",
		Kind:   model.KindScrape,
		Status: model.StatusSucceeded,
		// No ResultPath
	}

	_, err := srv.loadInterceptedEntries(job)
	if err == nil {
		t.Error("expected error for missing result path")
	}
}
