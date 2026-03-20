// Package api provides integration tests for health check endpoint (/healthz).
//
// Purpose:
// - Verify the structured health contract for normal, degraded, and setup-mode responses.
//
// Responsibilities:
// - Assert `/healthz` returns component status, notices, and setup metadata.
// - Confirm optional subsystem failures degrade health without changing the transport status code.
// - Confirm setup-mode servers surface guided recovery information.
//
// Scope:
// - Health endpoint behavior only; individual subsystem implementations are tested elsewhere.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - `/healthz` remains unauthenticated.
// - Setup and degraded states still return HTTP 200 so browser clients can read diagnostics.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

func decodeHealthResponse(t *testing.T, rr *httptest.ResponseRecorder) HealthResponse {
	t.Helper()
	var health HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &health); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	return health
}

func TestHealth(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	health := decodeHealthResponse(t, rr)
	if health.Status != "ok" && health.Status != "degraded" {
		t.Fatalf("unexpected health status %q", health.Status)
	}
	if health.Version != buildinfo.Version {
		t.Fatalf("version = %q, want %q", health.Version, buildinfo.Version)
	}
	if _, ok := health.Components["database"]; !ok {
		t.Fatalf("expected database component in health response")
	}
	if _, ok := health.Components["queue"]; !ok {
		t.Fatalf("expected queue component in health response")
	}
	if _, ok := health.Components["ai"]; !ok {
		t.Fatalf("expected ai component in health response")
	}
	proxyPool, ok := health.Components["proxy_pool"]
	if !ok {
		t.Fatalf("expected proxy_pool component in health response")
	}
	if proxyPool.Status != "disabled" {
		t.Fatalf("expected disabled proxy_pool component by default, got %#v", proxyPool)
	}
}

func TestHealthLeavesAIComponentDisabledWhenUnconfigured(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	health := decodeHealthResponse(t, rr)
	ai, ok := health.Components["ai"]
	if !ok {
		t.Fatal("expected ai component in health response")
	}
	if ai.Status != "disabled" {
		t.Fatalf("expected disabled ai component by default, got %#v", ai)
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

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	health := decodeHealthResponse(t, rr)
	ai := health.Components["ai"]
	if ai.Status != "ok" {
		t.Fatalf("expected healthy ai component, got %#v", ai)
	}
}

func TestHealthIncludesCapabilityAwareAIDetails(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	srv.aiExtractor = extract.NewAIExtractorWithProvider(
		config.AIConfig{Enabled: true, Mode: "sdk", Routing: config.DefaultAIRoutingConfig()},
		srv.cfg.DataDir,
		&fakeAIProvider{
			healthSnapshot: extract.AIHealthSnapshot{
				Status:  "degraded",
				Mode:    "sdk",
				Message: "AI helpers are partially available. Ready: extract.natural_language. Degraded: template.generate.",
				Capabilities: map[string]extract.AICapabilityHealth{
					config.AICapabilityExtractNatural: {
						Status:          "ok",
						AvailableRoutes: []string{"openai/gpt-5.4"},
					},
					config.AICapabilityTemplateGeneration: {
						Status:           "degraded",
						ConfiguredRoutes: []string{"kimi-coding/k2p5"},
					},
				},
			},
		},
	)
	srv.cfg.AI = config.AIConfig{
		Enabled: true,
		Mode:    "sdk",
		Routing: config.DefaultAIRoutingConfig(),
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	health := decodeHealthResponse(t, rr)
	ai := health.Components["ai"]
	if ai.Status != "degraded" {
		t.Fatalf("expected degraded ai component, got %#v", ai)
	}
	details, ok := ai.Details.(map[string]any)
	if !ok {
		t.Fatalf("expected ai details map, got %#v", ai.Details)
	}
	capabilities, ok := details["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("expected capability details map, got %#v", details["capabilities"])
	}
	templateCapability, ok := capabilities[config.AICapabilityTemplateGeneration].(map[string]any)
	if !ok {
		t.Fatalf("expected template capability map, got %#v", capabilities[config.AICapabilityTemplateGeneration])
	}
	if status, _ := templateCapability["status"].(string); status != "degraded" {
		t.Fatalf("expected template capability degraded, got %#v", templateCapability)
	}
}

func TestHealthIncludesConfigNotices(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	srv.cfg.StartupNotices = []config.StartupNotice{{
		ID:       "retention-disabled-with-limits",
		Severity: "warning",
		Title:    "Retention limits are configured but inactive",
		Message:  "Retention limits are set while RETENTION_ENABLED is false.",
	}}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	health := decodeHealthResponse(t, rr)
	if health.Status != "degraded" {
		t.Fatalf("expected degraded status when notices are present, got %q", health.Status)
	}
	if len(health.Notices) != 1 {
		t.Fatalf("expected one notice, got %d", len(health.Notices))
	}
	if health.Notices[0].Scope != "config" {
		t.Fatalf("expected config-scope notice, got %#v", health.Notices[0])
	}
}

func TestSetupServerHealth(t *testing.T) {
	srv := NewSetupServer(config.Config{DataDir: "/tmp/spartan"}, SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Stored data needs a one-time reset",
		Message:  "Detected legacy persisted state.",
		DataDir:  "/tmp/spartan",
		Actions: []RecommendedAction{{
			Label: "Archive and recreate the data directory",
			Kind:  ActionKindCommand,
			Value: "./bin/spartan reset-data",
		}},
	})
	defer srv.Stop()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for setup-mode health, got %d", rr.Code)
	}

	health := decodeHealthResponse(t, rr)
	if health.Status != "setup_required" {
		t.Fatalf("expected setup_required status, got %q", health.Status)
	}
	if health.Setup == nil || !health.Setup.Required {
		t.Fatalf("expected setup payload, got %#v", health.Setup)
	}
	if len(health.Notices) != 1 || health.Notices[0].Scope != "setup" {
		t.Fatalf("expected setup notice, got %#v", health.Notices)
	}
	if database := health.Components["database"]; database.Status != "setup_required" {
		t.Fatalf("expected setup-required database component, got %#v", database)
	}
	if queue := health.Components["queue"]; queue.Status != "setup_required" {
		t.Fatalf("expected setup-required queue component, got %#v", queue)
	}
}

func TestSetupServerHealthLeavesDisabledOptionalSubsystemsOutOfNotices(t *testing.T) {
	srv := NewSetupServer(config.Config{DataDir: "/tmp/spartan"}, SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Stored data needs a one-time reset",
		Message:  "Detected legacy persisted state.",
		DataDir:  "/tmp/spartan",
	})
	defer srv.Stop()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	health := decodeHealthResponse(t, rr)
	if ai := health.Components["ai"]; ai.Status != "disabled" {
		t.Fatalf("expected disabled ai status in setup mode without config, got %#v", ai)
	}
	if proxy := health.Components["proxy_pool"]; proxy.Status != "disabled" {
		t.Fatalf("expected disabled proxy_pool status in setup mode without config, got %#v", proxy)
	}
	if len(health.Notices) != 1 {
		t.Fatalf("expected only the setup notice; optional disabled subsystems must stay quiet, got %#v", health.Notices)
	}
}

func TestSetupServerHealthIncludesOptionalSubsystems(t *testing.T) {
	srv := NewSetupServer(config.Config{
		DataDir:       "/tmp/spartan",
		ProxyPoolFile: "/definitely/missing/proxy-pool.json",
		AI: config.AIConfig{
			Enabled: true,
			Mode:    "sdk",
			Routing: config.DefaultAIRoutingConfig(),
		},
	}, SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Stored data needs a one-time reset",
		Message:  "Detected legacy persisted state.",
		DataDir:  "/tmp/spartan",
		Actions: []RecommendedAction{{
			Label: "Copy reset command",
			Kind:  ActionKindCopy,
			Value: "./bin/spartan reset-data",
		}},
	})
	defer srv.Stop()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for setup-mode health, got %d", rr.Code)
	}

	health := decodeHealthResponse(t, rr)
	for _, name := range []string{"database", "queue", "browser", "ai", "proxy_pool"} {
		if _, ok := health.Components[name]; !ok {
			t.Fatalf("expected %s component in setup-mode health payload", name)
		}
	}

	if browser := health.Components["browser"]; browser.Status == "" {
		t.Fatalf("expected browser status in setup mode, got %#v", browser)
	}
	if ai := health.Components["ai"]; ai.Status != "degraded" {
		t.Fatalf("expected degraded ai status in setup mode, got %#v", ai)
	}
	if proxy := health.Components["proxy_pool"]; proxy.Status != "degraded" {
		t.Fatalf("expected degraded proxy_pool status in setup mode, got %#v", proxy)
	}
}
