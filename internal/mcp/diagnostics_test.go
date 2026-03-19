// Package mcp tests the diagnostics-focused MCP tools.
//
// Purpose:
// - Verify MCP exposes the same structured health and recovery model as the other primary operator surfaces.
//
// Responsibilities:
// - Assert normal-mode tool lists include health_status and diagnostic_check.
// - Assert diagnostic actions are translated into MCP-native follow-up commands.
// - Assert setup mode only exposes diagnostics tools and rejects unrelated tool calls.
//
// Scope:
// - MCP diagnostics behavior only; REST and CLI diagnostics are tested elsewhere.
//
// Usage:
// - Run with `go test ./internal/mcp`.
//
// Invariants/Assumptions:
// - Setup mode must still provide actionable diagnostics.
// - MCP should not surface raw HTTP diagnostic endpoints as follow-up actions.
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestDiagnosticsToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer func() {
		srv.Close()
		_ = os.RemoveAll(tmpDir)
	}()

	tools := srv.toolsList()
	toolNames := make(map[string]bool, len(tools))
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	for _, name := range []string{"health_status", "diagnostic_check"} {
		if !toolNames[name] {
			t.Fatalf("expected tool %s in toolsList", name)
		}
	}
}

func TestHealthStatusLeavesProxyPoolDisabledWhenUnconfigured(t *testing.T) {
	srv := NewSetupServer(testConfig(config.Config{DataDir: t.TempDir()}), api.SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Stored data needs a one-time reset",
		Message:  "Detected legacy persisted state.",
	})
	defer srv.Close()

	result, callErr := srv.handleToolCall(context.Background(), map[string]json.RawMessage{
		"params": mustMarshalJSON(callParams{Name: "health_status", Arguments: map[string]interface{}{}}),
	})
	if callErr != nil {
		t.Fatalf("health_status failed: %v", callErr)
	}

	health, ok := result.(api.HealthResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", result)
	}
	proxy := health.Components[api.DiagnosticTargetProxyPool]
	if proxy.Status != "disabled" {
		t.Fatalf("unexpected proxy status %#v", proxy)
	}
}

func TestHealthStatusReturnsMCPTranslatedActions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := testConfig(config.Config{DataDir: tmpDir, ProxyPoolFile: "/definitely/missing/proxy-pool.json"})
	srv := NewSetupServer(cfg, api.SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Stored data needs a one-time reset",
		Message:  "Detected legacy persisted state.",
		DataDir:  tmpDir,
		Actions: []api.RecommendedAction{{
			Label: "Archive and recreate the data directory",
			Kind:  api.ActionKindCommand,
			Value: "spartan reset-data",
		}},
	})
	defer srv.Close()

	result, callErr := srv.handleToolCall(context.Background(), map[string]json.RawMessage{
		"params": mustMarshalJSON(callParams{Name: "health_status", Arguments: map[string]interface{}{}}),
	})
	if callErr != nil {
		t.Fatalf("health_status failed: %v", callErr)
	}

	health, ok := result.(api.HealthResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", result)
	}
	proxy := health.Components[api.DiagnosticTargetProxyPool]
	if proxy.Status != "degraded" {
		t.Fatalf("unexpected proxy status %#v", proxy)
	}
	if len(proxy.Actions) == 0 || proxy.Actions[0].Value != "diagnostic_check component=proxy_pool" {
		t.Fatalf("expected MCP-translated action, got %#v", proxy.Actions)
	}
}

func TestDiagnosticCheckReturnsTranslatedAction(t *testing.T) {
	srv, tmpDir := testServer()
	defer func() {
		srv.Close()
		_ = os.RemoveAll(tmpDir)
	}()

	result, err := srv.handleToolCall(context.Background(), map[string]json.RawMessage{
		"params": mustMarshalJSON(callParams{Name: "diagnostic_check", Arguments: map[string]interface{}{"component": "browser"}}),
	})
	if err != nil {
		t.Fatalf("diagnostic_check failed: %v", err)
	}

	response, ok := result.(api.DiagnosticActionResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", result)
	}
	if len(response.Actions) > 0 && response.Actions[0].Value == api.DiagnosticActionPath(api.DiagnosticTargetBrowser) {
		t.Fatalf("expected MCP-native follow-up action, got %#v", response.Actions)
	}
}

func TestDiagnosticCheckReturnsDisabledForOptionalAIWhenOff(t *testing.T) {
	srv := NewSetupServer(testConfig(config.Config{DataDir: t.TempDir()}), api.SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Stored data needs a one-time reset",
		Message:  "Detected legacy persisted state.",
	})
	defer srv.Close()

	result, err := srv.handleToolCall(context.Background(), map[string]json.RawMessage{
		"params": mustMarshalJSON(callParams{Name: "diagnostic_check", Arguments: map[string]interface{}{"component": "ai"}}),
	})
	if err != nil {
		t.Fatalf("diagnostic_check failed: %v", err)
	}

	response, ok := result.(api.DiagnosticActionResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", result)
	}
	if response.Status != "disabled" {
		t.Fatalf("expected disabled AI diagnostic, got %#v", response)
	}
}

func TestSetupModeOnlyExposesDiagnostics(t *testing.T) {
	srv := NewSetupServer(testConfig(config.Config{DataDir: t.TempDir()}), api.SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Stored data needs a one-time reset",
		Message:  "Detected legacy persisted state.",
	})
	defer srv.Close()

	tools := srv.toolsList()
	if len(tools) != 2 {
		t.Fatalf("expected diagnostics-only tool list, got %#v", tools)
	}

	_, err := srv.handleToolCall(context.Background(), map[string]json.RawMessage{
		"params": mustMarshalJSON(callParams{Name: "job_list", Arguments: map[string]interface{}{}}),
	})
	if err == nil {
		t.Fatal("expected setup-mode rejection for non-diagnostic tool")
	}
}
