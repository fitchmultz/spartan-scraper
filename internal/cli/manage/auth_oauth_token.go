// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Inspect and delete persisted OAuth tokens from the CLI.
//
// Responsibilities:
// - List profiles with stored tokens.
// - Report token expiry and refresh availability.
// - Delete stored tokens on demand.
//
// Scope:
// - `spartan auth oauth token *` subcommands only.
//
// Usage:
// - Invoked by `RunAuthOAuth` for token inspection and cleanup.
//
// Invariants/Assumptions:
// - Token status never prints the access token value.
// - Missing tokens return a non-zero exit code for automation.
package manage

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

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
		payload, _ := json.MarshalIndent(map[string]any{
			"profile":     *profileName,
			"has_token":   true,
			"token_type":  token.TokenType,
			"has_refresh": hasRefresh,
			"is_expired":  isExpired,
			"expires_at":  formatOAuthTokenExpiry(token.ExpiresAt),
			"scope":       token.Scope,
		}, "", "  ")
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

func formatOAuthTokenExpiry(expiresAt *time.Time) any {
	if expiresAt == nil {
		return nil
	}
	return expiresAt.Format(time.RFC3339)
}
