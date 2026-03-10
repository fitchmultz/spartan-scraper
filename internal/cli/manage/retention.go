// Package manage contains retention CLI command wiring.
//
// This file provides commands for:
// - Viewing retention configuration and status
// - Running manual cleanup with various options
// - Previewing what would be deleted (dry-run)
//
// It does NOT handle:
// - Scheduled cleanup execution (scheduler handles this)
// - Policy evaluation logic (retention package handles this)
package manage

import (
	"context"
	"flag"
	"fmt"
	"os"

	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/retention"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

const retentionCommandHelpText = `Manage retention policy status and manual cleanup.

Usage: spartan retention <command> [options]

Commands:
  status              Show retention configuration and statistics
  cleanup             Run retention cleanup immediately

Cleanup Options:
  --dry-run           Preview what would be deleted without removing
  --force             Run cleanup even if retention is disabled
  --older-than=N      Override age threshold (days)
  --kind=KIND         Only cleanup specific job kind (scrape|crawl|research)

Environment Variables:
  RETENTION_ENABLED                 Enable automatic retention (default: false)
  RETENTION_JOB_DAYS                Max age for jobs in days (default: 30, 0 = unlimited)
  RETENTION_CRAWL_STATE_DAYS        Max age for crawl states in days (default: 90, 0 = unlimited)
  RETENTION_MAX_JOBS                Max total jobs to keep (default: 10000, 0 = unlimited)
  RETENTION_MAX_STORAGE_GB          Max storage in GB (default: 10, 0 = unlimited)
  RETENTION_CLEANUP_INTERVAL_HOURS  Hours between cleanup runs (default: 24)
  RETENTION_DRY_RUN_DEFAULT         Default dry-run mode (default: false)

Examples:
  spartan retention status
  spartan retention cleanup --dry-run
  spartan retention cleanup --older-than=7 --kind=scrape
  spartan retention cleanup --force
`

// RunRetention handles the retention subcommand.
func RunRetention(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printRetentionHelp()
		return 1
	}

	switch args[0] {
	case "status":
		return runRetentionStatus(ctx, cfg, args[1:])
	case "cleanup":
		return runRetentionCleanup(ctx, cfg, args[1:])
	case "help", "--help", "-h":
		printRetentionHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown retention subcommand: %s\n", args[0])
		printRetentionHelp()
		return 1
	}
}

func runRetentionStatus(ctx context.Context, cfg config.Config, args []string) int {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()

	engine := retention.NewEngine(st, cfg)
	status, err := engine.GetStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get retention status: %v\n", err)
		return 1
	}

	fmt.Println("Retention Configuration:")
	fmt.Printf("  Enabled:              %v\n", status.Enabled)
	fmt.Printf("  Job Retention Days:   %d (0 = unlimited)\n", status.JobRetentionDays)
	fmt.Printf("  Crawl State Days:     %d (0 = unlimited)\n", status.CrawlStateDays)
	fmt.Printf("  Max Jobs:             %d (0 = unlimited)\n", status.MaxJobs)
	fmt.Printf("  Max Storage:          %d GB (0 = unlimited)\n", status.MaxStorageGB)
	fmt.Println()
	fmt.Println("Current Statistics:")
	fmt.Printf("  Total Jobs:           %d\n", status.TotalJobs)
	fmt.Printf("  Jobs Eligible:        %d\n", status.JobsEligible)
	fmt.Printf("  Storage Used:         %d MB\n", status.StorageUsedMB)

	return 0
}

func runRetentionCleanup(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("retention cleanup", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", cfg.RetentionDryRunDefault, "Preview what would be deleted without removing")
	force := fs.Bool("force", false, "Run cleanup even if retention is disabled")
	olderThan := fs.Int("older-than", 0, "Override age threshold (days, 0 = use config)")
	kind := fs.String("kind", "", "Only cleanup specific job kind (scrape|crawl|research)")
	_ = fs.Parse(args)

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()

	engine := retention.NewEngine(st, cfg)

	opts := retention.CleanupOptions{
		DryRun: *dryRun,
		Force:  *force,
	}

	if *olderThan > 0 {
		cutoff := time.Now().AddDate(0, 0, -*olderThan)
		opts.OlderThan = &cutoff
	}

	if *kind != "" {
		k := model.Kind(*kind)
		if k != model.KindScrape && k != model.KindCrawl && k != model.KindResearch {
			fmt.Fprintf(os.Stderr, "invalid kind: %s (must be scrape, crawl, or research)\n", *kind)
			return 1
		}
		opts.Kind = &k
	}

	fmt.Println("Running retention cleanup...")
	if *dryRun {
		fmt.Println("(DRY RUN - no actual deletions)")
	}

	result, err := engine.RunCleanup(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cleanup failed: %v\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println(retention.FormatResult(result, *dryRun))

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Printf("Errors encountered (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}

	return 0
}

func printRetentionHelp() {
	fmt.Fprint(os.Stderr, retentionCommandHelpText)
}
