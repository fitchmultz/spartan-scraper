package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"spartan-scraper/internal/config"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/store"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	dataDir := t.TempDir()
	cfg := config.Config{
		DataDir:            dataDir,
		RequestTimeoutSecs: 30,
		MaxConcurrency:     4,
		RateLimitQPS:       10,
		RateLimitBurst:     20,
		MaxRetries:         3,
		RetryBaseMs:        100,
		UserAgent:          "SpartanTest/1.0",
		Port:               "0", // not used for Routes() test
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	manager := jobs.NewManager(
		st,
		dataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		false,
	)
	ctx, cancel := context.WithCancel(context.Background())
	manager.Start(ctx)

	srv := NewServer(manager, st, cfg)

	cleanup := func() {
		cancel()
		manager.Wait()
		st.Close()
	}

	return srv, cleanup
}

func TestHealth(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"database"`) {
		t.Errorf("handler missing database status: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"queue"`) {
		t.Errorf("handler missing queue status: got %v", rr.Body.String())
	}
}

func TestHandleJobs(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleScrape(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleScrapeValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "missing content-type",
			body:           `{"url": "https://example.com"}`,
			contentType:    "",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "invalid json",
			body:           `{"url": "https://example.com"`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing url",
			body:           `{}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid url scheme",
			body:           `{"url": "ftp://example.com"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid url host",
			body:           `{"url": "https://"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid auth profile",
			body:           `{"url": "https://example.com", "authProfile": "invalid name!"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "timeout too low",
			body:           `{"url": "https://example.com", "timeoutSeconds": 1}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "timeout too high",
			body:           `{"url": "https://example.com", "timeoutSeconds": 600}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestHandleCrawlValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid maxDepth",
			body:           `{"url": "https://example.com", "maxDepth": 11}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid maxPages",
			body:           `{"url": "https://example.com", "maxPages": 20000}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/crawl", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestHandleResearch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"query": "test query", "urls": ["https://example.com"]}`
	req := httptest.NewRequest("POST", "/v1/research", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
