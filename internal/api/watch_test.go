// Package api provides HTTP handlers for watch monitoring endpoints.
// This file contains tests for the watch API handlers.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	var emptyList map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &emptyList); err != nil {
		t.Fatalf("Failed to parse empty response: %v", err)
	}
	watches, ok := emptyList["watches"].([]interface{})
	if !ok {
		t.Error("Expected watches array in response")
	}
	if len(watches) != 0 {
		t.Errorf("Expected empty watches array, got %d items", len(watches))
	}

	// Create a test watch
	storage := watch.NewFileStorage(srv.cfg.DataDir)
	createTestWatch(t, storage, "https://example.com/test")

	// Test list with watches
	req = httptest.NewRequest(http.MethodGet, "/v1/watch", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var listResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	watches, ok = listResp["watches"].([]interface{})
	if !ok {
		t.Fatal("Expected watches array in response")
	}
	if len(watches) != 1 {
		t.Errorf("Expected 1 watch, got %d", len(watches))
	}
}

func TestHandleCreateWatch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Test successful creation
	reqBody := WatchRequest{
		URL:             "https://example.com/watch",
		IntervalSeconds: 1800,
		Enabled:         true,
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
	reqBody := WatchRequest{
		URL:             "https://example.com/updated",
		IntervalSeconds: 7200,
		Enabled:         false,
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
	if resp.Enabled != reqBody.Enabled {
		t.Errorf("Expected enabled %v, got %v", reqBody.Enabled, resp.Enabled)
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

	var resp WatchCheckResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.WatchID != created.ID {
		t.Errorf("Expected watch ID %s, got %s", created.ID, resp.WatchID)
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
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for oversized request, got %d", rr.Code)
	}
}

func TestWatchStoragePersistence(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a watch using the API
	reqBody := WatchRequest{
		URL:             "https://example.com/persist",
		IntervalSeconds: 3600,
		Enabled:         true,
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
