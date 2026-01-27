// Package cli provides Spartan Scraper command-line interface router.
//
// It does NOT implement command handlers; those live in internal/cli/* domain packages.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fitchmultz/spartan-scraper/internal/cli/manage"
	"github.com/fitchmultz/spartan-scraper/internal/cli/scrape"
	"github.com/fitchmultz/spartan-scraper/internal/cli/server"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// Run executes CLI application. It parses command-line arguments and
// routes to appropriate subcommand. It returns an exit code.
func Run(ctx context.Context) int {
	cfg := config.Load()
	config.InitLogger(cfg)
	if len(os.Args) < 2 {
		printHelp()
		return 1
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch os.Args[1] {
	case "scrape":
		return scrape.RunScrape(ctx, cfg, os.Args[2:])
	case "crawl":
		return scrape.RunCrawl(ctx, cfg, os.Args[2:])
	case "research":
		return scrape.RunResearch(ctx, cfg, os.Args[2:])

	case "auth":
		return manage.RunAuth(ctx, cfg, os.Args[2:])
	case "export":
		return manage.RunExport(ctx, cfg, os.Args[2:])
	case "schedule":
		return manage.RunSchedule(ctx, cfg, os.Args[2:])
	case "templates":
		return manage.RunTemplates(ctx, cfg, os.Args[2:])
	case "crawl-states":
		return manage.RunCrawlStates(ctx, cfg, os.Args[2:])
	case "jobs":
		return manage.RunJobs(ctx, cfg, os.Args[2:])

	case "server":
		return server.RunServer(ctx, cfg, os.Args[2:])
	case "mcp":
		return server.RunMCP(ctx, cfg, os.Args[2:])
	case "health":
		return server.RunHealth(ctx, cfg, os.Args[2:])
	case "tui":
		return server.RunTUI(ctx, cfg, os.Args[2:])

	case "help", "--help", "-h":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printHelp()
		return 1
	}
}
