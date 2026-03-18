// Package server contains MCP CLI command wiring.
//
// It does NOT implement MCP protocol; internal/mcp does.
package server

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/mcp"
)

func RunMCP(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan mcp

Examples:
  spartan mcp

Notes:
  MCP uses stdio. One JSON-RPC request per line.
  In setup mode, MCP still starts but only exposes health_status and diagnostic_check.
`)
	}
	_ = fs.Parse(args)

	if fs.NArg() > 0 && (fs.Arg(0) == "--help" || fs.Arg(0) == "-h" || fs.Arg(0) == "help") {
		fs.Usage()
		return 0
	}

	setupStatus, err := inspectStartupPreflight(cfg, currentCommandName())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	var srv *mcp.Server
	if setupStatus != nil {
		srv = mcp.NewSetupServer(cfg, *setupStatus)
	} else {
		srv, err = mcp.NewServer(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	defer srv.Close()

	if err := srv.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
