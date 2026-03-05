// Package auth provides OAuth 2.0 and OIDC authentication support.
// This file implements an HTTP transport wrapper for automatic token refresh.
package auth

import (
	"context"
	"net/http"
	"sync"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// OAuthTransport wraps an http.RoundTripper with automatic token refresh.
// It intercepts requests to add the Authorization header and automatically
// refreshes expired tokens before they are used.
type OAuthTransport struct {
	base         http.RoundTripper
	store        *OAuthStore
	profileName  string
	config       OAuth2Config
	mu           sync.RWMutex
	currentToken *OAuth2Token
}

// NewOAuthTransport creates a new OAuth-enabled transport.
// If base is nil, http.DefaultTransport is used.
func NewOAuthTransport(base http.RoundTripper, store *OAuthStore, profileName string, config OAuth2Config) *OAuthTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &OAuthTransport{
		base:        base,
		store:       store,
		profileName: profileName,
		config:      config,
	}
}

// RoundTrip implements http.RoundTripper with token refresh.
// It adds the Authorization header and handles automatic token refresh.
func (t *OAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get a valid token (refreshing if necessary)
	token, err := t.getValidToken(req.Context())
	if err != nil {
		return nil, err
	}

	// Clone the request to avoid modifying the original
	newReq := req.Clone(req.Context())

	// Add Authorization header
	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	newReq.Header.Set("Authorization", tokenType+" "+token.AccessToken)

	// Perform the request
	resp, err := t.base.RoundTrip(newReq)
	if err != nil {
		return nil, err
	}

	// If we get a 401, try to refresh the token and retry once
	if resp.StatusCode == http.StatusUnauthorized && t.config.FlowType == OAuth2FlowAuthorizationCode {
		resp.Body.Close()

		// Force a token refresh
		if err := t.refreshToken(req.Context()); err != nil {
			return nil, err
		}

		// Get the new token and retry
		token, err = t.getValidToken(req.Context())
		if err != nil {
			return nil, err
		}

		// Clone and retry
		retryReq := req.Clone(req.Context())
		retryReq.Header.Set("Authorization", tokenType+" "+token.AccessToken)

		return t.base.RoundTrip(retryReq)
	}

	return resp, nil
}

// getValidToken returns a non-expired token, refreshing if necessary.
// This method is thread-safe.
func (t *OAuthTransport) getValidToken(ctx context.Context) (*OAuth2Token, error) {
	t.mu.RLock()
	token := t.currentToken
	t.mu.RUnlock()

	// Check if we have a valid cached token
	if token != nil && !IsTokenExpired(*token, defaultExpiryBuffer) {
		return token, nil
	}

	// Need to load/refresh token - acquire write lock
	t.mu.Lock()
	defer t.mu.Unlock()

	// Double-check after acquiring lock
	if t.currentToken != nil && !IsTokenExpired(*t.currentToken, defaultExpiryBuffer) {
		return t.currentToken, nil
	}

	// Try to load from store
	storedToken, found, err := t.store.LoadToken(t.profileName)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, apperrors.NotFound("no OAuth token found for profile: " + t.profileName)
	}

	// Check if stored token is expired
	if IsTokenExpired(storedToken, defaultExpiryBuffer) {
		// Token is expired, try to refresh
		if storedToken.RefreshToken == "" {
			return nil, apperrors.Validation("OAuth token expired and no refresh token available")
		}

		refreshedToken, err := RefreshOAuth2Token(ctx, t.config, storedToken.RefreshToken)
		if err != nil {
			return nil, err
		}

		// Save the refreshed token
		if err := t.store.SaveToken(t.profileName, *refreshedToken); err != nil {
			return nil, err
		}

		storedToken = *refreshedToken
	}

	t.currentToken = &storedToken
	return t.currentToken, nil
}

// refreshToken performs token refresh and updates storage.
// This method should be called with the write lock held.
func (t *OAuthTransport) refreshToken(ctx context.Context) error {
	if t.config.FlowType != OAuth2FlowAuthorizationCode {
		return apperrors.Validation("token refresh only supported for authorization_code flow")
	}

	// Load current token to get refresh token
	storedToken, found, err := t.store.LoadToken(t.profileName)
	if err != nil {
		return err
	}
	if !found {
		return apperrors.NotFound("no OAuth token found for profile: " + t.profileName)
	}

	if storedToken.RefreshToken == "" {
		return apperrors.Validation("no refresh token available")
	}

	// Refresh the token
	refreshedToken, err := RefreshOAuth2Token(ctx, t.config, storedToken.RefreshToken)
	if err != nil {
		return err
	}

	// Save the refreshed token
	if err := t.store.SaveToken(t.profileName, *refreshedToken); err != nil {
		return err
	}

	t.currentToken = refreshedToken
	return nil
}

// GetCurrentToken returns the current token (for inspection/debugging).
// This returns a copy of the token to prevent external modification.
func (t *OAuthTransport) GetCurrentToken() (*OAuth2Token, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.currentToken == nil {
		// Try to load from store
		storedToken, found, err := t.store.LoadToken(t.profileName)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}
		t.currentToken = &storedToken
	}

	// Return a copy
	tokenCopy := *t.currentToken
	return &tokenCopy, nil
}

// InvalidateToken clears the cached token, forcing a reload from storage on next use.
// This is useful when you know the token has been revoked or updated externally.
func (t *OAuthTransport) InvalidateToken() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentToken = nil
}

// OAuthTransportConfig provides configuration for creating OAuth transports.
type OAuthTransportConfig struct {
	// ProfileName is the name of the profile with OAuth2 configuration
	ProfileName string
	// DataDir is the data directory for token storage
	DataDir string
}

// CreateOAuthTransport creates an OAuth transport from a profile.
// It loads the profile and its OAuth2 configuration from the vault.
func CreateOAuthTransport(base http.RoundTripper, cfg OAuthTransportConfig) (*OAuthTransport, error) {
	if cfg.ProfileName == "" {
		return nil, apperrors.Validation("profile name is required")
	}

	// Load profile
	profile, found, err := GetProfile(cfg.DataDir, cfg.ProfileName)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, apperrors.NotFound("profile not found: " + cfg.ProfileName)
	}

	if profile.OAuth2 == nil {
		return nil, apperrors.Validation("profile does not have OAuth2 configuration: " + cfg.ProfileName)
	}

	// Create store and transport
	store := NewOAuthStore(cfg.DataDir)
	return NewOAuthTransport(base, store, cfg.ProfileName, *profile.OAuth2), nil
}
