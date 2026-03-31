// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Route `spartan auth oauth` subcommands onto focused OAuth configuration, flow, discovery, and token handlers.
//
// Responsibilities:
// - Validate the top-level OAuth subcommand shape.
// - Dispatch to the correct focused handler file.
// - Return stable exit codes for the CLI entrypoint.
//
// Scope:
// - Top-level OAuth command routing only.
//
// Usage:
// - Called by `RunAuth` when operators run `spartan auth oauth ...`.
//
// Invariants/Assumptions:
// - Subcommand handlers own their own flag parsing and output.
// - Unknown or incomplete subcommands print the relevant help text and return a non-zero exit code.
package manage

import (
	"context"
	"fmt"
	"os"

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
