// Package api provides unit tests for transformation helper functions.
// Tests cover loadJobResults and ApplyTransformation functions.
// Does NOT test HTTP handlers directly (those are in other test files).
package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestLoadJobResults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a job with results
	jobID := "test-job-load"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Params:     map[string]interface{}{"url": "https://example.com"},
		ResultPath: filepath.Join(srv.cfg.DataDir, "jobs", jobID, "results.jsonl"),
	}

	// Create results directory and file
	jobDir := filepath.Join(srv.cfg.DataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("failed to create job dir: %v", err)
	}

	// Write 15 test results
	file, err := os.Create(job.ResultPath)
	if err != nil {
		t.Fatalf("failed to create results file: %v", err)
	}
	for i := 0; i < 15; i++ {
		data, _ := json.Marshal(map[string]interface{}{
			"id":    i,
			"title": "Article " + string(rune('A'+i)),
		})
		file.WriteString(string(data) + "\n")
	}
	file.Close()

	// Test loading with limit
	results, err := srv.loadJobResults(job, 5)
	if err != nil {
		t.Fatalf("failed to load job results: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	// Test loading with higher limit than available
	results, err = srv.loadJobResults(job, 100)
	if err != nil {
		t.Fatalf("failed to load job results: %v", err)
	}

	if len(results) != 15 {
		t.Errorf("expected 15 results, got %d", len(results))
	}
}

func TestLoadJobResults_EmptyResultPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	job := model.Job{
		ID:         "test-empty",
		ResultPath: "",
	}

	results, err := srv.loadJobResults(job, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestApplyTransformation(t *testing.T) {
	testData := []interface{}{
		map[string]interface{}{"name": "Alice", "age": 30},
		map[string]interface{}{"name": "Bob", "age": 25},
	}

	t.Run("JMESPath projection", func(t *testing.T) {
		results, err := ApplyTransformation(testData, "{name: name}", "jmespath")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		// Verify each result only has name field
		for i, r := range results {
			result, ok := r.(map[string]interface{})
			if !ok {
				t.Fatalf("result %d is not an object", i)
			}
			if _, hasName := result["name"]; !hasName {
				t.Errorf("result %d missing name", i)
			}
			if _, hasAge := result["age"]; hasAge {
				t.Errorf("result %d should not have age", i)
			}
		}
	})

	t.Run("JSONata calculation", func(t *testing.T) {
		results, err := ApplyTransformation(testData, `{"person": name, "category": age > 25 ? "senior" : "junior"}`, "jsonata")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		// Verify conditional logic worked
		result0 := results[0].(map[string]interface{})
		if result0["category"] != "senior" {
			t.Errorf("expected senior, got %v", result0["category"])
		}

		result1 := results[1].(map[string]interface{})
		if result1["category"] != "junior" {
			t.Errorf("expected junior, got %v", result1["category"])
		}
	})

	t.Run("Empty data", func(t *testing.T) {
		results, err := ApplyTransformation([]interface{}{}, "{name: name}", "jmespath")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("Invalid expression", func(t *testing.T) {
		_, err := ApplyTransformation(testData, "{invalid", "jmespath")
		if err == nil {
			t.Error("expected error for invalid expression")
		}
	})
}
