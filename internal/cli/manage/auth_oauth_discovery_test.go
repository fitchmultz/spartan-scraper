// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Verify OAuth discovery flows.
//
// Responsibilities:
// - Cover well-known discovery and profile-apply behavior for OAuth providers.
//
// Scope:
// - OAuth CLI behavior only.
//
// Usage:
// - Run with `go test ./internal/cli/manage`.
//
// Invariants/Assumptions:
// - Tests use temp data dirs and mocked HTTP servers only.
package manage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestOAuthDiscover_MissingURL(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthDiscover(context.Background(), cfg, []string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthDiscover_BothURLs tests discover with both discovery-url and issuer.
func TestOAuthDiscover_BothURLs(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthDiscover(context.Background(), cfg, []string{
		"--discovery-url", "https://example.com/.well-known/openid-configuration",
		"--issuer", "https://example.com",
	})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthDiscover_Success tests successful OIDC discovery.
func TestOAuthDiscover_Success(t *testing.T) {
	// Create a mock OIDC discovery server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"issuer":                 "https://example.com",
			"authorization_endpoint": "https://example.com/authorize",
			"token_endpoint":         "https://example.com/token",
			"revocation_endpoint":    "https://example.com/revoke",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthDiscover(context.Background(), cfg, []string{
		"--discovery-url", mockServer.URL + "/.well-known/openid-configuration",
	})

	w.Close()
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "https://example.com") {
		t.Error("output should contain issuer URL")
	}
	if !strings.Contains(output, "https://example.com/authorize") {
		t.Error("output should contain authorization endpoint")
	}
	if !strings.Contains(output, "https://example.com/token") {
		t.Error("output should contain token endpoint")
	}
}

// TestOAuthDiscover_ApplyToProfile tests discovery with --apply flag.
func TestOAuthDiscover_ApplyToProfile(t *testing.T) {
	// Create a mock OIDC discovery server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"issuer":                 "https://example.com",
			"authorization_endpoint": "https://example.com/authorize",
			"token_endpoint":         "https://example.com/token",
			"revocation_endpoint":    "https://example.com/revoke",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Create a profile with existing OAuth2 config
	oauth2 := &auth.OAuth2Config{
		FlowType: auth.OAuth2FlowAuthorizationCode,
		ClientID: "test-client",
		TokenURL: "https://old.example.com/token",
	}
	createTestProfile(t, dataDir, "testprofile", oauth2)

	exitCode := runOAuthDiscover(context.Background(), cfg, []string{
		"--discovery-url", mockServer.URL + "/.well-known/openid-configuration",
		"--apply",
		"--profile", "testprofile",
	})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify the profile was updated
	profile, _, _ := auth.GetProfile(dataDir, "testprofile")
	if profile.OAuth2.TokenURL != "https://example.com/token" {
		t.Errorf("expected token_url to be updated, got '%s'", profile.OAuth2.TokenURL)
	}
	if profile.OAuth2.AuthorizeURL != "https://example.com/authorize" {
		t.Errorf("expected authorize_url to be updated, got '%s'", profile.OAuth2.AuthorizeURL)
	}
}

// TestOAuthInitiate_MissingProfile tests initiate with missing --profile.
