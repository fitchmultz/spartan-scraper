// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Verify OAuth config set and clear flows.
//
// Responsibilities:
// - Cover write-path validation plus profile persistence and clearing for OAuth config commands.
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
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestOAuthConfigSet_MissingProfile(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigSet(cfg, []string{"--client-id", "test"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigSet_MissingClientID tests config set with missing --client-id.
func TestOAuthConfigSet_MissingClientID(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigSet(cfg, []string{"--profile", "test"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigSet_MissingTokenURL tests config set with missing --token-url.
func TestOAuthConfigSet_MissingTokenURL(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigSet(cfg, []string{"--profile", "test", "--client-id", "test"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigSet_MissingAuthorizeURL tests config set for auth code flow without authorize-url.
func TestOAuthConfigSet_MissingAuthorizeURL(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigSet(cfg, []string{
		"--profile", "test",
		"--client-id", "test",
		"--token-url", "https://example.com/token",
	})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigSet_Success tests successful config set.
func TestOAuthConfigSet_Success(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigSet(cfg, []string{
		"--profile", "testprofile",
		"--client-id", "my-client-id",
		"--client-secret", "my-secret",
		"--token-url", "https://example.com/token",
		"--authorize-url", "https://example.com/authorize",
		"--scope", "read",
		"--scope", "write",
		"--use-pkce",
		"--redirect-uri", "https://example.com/callback",
	})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify the profile was created
	profile, found, err := auth.GetProfile(dataDir, "testprofile")
	if err != nil {
		t.Fatalf("failed to get profile: %v", err)
	}
	if !found {
		t.Fatal("profile should exist")
	}
	if profile.OAuth2 == nil {
		t.Fatal("profile should have OAuth2 config")
	}
	if profile.OAuth2.ClientID != "my-client-id" {
		t.Errorf("expected client_id 'my-client-id', got '%s'", profile.OAuth2.ClientID)
	}
	if profile.OAuth2.ClientSecret != "my-secret" {
		t.Errorf("expected client_secret 'my-secret', got '%s'", profile.OAuth2.ClientSecret)
	}
	if len(profile.OAuth2.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(profile.OAuth2.Scopes))
	}
	if !profile.OAuth2.UsePKCE {
		t.Error("expected UsePKCE to be true")
	}
}

// TestOAuthConfigSet_InvalidFlowType tests config set with invalid flow type.
func TestOAuthConfigSet_InvalidFlowType(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigSet(cfg, []string{
		"--profile", "test",
		"--client-id", "test",
		"--token-url", "https://example.com/token",
		"--flow-type", "invalid_flow",
	})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigClear_MissingProfile tests config clear with missing --profile.
func TestOAuthConfigClear_MissingProfile(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigClear(cfg, []string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigClear_ProfileNotFound tests config clear with non-existent profile.
func TestOAuthConfigClear_ProfileNotFound(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigClear(cfg, []string{"--profile", "nonexistent"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigClear_Success tests successful config clear.
func TestOAuthConfigClear_Success(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
	}
	createTestProfile(t, dataDir, "testprofile", oauth2)

	exitCode := runOAuthConfigClear(cfg, []string{"--profile", "testprofile"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify OAuth2 config was removed
	profile, _, _ := auth.GetProfile(dataDir, "testprofile")
	if profile.OAuth2 != nil {
		t.Error("OAuth2 config should be nil after clear")
	}
}

// TestOAuthDiscover_MissingURL tests discover with neither discovery-url nor issuer.
