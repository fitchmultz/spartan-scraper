// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Verify OAuth initiate and callback flows.
//
// Responsibilities:
// - Cover authorization URL generation, redirect overrides, PKCE initiation, and callback validation.
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
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestOAuthInitiate_MissingProfile(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthInitiate(cfg, []string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthInitiate_ProfileNotFound tests initiate with non-existent profile.
func TestOAuthInitiate_ProfileNotFound(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthInitiate(cfg, []string{"--profile", "nonexistent"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthInitiate_NoOAuthConfig tests initiate with profile that has no OAuth2 config.
func TestOAuthInitiate_NoOAuthConfig(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	createTestProfile(t, dataDir, "testprofile", nil)

	exitCode := runOAuthInitiate(cfg, []string{"--profile", "testprofile"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthInitiate_Success tests successful initiation.
func TestOAuthInitiate_Success(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
		RedirectURI:  "https://example.com/callback",
	}
	createTestProfile(t, dataDir, "testprofile", oauth2)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthInitiate(cfg, []string{"--profile", "testprofile"})

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

	if !strings.Contains(output, "Authorization URL:") {
		t.Error("output should contain 'Authorization URL:'")
	}
	if !strings.Contains(output, "https://example.com/authorize") {
		t.Error("output should contain the authorize URL")
	}
	if !strings.Contains(output, "State:") {
		t.Error("output should contain 'State:'")
	}
}

// TestOAuthInitiate_WithPKCE tests initiation with PKCE enabled.
func TestOAuthInitiate_WithPKCE(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
		UsePKCE:      true,
	}
	createTestProfile(t, dataDir, "testprofile", oauth2)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthInitiate(cfg, []string{"--profile", "testprofile"})

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

	// Check that the authorization URL contains PKCE parameters
	if !strings.Contains(output, "code_challenge=") {
		t.Error("authorization URL should contain code_challenge")
	}
	if !strings.Contains(output, "code_challenge_method=S256") {
		t.Error("authorization URL should contain code_challenge_method=S256")
	}
}

// TestOAuthInitiate_WithRedirectOverride tests initiation with redirect URI override.
func TestOAuthInitiate_WithRedirectOverride(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
		RedirectURI:  "https://default.example.com/callback",
	}
	createTestProfile(t, dataDir, "testprofile", oauth2)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthInitiate(cfg, []string{
		"--profile", "testprofile",
		"--redirect-uri", "https://override.example.com/callback",
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

	// Check that the authorization URL contains the override redirect URI
	// The URL may be encoded, so check for both encoded and decoded forms
	if !strings.Contains(output, "override.example.com") {
		t.Error("authorization URL should contain the override redirect URI")
	}
}

// TestOAuthCallback_MissingState tests callback with missing --state.
func TestOAuthCallback_MissingState(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthCallback(cfg, []string{"--code", "testcode"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthCallback_MissingCode tests callback with missing --code.
func TestOAuthCallback_MissingCode(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthCallback(cfg, []string{"--state", "teststate"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthCallback_InvalidState tests callback with invalid state.
func TestOAuthCallback_InvalidState(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthCallback(cfg, []string{
		"--state", "invalid-state",
		"--code", "testcode",
	})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthCallback_FromCallbackURL tests callback parsing from callback URL.
func TestOAuthCallback_FromCallbackURL(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Create a valid state
	oauthStore := auth.NewOAuthStore(dataDir)
	state, _, err := oauthStore.CreateOAuthState("testprofile", "", false)
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}

	callbackURL := fmt.Sprintf("https://example.com/callback?state=%s&code=testcode", state)

	exitCode := runOAuthCallback(cfg, []string{
		"--callback-url", callbackURL,
	})
	// Will fail because profile doesn't exist, but should parse the URL correctly
	if exitCode != 1 {
		t.Errorf("expected exit code 1 (profile not found), got %d", exitCode)
	}
}

// TestOAuthTokenList_Empty tests token list when no tokens exist.
