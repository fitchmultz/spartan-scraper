// Package manage contains CLI commands for configuration/data management.
//
// This file implements OAuth 2.0 CLI subcommands for auth profile management.
// It provides commands for configuring OAuth2 on profiles and executing OAuth flows.
//
// Responsibilities:
// - OAuth2 profile configuration (set, clear, show)
// - OAuth2 flow execution (initiate, callback, refresh, revoke)
// - OIDC discovery
// - Token management (list, status, delete)
//
// Does NOT handle:
// - Interactive browser-based callback listener (no local HTTP server)
// - Token storage format changes (uses existing oauth_tokens.json)
//
// Invariants/Assumptions:
// - Never prints secrets by default (client secrets, tokens)
// - Requires explicit --show-secret or --print-access-token flags to reveal sensitive data
// - Uses the existing auth vault and OAuth store from internal/auth
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// RunAuthOAuth handles the 'spartan auth oauth' subcommand group.
func RunAuthOAuth(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printAuthOAuthHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printAuthOAuthHelp()
		return 0
	}

	switch args[0] {
	case "config":
		if len(args) < 2 {
			printAuthOAuthConfigHelp()
			return 1
		}
		switch args[1] {
		case "show":
			return runOAuthConfigShow(cfg, args[2:])
		case "set":
			return runOAuthConfigSet(cfg, args[2:])
		case "clear":
			return runOAuthConfigClear(cfg, args[2:])
		default:
			printAuthOAuthConfigHelp()
			return 1
		}

	case "discover":
		return runOAuthDiscover(ctx, cfg, args[1:])

	case "initiate":
		return runOAuthInitiate(cfg, args[1:])

	case "callback":
		return runOAuthCallback(cfg, args[1:])

	case "refresh":
		return runOAuthRefresh(ctx, cfg, args[1:])

	case "revoke":
		return runOAuthRevoke(ctx, cfg, args[1:])

	case "token":
		if len(args) < 2 {
			printAuthOAuthTokenHelp()
			return 1
		}
		switch args[1] {
		case "list":
			return runOAuthTokenList(cfg)
		case "status":
			return runOAuthTokenStatus(cfg, args[2:])
		case "delete":
			return runOAuthTokenDelete(cfg, args[2:])
		default:
			printAuthOAuthTokenHelp()
			return 1
		}

	default:
		fmt.Fprintln(os.Stderr, "unknown oauth subcommand:", args[0])
		return 1
	}
}

func printAuthOAuthHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan auth oauth <subcommand> [options]

Subcommands:
  config        Manage OAuth2 configuration for profiles
  discover      Perform OIDC discovery
  initiate      Start OAuth2 authorization flow
  callback      Exchange authorization code for token
  refresh       Refresh access token
  revoke        Revoke OAuth2 token
  token         Manage stored OAuth2 tokens

Examples:
  spartan auth oauth config show --profile myapp
  spartan auth oauth config set --profile myapp --client-id abc --token-url https://oauth/token
  spartan auth oauth discover --issuer https://accounts.google.com
  spartan auth oauth initiate --profile myapp
  spartan auth oauth callback --state xyz --code abc
  spartan auth oauth refresh --profile myapp
  spartan auth oauth revoke --profile myapp
  spartan auth oauth token list

Use "spartan auth oauth <subcommand> --help" for more details.
`)
}

func printAuthOAuthConfigHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan auth oauth config <subcommand> [options]

Subcommands:
  show    Display OAuth2 configuration for a profile
  set     Set OAuth2 configuration for a profile
  clear   Remove OAuth2 configuration from a profile

Examples:
  spartan auth oauth config show --profile myapp
  spartan auth oauth config set --profile myapp --client-id abc --token-url https://oauth/token
  spartan auth oauth config clear --profile myapp
`)
}

func printAuthOAuthTokenHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan auth oauth token <subcommand> [options]

Subcommands:
  list     List profiles with stored OAuth tokens
  status   Show token status for a profile
  delete   Delete stored token for a profile

Examples:
  spartan auth oauth token list
  spartan auth oauth token status --profile myapp
  spartan auth oauth token delete --profile myapp
`)
}

// runOAuthConfigShow displays OAuth2 configuration for a profile.
func runOAuthConfigShow(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth config show", flag.ExitOnError)
	profileName := fs.String("profile", "", "Profile name (required)")
	showSecret := fs.Bool("show-secret", false, "Show client secret")
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
		fmt.Println("Profile has no OAuth2 configuration")
		return 0
	}

	config := *profile.OAuth2
	if !*showSecret && config.ClientSecret != "" {
		config.ClientSecret = "***"
	}

	if *jsonOutput {
		payload, _ := json.MarshalIndent(config, "", "  ")
		fmt.Println(string(payload))
		return 0
	}

	fmt.Printf("OAuth2 Configuration for profile '%s':\n", *profileName)
	fmt.Printf("  Flow Type:    %s\n", config.FlowType)
	fmt.Printf("  Client ID:    %s\n", config.ClientID)
	if *showSecret {
		fmt.Printf("  Client Secret: %s\n", config.ClientSecret)
	} else if config.ClientSecret != "" {
		fmt.Printf("  Client Secret: *** (use --show-secret to reveal)\n")
	}
	fmt.Printf("  Token URL:    %s\n", config.TokenURL)
	if config.AuthorizeURL != "" {
		fmt.Printf("  Authorize URL: %s\n", config.AuthorizeURL)
	}
	if config.RevokeURL != "" {
		fmt.Printf("  Revoke URL:   %s\n", config.RevokeURL)
	}
	if len(config.Scopes) > 0 {
		fmt.Printf("  Scopes:       %s\n", strings.Join(config.Scopes, " "))
	}
	fmt.Printf("  Use PKCE:     %v\n", config.UsePKCE)
	if config.RedirectURI != "" {
		fmt.Printf("  Redirect URI: %s\n", config.RedirectURI)
	}
	if config.DiscoveryURL != "" {
		fmt.Printf("  Discovery URL: %s\n", config.DiscoveryURL)
	}
	if config.Issuer != "" {
		fmt.Printf("  Issuer:       %s\n", config.Issuer)
	}

	return 0
}

// runOAuthConfigSet sets OAuth2 configuration for a profile.
func runOAuthConfigSet(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth config set", flag.ExitOnError)
	profileName := fs.String("profile", "", "Profile name (required)")
	flowType := fs.String("flow-type", "authorization_code", "Flow type: authorization_code|client_credentials|device_code")
	clientID := fs.String("client-id", "", "Client ID (required)")
	clientSecret := fs.String("client-secret", "", "Client secret (for confidential clients)")
	authorizeURL := fs.String("authorize-url", "", "Authorization URL (required for authorization_code)")
	tokenURL := fs.String("token-url", "", "Token URL (required)")
	revokeURL := fs.String("revoke-url", "", "Revoke URL (optional)")
	redirectURI := fs.String("redirect-uri", "", "Redirect URI (optional)")
	discoveryURL := fs.String("discovery-url", "", "OIDC discovery URL (optional)")
	issuer := fs.String("issuer", "", "OIDC issuer (optional)")
	usePKCE := fs.Bool("use-pkce", false, "Use PKCE (for authorization_code flow)")
	noPKCE := fs.Bool("no-pkce", false, "Disable PKCE")

	var scopes stringSliceFlag
	fs.Var(&scopes, "scope", "OAuth scope (repeatable)")

	_ = fs.Parse(args)

	if *profileName == "" {
		fmt.Fprintln(os.Stderr, "--profile is required")
		return 1
	}

	if *clientID == "" {
		fmt.Fprintln(os.Stderr, "--client-id is required")
		return 1
	}

	if *tokenURL == "" {
		fmt.Fprintln(os.Stderr, "--token-url is required")
		return 1
	}

	// Validate flow type
	var oauthFlowType auth.OAuth2FlowType
	switch *flowType {
	case "authorization_code":
		oauthFlowType = auth.OAuth2FlowAuthorizationCode
	case "client_credentials":
		oauthFlowType = auth.OAuth2FlowClientCredentials
	case "device_code":
		oauthFlowType = auth.OAuth2FlowDeviceCode
	default:
		fmt.Fprintf(os.Stderr, "invalid --flow-type: %s (must be authorization_code, client_credentials, or device_code)\n", *flowType)
		return 1
	}

	// Validate flow-specific requirements
	if oauthFlowType == auth.OAuth2FlowAuthorizationCode && *authorizeURL == "" {
		fmt.Fprintln(os.Stderr, "--authorize-url is required for authorization_code flow")
		return 1
	}

	// Load existing profile or create new one
	profile, found, err := auth.GetProfile(cfg.DataDir, *profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if !found {
		profile = auth.Profile{Name: *profileName}
	}

	// Determine PKCE setting
	pkceEnabled := *usePKCE
	if *noPKCE {
		pkceEnabled = false
	}

	// Set OAuth2 config
	profile.OAuth2 = &auth.OAuth2Config{
		FlowType:     oauthFlowType,
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		AuthorizeURL: *authorizeURL,
		TokenURL:     *tokenURL,
		RevokeURL:    *revokeURL,
		Scopes:       []string(scopes),
		UsePKCE:      pkceEnabled,
		RedirectURI:  *redirectURI,
		DiscoveryURL: *discoveryURL,
		Issuer:       *issuer,
	}

	if err := auth.UpsertProfile(cfg.DataDir, profile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("OAuth2 configuration saved for profile '%s'\n", *profileName)
	return 0
}

// runOAuthConfigClear removes OAuth2 configuration from a profile.
func runOAuthConfigClear(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth config clear", flag.ExitOnError)
	profileName := fs.String("profile", "", "Profile name (required)")
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

	profile.OAuth2 = nil
	if err := auth.UpsertProfile(cfg.DataDir, profile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("OAuth2 configuration cleared for profile '%s'\n", *profileName)
	return 0
}

// runOAuthDiscover performs OIDC discovery.
func runOAuthDiscover(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth discover", flag.ExitOnError)
	discoveryURL := fs.String("discovery-url", "", "OIDC discovery URL")
	issuer := fs.String("issuer", "", "OIDC issuer URL")
	apply := fs.Bool("apply", false, "Apply discovered endpoints to profile")
	profileName := fs.String("profile", "", "Profile name (required with --apply)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	_ = fs.Parse(args)

	// Validate that exactly one of discovery-url or issuer is provided
	if (*discoveryURL == "" && *issuer == "") || (*discoveryURL != "" && *issuer != "") {
		fmt.Fprintln(os.Stderr, "exactly one of --discovery-url or --issuer is required")
		return 1
	}

	if *apply && *profileName == "" {
		fmt.Fprintln(os.Stderr, "--profile is required when using --apply")
		return 1
	}

	var metadata *auth.OIDCProviderMetadata
	var err error

	if *discoveryURL != "" {
		metadata, err = auth.OIDCDiscover(ctx, *discoveryURL)
	} else {
		metadata, err = auth.OIDCDiscoverFromIssuer(ctx, *issuer)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "discovery failed:", err)
		return 1
	}

	if *apply {
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
			fmt.Fprintf(os.Stderr, "profile '%s' has no OAuth2 configuration; run 'config set' first\n", *profileName)
			return 1
		}

		// Apply discovered endpoints
		profile.OAuth2.AuthorizeURL = metadata.AuthorizationEndpoint
		profile.OAuth2.TokenURL = metadata.TokenEndpoint
		if metadata.RevocationEndpoint != "" {
			profile.OAuth2.RevokeURL = metadata.RevocationEndpoint
		}
		profile.OAuth2.Issuer = metadata.Issuer

		if err := auth.UpsertProfile(cfg.DataDir, profile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		fmt.Printf("Applied discovered endpoints to profile '%s'\n", *profileName)
	}

	if *jsonOutput {
		payload, _ := json.MarshalIndent(metadata, "", "  ")
		fmt.Println(string(payload))
		return 0
	}

	fmt.Println("OIDC Discovery Results:")
	fmt.Printf("  Issuer:                 %s\n", metadata.Issuer)
	fmt.Printf("  Authorization Endpoint: %s\n", metadata.AuthorizationEndpoint)
	fmt.Printf("  Token Endpoint:         %s\n", metadata.TokenEndpoint)
	if metadata.UserinfoEndpoint != "" {
		fmt.Printf("  Userinfo Endpoint:      %s\n", metadata.UserinfoEndpoint)
	}
	if metadata.RevocationEndpoint != "" {
		fmt.Printf("  Revocation Endpoint:    %s\n", metadata.RevocationEndpoint)
	}
	if len(metadata.ScopesSupported) > 0 {
		fmt.Printf("  Scopes Supported:       %s\n", strings.Join(metadata.ScopesSupported, ", "))
	}

	return 0
}

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

	oauthConfig := profile.OAuth2

	if oauthConfig.FlowType != auth.OAuth2FlowAuthorizationCode {
		fmt.Fprintln(os.Stderr, "only authorization_code flow is supported for initiate")
		return 1
	}

	if oauthConfig.AuthorizeURL == "" {
		fmt.Fprintln(os.Stderr, "profile has no authorize_url configured")
		return 1
	}

	// Determine redirect URI
	finalRedirectURI := *redirectURI
	if finalRedirectURI == "" {
		finalRedirectURI = oauthConfig.RedirectURI
	}

	// Create OAuth state store
	oauthStore := auth.NewOAuthStore(cfg.DataDir)

	// Generate state and PKCE (if enabled)
	stateStr, codeVerifier, err := oauthStore.CreateOAuthState(*profileName, finalRedirectURI, oauthConfig.UsePKCE)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Build PKCE challenge if using PKCE
	var codeChallenge, codeChallengeMethod string
	if oauthConfig.UsePKCE {
		challenge, method, err := auth.DerivePKCEChallengeS256(codeVerifier)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		codeChallenge = challenge
		codeChallengeMethod = method
	}

	// Build authorization URL using the override redirect URI
	// Create a copy of the config with the override redirect URI
	authConfigCopy := *oauthConfig
	authConfigCopy.RedirectURI = finalRedirectURI
	authURL, err := auth.BuildAuthorizationURL(authConfigCopy, stateStr, codeChallenge, codeChallengeMethod)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *jsonOutput {
		result := map[string]string{
			"authorization_url": authURL,
			"state":             stateStr,
		}
		payload, _ := json.MarshalIndent(result, "", "  ")
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

	// Parse callback URL if provided
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

	// Load and validate state
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

	// Delete state after use (one-time use)
	defer oauthStore.DeleteState(*stateStr)

	// Load profile to get OAuth config
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

	// Exchange code for token
	ctx := context.Background()
	token, err := auth.ExchangeAuthorizationCode(ctx, *profile.OAuth2, *code, oauthState.CodeVerifier, oauthState.RedirectURI)
	if err != nil {
		fmt.Fprintln(os.Stderr, "token exchange failed:", err)
		return 1
	}

	// Save token
	if err := oauthStore.SaveToken(oauthState.ProfileName, *token); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *jsonOutput {
		result := map[string]any{
			"profile":      oauthState.ProfileName,
			"token_type":   token.TokenType,
			"scope":        token.Scope,
			"has_refresh":  token.RefreshToken != "",
			"expires_at":   nil,
			"access_token": nil,
		}
		if token.ExpiresAt != nil {
			result["expires_at"] = token.ExpiresAt.Format(time.RFC3339)
		}
		if *printAccessToken {
			result["access_token"] = token.AccessToken
		}
		payload, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(payload))
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

	// Load existing token
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

	// Load profile to get OAuth config
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

	// Refresh token
	newToken, err := auth.RefreshOAuth2Token(ctx, *profile.OAuth2, existingToken.RefreshToken)
	if err != nil {
		fmt.Fprintln(os.Stderr, "token refresh failed:", err)
		return 1
	}

	// Save new token
	if err := oauthStore.SaveToken(*profileName, *newToken); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *jsonOutput {
		result := map[string]any{
			"profile":      *profileName,
			"token_type":   newToken.TokenType,
			"scope":        newToken.Scope,
			"has_refresh":  newToken.RefreshToken != "",
			"expires_at":   nil,
			"access_token": nil,
		}
		if newToken.ExpiresAt != nil {
			result["expires_at"] = newToken.ExpiresAt.Format(time.RFC3339)
		}
		if *printAccessToken {
			result["access_token"] = newToken.AccessToken
		}
		payload, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(payload))
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

	// Load existing token
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

	// Load profile to get OAuth config
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

	// Revoke token if revoke URL is configured and not local-only
	if profile.OAuth2.RevokeURL != "" && !*localOnly {
		if err := auth.RevokeOAuth2Token(ctx, profile.OAuth2.RevokeURL, profile.OAuth2.ClientID, profile.OAuth2.ClientSecret, existingToken.AccessToken, "access_token"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to revoke token at provider: %v\n", err)
		}
	} else if profile.OAuth2.RevokeURL == "" && !*localOnly {
		fmt.Println("Note: remote revoke is not configured (no revoke_url)")
	}

	// Delete token locally
	if err := oauthStore.DeleteToken(*profileName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("OAuth2 token revoked for profile '%s'\n", *profileName)
	return 0
}

// runOAuthTokenList lists profiles with stored OAuth tokens.
func runOAuthTokenList(cfg config.Config) int {
	oauthStore := auth.NewOAuthStore(cfg.DataDir)
	names, err := oauthStore.ListTokens()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if len(names) == 0 {
		fmt.Println("No OAuth tokens found.")
		return 0
	}

	fmt.Println("Profiles with stored OAuth tokens:")
	for _, name := range names {
		fmt.Printf("  %s\n", name)
	}

	return 0
}

// runOAuthTokenStatus shows token status for a profile.
func runOAuthTokenStatus(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth token status", flag.ExitOnError)
	profileName := fs.String("profile", "", "Profile name (required)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	_ = fs.Parse(args)

	if *profileName == "" {
		fmt.Fprintln(os.Stderr, "--profile is required")
		return 1
	}

	oauthStore := auth.NewOAuthStore(cfg.DataDir)
	token, found, err := oauthStore.LoadToken(*profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "no OAuth token found for profile '%s'\n", *profileName)
		return 1
	}

	isExpired := auth.IsTokenExpired(token, 0)
	hasRefresh := token.RefreshToken != ""

	if *jsonOutput {
		result := map[string]any{
			"profile":     *profileName,
			"has_token":   true,
			"token_type":  token.TokenType,
			"has_refresh": hasRefresh,
			"is_expired":  isExpired,
			"expires_at":  nil,
			"scope":       token.Scope,
		}
		if token.ExpiresAt != nil {
			result["expires_at"] = token.ExpiresAt.Format(time.RFC3339)
		}
		payload, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(payload))
		return 0
	}

	fmt.Printf("OAuth2 Token Status for profile '%s':\n", *profileName)
	fmt.Printf("  Token type:   %s\n", token.TokenType)
	fmt.Printf("  Has refresh:  %v\n", hasRefresh)
	if token.Scope != "" {
		fmt.Printf("  Scope:        %s\n", token.Scope)
	}
	if token.ExpiresAt != nil {
		status := "valid"
		if isExpired {
			status = "expired"
		}
		fmt.Printf("  Expires:      %s (%s)\n", token.ExpiresAt.Format(time.RFC3339), status)
	} else {
		fmt.Println("  Expires:      never")
	}

	return 0
}

// runOAuthTokenDelete deletes a stored OAuth token.
func runOAuthTokenDelete(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth token delete", flag.ExitOnError)
	profileName := fs.String("profile", "", "Profile name (required)")
	_ = fs.Parse(args)

	if *profileName == "" {
		fmt.Fprintln(os.Stderr, "--profile is required")
		return 1
	}

	oauthStore := auth.NewOAuthStore(cfg.DataDir)
	if err := oauthStore.DeleteToken(*profileName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("OAuth2 token deleted for profile '%s'\n", *profileName)
	return 0
}

// stringSliceFlag is a custom flag type that accumulates multiple values.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
