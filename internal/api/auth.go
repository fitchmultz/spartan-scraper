// Package api provides HTTP handlers for authentication profile management endpoints.
// Auth handlers support listing, creating, updating, deleting auth profiles,
// importing/exporting profiles, and managing profile presets.
// It also provides OAuth 2.0 and OIDC authentication flow support.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

// OAuthInitiateRequest represents a request to initiate an OAuth flow.
type OAuthInitiateRequest struct {
	ProfileName string `json:"profile_name"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}

// OAuthInitiateResponse represents the response from initiating an OAuth flow.
type OAuthInitiateResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
}

// OAuthCallbackResponse represents the response from an OAuth callback.
type OAuthCallbackResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

// OAuthRefreshRequest represents a request to refresh an OAuth token.
type OAuthRefreshRequest struct {
	ProfileName string `json:"profile_name"`
}

// OIDCDiscoverRequest represents a request to perform OIDC discovery.
type OIDCDiscoverRequest struct {
	DiscoveryURL string `json:"discovery_url,omitempty"`
	Issuer       string `json:"issuer,omitempty"`
}

func (s *Server) handleAuthProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	vault, err := auth.LoadVault(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, map[string]any{"profiles": vault.Profiles})
}

func (s *Server) handleAuthProfile(w http.ResponseWriter, r *http.Request) {
	name := extractID(r.URL.Path, "profiles")
	if name == "" {
		writeError(w, r, apperrors.Validation("name required"))
		return
	}
	switch r.Method {
	case http.MethodPut:
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		var profile auth.Profile
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&profile); err != nil {
			writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
			return
		}
		if profile.Name == "" {
			profile.Name = name
		}
		if profile.Name != name {
			writeError(w, r, apperrors.Validation("profile name mismatch"))
			return
		}
		if err := validate.ValidateAuthProfileName(profile.Name); err != nil {
			writeError(w, r, apperrors.Validation(err.Error()))
			return
		}
		if err := auth.UpsertProfile(s.cfg.DataDir, profile); err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, profile)
	case http.MethodDelete:
		if err := auth.DeleteProfile(s.cfg.DataDir, name); err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleAuthImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var payload struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
		return
	}
	if err := auth.ImportVault(s.cfg.DataDir, payload.Path); err != nil {
		if errors.Is(err, auth.ErrInvalidPath) || err.Error() == "path is required" {
			writeError(w, r, err)
			return
		}
		writeError(w, r, err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleAuthExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var payload struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
		return
	}
	if err := auth.ExportVault(s.cfg.DataDir, payload.Path); err != nil {
		if errors.Is(err, auth.ErrInvalidPath) || err.Error() == "path is required" {
			writeError(w, r, err)
			return
		}
		writeError(w, r, err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// handleOAuthInitiate starts an OAuth 2.0 authorization flow.
// POST /v1/auth/oauth/initiate
func (s *Server) handleOAuthInitiate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req OAuthInitiateRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
		return
	}

	if req.ProfileName == "" {
		writeError(w, r, apperrors.Validation("profile_name is required"))
		return
	}

	// Load the profile to get OAuth config
	profile, found, err := auth.GetProfile(s.cfg.DataDir, req.ProfileName)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("profile not found"))
		return
	}

	if profile.OAuth2 == nil {
		writeError(w, r, apperrors.Validation("profile does not have OAuth2 configuration"))
		return
	}

	oauthConfig := profile.OAuth2

	// Validate OAuth configuration
	if oauthConfig.FlowType != auth.OAuth2FlowAuthorizationCode {
		writeError(w, r, apperrors.Validation("only authorization_code flow is supported for initiate"))
		return
	}

	if oauthConfig.AuthorizeURL == "" {
		writeError(w, r, apperrors.Validation("authorize_url is required for authorization_code flow"))
		return
	}

	// Create OAuth state store
	oauthStore := auth.NewOAuthStore(s.cfg.DataDir)

	// Determine redirect URI: use request override if provided, otherwise use profile default
	redirectURI := req.RedirectURI
	if redirectURI == "" {
		redirectURI = oauthConfig.RedirectURI
	}

	// Generate state and PKCE (if enabled)
	stateStr, codeVerifier, err := oauthStore.CreateOAuthState(req.ProfileName, redirectURI, oauthConfig.UsePKCE)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Build PKCE challenge if using PKCE - derive from stored verifier
	var codeChallenge, codeChallengeMethod string
	if oauthConfig.UsePKCE {
		challenge, method, err := auth.DerivePKCEChallengeS256(codeVerifier)
		if err != nil {
			writeError(w, r, err)
			return
		}
		codeChallenge = challenge
		codeChallengeMethod = method
	}

	// Build authorization URL using the redirect URI stored in state
	// Create a copy of the config with the override redirect URI
	authConfigCopy := *oauthConfig
	authConfigCopy.RedirectURI = redirectURI
	authURL, err := auth.BuildAuthorizationURL(authConfigCopy, stateStr, codeChallenge, codeChallengeMethod)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, OAuthInitiateResponse{
		AuthorizationURL: authURL,
		State:            stateStr,
	})
}

// handleOAuthCallback handles the OAuth 2.0 callback from the provider.
// GET /v1/auth/oauth/callback?code=...&state=...
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	query := r.URL.Query()
	code := query.Get("code")
	state := query.Get("state")
	errorParam := query.Get("error")

	if errorParam != "" {
		writeError(w, r, apperrors.Internal("OAuth provider error: "+errorParam))
		return
	}

	if state == "" {
		writeError(w, r, apperrors.Validation("state parameter is required"))
		return
	}

	if code == "" {
		writeError(w, r, apperrors.Validation("code parameter is required"))
		return
	}

	// Load and validate state
	oauthStore := auth.NewOAuthStore(s.cfg.DataDir)
	oauthState, valid, err := oauthStore.LoadState(state)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !valid {
		writeError(w, r, apperrors.Validation("invalid or expired state"))
		return
	}

	// Delete state after use (one-time use)
	defer oauthStore.DeleteState(state)

	// Load profile to get OAuth config
	profile, found, err := auth.GetProfile(s.cfg.DataDir, oauthState.ProfileName)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("profile not found"))
		return
	}

	if profile.OAuth2 == nil {
		writeError(w, r, apperrors.Validation("profile does not have OAuth2 configuration"))
		return
	}

	// Exchange code for token
	token, err := auth.ExchangeAuthorizationCode(r.Context(), *profile.OAuth2, code, oauthState.CodeVerifier, oauthState.RedirectURI)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Save token
	if err := oauthStore.SaveToken(oauthState.ProfileName, *token); err != nil {
		writeError(w, r, err)
		return
	}

	// Calculate expires_in
	var expiresIn int
	if token.ExpiresAt != nil {
		expiresIn = int(time.Until(*token.ExpiresAt).Seconds())
	}

	writeJSON(w, OAuthCallbackResponse{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresIn:   expiresIn,
		Scope:       token.Scope,
	})
}

// handleOAuthRefresh manually refreshes an OAuth 2.0 token.
// POST /v1/auth/oauth/refresh
func (s *Server) handleOAuthRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req OAuthRefreshRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
		return
	}

	if req.ProfileName == "" {
		writeError(w, r, apperrors.Validation("profile_name is required"))
		return
	}

	// Load existing token
	oauthStore := auth.NewOAuthStore(s.cfg.DataDir)
	existingToken, found, err := oauthStore.LoadToken(req.ProfileName)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("no OAuth token found for profile"))
		return
	}

	if existingToken.RefreshToken == "" {
		writeError(w, r, apperrors.Validation("no refresh token available"))
		return
	}

	// Load profile to get OAuth config
	profile, found, err := auth.GetProfile(s.cfg.DataDir, req.ProfileName)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("profile not found"))
		return
	}

	if profile.OAuth2 == nil {
		writeError(w, r, apperrors.Validation("profile does not have OAuth2 configuration"))
		return
	}

	// Refresh token
	newToken, err := auth.RefreshOAuth2Token(r.Context(), *profile.OAuth2, existingToken.RefreshToken)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Save new token
	if err := oauthStore.SaveToken(req.ProfileName, *newToken); err != nil {
		writeError(w, r, err)
		return
	}

	// Calculate expires_in
	var expiresIn int
	if newToken.ExpiresAt != nil {
		expiresIn = int(time.Until(*newToken.ExpiresAt).Seconds())
	}

	writeJSON(w, OAuthCallbackResponse{
		AccessToken: newToken.AccessToken,
		TokenType:   newToken.TokenType,
		ExpiresIn:   expiresIn,
		Scope:       newToken.Scope,
	})
}

// handleOIDCDiscover performs OIDC discovery for a provider.
// POST /v1/auth/oauth/discover
func (s *Server) handleOIDCDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req OIDCDiscoverRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
		return
	}

	var metadata *auth.OIDCProviderMetadata
	var err error

	if req.DiscoveryURL != "" {
		metadata, err = auth.OIDCDiscover(r.Context(), req.DiscoveryURL)
	} else if req.Issuer != "" {
		metadata, err = auth.OIDCDiscoverFromIssuer(r.Context(), req.Issuer)
	} else {
		writeError(w, r, apperrors.Validation("discovery_url or issuer is required"))
		return
	}

	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, metadata)
}

// handleOAuthRevoke revokes an OAuth 2.0 token.
// POST /v1/auth/oauth/revoke
func (s *Server) handleOAuthRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req struct {
		ProfileName string `json:"profile_name"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
		return
	}

	if req.ProfileName == "" {
		writeError(w, r, apperrors.Validation("profile_name is required"))
		return
	}

	// Load existing token
	oauthStore := auth.NewOAuthStore(s.cfg.DataDir)
	existingToken, found, err := oauthStore.LoadToken(req.ProfileName)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("no OAuth token found for profile"))
		return
	}

	// Load profile to get OAuth config
	profile, found, err := auth.GetProfile(s.cfg.DataDir, req.ProfileName)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("profile not found"))
		return
	}

	if profile.OAuth2 == nil {
		writeError(w, r, apperrors.Validation("profile does not have OAuth2 configuration"))
		return
	}

	// Revoke token if revoke URL is configured
	if profile.OAuth2.RevokeURL != "" {
		if err := auth.RevokeOAuth2Token(r.Context(), profile.OAuth2.RevokeURL, profile.OAuth2.ClientID, profile.OAuth2.ClientSecret, existingToken.AccessToken, "access_token"); err != nil {
			// Log but don't fail - still delete the token locally
			slog.Error("failed to revoke token at provider", "error", apperrors.SafeMessage(err))
		}
	}

	// Delete token locally
	if err := oauthStore.DeleteToken(req.ProfileName); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}
