// Package api provides integration tests for schedules endpoints (/v1/schedules).
// Tests cover schedule listing, addition, and deletion.
// Does NOT test schedule execution or recurrence logic handled by scheduler package.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSchedulesList(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/schedules", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse JSON response: %v", err)
	}
	if _, ok := resp["schedules"]; !ok {
		t.Errorf("expected 'schedules' field in response, got: %v", resp)
	}
}

func TestHandleSchedulesAdd(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid scrape schedule",
			body:           `{"kind":"scrape","intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid crawl schedule",
			body:           `{"kind":"crawl","intervalSeconds":7200,"specVersion":1,"spec":{"version":1,"url":"https://example.com","maxDepth":2,"maxPages":200,"execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid research schedule",
			body:           `{"kind":"research","intervalSeconds":86400,"specVersion":1,"spec":{"version":1,"query":"test query","urls":["https://example.com"],"maxDepth":2,"maxPages":200,"execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing kind",
			body:           `{"intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid interval (negative)",
			body:           `{"kind":"scrape","intervalSeconds":-1,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid kind value",
			body:           `{"kind":"invalid","intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing content-type",
			body:           `{"kind":"scrape","intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "trailing json payload is rejected",
			body:           `{"kind":"scrape","intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":30}}}{"extra":true}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing url for scrape",
			body:           `{"kind":"scrape","intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing url for crawl",
			body:           `{"kind":"crawl","intervalSeconds":7200,"specVersion":1,"spec":{"version":1,"maxDepth":2,"maxPages":200,"execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing query and urls for research",
			body:           `{"kind":"research","intervalSeconds":86400,"specVersion":1,"spec":{"version":1,"maxDepth":2,"maxPages":200,"execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid timeout too low",
			body:           `{"kind":"scrape","intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":1}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid timeout too high",
			body:           `{"kind":"scrape","intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":600}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing specVersion",
			body:           `{"kind":"scrape","intervalSeconds":3600,"spec":{"version":1,"url":"https://example.com","execution":{"timeoutSeconds":30}}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing spec",
			body:           `{"kind":"scrape","intervalSeconds":3600,"specVersion":1}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/schedules", strings.NewReader(tt.body))
			if tt.name != "missing content-type" {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
			}

			if tt.expectedStatus == http.StatusCreated {
				var resp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to parse JSON response: %v", err)
				}
				if _, ok := resp["id"]; !ok {
					t.Errorf("expected 'id' field in schedule response, got: %v", resp)
				}
				if _, ok := resp["kind"]; !ok {
					t.Errorf("expected 'kind' field in schedule response, got: %v", resp)
				}
				if _, ok := resp["intervalSeconds"]; !ok {
					t.Errorf("expected 'intervalSeconds' field in schedule response, got: %v", resp)
				}
				if _, ok := resp["nextRun"]; !ok {
					t.Errorf("expected 'nextRun' field in schedule response, got: %v", resp)
				}
				if _, ok := resp["specVersion"]; !ok {
					t.Errorf("expected 'specVersion' field in schedule response, got: %v", resp)
				}
				if _, ok := resp["spec"]; !ok {
					t.Errorf("expected 'spec' field in schedule response, got: %v", resp)
				}
			} else {
				var resp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to parse JSON response: %v", err)
				}
				if _, ok := resp["error"]; !ok {
					t.Errorf("expected 'error' field in error response, got: %v", resp)
				}
			}
		})
	}
}

func TestHandleScheduleDelete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	addBody := `{"kind":"scrape","intervalSeconds":3600,"specVersion":1,"spec":{"version":1,"url":"https://example.com","execution":{"headless":false,"timeoutSeconds":30}}}`
	req := httptest.NewRequest("POST", "/v1/schedules", strings.NewReader(addBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("failed to add schedule: got status %v, body: %s", status, rr.Body.String())
	}

	var addResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &addResp); err != nil {
		t.Fatalf("failed to parse add response: %v, body: %s", err, rr.Body.String())
	}

	scheduleID, ok := addResp["id"].(string)
	if !ok {
		t.Fatalf("add response missing id field, got: %+v", addResp)
	}

	t.Logf("Schedule ID: %s", scheduleID)

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/schedules/%s", scheduleID), nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse delete response: %v", err)
	}

	if status, ok := resp["status"].(string); !ok || status != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
}

func TestHandleScheduleDeleteNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("DELETE", "/v1/schedules/nonexistent-id", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("deleting non-existent schedule should succeed (idempotent), got status %v", status)
	}
}

func TestHandleScheduleDeleteInvalidID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "slash only",
			path:           "/v1/schedules/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "base path without ID",
			path:           "/v1/schedules",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", tt.path, nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %v for invalid path %s, got %v", tt.expectedStatus, tt.path, status)
			}
		})
	}
}
