// Package api provides tests for read-only recovery diagnostics and action builders.
//
// Purpose:
// - Verify guided recovery actions and safe setup-mode diagnostic endpoints.
//
// Responsibilities:
// - Assert one-click and copy-ready actions are emitted for degraded subsystems.
// - Confirm setup-mode servers still answer safe diagnostic routes.
// - Keep recovery payloads stable for web and CLI consumers.
//
// Scope:
// - Diagnostic action builders and endpoint behavior only.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Diagnostic endpoints stay read-only and return JSON payloads even in setup mode.
// - Recovery actions always include concrete next steps.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func decodeDiagnosticResponse(t *testing.T, rr *httptest.ResponseRecorder) DiagnosticActionResponse {
	t.Helper()
	var response DiagnosticActionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode diagnostic response: %v", err)
	}
	return response
}

func TestBrowserRecoveryActionsIncludeOneClickAndPlaywrightInstall(t *testing.T) {
	actions := browserRecoveryActions("darwin", true)

	var foundOneClick bool
	var foundPlaywright bool
	for _, action := range actions {
		if action.Kind == ActionKindOneClick && action.Value == "/v1/diagnostics/browser-check" {
			foundOneClick = true
		}
		if action.Kind == ActionKindCopy && action.Value == "go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install --with-deps" {
			foundPlaywright = true
		}
	}

	if !foundOneClick {
		t.Fatal("expected browser one-click action")
	}
	if !foundPlaywright {
		t.Fatal("expected playwright install action")
	}
}

func TestProxyPoolRecoveryActionsIncludeDisableOverride(t *testing.T) {
	actions := proxyPoolRecoveryActions(".data/proxy_pool.json")

	for _, action := range actions {
		if action.Kind == ActionKindCopy && action.Value == "PROXY_POOL_FILE=" {
			return
		}
	}

	t.Fatal("expected disable override action")
}

func TestActionTranslationsUseSurfaceNativeCommands(t *testing.T) {
	actions := []RecommendedAction{{
		Label: "Re-check browser tooling",
		Kind:  ActionKindOneClick,
		Value: DiagnosticActionPath(DiagnosticTargetBrowser),
	}}

	cli := CLIRecommendedActions(actions, "spartan")
	if len(cli) != 1 || cli[0].Kind != ActionKindCommand || cli[0].Value != "spartan health --check browser" {
		t.Fatalf("unexpected CLI actions %#v", cli)
	}

	mcp := MCPRecommendedActions(actions)
	if len(mcp) != 1 || mcp[0].Kind != ActionKindCommand || mcp[0].Value != "diagnostic_check component=browser" {
		t.Fatalf("unexpected MCP actions %#v", mcp)
	}
}

func TestProxyPoolDiagnosticResponseHandlesUnavailableRuntime(t *testing.T) {
	path := t.TempDir() + "/proxy_pool.json"
	if err := os.WriteFile(path, []byte("[]"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	response := BuildProxyPoolDiagnosticResponse(config.Config{ProxyPoolFile: path}, ProxyPoolRuntimeUnavailable)
	if response.Status != "degraded" {
		t.Fatalf("status = %q, want degraded", response.Status)
	}
	if response.Title != "Proxy pool is waiting on the runtime" {
		t.Fatalf("title = %q, want runtime waiting guidance", response.Title)
	}
}

func TestSetupServerProxyPoolDiagnosticRemainsAvailable(t *testing.T) {
	srv := NewSetupServer(config.Config{}, SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Setup required",
		Message:  "Detected legacy persisted state.",
	})
	defer srv.Stop()

	req := httptest.NewRequest(http.MethodPost, "/v1/diagnostics/proxy-pool-check", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for setup-mode diagnostic, got %d", rr.Code)
	}

	response := decodeDiagnosticResponse(t, rr)
	if response.Status != "disabled" {
		t.Fatalf("expected disabled proxy-pool diagnostic, got %#v", response)
	}
}

func TestSetupServerAIDiagnosticRemainsAvailable(t *testing.T) {
	srv := NewSetupServer(config.Config{}, SetupStatus{
		Required: true,
		Code:     "legacy_data_dir",
		Title:    "Setup required",
		Message:  "Detected legacy persisted state.",
	})
	defer srv.Stop()

	req := httptest.NewRequest(http.MethodPost, "/v1/diagnostics/ai-check", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for setup-mode diagnostic, got %d", rr.Code)
	}

	response := decodeDiagnosticResponse(t, rr)
	if response.Status != "disabled" {
		t.Fatalf("expected disabled ai diagnostic, got %#v", response)
	}
}
