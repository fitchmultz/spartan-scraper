// Package api provides integration tests for health check endpoint (/healthz).
// Tests verify health endpoint returns correct status and includes required fields.
// Does NOT test individual component health checks in depth.
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

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
	expectedVersion := `"version":"` + buildinfo.Version + `"`
	if !strings.Contains(rr.Body.String(), expectedVersion) {
		t.Errorf("handler missing or incorrect version: got %v, want %s", rr.Body.String(), expectedVersion)
	}
	if !strings.Contains(rr.Body.String(), `"database"`) {
		t.Errorf("handler missing database status: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"queue"`) {
		t.Errorf("handler missing queue status: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ai"`) {
		t.Errorf("handler missing ai status: got %v", rr.Body.String())
	}
}

func TestHealthIncludesAIComponentWhenEnabled(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	srv.aiExtractor = extract.NewAIExtractorWithProvider(
		config.AIConfig{Enabled: true, Mode: "sdk", Routing: config.DefaultAIRoutingConfig()},
		srv.cfg.DataDir,
		&fakeAIProvider{},
	)
	srv.cfg.AI = config.AIConfig{
		Enabled: true,
		Mode:    "sdk",
		Routing: config.DefaultAIRoutingConfig(),
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `"ai":{"status":"ok"`) {
		t.Fatalf("handler missing healthy ai component: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"mode":"sdk"`) {
		t.Fatalf("handler missing ai mode detail: got %v", rr.Body.String())
	}
}
