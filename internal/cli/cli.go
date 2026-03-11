// Package cli provides the Spartan Scraper command-line interface router.
//
// Responsibilities:
// - Route top-level commands to their respective handlers.
// - Handle signal interrupts (SIGINT, SIGTERM) for graceful shutdown.
// - Provide basic global flag parsing (e.g., version, help).
//
// Does NOT handle:
// - Implementation of specific command logic (subcommands).
// - Complex argument parsing (delegated to subcommands).
//
// Invariants/Assumptions:
// - Assumes os.Args is available and has at least one element (program name).
// - Expects a valid context and configuration for command routing.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fitchmultz/spartan-scraper/internal/cli/batch"
	"github.com/fitchmultz/spartan-scraper/internal/cli/manage"
	"github.com/fitchmultz/spartan-scraper/internal/cli/scrape"
	"github.com/fitchmultz/spartan-scraper/internal/cli/server"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// Run executes CLI application. It parses command-line arguments and
// routes to appropriate subcommand. It returns an exit code.
func Run(ctx context.Context) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		return 1
	}
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
	case "render-profiles":
		return manage.RunRenderProfiles(ctx, cfg, os.Args[2:])
	case "pipeline-js":
		return manage.RunPipelineJS(ctx, cfg, os.Args[2:])
	case "export":
		return manage.RunExport(ctx, cfg, os.Args[2:])
	case "schedule":
		return manage.RunSchedule(ctx, cfg, os.Args[2:])
	case "export-schedule":
		return manage.RunExportSchedule(ctx, cfg, os.Args[2:])
	case "templates":
		return manage.RunTemplates(ctx, cfg, os.Args[2:])
	case "crawl-states":
		return manage.RunCrawlStates(ctx, cfg, os.Args[2:])
	case "retention":
		return manage.RunRetention(ctx, cfg, os.Args[2:])
	case "backup":
		return manage.RunBackup(ctx, cfg, os.Args[2:])
	case "restore":
		return manage.RunRestore(ctx, cfg, os.Args[2:])
	case "jobs":
		return manage.RunJobs(ctx, cfg, os.Args[2:])
	case "chains":
		return manage.RunChains(ctx, cfg, os.Args[2:])
	case "batch":
		return batch.RunBatch(ctx, cfg, os.Args[2:])
	case "watch":
		return manage.RunWatch(ctx, cfg, os.Args[2:])

	case "server":
		return server.RunServer(ctx, cfg, os.Args[2:])
	case "mcp":
		return server.RunMCP(ctx, cfg, os.Args[2:])
	case "health":
		return server.RunHealth(ctx, cfg, os.Args[2:])
	case "tui":
		return server.RunTUI(ctx, cfg, os.Args[2:])

	case "version", "--version", "-v":
		if err := RunVersion(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0

	case "help", "--help", "-h":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printHelp()
		return 1
	}
}
