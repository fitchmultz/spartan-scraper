// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Execute OAuth authorization, callback, refresh, and revoke flows from the CLI.
//
// Responsibilities:
// - Initiate authorization_code flows with optional PKCE.
// - Exchange callback codes, refresh tokens, and revoke stored credentials.
// - Print stable human-readable and JSON summaries without exposing secrets by default.
//
// Scope:
// - `spartan auth oauth initiate|callback|refresh|revoke` only.
//
// Usage:
// - Invoked by `RunAuthOAuth` for operator-driven OAuth flows.
//
// Invariants/Assumptions:
// - Only authorization_code supports `initiate`.
// - Callback state is one-time-use and removed after successful load.
// - Access tokens are only printed when explicitly requested.
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// runOAuthInitiate starts an OAuth2 authorization flow.
func runOAuthInitiate(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth initiate", flag.ExitOnError)
	profileName := fs.String("profile", "", "Profile name (required)")
	redirectURI := fs.String("redirect-uri", "", "Redirect URI override (optional)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	_ = fs.Parse(args)

	if *profileName == "" {
		fmt.Fprintln(os.Stderr, "--profile is required")
		return 1
	}

	profile, found, err := auth.GetProfile(cfg.DataDir, *profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "profile not found: %s\n", *profileName)
		return 1
	}
	if profile.OAuth2 == nil {
		fmt.Fprintf(os.Stderr, "profile '%s' has no OAuth2 configuration\n", *profileName)
		return 1
	}
	if profile.OAuth2.FlowType != auth.OAuth2FlowAuthorizationCode {
		fmt.Fprintln(os.Stderr, "only authorization_code flow is supported for initiate")
		return 1
	}
	if profile.OAuth2.AuthorizeURL == "" {
		fmt.Fprintln(os.Stderr, "profile has no authorize_url configured")
		return 1
	}

	finalRedirectURI := *redirectURI
	if finalRedirectURI == "" {
		finalRedirectURI = profile.OAuth2.RedirectURI
	}

	oauthStore := auth.NewOAuthStore(cfg.DataDir)
	stateStr, codeVerifier, err := oauthStore.CreateOAuthState(*profileName, finalRedirectURI, profile.OAuth2.UsePKCE)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	var codeChallenge, codeChallengeMethod string
	if profile.OAuth2.UsePKCE {
		challenge, method, err := auth.DerivePKCEChallengeS256(codeVerifier)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		codeChallenge = challenge
		codeChallengeMethod = method
	}

	authConfigCopy := *profile.OAuth2
	authConfigCopy.RedirectURI = finalRedirectURI
	authURL, err := auth.BuildAuthorizationURL(authConfigCopy, stateStr, codeChallenge, codeChallengeMethod)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *jsonOutput {
		payload, _ := json.MarshalIndent(map[string]string{
			"authorization_url": authURL,
			"state":             stateStr,
		}, "", "  ")
		fmt.Println(string(payload))
		return 0
	}

	fmt.Println("OAuth2 Authorization Flow Initiated")
	fmt.Println()
	fmt.Println("Authorization URL:")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Printf("State: %s\n", stateStr)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Open the Authorization URL in your browser")
	fmt.Println("2. Complete the authorization")
	fmt.Println("3. Copy the 'code' parameter from the callback URL")
	fmt.Println("4. Run: spartan auth oauth callback --state", stateStr, "--code <code>")
	fmt.Println()
	fmt.Println("Note: The state expires in 10 minutes.")

	return 0
}

// runOAuthCallback exchanges an authorization code for a token.
func runOAuthCallback(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth callback", flag.ExitOnError)
	stateStr := fs.String("state", "", "State parameter (required)")
	code := fs.String("code", "", "Authorization code (required unless using --callback-url)")
	callbackURL := fs.String("callback-url", "", "Callback URL to parse code and state from")
	printAccessToken := fs.Bool("print-access-token", false, "Print the access token")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	_ = fs.Parse(args)

	if *callbackURL != "" {
		u, err := url.Parse(*callbackURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid callback URL: %v\n", err)
			return 1
		}
		q := u.Query()
		if *stateStr == "" {
			*stateStr = q.Get("state")
		}
		if *code == "" {
			*code = q.Get("code")
		}
	}

	if *stateStr == "" {
		fmt.Fprintln(os.Stderr, "--state is required (or must be present in --callback-url)")
		return 1
	}
	if *code == "" {
		fmt.Fprintln(os.Stderr, "--code is required (or must be present in --callback-url)")
		return 1
	}

	oauthStore := auth.NewOAuthStore(cfg.DataDir)
	oauthState, valid, err := oauthStore.LoadState(*stateStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !valid {
		fmt.Fprintln(os.Stderr, "invalid or expired state")
		return 1
	}
	defer oauthStore.DeleteState(*stateStr)

	profile, found, err := auth.GetProfile(cfg.DataDir, oauthState.ProfileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "profile not found: %s\n", oauthState.ProfileName)
		return 1
	}
	if profile.OAuth2 == nil {
		fmt.Fprintln(os.Stderr, "profile does not have OAuth2 configuration")
		return 1
	}

	token, err := auth.ExchangeAuthorizationCode(context.Background(), *profile.OAuth2, *code, oauthState.CodeVerifier, oauthState.RedirectURI)
	if err != nil {
		fmt.Fprintln(os.Stderr, "token exchange failed:", err)
		return 1
	}
	if err := oauthStore.SaveToken(oauthState.ProfileName, *token); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *jsonOutput {
		fmt.Println(string(marshalOAuthTokenSummary(oauthState.ProfileName, *token, *printAccessToken)))
		return 0
	}

	fmt.Printf("OAuth2 token obtained for profile '%s'\n", oauthState.ProfileName)
	fmt.Printf("Token type: %s\n", token.TokenType)
	if token.Scope != "" {
		fmt.Printf("Scope: %s\n", token.Scope)
	}
	if token.RefreshToken != "" {
		fmt.Println("Refresh token: present")
	}
	if token.ExpiresAt != nil {
		fmt.Printf("Expires: %s\n", token.ExpiresAt.Format(time.RFC3339))
	}
	if *printAccessToken {
		fmt.Printf("Access token: %s\n", token.AccessToken)
	}

	return 0
}

// runOAuthRefresh refreshes an OAuth2 access token.
func runOAuthRefresh(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth refresh", flag.ExitOnError)
	profileName := fs.String("profile", "", "Profile name (required)")
	printAccessToken := fs.Bool("print-access-token", false, "Print the access token")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	_ = fs.Parse(args)

	if *profileName == "" {
		fmt.Fprintln(os.Stderr, "--profile is required")
		return 1
	}

	oauthStore := auth.NewOAuthStore(cfg.DataDir)
	existingToken, found, err := oauthStore.LoadToken(*profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !found {
		fmt.Fprintln(os.Stderr, "no OAuth token found for profile")
		return 1
	}
	if existingToken.RefreshToken == "" {
		fmt.Fprintln(os.Stderr, "no refresh token available")
		return 1
	}

	profile, found, err := auth.GetProfile(cfg.DataDir, *profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "profile not found: %s\n", *profileName)
		return 1
	}
	if profile.OAuth2 == nil {
		fmt.Fprintln(os.Stderr, "profile does not have OAuth2 configuration")
		return 1
	}

	newToken, err := auth.RefreshOAuth2Token(ctx, *profile.OAuth2, existingToken.RefreshToken)
	if err != nil {
		fmt.Fprintln(os.Stderr, "token refresh failed:", err)
		return 1
	}
	if err := oauthStore.SaveToken(*profileName, *newToken); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *jsonOutput {
		fmt.Println(string(marshalOAuthTokenSummary(*profileName, *newToken, *printAccessToken)))
		return 0
	}

	fmt.Printf("OAuth2 token refreshed for profile '%s'\n", *profileName)
	fmt.Printf("Token type: %s\n", newToken.TokenType)
	if newToken.Scope != "" {
		fmt.Printf("Scope: %s\n", newToken.Scope)
	}
	if newToken.RefreshToken != "" {
		fmt.Println("Refresh token: present")
	}
	if newToken.ExpiresAt != nil {
		fmt.Printf("Expires: %s\n", newToken.ExpiresAt.Format(time.RFC3339))
	}
	if *printAccessToken {
		fmt.Printf("Access token: %s\n", newToken.AccessToken)
	}

	return 0
}

// runOAuthRevoke revokes an OAuth2 token.
func runOAuthRevoke(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth revoke", flag.ExitOnError)
	profileName := fs.String("profile", "", "Profile name (required)")
	localOnly := fs.Bool("local-only", false, "Skip remote revoke even if configured")
	_ = fs.Parse(args)

	if *profileName == "" {
		fmt.Fprintln(os.Stderr, "--profile is required")
		return 1
	}

	oauthStore := auth.NewOAuthStore(cfg.DataDir)
	existingToken, found, err := oauthStore.LoadToken(*profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !found {
		fmt.Fprintln(os.Stderr, "no OAuth token found for profile")
		return 1
	}

	profile, found, err := auth.GetProfile(cfg.DataDir, *profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "profile not found: %s\n", *profileName)
		return 1
	}
	if profile.OAuth2 == nil {
		fmt.Fprintln(os.Stderr, "profile does not have OAuth2 configuration")
		return 1
	}

	if profile.OAuth2.RevokeURL != "" && !*localOnly {
		if err := auth.RevokeOAuth2Token(ctx, profile.OAuth2.RevokeURL, profile.OAuth2.ClientID, profile.OAuth2.ClientSecret, existingToken.AccessToken, "access_token"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to revoke token at provider: %v\n", err)
		}
	} else if profile.OAuth2.RevokeURL == "" && !*localOnly {
		fmt.Println("Note: remote revoke is not configured (no revoke_url)")
	}

	if err := oauthStore.DeleteToken(*profileName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("OAuth2 token revoked for profile '%s'\n", *profileName)
	return 0
}

func marshalOAuthTokenSummary(profileName string, token auth.OAuth2Token, includeAccessToken bool) []byte {
	result := map[string]any{
		"profile":      profileName,
		"token_type":   token.TokenType,
		"scope":        token.Scope,
		"has_refresh":  token.RefreshToken != "",
		"expires_at":   nil,
		"access_token": nil,
	}
	if token.ExpiresAt != nil {
		result["expires_at"] = token.ExpiresAt.Format(time.RFC3339)
	}
	if includeAccessToken {
		result["access_token"] = token.AccessToken
	}
	payload, _ := json.MarshalIndent(result, "", "  ")
	return payload
}
