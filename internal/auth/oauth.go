// Package auth provides OAuth 2.0 and OIDC authentication support.
// It handles authorization flows, token refresh, and OIDC discovery.
// It does NOT handle the actual HTTP fetch operations (see fetch package).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// PKCE code challenge methods.
const (
	PKCEMethodS256  = "S256"
	PKCEMethodPlain = "plain"
)

// Clock skew buffer for token expiration checks.
const defaultExpiryBuffer = 60 * time.Second

// GeneratePKCE generates a PKCE code verifier and challenge.
// Returns the verifier, challenge, method, and any error.
func GeneratePKCE() (verifier string, challenge string, method string, err error) {
	// Generate 32 bytes of random data for the verifier (recommended minimum)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", "", "", apperrors.Wrap(apperrors.KindInternal, "failed to generate PKCE verifier", err)
	}

	// Base64URL encode without padding
	verifier = base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate S256 challenge
	challenge, method, err = DerivePKCEChallengeS256(verifier)
	if err != nil {
		return "", "", "", err
	}

	return verifier, challenge, method, nil
}

// DerivePKCEChallengeS256 derives the S256 code challenge from a verifier.
// This ensures the challenge is always derived from the stored verifier.
// Returns the challenge, method (S256), and any error.
func DerivePKCEChallengeS256(verifier string) (challenge string, method string, err error) {
	if strings.TrimSpace(verifier) == "" {
		return "", "", apperrors.Validation("PKCE verifier cannot be empty")
	}

	// Generate S256 challenge: base64url(sha256(verifier))
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return challenge, PKCEMethodS256, nil
}

// GenerateOAuthState generates a cryptographically secure state parameter.
// The state parameter is used for CSRF protection in OAuth flows.
func GenerateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to generate OAuth state", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// OIDCDiscover fetches OIDC provider metadata from discovery URL.
// The discoveryURL should be the full URL to the .well-known/openid-configuration endpoint.
func OIDCDiscover(ctx context.Context, discoveryURL string) (*OIDCProviderMetadata, error) {
	if strings.TrimSpace(discoveryURL) == "" {
		return nil, apperrors.Validation("discovery URL is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create discovery request", err)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "discovery request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, apperrors.Internal(fmt.Sprintf("discovery returned status %d: %s", resp.StatusCode, string(body)))
	}

	var metadata OIDCProviderMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decode discovery response", err)
	}

	// Validate required fields
	if metadata.Issuer == "" {
		return nil, apperrors.Validation("OIDC discovery response missing issuer")
	}
	if metadata.AuthorizationEndpoint == "" {
		return nil, apperrors.Validation("OIDC discovery response missing authorization_endpoint")
	}
	if metadata.TokenEndpoint == "" {
		return nil, apperrors.Validation("OIDC discovery response missing token_endpoint")
	}

	return &metadata, nil
}

// OIDCDiscoverFromIssuer performs discovery using issuer URL.
// It appends the standard .well-known/openid-configuration path to the issuer.
func OIDCDiscoverFromIssuer(ctx context.Context, issuer string) (*OIDCProviderMetadata, error) {
	if strings.TrimSpace(issuer) == "" {
		return nil, apperrors.Validation("issuer is required")
	}

	// Ensure issuer doesn't have trailing slash before appending path
	issuer = strings.TrimSuffix(issuer, "/")
	discoveryURL := issuer + "/.well-known/openid-configuration"

	return OIDCDiscover(ctx, discoveryURL)
}

// BuildAuthorizationURL constructs the OAuth 2.0 authorization URL.
// For PKCE, provide the codeChallenge and codeChallengeMethod (typically "S256").
func BuildAuthorizationURL(config OAuth2Config, state string, codeChallenge string, codeChallengeMethod string) (string, error) {
	if strings.TrimSpace(config.AuthorizeURL) == "" {
		return "", apperrors.Validation("authorize URL is required")
	}
	if strings.TrimSpace(config.ClientID) == "" {
		return "", apperrors.Validation("client ID is required")
	}
	if strings.TrimSpace(state) == "" {
		return "", apperrors.Validation("state is required")
	}

	u, err := url.Parse(config.AuthorizeURL)
	if err != nil {
		return "", apperrors.Validation(fmt.Sprintf("invalid authorize URL: %v", err))
	}

	q := u.Query()
	q.Set("client_id", config.ClientID)
	q.Set("response_type", "code")
	q.Set("state", state)

	if config.RedirectURI != "" {
		q.Set("redirect_uri", config.RedirectURI)
	}

	if len(config.Scopes) > 0 {
		q.Set("scope", strings.Join(config.Scopes, " "))
	}

	// Add PKCE parameters if provided
	if codeChallenge != "" {
		q.Set("code_challenge", codeChallenge)
		if codeChallengeMethod != "" {
			q.Set("code_challenge_method", codeChallengeMethod)
		}
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// TokenExchangeResponse represents the response from a token exchange.
type TokenExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ExchangeAuthorizationCode exchanges an authorization code for tokens.
// The codeVerifier is required for PKCE flows.
func ExchangeAuthorizationCode(ctx context.Context, config OAuth2Config, code string, codeVerifier string, redirectURI string) (*OAuth2Token, error) {
	if strings.TrimSpace(config.TokenURL) == "" {
		return nil, apperrors.Validation("token URL is required")
	}
	if strings.TrimSpace(config.ClientID) == "" {
		return nil, apperrors.Validation("client ID is required")
	}
	if strings.TrimSpace(code) == "" {
		return nil, apperrors.Validation("authorization code is required")
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", config.ClientID)
	data.Set("code", code)

	if config.ClientSecret != "" {
		data.Set("client_secret", config.ClientSecret)
	}

	if redirectURI != "" {
		data.Set("redirect_uri", redirectURI)
	}

	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create token request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "token exchange request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read token response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, apperrors.Internal(fmt.Sprintf("token exchange returned status %d: %s", resp.StatusCode, string(body)))
	}

	var exchangeResp TokenExchangeResponse
	if err := json.Unmarshal(body, &exchangeResp); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decode token response", err)
	}

	token := &OAuth2Token{
		AccessToken:  exchangeResp.AccessToken,
		RefreshToken: exchangeResp.RefreshToken,
		TokenType:    exchangeResp.TokenType,
		Scope:        exchangeResp.Scope,
	}

	if exchangeResp.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(exchangeResp.ExpiresIn) * time.Second)
		token.ExpiresAt = &expiry
	}

	return token, nil
}

// RefreshOAuth2Token refreshes an OAuth 2.0 access token using the refresh token.
func RefreshOAuth2Token(ctx context.Context, config OAuth2Config, refreshToken string) (*OAuth2Token, error) {
	if strings.TrimSpace(config.TokenURL) == "" {
		return nil, apperrors.Validation("token URL is required")
	}
	if strings.TrimSpace(config.ClientID) == "" {
		return nil, apperrors.Validation("client ID is required")
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, apperrors.Validation("refresh token is required")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", config.ClientID)
	data.Set("refresh_token", refreshToken)

	if config.ClientSecret != "" {
		data.Set("client_secret", config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create refresh request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "token refresh request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read refresh response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, apperrors.Internal(fmt.Sprintf("token refresh returned status %d: %s", resp.StatusCode, string(body)))
	}

	var exchangeResp TokenExchangeResponse
	if err := json.Unmarshal(body, &exchangeResp); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decode refresh response", err)
	}

	token := &OAuth2Token{
		AccessToken: exchangeResp.AccessToken,
		TokenType:   exchangeResp.TokenType,
		Scope:       exchangeResp.Scope,
	}

	// Use new refresh token if provided, otherwise keep the old one
	if exchangeResp.RefreshToken != "" {
		token.RefreshToken = exchangeResp.RefreshToken
	} else {
		token.RefreshToken = refreshToken
	}

	if exchangeResp.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(exchangeResp.ExpiresIn) * time.Second)
		token.ExpiresAt = &expiry
	}

	return token, nil
}

// IsTokenExpired checks if an OAuth2 token is expired (with clock skew buffer).
func IsTokenExpired(token OAuth2Token, buffer time.Duration) bool {
	if token.ExpiresAt == nil {
		// No expiration = never expired
		return false
	}

	if buffer <= 0 {
		buffer = defaultExpiryBuffer
	}

	return time.Now().Add(buffer).After(*token.ExpiresAt)
}

// RevokeOAuth2Token revokes an OAuth 2.0 token at the revocation endpoint.
func RevokeOAuth2Token(ctx context.Context, revokeURL, clientID, clientSecret, token string, tokenTypeHint string) error {
	if strings.TrimSpace(revokeURL) == "" {
		return apperrors.Validation("revoke URL is required")
	}
	if strings.TrimSpace(clientID) == "" {
		return apperrors.Validation("client ID is required")
	}
	if strings.TrimSpace(token) == "" {
		return apperrors.Validation("token is required")
	}

	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", clientID)

	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	if tokenTypeHint != "" {
		data.Set("token_type_hint", tokenTypeHint)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create revoke request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "token revoke request failed", err)
	}
	defer resp.Body.Close()

	// Revocation endpoint returns 200 on success
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return apperrors.Internal(fmt.Sprintf("token revoke returned status %d: %s", resp.StatusCode, string(body)))
	}

	return nil
}
