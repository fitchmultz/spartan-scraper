// Package api provides integration tests for job results endpoint pagination.
// Tests cover pagination parameters (limit, offset), X-Total-Count header, edge cases,
// and data integrity across pages.
// Does NOT test result file generation or export logic handled by exporter package.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleJobResultsWithPagination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	jobID := "00000000-0000-4000-8000-000000000001"

	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	var resultLines []string
	for i := 1; i <= 150; i++ {
		resultLines = append(resultLines, fmt.Sprintf(`{"url":"https://example.com/page%d","status":200,"title":"Page %d"}`, i, i))
	}
	resultContent := strings.Join(resultLines, "\n")
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: resultPath,
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job in store: %v", err)
	}

	req := httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=0", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200, got %v", status)
	}

	totalCountStr := rr.Header().Get("X-Total-Count")
	if totalCountStr == "" {
		t.Error("expected X-Total-Count header")
	}
	totalCount, _ := strconv.Atoi(totalCountStr)
	if totalCount != 150 {
		t.Errorf("expected total count 150, got %d", totalCount)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(items) != 50 {
		t.Errorf("expected 50 items, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=50", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 50 {
		t.Errorf("expected 50 items with offset 50, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=100", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 50 {
		t.Errorf("expected 50 items with offset 100, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=150", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 0 {
		t.Errorf("expected 0 items with offset beyond total, got %d", len(items))
	}

	// Test negative limit returns 400
	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=-1&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected 400 for negative limit, got %v", status)
	}

	// Test non-numeric limit returns 400
	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=abc&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric limit, got %v", status)
	}

	// Test negative offset returns 400
	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=-1", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected 400 for negative offset, got %v", status)
	}

	// Test non-numeric offset returns 400
	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=xyz", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric offset, got %v", status)
	}

	// Test limit > 1000 is clamped to 1000 (still returns 200)
	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=2000&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200 for limit > 1000 (clamped), got %v", status)
	}

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 150 {
		t.Errorf("expected max limit of 1000 with limit > 1000, but only 150 items in file, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	firstTitle, _ := items[0]["title"].(string)
	if firstTitle != "Page 1" {
		t.Errorf("expected first item to be Page 1, got %v", firstTitle)
	}

	fiftiethTitle, _ := items[49]["title"].(string)
	if fiftiethTitle != "Page 50" {
		t.Errorf("expected 50th item to be Page 50, got %v", fiftiethTitle)
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=50", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	firstTitle, _ = items[0]["title"].(string)
	if firstTitle != "Page 51" {
		t.Errorf("expected first item on second page to be Page 51, got %v", firstTitle)
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=json&limit=50&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200 for json format, got %v", status)
	}

	if ct := rr.Header().Get("X-Total-Count"); ct != "" {
		t.Error("expected no X-Total-Count header for non-jsonl format")
	}
}
