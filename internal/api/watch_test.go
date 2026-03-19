// Package api provides HTTP handler tests for watch monitoring endpoints.
//
// Purpose:
// - Verify watch CRUD and manual-check behavior through the API layer.
//
// Responsibilities:
// - Assert defaults and validation for create/update requests.
// - Cover delete and manual-check not-found semantics.
// - Confirm watch payloads round-trip through storage-backed handlers.
//
// Scope:
// - `/v1/watch` route behavior only.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Tests use setupTestServer with isolated temp-backed storage.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

// createTestWatch creates a test watch in the given storage
func createTestWatch(t *testing.T, storage *watch.FileStorage, url string) *watch.Watch {
	w := &watch.Watch{
		URL:             url,
		IntervalSeconds: 3600,
		Enabled:         true,
	}
	created, err := storage.Add(w)
	if err != nil {
		t.Fatalf("Failed to create test watch: %v", err)
	}
	return created
}

func TestHandleListWatches(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Test empty list
	req := httptest.NewRequest(http.MethodGet, "/v1/watch", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var emptyList WatchListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &emptyList); err != nil {
		t.Fatalf("Failed to parse empty response: %v", err)
	}
	if len(emptyList.Watches) != 0 {
		t.Errorf("Expected empty watches array, got %d items", len(emptyList.Watches))
	}
	if emptyList.Total != 0 || emptyList.Limit != 100 || emptyList.Offset != 0 {
		t.Fatalf("unexpected empty pagination metadata: %#v", emptyList)
	}

	storage := watch.NewFileStorage(srv.cfg.DataDir)
	older := createTestWatch(t, storage, "https://example.com/older")
	newer := createTestWatch(t, storage, "https://example.com/newer")

	// Test default list envelope
	req = httptest.NewRequest(http.MethodGet, "/v1/watch", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var listResp WatchListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if listResp.Total != 2 || listResp.Limit != 100 || listResp.Offset != 0 {
		t.Fatalf("unexpected pagination metadata: %#v", listResp)
	}
	if len(listResp.Watches) != 2 {
		t.Fatalf("Expected 2 watches, got %d", len(listResp.Watches))
	}
	if listResp.Watches[0].ID != newer.ID || listResp.Watches[1].ID != older.ID {
		t.Fatalf("expected newest-first order, got %#v", listResp.Watches)
	}

	// Test explicit pagination
	req = httptest.NewRequest(http.MethodGet, "/v1/watch?limit=1&offset=1", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected paginated status 200, got %d", rr.Code)
	}
	var pagedResp WatchListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &pagedResp); err != nil {
		t.Fatalf("Failed to parse paged response: %v", err)
	}
	if pagedResp.Total != 2 || pagedResp.Limit != 1 || pagedResp.Offset != 1 {
		t.Fatalf("unexpected paged metadata: %#v", pagedResp)
	}
	if len(pagedResp.Watches) != 1 || pagedResp.Watches[0].ID != older.ID {
		t.Fatalf("expected older watch on second page, got %#v", pagedResp.Watches)
	}
}

func TestHandleCreateWatch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Test successful creation
	enabled := true
	reqBody := WatchRequest{
		URL:             "https://example.com/watch",
		IntervalSeconds: 1800,
		Enabled:         &enabled,
		DiffFormat:      "unified",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.URL != reqBody.URL {
		t.Errorf("Expected URL %s, got %s", reqBody.URL, resp.URL)
	}
	if resp.IntervalSeconds != reqBody.IntervalSeconds {
		t.Errorf("Expected interval %d, got %d", reqBody.IntervalSeconds, resp.IntervalSeconds)
	}
	if resp.Status != "active" {
		t.Errorf("Expected status active, got %s", resp.Status)
	}

	// Test validation error - missing URL
	invalidReq := WatchRequest{
		IntervalSeconds: 1800,
	}
	body, _ = json.Marshal(invalidReq)
	req = httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for validation error, got %d", rr.Code)
	}

	// Test validation error - interval too small
	invalidReq = WatchRequest{
		URL:             "https://example.com/test",
		IntervalSeconds: 30, // Less than minimum 60
	}
	body, _ = json.Marshal(invalidReq)
	req = httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for interval validation error, got %d", rr.Code)
	}

	// Test invalid content type
	req = httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Expected status 415 for invalid content type, got %d", rr.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestHandleCreateWatchRejectsInvalidWebhookURL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := WatchRequest{
		URL:             "https://example.com/watch",
		IntervalSeconds: 1800,
		WebhookConfig:   &model.WebhookSpec{URL: "ftp://hooks.example.com/watch"},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("webhook URL must use http or https scheme")) {
		t.Fatalf("expected webhook URL validation error, got %s", rr.Body.String())
	}
}

func TestHandleCreateWatchRejectsInvalidJobTrigger(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := WatchRequest{
		URL:             "https://example.com/watch",
		IntervalSeconds: 1800,
		JobTrigger: &watch.JobTrigger{
			Kind:    model.KindScrape,
			Request: json.RawMessage(`{"url":"https://example.com","unknown":true}`),
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleWatchCheckTriggersConfiguredJob(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	currentContent := "before"
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body><main>" + currentContent + "</main></body></html>"))
	}))
	defer site.Close()

	reqBody := WatchRequest{
		URL:             site.URL,
		IntervalSeconds: 1800,
		JobTrigger: &watch.JobTrigger{
			Kind:    model.KindScrape,
			Request: json.RawMessage(`{"url":"` + site.URL + `","timeoutSeconds":15}`),
		},
	}
	body, _ := json.Marshal(reqBody)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created WatchResponse
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to parse create response: %v", err)
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/v1/watch/"+created.ID+"/check", nil)
	firstRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(firstRR, firstReq)
	if firstRR.Code != http.StatusOK {
		t.Fatalf("Expected first check status 200, got %d: %s", firstRR.Code, firstRR.Body.String())
	}

	currentContent = "after"
	secondReq := httptest.NewRequest(http.MethodPost, "/v1/watch/"+created.ID+"/check", nil)
	secondRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(secondRR, secondReq)
	if secondRR.Code != http.StatusOK {
		t.Fatalf("Expected second check status 200, got %d: %s", secondRR.Code, secondRR.Body.String())
	}

	var checkResp WatchCheckInspectionResponse
	if err := json.Unmarshal(secondRR.Body.Bytes(), &checkResp); err != nil {
		t.Fatalf("Failed to parse check response: %v", err)
	}
	if !checkResp.Check.Changed {
		t.Fatalf("expected changed=true on second check")
	}
	if len(checkResp.Check.TriggeredJobs) != 1 {
		t.Fatalf("expected one triggered job, got %#v", checkResp.Check.TriggeredJobs)
	}

	jobs, err := srv.store.List(context.Background())
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 triggered job, got %d", len(jobs))
	}
	if jobs[0].ID != checkResp.Check.TriggeredJobs[0] {
		t.Fatalf("expected triggered job ID %s, got %s", jobs[0].ID, checkResp.Check.TriggeredJobs[0])
	}
	if jobs[0].Kind != model.KindScrape {
		t.Fatalf("expected triggered scrape job, got %s", jobs[0].Kind)
	}
}

func TestHandleGetWatch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test watch
	storage := watch.NewFileStorage(srv.cfg.DataDir)
	created := createTestWatch(t, storage, "https://example.com/get-test")

	// Test successful get
	req := httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID, nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, resp.ID)
	}

	// Test not found
	req = httptest.NewRequest(http.MethodGet, "/v1/watch/nonexistent-id", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestHandleUpdateWatch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test watch
	storage := watch.NewFileStorage(srv.cfg.DataDir)
	created := createTestWatch(t, storage, "https://example.com/update-test")

	// Test successful update
	enabled := false
	reqBody := WatchRequest{
		URL:             "https://example.com/updated",
		IntervalSeconds: 7200,
		Enabled:         &enabled,
		DiffFormat:      "html-side-by-side",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/v1/watch/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.URL != reqBody.URL {
		t.Errorf("Expected URL %s, got %s", reqBody.URL, resp.URL)
	}
	if resp.Enabled != *reqBody.Enabled {
		t.Errorf("Expected enabled %v, got %v", *reqBody.Enabled, resp.Enabled)
	}
	if resp.Status != "disabled" {
		t.Errorf("Expected status disabled, got %s", resp.Status)
	}

	// Test not found
	req = httptest.NewRequest(http.MethodPut, "/v1/watch/nonexistent-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}

	// Test validation error
	invalidReq := WatchRequest{
		URL:             "", // Missing URL
		IntervalSeconds: 7200,
	}
	body, _ = json.Marshal(invalidReq)
	req = httptest.NewRequest(http.MethodPut, "/v1/watch/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for validation error, got %d", rr.Code)
	}
}

func TestHandleUpdateWatchRejectsInvalidWebhookURL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	storage := watch.NewFileStorage(srv.cfg.DataDir)
	created := createTestWatch(t, storage, "https://example.com/update-webhook")

	body, _ := json.Marshal(WatchRequest{
		URL:             "https://example.com/update-webhook",
		IntervalSeconds: 3600,
		WebhookConfig:   &model.WebhookSpec{URL: "ftp://hooks.example.com/watch"},
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/watch/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("webhook URL must use http or https scheme")) {
		t.Fatalf("expected webhook URL validation error, got %s", rr.Body.String())
	}
}

func TestHandleUpdateWatchPreservesOmittedOptionalFields(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	storage := watch.NewFileStorage(srv.cfg.DataDir)
	enabled := false
	threshold := 0.25
	created, err := storage.Add(&watch.Watch{
		URL:                 "https://example.com/original",
		IntervalSeconds:     3600,
		Enabled:             enabled,
		DiffFormat:          "html-inline",
		VisualDiffThreshold: threshold,
	})
	if err != nil {
		t.Fatalf("failed to seed watch: %v", err)
	}

	body := []byte(`{"url":"https://example.com/updated"}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/watch/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.URL != "https://example.com/updated" {
		t.Fatalf("expected updated URL, got %s", resp.URL)
	}
	if resp.Enabled {
		t.Fatalf("expected enabled to remain false")
	}
	if resp.DiffFormat != "html-inline" {
		t.Fatalf("expected diff format to remain html-inline, got %s", resp.DiffFormat)
	}
	if resp.VisualDiffThreshold != threshold {
		t.Fatalf("expected threshold %.2f, got %.2f", threshold, resp.VisualDiffThreshold)
	}
}

func TestHandleDeleteWatch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test watch
	storage := watch.NewFileStorage(srv.cfg.DataDir)
	created := createTestWatch(t, storage, "https://example.com/delete-test")

	// Verify watch exists
	watches, _ := storage.List()
	if len(watches) != 1 {
		t.Fatal("Expected 1 watch before deletion")
	}

	sourcePath := filepath.Join(t.TempDir(), "current.png")
	if err := os.WriteFile(sourcePath, append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("watch-delete")...), 0o600); err != nil {
		t.Fatalf("write test artifact: %v", err)
	}
	if _, _, err := watch.NewArtifactStore(srv.cfg.DataDir).ReplaceCurrent(created.ID, sourcePath); err != nil {
		t.Fatalf("persist watch artifact: %v", err)
	}
	artifactDir := filepath.Join(srv.cfg.DataDir, "watch_artifacts", created.ID)
	if _, err := os.Stat(artifactDir); err != nil {
		t.Fatalf("expected artifact directory before deletion: %v", err)
	}

	// Test successful delete
	req := httptest.NewRequest(http.MethodDelete, "/v1/watch/"+created.ID, nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify watch is deleted
	watches, _ = storage.List()
	if len(watches) != 0 {
		t.Errorf("Expected 0 watches after deletion, got %d", len(watches))
	}
	if _, err := os.Stat(artifactDir); !os.IsNotExist(err) {
		t.Fatalf("expected artifact directory removal, got err=%v", err)
	}

	// Test not found (deleting already deleted watch)
	req = httptest.NewRequest(http.MethodDelete, "/v1/watch/"+created.ID, nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestHandleWatchCheck(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test watch
	storage := watch.NewFileStorage(srv.cfg.DataDir)
	created := createTestWatch(t, storage, "https://httpbin.org/html")

	// Test check (may succeed or fail depending on network, but shouldn't panic)
	req := httptest.NewRequest(http.MethodPost, "/v1/watch/"+created.ID+"/check", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	// Should return 200 even if check fails (error is in response body)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchCheckInspectionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.Check.WatchID != created.ID {
		t.Errorf("Expected watch ID %s, got %s", created.ID, resp.Check.WatchID)
	}

	// Test not found
	req = httptest.NewRequest(http.MethodPost, "/v1/watch/nonexistent-id/check", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestWatchMethodNotAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Test PATCH not allowed
	req := httptest.NewRequest(http.MethodPatch, "/v1/watch", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rr.Code)
	}

	// Test POST to individual watch not allowed
	req = httptest.NewRequest(http.MethodPost, "/v1/watch/some-id", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rr.Code)
	}
}

func TestToWatchResponse(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		watch    watch.Watch
		expected string
	}{
		{
			name: "active watch",
			watch: watch.Watch{
				ID:              "test-id",
				URL:             "https://example.com",
				IntervalSeconds: 3600,
				Enabled:         true,
				CreatedAt:       now,
				ChangeCount:     5,
				DiffFormat:      "unified",
			},
			expected: "active",
		},
		{
			name: "disabled watch",
			watch: watch.Watch{
				ID:              "test-id-2",
				URL:             "https://example.com",
				IntervalSeconds: 3600,
				Enabled:         false,
				CreatedAt:       now,
				DiffFormat:      "unified",
			},
			expected: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := toWatchResponse(tt.watch)
			if resp.Status != tt.expected {
				t.Errorf("Expected status %s, got %s", tt.expected, resp.Status)
			}
			if resp.ID != tt.watch.ID {
				t.Errorf("Expected ID %s, got %s", tt.watch.ID, resp.ID)
			}
		})
	}
}

func TestWatchRequestBodySize(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a very large request body
	largeBody := make([]byte, maxRequestBodySize+1000)
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	// Should fail due to size limit
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for oversized request, got %d", rr.Code)
	}
}

func TestWatchStoragePersistence(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a watch using the API
	enabled := true
	reqBody := WatchRequest{
		URL:             "https://example.com/persist",
		IntervalSeconds: 3600,
		Enabled:         &enabled,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create watch: %d - %s", rr.Code, rr.Body.String())
	}

	var created WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify file exists
	watchesPath := filepath.Join(srv.cfg.DataDir, "watches.json")
	if _, err := os.Stat(watchesPath); os.IsNotExist(err) {
		t.Error("Expected watches.json to exist")
	}

	// Verify watch can be retrieved
	req = httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID, nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.URL != reqBody.URL {
		t.Errorf("Expected URL %s, got %s", reqBody.URL, resp.URL)
	}
}

func TestCreateWatchDefaultsEnabledToTrue(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create watch without specifying enabled field
	reqBody := map[string]interface{}{
		"url":             "https://example.com/default-enabled",
		"intervalSeconds": 3600,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if !resp.Enabled {
		t.Errorf("Expected enabled to default to true, got false")
	}
}

func TestCreateWatchWithScreenshotFields(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	enabled := true
	threshold := 0.15
	reqBody := WatchRequest{
		URL:                 "https://example.com/screenshot",
		IntervalSeconds:     3600,
		Enabled:             &enabled,
		ScreenshotEnabled:   true,
		VisualDiffThreshold: &threshold,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if !resp.ScreenshotEnabled {
		t.Errorf("Expected screenshotEnabled to be true, got false")
	}
	if resp.VisualDiffThreshold != 0.15 {
		t.Errorf("Expected visualDiffThreshold 0.15, got %f", resp.VisualDiffThreshold)
	}

	// Verify fields round-trip on GET
	req = httptest.NewRequest(http.MethodGet, "/v1/watch/"+resp.ID, nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var getResp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if !getResp.ScreenshotEnabled {
		t.Errorf("Expected screenshotEnabled to round-trip as true, got false")
	}
	if getResp.VisualDiffThreshold != 0.15 {
		t.Errorf("Expected visualDiffThreshold to round-trip as 0.15, got %f", getResp.VisualDiffThreshold)
	}
}

func TestCreateWatchDefaultsVisualDiffThreshold(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := []byte(`{"url":"https://example.com/default-threshold","intervalSeconds":3600,"screenshotEnabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.VisualDiffThreshold != 0.1 {
		t.Fatalf("expected default threshold 0.1, got %f", resp.VisualDiffThreshold)
	}
}
