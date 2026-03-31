// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Keep OAuth-specific CLI help text separate from command execution logic.
//
// Responsibilities:
// - Print top-level, config, and token help menus for `spartan auth oauth`.
// - Keep examples aligned with the supported command contract.
//
// Scope:
// - Help text only; no flag parsing or runtime behavior.
//
// Usage:
// - Called by OAuth command dispatchers when arguments are missing or help is requested.
//
// Invariants/Assumptions:
// - Help output is written to stderr to match the rest of the CLI command family.
// - Examples describe currently supported subcommands and flags.
package manage

import (
	"fmt"
	"os"
)

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
