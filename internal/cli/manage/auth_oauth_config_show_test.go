// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Verify OAuth config show flows.
//
// Responsibilities:
// - Cover missing-profile handling, redaction, and JSON output for OAuth config inspection.
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
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestOAuthConfigShow_MissingProfile(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigShow(cfg, []string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigShow_ProfileNotFound tests config show with non-existent profile.
func TestOAuthConfigShow_ProfileNotFound(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}

	exitCode := runOAuthConfigShow(cfg, []string{"--profile", "nonexistent"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestOAuthConfigShow_NoOAuthConfig tests config show with profile that has no OAuth2 config.
func TestOAuthConfigShow_NoOAuthConfig(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	createTestProfile(t, dataDir, "testprofile", nil)

	exitCode := runOAuthConfigShow(cfg, []string{"--profile", "testprofile"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

// TestOAuthConfigShow_RedactsSecret tests that client_secret is redacted by default.
func TestOAuthConfigShow_RedactsSecret(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		ClientSecret: "super-secret-value",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
	}
	createTestProfile(t, dataDir, "testprofile", oauth2)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthConfigShow(cfg, []string{"--profile", "testprofile"})

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

	if strings.Contains(output, "super-secret-value") {
		t.Error("output should not contain the secret value without --show-secret")
	}
	if !strings.Contains(output, "***") {
		t.Error("output should contain '***' to indicate redacted secret")
	}
}

// TestOAuthConfigShow_ShowsSecretWithFlag tests that --show-secret reveals the secret.
func TestOAuthConfigShow_ShowsSecretWithFlag(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		ClientSecret: "super-secret-value",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
	}
	createTestProfile(t, dataDir, "testprofile", oauth2)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthConfigShow(cfg, []string{"--profile", "testprofile", "--show-secret"})

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

	if !strings.Contains(output, "super-secret-value") {
		t.Error("output should contain the secret value with --show-secret")
	}
}

// TestOAuthConfigShow_JSONOutput tests JSON output format.
func TestOAuthConfigShow_JSONOutput(t *testing.T) {
	dataDir := setupTestDataDir(t)
	cfg := config.Config{DataDir: dataDir}
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		ClientSecret: "super-secret-value",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
		Scopes:       []string{"read", "write"},
		UsePKCE:      true,
	}
	createTestProfile(t, dataDir, "testprofile", oauth2)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runOAuthConfigShow(cfg, []string{"--profile", "testprofile", "--json"})

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

	var result auth.OAuth2Config
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.ClientID != "test-client-id" {
		t.Errorf("expected client_id 'test-client-id', got '%s'", result.ClientID)
	}
	// Secret should be redacted in JSON output too
	if result.ClientSecret != "***" {
		t.Errorf("expected redacted secret '***', got '%s'", result.ClientSecret)
	}
}

// TestOAuthConfigSet_MissingProfile tests config set with missing --profile.
