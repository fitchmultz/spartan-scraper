// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Verify OAuth token refresh and revoke flows.
//
// Responsibilities:
// - Cover refresh and revoke command behavior across missing-token and remote/local cases.
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
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestOAuthRefresh_MissingProfile(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthRefresh(context.Background(), cfg, []string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthRefresh_NoToken tests refresh when no token exists.
func TestOAuthRefresh_NoToken(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthRefresh(context.Background(), cfg, []string{"--profile", "testprofile"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthRefresh_NoRefreshToken tests refresh when token has no refresh token.
func TestOAuthRefresh_NoRefreshToken(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	// Create a token without refresh token
	oauthStore := auth.NewOAuthStore(dataDir)
	token := auth.OAuth2Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		// No RefreshToken
	}
	if err := oauthStore.SaveToken("testprofile", token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Create profile
	createTestProfile(t, dataDir, "testprofile", &auth.OAuth2Config{
		FlowType: auth.OAuth2FlowAuthorizationCode,
		ClientID: "test",
		TokenURL: "https://example.com/token",
	})

	exitCode := runOAuthRefresh(context.Background(), cfg, []string{"--profile", "testprofile"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthRevoke_MissingProfile tests revoke with missing --profile.
func TestOAuthRevoke_MissingProfile(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthRevoke(context.Background(), cfg, []string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthRevoke_NoToken tests revoke when no token exists.
func TestOAuthRevoke_NoToken(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthRevoke(context.Background(), cfg, []string{"--profile", "testprofile"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthRevoke_LocalOnly tests revoke with --local-only flag.
func TestOAuthRevoke_LocalOnly(t *testing.T) {
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

	// Create profile without revoke URL
	createTestProfile(t, dataDir, "testprofile", &auth.OAuth2Config{
		FlowType: auth.OAuth2FlowAuthorizationCode,
		ClientID: "test",
		TokenURL: "https://example.com/token",
	})

	exitCode := runOAuthRevoke(context.Background(), cfg, []string{
		"--profile", "testprofile",
		"--local-only",
	})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify token was deleted locally
	_, found, _ := oauthStore.LoadToken("testprofile")
	if found {
		t.Error("token should be deleted locally")
	}
}

// TestOAuthRevoke_NoRevokeURL tests revoke when profile has no revoke URL.
func TestOAuthRevoke_NoRevokeURL(t *testing.T) {
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

	// Create profile without revoke URL
	createTestProfile(t, dataDir, "testprofile", &auth.OAuth2Config{
		FlowType: auth.OAuth2FlowAuthorizationCode,
		ClientID: "test",
		TokenURL: "https://example.com/token",
	})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthRevoke(context.Background(), cfg, []string{
		"--profile", "testprofile",
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

	if !strings.Contains(output, "not configured") {
		t.Error("output should indicate remote revoke is not configured")
	}
}

// TestDerivePKCEChallengeS256 tests the DerivePKCEChallengeS256 helper.
