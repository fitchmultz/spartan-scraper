// Package manage contains CLI commands for configuration/data management.
//
// This file contains unit tests for OAuth CLI subcommands.
package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// setupTestDataDir creates a temporary data directory for tests.
func setupTestDataDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// createTestProfile creates a test profile with optional OAuth2 config.
func createTestProfile(t *testing.T, dataDir, name string, oauth2 *auth.OAuth2Config) {
	t.Helper()
	profile := auth.Profile{
		Name:   name,
		OAuth2: oauth2,
	}
	if err := auth.UpsertProfile(dataDir, profile); err != nil {
		t.Fatalf("failed to create test profile: %v", err)
	}
}

// TestOAuthConfigShow_MissingProfile tests config show with missing --profile.
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
func TestDerivePKCEChallengeS256(t *testing.T) {
	// Generate a verifier first
	verifier, _, _, err := auth.GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE failed: %v", err)
	}

	// Derive challenge from verifier
	challenge, method, err := auth.DerivePKCEChallengeS256(verifier)
	if err != nil {
		t.Fatalf("DerivePKCEChallengeS256 failed: %v", err)
	}

	if challenge == "" {
		t.Error("challenge should not be empty")
	}
	if method != auth.PKCEMethodS256 {
		t.Errorf("expected method '%s', got '%s'", auth.PKCEMethodS256, method)
	}
}

// TestDerivePKCEChallengeS256_EmptyVerifier tests with empty verifier.
func TestDerivePKCEChallengeS256_EmptyVerifier(t *testing.T) {
	_, _, err := auth.DerivePKCEChallengeS256("")
	if err == nil {
		t.Error("expected error for empty verifier")
	}
}

// TestDerivePKCEChallengeS256_Consistency tests that the same verifier produces the same challenge.
func TestDerivePKCEChallengeS256_Consistency(t *testing.T) {
	verifier := "test-verifier-string"

	challenge1, method1, err1 := auth.DerivePKCEChallengeS256(verifier)
	challenge2, method2, err2 := auth.DerivePKCEChallengeS256(verifier)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v, %v", err1, err2)
	}

	if challenge1 != challenge2 {
		t.Error("same verifier should produce same challenge")
	}
	if method1 != method2 {
		t.Error("same verifier should produce same method")
	}
}
