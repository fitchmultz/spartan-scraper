// Package api provides HTTP handlers for authentication profile management endpoints.
//
// This file contains tests for OAuth handlers, specifically testing PKCE challenge
// correctness and redirect URI override behavior.
package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
)

// setupTestOAuthServer creates a test server with a temporary data directory.
// Uses the shared setupTestServer from setup_test.go.
func setupTestOAuthServer(t *testing.T) (*Server, string, func()) {
	t.Helper()
	server, cleanup := setupTestServer(t)
	return server, server.cfg.DataDir, cleanup
}

// createTestProfileWithOAuth creates a test profile with OAuth2 configuration.
func createTestProfileWithOAuth(t *testing.T, dataDir string, name string, oauth2 *auth.OAuth2Config) {
	t.Helper()
	profile := auth.Profile{
		Name:   name,
		OAuth2: oauth2,
	}
	if err := auth.UpsertProfile(dataDir, profile); err != nil {
		t.Fatalf("failed to create test profile: %v", err)
	}
}

// TestHandleOAuthInitiate_PKCEChallengeMatchesStoredVerifier tests that the PKCE challenge
// in the authorization URL matches the stored verifier.
func TestHandleOAuthInitiate_PKCEChallengeMatchesStoredVerifier(t *testing.T) {
	server, dataDir, cleanup := setupTestOAuthServer(t)
	defer cleanup()

	// Create a profile with PKCE enabled
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
		UsePKCE:      true,
		RedirectURI:  "https://example.com/callback",
	}
	createTestProfileWithOAuth(t, dataDir, "testprofile", oauth2)

	// Make the request
	reqBody := OAuthInitiateRequest{
		ProfileName: "testprofile",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/initiate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleOAuthInitiate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response OAuthInitiateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Load the stored state to get the verifier
	oauthStore := auth.NewOAuthStore(dataDir)
	oauthState, valid, err := oauthStore.LoadState(response.State)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if !valid {
		t.Fatal("state should be valid")
	}

	// Verify the verifier is not empty
	if oauthState.CodeVerifier == "" {
		t.Fatal("code verifier should not be empty")
	}

	// Compute the expected challenge from the stored verifier
	hash := sha256.Sum256([]byte(oauthState.CodeVerifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Check that the authorization URL contains the correct challenge
	if !strings.Contains(response.AuthorizationURL, "code_challenge="+expectedChallenge) {
		t.Errorf("authorization URL should contain code_challenge=%s", expectedChallenge)
	}
	if !strings.Contains(response.AuthorizationURL, "code_challenge_method=S256") {
		t.Error("authorization URL should contain code_challenge_method=S256")
	}
}

// TestHandleOAuthInitiate_RedirectURIOverride tests that a redirect URI override
// is correctly applied to the authorization URL and stored in state.
func TestHandleOAuthInitiate_RedirectURIOverride(t *testing.T) {
	server, dataDir, cleanup := setupTestOAuthServer(t)
	defer cleanup()

	// Create a profile with a default redirect URI
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
		RedirectURI:  "https://default.example.com/callback",
	}
	createTestProfileWithOAuth(t, dataDir, "testprofile", oauth2)

	// Make the request with redirect URI override
	reqBody := OAuthInitiateRequest{
		ProfileName: "testprofile",
		RedirectURI: "https://override.example.com/callback",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/initiate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleOAuthInitiate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response OAuthInitiateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check that the authorization URL contains the override redirect URI
	if !strings.Contains(response.AuthorizationURL, "override.example.com") {
		t.Errorf("authorization URL should contain the override redirect URI, got: %s", response.AuthorizationURL)
	}

	// Load the stored state to verify the redirect URI was stored
	oauthStore := auth.NewOAuthStore(dataDir)
	oauthState, valid, err := oauthStore.LoadState(response.State)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if !valid {
		t.Fatal("state should be valid")
	}

	// Verify the redirect URI in state matches the override
	if oauthState.RedirectURI != "https://override.example.com/callback" {
		t.Errorf("stored redirect URI should be 'https://override.example.com/callback', got: %s", oauthState.RedirectURI)
	}
}

// TestHandleOAuthInitiate_NoPKCE tests initiation without PKCE.
func TestHandleOAuthInitiate_NoPKCE(t *testing.T) {
	server, dataDir, cleanup := setupTestOAuthServer(t)
	defer cleanup()

	// Create a profile without PKCE
	oauth2 := &auth.OAuth2Config{
		FlowType:     auth.OAuth2FlowAuthorizationCode,
		ClientID:     "test-client-id",
		TokenURL:     "https://example.com/token",
		AuthorizeURL: "https://example.com/authorize",
		UsePKCE:      false,
		RedirectURI:  "https://example.com/callback",
	}
	createTestProfileWithOAuth(t, dataDir, "testprofile", oauth2)

	// Make the request
	reqBody := OAuthInitiateRequest{
		ProfileName: "testprofile",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/initiate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleOAuthInitiate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response OAuthInitiateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check that the authorization URL does NOT contain PKCE parameters
	if strings.Contains(response.AuthorizationURL, "code_challenge=") {
		t.Error("authorization URL should not contain code_challenge when PKCE is disabled")
	}

	// Load the stored state to verify no verifier was stored
	oauthStore := auth.NewOAuthStore(dataDir)
	oauthState, valid, err := oauthStore.LoadState(response.State)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if !valid {
		t.Fatal("state should be valid")
	}

	// Verify the verifier is empty
	if oauthState.CodeVerifier != "" {
		t.Error("code verifier should be empty when PKCE is disabled")
	}
}

// TestHandleOAuthInitiate_MissingProfile tests initiation with non-existent profile.
func TestHandleOAuthInitiate_MissingProfile(t *testing.T) {
	server, _, cleanup := setupTestOAuthServer(t)
	defer cleanup()

	// Make the request with non-existent profile
	reqBody := OAuthInitiateRequest{
		ProfileName: "nonexistent",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/initiate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleOAuthInitiate(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestHandleOAuthInitiate_NoOAuthConfig tests initiation with profile that has no OAuth2 config.
func TestHandleOAuthInitiate_NoOAuthConfig(t *testing.T) {
	server, dataDir, cleanup := setupTestOAuthServer(t)
	defer cleanup()

	// Create a profile without OAuth2 config
	profile := auth.Profile{Name: "testprofile"}
	if err := auth.UpsertProfile(dataDir, profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// Make the request
	reqBody := OAuthInitiateRequest{
		ProfileName: "testprofile",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/initiate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleOAuthInitiate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleOAuthInitiate_WrongFlowType tests initiation with wrong flow type.
func TestHandleOAuthInitiate_WrongFlowType(t *testing.T) {
	server, dataDir, cleanup := setupTestOAuthServer(t)
	defer cleanup()

	// Create a profile with client_credentials flow
	oauth2 := &auth.OAuth2Config{
		FlowType: auth.OAuth2FlowClientCredentials,
		ClientID: "test-client-id",
		TokenURL: "https://example.com/token",
	}
	createTestProfileWithOAuth(t, dataDir, "testprofile", oauth2)

	// Make the request
	reqBody := OAuthInitiateRequest{
		ProfileName: "testprofile",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/initiate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleOAuthInitiate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleOAuthInitiate_MissingAuthorizeURL tests initiation without authorize URL.
func TestHandleOAuthInitiate_MissingAuthorizeURL(t *testing.T) {
	server, dataDir, cleanup := setupTestOAuthServer(t)
	defer cleanup()

	// Create a profile without authorize URL
	oauth2 := &auth.OAuth2Config{
		FlowType: auth.OAuth2FlowAuthorizationCode,
		ClientID: "test-client-id",
		TokenURL: "https://example.com/token",
		// No AuthorizeURL
	}
	createTestProfileWithOAuth(t, dataDir, "testprofile", oauth2)

	// Make the request
	reqBody := OAuthInitiateRequest{
		ProfileName: "testprofile",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/initiate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleOAuthInitiate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestDerivePKCEChallengeS256_API tests the DerivePKCEChallengeS256 helper via the auth package.
func TestDerivePKCEChallengeS256_API(t *testing.T) {
	// Generate a verifier
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

	// Verify the challenge is correctly computed
	hash := sha256.Sum256([]byte(verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	if challenge != expectedChallenge {
		t.Errorf("challenge mismatch: expected %s, got %s", expectedChallenge, challenge)
	}
}

// TestDerivePKCEChallengeS256_EmptyVerifier_API tests with empty verifier.
func TestDerivePKCEChallengeS256_EmptyVerifier_API(t *testing.T) {
	_, _, err := auth.DerivePKCEChallengeS256("")
	if err == nil {
		t.Error("expected error for empty verifier")
	}
}
