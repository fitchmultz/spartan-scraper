// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Run OIDC discovery for OAuth profiles from the CLI.
//
// Responsibilities:
// - Resolve discovery metadata from either a discovery URL or issuer.
// - Optionally apply discovered endpoints back onto an existing profile.
// - Print human-readable or JSON discovery output.
//
// Scope:
// - `spartan auth oauth discover` only.
//
// Usage:
// - Invoked by `RunAuthOAuth` when operators request OIDC discovery.
//
// Invariants/Assumptions:
// - Exactly one discovery source must be supplied.
// - Applying discovery requires an existing profile with OAuth config already initialized.
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// runOAuthDiscover performs OIDC discovery.
func runOAuthDiscover(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth oauth discover", flag.ExitOnError)
	discoveryURL := fs.String("discovery-url", "", "OIDC discovery URL")
	issuer := fs.String("issuer", "", "OIDC issuer URL")
	apply := fs.Bool("apply", false, "Apply discovered endpoints to profile")
	profileName := fs.String("profile", "", "Profile name (required with --apply)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	_ = fs.Parse(args)

	if (*discoveryURL == "" && *issuer == "") || (*discoveryURL != "" && *issuer != "") {
		fmt.Fprintln(os.Stderr, "exactly one of --discovery-url or --issuer is required")
		return 1
	}
	if *apply && *profileName == "" {
		fmt.Fprintln(os.Stderr, "--profile is required when using --apply")
		return 1
	}

	var (
		metadata *auth.OIDCProviderMetadata
		err      error
	)
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
