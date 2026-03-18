// Package manage contains retention CLI command wiring.
//
// Purpose:
// - Present retention status and cleanup flows with capability-aware operator guidance.
//
// Responsibilities:
// - Explain whether automatic retention is disabled, healthy, or needs attention.
// - Run manual cleanup previews or executions with explicit destructive confirmation flags.
// - Keep retention CLI guidance aligned with the shared API-backed Settings flows.
//
// Scope:
// - Retention CLI rendering and command dispatch only.
//
// Usage:
// - Called by `spartan retention ...` subcommands.
//
// Invariants/Assumptions:
// - Status output should recommend the next useful action instead of dumping raw config alone.
// - Cleanup defaults remain safety-first.
package manage

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
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

	response := api.RetentionStatusResponse{
		Enabled:          status.Enabled,
		JobRetentionDays: status.JobRetentionDays,
		CrawlStateDays:   status.CrawlStateDays,
		MaxJobs:          status.MaxJobs,
		MaxStorageGB:     status.MaxStorageGB,
		TotalJobs:        status.TotalJobs,
		JobsEligible:     status.JobsEligible,
		StorageUsedMB:    status.StorageUsedMB,
	}
	response.Guidance = api.BuildRetentionCapabilityGuidance(response)

	fmt.Println("Retention Status")
	fmt.Printf("Capability: %s\n", strings.ToUpper(response.Guidance.Status))
	if title := strings.TrimSpace(response.Guidance.Title); title != "" {
		fmt.Printf("%s\n", title)
	}
	if message := strings.TrimSpace(response.Guidance.Message); message != "" {
		fmt.Printf("%s\n", message)
	}
	renderRecommendedActions(response.Guidance.Actions, "spartan")
	fmt.Println()
	fmt.Println("Configuration")
	fmt.Printf("  Enabled:              %v\n", response.Enabled)
	fmt.Printf("  Job Retention Days:   %d (0 = unlimited)\n", response.JobRetentionDays)
	fmt.Printf("  Crawl State Days:     %d (0 = unlimited)\n", response.CrawlStateDays)
	fmt.Printf("  Max Jobs:             %d (0 = unlimited)\n", response.MaxJobs)
	fmt.Printf("  Max Storage:          %d GB (0 = unlimited)\n", response.MaxStorageGB)
	fmt.Println()
	fmt.Println("Current Statistics")
	fmt.Printf("  Total Jobs:           %d\n", response.TotalJobs)
	fmt.Printf("  Jobs Eligible:        %d\n", response.JobsEligible)
	fmt.Printf("  Storage Used:         %d MB\n", response.StorageUsedMB)

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
		for _, cleanupErr := range result.Errors {
			fmt.Printf("  - %v\n", cleanupErr)
		}
	}

	return 0
}

func printRetentionHelp() {
	fmt.Fprint(os.Stderr, retentionCommandHelpText)
}
