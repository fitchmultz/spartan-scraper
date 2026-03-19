// Package server tests CLI health rendering and local fallback diagnostics.
//
// Purpose:
// - Verify `spartan health` keeps structured recovery guidance consistent when rendered in the terminal.
//
// Responsibilities:
// - Assert one-click recovery actions are translated into CLI commands.
// - Assert local fallback health output surfaces offline-runtime guidance.
// - Assert setup-mode local health preserves guided recovery actions.
//
// Scope:
// - CLI health helpers only; HTTP endpoint and MCP diagnostics are tested elsewhere.
//
// Usage:
// - Run with `go test ./internal/cli/server`.
//
// Invariants/Assumptions:
// - Human-readable health output should stay operator-friendly.
// - Local fallback should never hide setup recovery requirements.
package server

import (
	"context"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestRenderHealthResponseTranslatesOneClickActions(t *testing.T) {
	health := api.HealthResponse{
		Status:  "degraded",
		Version: "test",
		Components: map[string]api.ComponentStatus{
			api.DiagnosticTargetBrowser: {
				Status:  "degraded",
				Message: "Browser tooling needs attention.",
				Actions: []api.RecommendedAction{{
					Label: "Re-check browser tooling",
					Kind:  api.ActionKindOneClick,
					Value: api.DiagnosticActionPath(api.DiagnosticTargetBrowser),
				}},
			},
		},
	}

	rendered := renderHealthResponse(health, "spartan", "runtime")
	if !strings.Contains(rendered, "spartan health --check browser") {
		t.Fatalf("expected translated CLI diagnostic command, got %q", rendered)
	}
}

func TestBuildLocalHealthResponseIncludesOfflineRuntimeNotice(t *testing.T) {
	dataDir := t.TempDir()
	health, source, err := buildLocalHealthResponse(context.Background(), config.Config{DataDir: dataDir}, "spartan")
	if err != nil {
		t.Fatalf("buildLocalHealthResponse failed: %v", err)
	}
	if source != "local" {
		t.Fatalf("source = %q, want local", source)
	}
	if health.Status != "degraded" {
		t.Fatalf("status = %q, want degraded", health.Status)
	}
	if queue := health.Components["queue"]; queue.Status != "degraded" {
		t.Fatalf("queue status = %#v, want degraded", queue)
	}
	if proxy := health.Components[api.DiagnosticTargetProxyPool]; proxy.Status != "disabled" {
		t.Fatalf("expected disabled proxy_pool status when unconfigured, got %#v", proxy)
	}
	if len(health.Notices) == 0 || health.Notices[0].ID != "server_offline" {
		t.Fatalf("expected offline notice, got %#v", health.Notices)
	}
}

func TestBuildLocalHealthResponsePreservesSetupRecovery(t *testing.T) {
	dataDir := t.TempDir()
	status, err := inspectStartupPreflight(config.Config{DataDir: dataDir}, "spartan")
	if err != nil {
		t.Fatalf("inspectStartupPreflight failed: %v", err)
	}
	if status != nil {
		t.Fatalf("expected empty data dir to avoid setup mode, got %#v", status)
	}

	legacyStatus := &api.SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Stored data needs a one-time reset",
		Message:  "Detected legacy persisted state.",
		DataDir:  dataDir,
		Actions: []api.RecommendedAction{{
			Label: "Archive and recreate the data directory",
			Kind:  api.ActionKindCommand,
			Value: "spartan reset-data",
		}},
	}
	health := api.HealthResponse{
		Status:  "setup_required",
		Version: "test",
		Setup:   legacyStatus,
		Components: map[string]api.ComponentStatus{
			"database": {
				Status:  "setup_required",
				Message: legacyStatus.Message,
				Actions: legacyStatus.Actions,
			},
		},
	}

	rendered := renderHealthResponse(health, "spartan", "local")
	if !strings.Contains(rendered, "spartan reset-data") {
		t.Fatalf("expected setup recovery action in rendered output, got %q", rendered)
	}
}
