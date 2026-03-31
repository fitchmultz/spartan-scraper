// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Manage persisted OAuth profile configuration from the CLI.
//
// Responsibilities:
// - Show OAuth settings for a profile with secret redaction by default.
// - Set or clear OAuth profile configuration.
// - Parse repeatable scope flags for OAuth config authoring.
//
// Scope:
// - `spartan auth oauth config *` subcommands only.
//
// Usage:
// - Invoked by `RunAuthOAuth` for `config show|set|clear`.
//
// Invariants/Assumptions:
// - Client secrets stay hidden unless explicitly requested.
// - Flow-specific validation happens before writing profile changes.
package manage

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

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

	oauthConfig := *profile.OAuth2
	if !*showSecret && oauthConfig.ClientSecret != "" {
		oauthConfig.ClientSecret = "***"
	}

	if *jsonOutput {
		payload, _ := json.MarshalIndent(oauthConfig, "", "  ")
		fmt.Println(string(payload))
		return 0
	}

	fmt.Printf("OAuth2 Configuration for profile '%s':\n", *profileName)
	fmt.Printf("  Flow Type:    %s\n", oauthConfig.FlowType)
	fmt.Printf("  Client ID:    %s\n", oauthConfig.ClientID)
	if *showSecret {
		fmt.Printf("  Client Secret: %s\n", oauthConfig.ClientSecret)
	} else if oauthConfig.ClientSecret != "" {
		fmt.Printf("  Client Secret: *** (use --show-secret to reveal)\n")
	}
	fmt.Printf("  Token URL:    %s\n", oauthConfig.TokenURL)
	if oauthConfig.AuthorizeURL != "" {
		fmt.Printf("  Authorize URL: %s\n", oauthConfig.AuthorizeURL)
	}
	if oauthConfig.RevokeURL != "" {
		fmt.Printf("  Revoke URL:   %s\n", oauthConfig.RevokeURL)
	}
	if len(oauthConfig.Scopes) > 0 {
		fmt.Printf("  Scopes:       %s\n", strings.Join(oauthConfig.Scopes, " "))
	}
	fmt.Printf("  Use PKCE:     %v\n", oauthConfig.UsePKCE)
	if oauthConfig.RedirectURI != "" {
		fmt.Printf("  Redirect URI: %s\n", oauthConfig.RedirectURI)
	}
	if oauthConfig.DiscoveryURL != "" {
		fmt.Printf("  Discovery URL: %s\n", oauthConfig.DiscoveryURL)
	}
	if oauthConfig.Issuer != "" {
		fmt.Printf("  Issuer:       %s\n", oauthConfig.Issuer)
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

	if oauthFlowType == auth.OAuth2FlowAuthorizationCode && *authorizeURL == "" {
		fmt.Fprintln(os.Stderr, "--authorize-url is required for authorization_code flow")
		return 1
	}

	profile, found, err := auth.GetProfile(cfg.DataDir, *profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !found {
		profile = auth.Profile{Name: *profileName}
	}

	pkceEnabled := *usePKCE
	if *noPKCE {
		pkceEnabled = false
	}

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

// stringSliceFlag is a custom flag type that accumulates multiple values.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
