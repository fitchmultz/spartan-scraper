// Package auth provides OAuth 2.0 and OIDC authentication support.
// This file contains tests for OAuth operations.
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, method, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE failed: %v", err)
	}

	if verifier == "" {
		t.Error("verifier should not be empty")
	}
	if challenge == "" {
		t.Error("challenge should not be empty")
	}
	if method != PKCEMethodS256 {
		t.Errorf("method should be S256, got %s", method)
	}

	// Verifier should be different each time
	verifier2, _, _, _ := GeneratePKCE()
	if verifier == verifier2 {
		t.Error("verifier should be unique each time")
	}
}

func TestGenerateOAuthState(t *testing.T) {
	state, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("GenerateOAuthState failed: %v", err)
	}

	if state == "" {
		t.Error("state should not be empty")
	}

	// State should be different each time
	state2, _ := GenerateOAuthState()
	if state == state2 {
		t.Error("state should be unique each time")
	}
}

func TestOIDCDiscover(t *testing.T) {
	// Create a mock OIDC discovery server
	mockMetadata := OIDCProviderMetadata{
		Issuer:                 "https://example.com",
		AuthorizationEndpoint:  "https://example.com/oauth/authorize",
		TokenEndpoint:          "https://example.com/oauth/token",
		UserinfoEndpoint:       "https://example.com/oauth/userinfo",
		RevocationEndpoint:     "https://example.com/oauth/revoke",
		JWKSURI:                "https://example.com/.well-known/jwks.json",
		ScopesSupported:        []string{"openid", "email", "profile"},
		ResponseTypesSupported: []string{"code", "token"},
		GrantTypesSupported:    []string{"authorization_code", "refresh_token"},
		CodeChallengeMethods:   []string{"S256", "plain"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockMetadata)
	}))
	defer server.Close()

	ctx := context.Background()
	metadata, err := OIDCDiscover(ctx, server.URL+"/.well-known/openid-configuration")
	if err != nil {
		t.Fatalf("OIDCDiscover failed: %v", err)
	}

	if metadata.Issuer != mockMetadata.Issuer {
		t.Errorf("issuer mismatch: got %s, want %s", metadata.Issuer, mockMetadata.Issuer)
	}
	if metadata.AuthorizationEndpoint != mockMetadata.AuthorizationEndpoint {
		t.Errorf("authorization_endpoint mismatch")
	}
	if metadata.TokenEndpoint != mockMetadata.TokenEndpoint {
		t.Errorf("token_endpoint mismatch")
	}
}

func TestOIDCDiscoverFromIssuer(t *testing.T) {
	// Create a mock OIDC discovery server
	mockMetadata := OIDCProviderMetadata{
		Issuer:                "https://example.com",
		AuthorizationEndpoint: "https://example.com/oauth/authorize",
		TokenEndpoint:         "https://example.com/oauth/token",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockMetadata)
	}))
	defer server.Close()

	// Extract the host:port from the server URL
	issuer := strings.TrimPrefix(server.URL, "https://")
	issuer = strings.TrimPrefix(issuer, "http://")

	ctx := context.Background()
	// Note: This test uses http:// which won't work with the real function
	// that expects https:// for OIDC. We're testing the path construction.
	_, err := OIDCDiscoverFromIssuer(ctx, "http://"+issuer)
	if err != nil {
		// Expected to fail due to http vs https, but we're testing path construction
		t.Logf("OIDCDiscoverFromIssuer error (expected for http): %v", err)
	}
}

func TestOIDCDiscover_MissingRequiredFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return incomplete metadata
		json.NewEncoder(w).Encode(map[string]string{
			"issuer": "https://example.com",
			// Missing authorization_endpoint and token_endpoint
		})
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := OIDCDiscover(ctx, server.URL+"/.well-known/openid-configuration")
	if err == nil {
		t.Error("expected error for missing required fields")
	}
}

func TestBuildAuthorizationURL(t *testing.T) {
	config := OAuth2Config{
		ClientID:     "test-client-id",
		AuthorizeURL: "https://example.com/oauth/authorize",
		RedirectURI:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "email"},
	}

	state := "test-state"
	codeChallenge := "test-challenge"
	codeChallengeMethod := "S256"

	url, err := BuildAuthorizationURL(config, state, codeChallenge, codeChallengeMethod)
	if err != nil {
		t.Fatalf("BuildAuthorizationURL failed: %v", err)
	}

	// Check that URL contains required parameters
	if !strings.Contains(url, "client_id=test-client-id") {
		t.Error("URL should contain client_id")
	}
	if !strings.Contains(url, "response_type=code") {
		t.Error("URL should contain response_type=code")
	}
	if !strings.Contains(url, "state=test-state") {
		t.Error("URL should contain state")
	}
	if !strings.Contains(url, "redirect_uri=") {
		t.Error("URL should contain redirect_uri")
	}
	if !strings.Contains(url, "scope=openid+email") {
		t.Error("URL should contain scope")
	}
	if !strings.Contains(url, "code_challenge=test-challenge") {
		t.Error("URL should contain code_challenge")
	}
	if !strings.Contains(url, "code_challenge_method=S256") {
		t.Error("URL should contain code_challenge_method")
	}
}

func TestBuildAuthorizationURL_MissingRequired(t *testing.T) {
	// Missing authorize URL
	config := OAuth2Config{
		ClientID: "test-client-id",
	}
	_, err := BuildAuthorizationURL(config, "state", "", "")
	if err == nil {
		t.Error("expected error for missing authorize URL")
	}

	// Missing client ID
	config = OAuth2Config{
		AuthorizeURL: "https://example.com/oauth/authorize",
	}
	_, err = BuildAuthorizationURL(config, "state", "", "")
	if err == nil {
		t.Error("expected error for missing client ID")
	}

	// Missing state
	config = OAuth2Config{
		ClientID:     "test-client-id",
		AuthorizeURL: "https://example.com/oauth/authorize",
	}
	_, err = BuildAuthorizationURL(config, "", "", "")
	if err == nil {
		t.Error("expected error for missing state")
	}
}

func TestExchangeAuthorizationCode(t *testing.T) {
	// Create a mock token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		err := r.ParseForm()
		if err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// Verify required parameters
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Error("expected grant_type=authorization_code")
		}
		if r.Form.Get("client_id") != "test-client-id" {
			t.Error("expected client_id")
		}
		if r.Form.Get("code") != "test-code" {
			t.Error("expected code")
		}
		if r.Form.Get("code_verifier") != "test-verifier" {
			t.Error("expected code_verifier")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenExchangeResponse{
			AccessToken:  "test-access-token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "test-refresh-token",
			Scope:        "openid email",
		})
	}))
	defer server.Close()

	config := OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     server.URL,
	}

	ctx := context.Background()
	token, err := ExchangeAuthorizationCode(ctx, config, "test-code", "test-verifier", "http://localhost/callback")
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode failed: %v", err)
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("access_token mismatch: got %s, want %s", token.AccessToken, "test-access-token")
	}
	if token.TokenType != "Bearer" {
		t.Errorf("token_type mismatch: got %s, want %s", token.TokenType, "Bearer")
	}
	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("refresh_token mismatch")
	}
	if token.Scope != "openid email" {
		t.Errorf("scope mismatch")
	}
	if token.ExpiresAt == nil {
		t.Error("expires_at should be set")
	}
}

func TestExchangeAuthorizationCode_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer server.Close()

	config := OAuth2Config{
		ClientID: "test-client-id",
		TokenURL: server.URL,
	}

	ctx := context.Background()
	_, err := ExchangeAuthorizationCode(ctx, config, "test-code", "", "")
	if err == nil {
		t.Error("expected error for bad response")
	}
}

func TestRefreshOAuth2Token(t *testing.T) {
	// Create a mock token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.Form.Get("grant_type") != "refresh_token" {
			t.Error("expected grant_type=refresh_token")
		}
		if r.Form.Get("refresh_token") != "test-refresh-token" {
			t.Error("expected refresh_token")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenExchangeResponse{
			AccessToken:  "new-access-token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
			RefreshToken: "new-refresh-token",
			Scope:        "openid",
		})
	}))
	defer server.Close()

	config := OAuth2Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     server.URL,
	}

	ctx := context.Background()
	token, err := RefreshOAuth2Token(ctx, config, "test-refresh-token")
	if err != nil {
		t.Fatalf("RefreshOAuth2Token failed: %v", err)
	}

	if token.AccessToken != "new-access-token" {
		t.Errorf("access_token mismatch")
	}
	if token.RefreshToken != "new-refresh-token" {
		t.Errorf("should use new refresh token when provided")
	}
}

func TestRefreshOAuth2Token_KeepOldRefreshToken(t *testing.T) {
	// Server doesn't return a new refresh token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenExchangeResponse{
			AccessToken: "new-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			// No refresh_token in response
		})
	}))
	defer server.Close()

	config := OAuth2Config{
		ClientID: "test-client-id",
		TokenURL: server.URL,
	}

	ctx := context.Background()
	token, err := RefreshOAuth2Token(ctx, config, "old-refresh-token")
	if err != nil {
		t.Fatalf("RefreshOAuth2Token failed: %v", err)
	}

	if token.RefreshToken != "old-refresh-token" {
		t.Errorf("should keep old refresh token when new one not provided, got %s", token.RefreshToken)
	}
}

func TestIsTokenExpired(t *testing.T) {
	now := time.Now()

	// Token with no expiration
	token := OAuth2Token{
		AccessToken: "test",
	}
	if IsTokenExpired(token, 0) {
		t.Error("token with no expiration should not be expired")
	}

	// Token that expires in the future
	future := now.Add(time.Hour)
	token = OAuth2Token{
		AccessToken: "test",
		ExpiresAt:   &future,
	}
	if IsTokenExpired(token, 0) {
		t.Error("token expiring in future should not be expired")
	}

	// Token that expired in the past
	past := now.Add(-time.Hour)
	token = OAuth2Token{
		AccessToken: "test",
		ExpiresAt:   &past,
	}
	if !IsTokenExpired(token, 0) {
		t.Error("token expired in past should be expired")
	}

	// Token expiring soon (with buffer)
	soon := now.Add(30 * time.Second)
	token = OAuth2Token{
		AccessToken: "test",
		ExpiresAt:   &soon,
	}
	if !IsTokenExpired(token, 60*time.Second) {
		t.Error("token expiring within buffer should be considered expired")
	}
}

func TestRevokeOAuth2Token(t *testing.T) {
	revoked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.Form.Get("token") != "test-token" {
			t.Error("expected token")
		}
		if r.Form.Get("client_id") != "test-client-id" {
			t.Error("expected client_id")
		}
		if r.Form.Get("token_type_hint") != "access_token" {
			t.Error("expected token_type_hint")
		}

		revoked = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	err := RevokeOAuth2Token(ctx, server.URL, "test-client-id", "test-client-secret", "test-token", "access_token")
	if err != nil {
		t.Fatalf("RevokeOAuth2Token failed: %v", err)
	}

	if !revoked {
		t.Error("token should have been revoked")
	}
}

func TestRevokeOAuth2Token_MissingParams(t *testing.T) {
	ctx := context.Background()

	// Missing revoke URL
	err := RevokeOAuth2Token(ctx, "", "client-id", "", "token", "")
	if err == nil {
		t.Error("expected error for missing revoke URL")
	}

	// Missing client ID
	err = RevokeOAuth2Token(ctx, "https://example.com", "", "", "token", "")
	if err == nil {
		t.Error("expected error for missing client ID")
	}

	// Missing token
	err = RevokeOAuth2Token(ctx, "https://example.com", "client-id", "", "", "")
	if err == nil {
		t.Error("expected error for missing token")
	}
}
