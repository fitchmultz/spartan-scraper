// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Verify OAuth token inspection flows.
//
// Responsibilities:
// - Cover token listing, token status, and token deletion behavior.
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
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestOAuthTokenList_Empty(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthTokenList(cfg)

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

	if !strings.Contains(output, "No OAuth tokens found") {
		t.Error("output should indicate no tokens found")
	}
}

// TestOAuthTokenList_WithTokens tests token list when tokens exist.
func TestOAuthTokenList_WithTokens(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Create a token
	oauthStore := auth.NewOAuthStore(dataDir)
	token := auth.OAuth2Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	if err := oauthStore.SaveToken("testprofile", token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthTokenList(cfg)

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

	if !strings.Contains(output, "testprofile") {
		t.Error("output should contain the profile name")
	}
}

// TestOAuthTokenStatus_MissingProfile tests token status with missing --profile.
func TestOAuthTokenStatus_MissingProfile(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthTokenStatus(cfg, []string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthTokenStatus_NoToken tests token status when no token exists.
func TestOAuthTokenStatus_NoToken(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthTokenStatus(cfg, []string{"--profile", "testprofile"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthTokenStatus_WithToken tests token status when token exists.
func TestOAuthTokenStatus_WithToken(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Create a token with expiration
	expiry := time.Now().Add(time.Hour)
	oauthStore := auth.NewOAuthStore(dataDir)
	token := auth.OAuth2Token{
		AccessToken:  "test-token",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		ExpiresAt:    &expiry,
		Scope:        "read write",
	}
	if err := oauthStore.SaveToken("testprofile", token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthTokenStatus(cfg, []string{"--profile", "testprofile"})

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

	if !strings.Contains(output, "Bearer") {
		t.Error("output should contain token type")
	}
	if !strings.Contains(output, "Has refresh:  true") {
		t.Error("output should indicate refresh token is present")
	}
	if !strings.Contains(output, "read write") {
		t.Error("output should contain scope")
	}
}

// TestOAuthTokenStatus_ExpiredToken tests token status with expired token.
func TestOAuthTokenStatus_ExpiredToken(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Create an expired token
	expiry := time.Now().Add(-time.Hour)
	oauthStore := auth.NewOAuthStore(dataDir)
	token := auth.OAuth2Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		ExpiresAt:   &expiry,
	}
	if err := oauthStore.SaveToken("testprofile", token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthTokenStatus(cfg, []string{"--profile", "testprofile"})

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

	if !strings.Contains(output, "expired") {
		t.Error("output should indicate token is expired")
	}
}

// TestOAuthTokenDelete_MissingProfile tests token delete with missing --profile.
func TestOAuthTokenDelete_MissingProfile(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthTokenDelete(cfg, []string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthTokenDelete_Success tests successful token deletion.
func TestOAuthTokenDelete_Success(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Create a token
	oauthStore := auth.NewOAuthStore(dataDir)
	token := auth.OAuth2Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	if err := oauthStore.SaveToken("testprofile", token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	exitCode := runOAuthTokenDelete(cfg, []string{"--profile", "testprofile"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify token was deleted
	_, found, _ := oauthStore.LoadToken("testprofile")
	if found {
		t.Error("token should be deleted")
	}
}

// TestOAuthRefresh_MissingProfile tests refresh with missing --profile.
