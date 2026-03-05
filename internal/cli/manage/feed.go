// Package manage provides CLI subcommands for managing spartan scraper resources.
//
// This file is responsible for:
// - feed add: Create a new RSS/Atom feed monitor
// - feed list: List all feeds
// - feed get: Get a feed by ID
// - feed update: Update a feed
// - feed delete: Delete a feed by ID
// - feed check: Manually check a feed
// - feed items: List seen items for a feed
// - feed start: Start the feed scheduler
//
// This file does NOT handle:
// - Feed scheduling (feed/scheduler.go handles this)
// - Feed execution (feed/feed.go handles this)
// - Feed parsing (feed/feed.go handles this)
//
// Invariants:
// - All commands validate inputs before execution
// - Errors are printed to stderr with helpful messages
// - Success output is formatted for readability
package manage

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/feed"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// RunFeed routes feed subcommands.
func RunFeed(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: feed subcommand required (add, list, get, update, delete, check, items, start)")
		printFeedHelp()
		return 1
	}

	switch args[0] {
	case "add":
		return runFeedAdd(cfg, args[1:])
	case "list":
		return runFeedList(cfg, args[1:])
	case "get":
		return runFeedGet(cfg, args[1:])
	case "update":
		return runFeedUpdate(cfg, args[1:])
	case "delete":
		return runFeedDelete(cfg, args[1:])
	case "check":
		return runFeedCheck(ctx, cfg, args[1:])
	case "items":
		return runFeedItems(cfg, args[1:])
	case "start":
		return runFeedStart(ctx, cfg, args[1:])
	case "help", "--help", "-h":
		printFeedHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown feed subcommand: %s\n", args[0])
		printFeedHelp()
		return 1
	}
}

func runFeedAdd(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("feed add", flag.ExitOnError)
	url := fs.String("url", "", "Feed URL to monitor (required)")
	feedType := fs.String("type", "auto", "Feed type: rss, atom, auto")
	interval := fs.Int("interval", 3600, "Check interval in seconds (min: 60)")
	autoScrape := fs.Bool("auto-scrape", true, "Create scrape jobs for new items")
	enabled := fs.Bool("enabled", true, "Enable the feed")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *url == "" {
		fmt.Fprintln(os.Stderr, "Error: --url is required")
		return 1
	}

	storage := feed.NewFileStorage(cfg.DataDir)

	f := &feed.Feed{
		URL:             *url,
		FeedType:        feed.FeedType(*feedType),
		IntervalSeconds: *interval,
		Enabled:         *enabled,
		AutoScrape:      *autoScrape,
		CreatedAt:       time.Now(),
	}

	if err := f.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	result, err := storage.Add(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to add feed: %v\n", err)
		return 1
	}

	fmt.Printf("Feed created:\n")
	fmt.Printf("  ID:         %s\n", result.ID)
	fmt.Printf("  URL:        %s\n", result.URL)
	fmt.Printf("  Type:       %s\n", result.FeedType)
	fmt.Printf("  Interval:   %d seconds\n", result.IntervalSeconds)
	fmt.Printf("  AutoScrape: %v\n", result.AutoScrape)
	fmt.Printf("  Status:     %s\n", result.GetStatus())

	return 0
}

func runFeedList(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("feed list", flag.ExitOnError)
	showAll := fs.Bool("all", false, "Show all feeds including disabled")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	storage := feed.NewFileStorage(cfg.DataDir)
	feeds, err := storage.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list feeds: %v\n", err)
		return 1
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds configured.")
		return 0
	}

	// Header
	fmt.Printf("%-36s %-40s %-10s %-12s %-12s %-20s\n", "ID", "URL", "TYPE", "STATUS", "INTERVAL", "LAST CHECKED")
	fmt.Println(strings.Repeat("-", 130))

	for _, f := range feeds {
		if !*showAll && !f.Enabled {
			continue
		}

		status := f.GetStatus()
		url := f.URL
		if len(url) > 38 {
			url = url[:35] + "..."
		}

		lastChecked := "never"
		if !f.LastCheckedAt.IsZero() {
			lastChecked = time.Since(f.LastCheckedAt).Round(time.Second).String() + " ago"
		}

		fmt.Printf("%-36s %-40s %-10s %-12s %-12d %-20s\n", f.ID, url, f.FeedType, status, f.IntervalSeconds, lastChecked)
	}

	return 0
}

func runFeedGet(cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: feed ID required")
		fmt.Fprintln(os.Stderr, "Usage: spartan feed get <id>")
		return 1
	}

	id := args[0]
	storage := feed.NewFileStorage(cfg.DataDir)

	f, err := storage.Get(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get feed: %v\n", err)
		return 1
	}

	fmt.Printf("Feed details:\n")
	fmt.Printf("  ID:                  %s\n", f.ID)
	fmt.Printf("  URL:                 %s\n", f.URL)
	fmt.Printf("  Type:                %s\n", f.FeedType)
	fmt.Printf("  Interval:            %d seconds\n", f.IntervalSeconds)
	fmt.Printf("  Enabled:             %v\n", f.Enabled)
	fmt.Printf("  AutoScrape:          %v\n", f.AutoScrape)
	fmt.Printf("  Status:              %s\n", f.GetStatus())
	fmt.Printf("  Created:             %s\n", f.CreatedAt.Format(time.RFC3339))
	if !f.LastCheckedAt.IsZero() {
		fmt.Printf("  Last Checked:        %s\n", f.LastCheckedAt.Format(time.RFC3339))
	}
	if f.LastError != "" {
		fmt.Printf("  Last Error:          %s\n", f.LastError)
	}
	if f.ConsecutiveFailures > 0 {
		fmt.Printf("  Consecutive Failures: %d\n", f.ConsecutiveFailures)
	}

	return 0
}

func runFeedUpdate(cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: feed ID required")
		fmt.Fprintln(os.Stderr, "Usage: spartan feed update <id> [options]")
		return 1
	}

	id := args[0]
	fs := flag.NewFlagSet("feed update", flag.ExitOnError)
	url := fs.String("url", "", "Feed URL")
	feedType := fs.String("type", "", "Feed type: rss, atom, auto")
	interval := fs.Int("interval", 0, "Check interval in seconds (min: 60)")
	enabled := fs.Bool("enabled", true, "Enable the feed")
	autoScrape := fs.Bool("auto-scrape", true, "Create scrape jobs for new items")

	if err := fs.Parse(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	storage := feed.NewFileStorage(cfg.DataDir)

	f, err := storage.Get(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get feed: %v\n", err)
		return 1
	}

	// Update fields if provided
	if *url != "" {
		f.URL = *url
	}
	if *feedType != "" {
		f.FeedType = feed.FeedType(*feedType)
	}
	if *interval > 0 {
		f.IntervalSeconds = *interval
	}
	f.Enabled = *enabled
	f.AutoScrape = *autoScrape

	if err := f.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if err := storage.Update(f); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to update feed: %v\n", err)
		return 1
	}

	fmt.Printf("Feed %s updated.\n", id)
	return 0
}

func runFeedDelete(cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: feed ID required")
		fmt.Fprintln(os.Stderr, "Usage: spartan feed delete <id>")
		return 1
	}

	id := args[0]
	storage := feed.NewFileStorage(cfg.DataDir)

	if err := storage.Delete(id); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to delete feed: %v\n", err)
		return 1
	}

	fmt.Printf("Feed %s deleted.\n", id)
	return 0
}

func runFeedCheck(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: feed ID required")
		fmt.Fprintln(os.Stderr, "Usage: spartan feed check <id>")
		return 1
	}

	id := args[0]
	storage := feed.NewFileStorage(cfg.DataDir)

	f, err := storage.Get(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get feed: %v\n", err)
		return 1
	}

	// Open store for job manager
	stateStore, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open store: %v\n", err)
		return 1
	}
	defer stateStore.Close()

	// Create job manager for auto-scrape
	jobManager := jobs.NewManager(
		stateStore,
		cfg.DataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.MaxResponseBytes,
		cfg.UsePlaywright,
		fetch.CircuitBreakerConfig{},
		nil,
	)

	// Create seen storage and checker
	seenStorage := feed.NewFileSeenStorage(cfg.DataDir)
	checker := feed.NewChecker(storage, seenStorage, jobManager)

	fmt.Printf("Checking feed %s (%s)...\n", f.ID, f.URL)
	result, err := checker.Check(ctx, f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: check failed: %v\n", err)
		return 1
	}

	fmt.Printf("Feed check complete:\n")
	fmt.Printf("  Total items: %d\n", result.TotalItems)
	fmt.Printf("  New items:   %d\n", len(result.NewItems))
	if result.FeedTitle != "" {
		fmt.Printf("  Feed title:  %s\n", result.FeedTitle)
	}

	if len(result.NewItems) > 0 {
		fmt.Println("\nNew items:")
		for _, item := range result.NewItems {
			fmt.Printf("  - %s\n", item.Title)
			fmt.Printf("    Link: %s\n", item.Link)
			if item.PubDate.IsZero() {
				fmt.Printf("    Published: %s\n", item.PubDate.Format(time.RFC3339))
			}
		}
	}

	return 0
}

func runFeedItems(cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: feed ID required")
		fmt.Fprintln(os.Stderr, "Usage: spartan feed items <id>")
		return 1
	}

	id := args[0]
	storage := feed.NewFileStorage(cfg.DataDir)

	// Check feed exists
	f, err := storage.Get(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get feed: %v\n", err)
		return 1
	}

	seenStorage := feed.NewFileSeenStorage(cfg.DataDir)
	items, err := seenStorage.GetSeen(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get seen items: %v\n", err)
		return 1
	}

	fmt.Printf("Seen items for feed %s (%s):\n", f.ID, f.URL)
	fmt.Printf("Total: %d items\n\n", len(items))

	for _, item := range items {
		fmt.Printf("  - %s\n", item.Title)
		fmt.Printf("    GUID: %s\n", item.GUID)
		fmt.Printf("    Link: %s\n", item.Link)
		fmt.Printf("    Seen: %s\n", item.SeenAt.Format(time.RFC3339))
		fmt.Println()
	}

	return 0
}

func runFeedStart(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("feed start", flag.ExitOnError)
	pollInterval := fs.Duration("poll-interval", 10*time.Second, "How often to check for due feeds")
	maxConcurrent := fs.Int("max-concurrent", 5, "Maximum concurrent feed checks")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	storage := feed.NewFileStorage(cfg.DataDir)

	// Open store for job manager
	stateStore, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open store: %v\n", err)
		return 1
	}
	defer stateStore.Close()

	// Create job manager for auto-scrape
	jobManager := jobs.NewManager(
		stateStore,
		cfg.DataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.MaxResponseBytes,
		cfg.UsePlaywright,
		fetch.CircuitBreakerConfig{},
		nil,
	)

	// Create seen storage and checker
	seenStorage := feed.NewFileSeenStorage(cfg.DataDir)
	checker := feed.NewChecker(storage, seenStorage, jobManager)

	scheduler := feed.NewScheduler(checker, storage, feed.SchedulerConfig{
		Interval:      *pollInterval,
		MaxConcurrent: *maxConcurrent,
	})

	fmt.Println("Starting feed scheduler...")
	fmt.Printf("  Poll interval: %v\n", *pollInterval)
	fmt.Printf("  Max concurrent: %d\n", *maxConcurrent)
	fmt.Println("Press Ctrl+C to stop.")

	if err := scheduler.Run(ctx); err != nil {
		if err == context.Canceled {
			fmt.Println("\nScheduler stopped.")
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: scheduler failed: %v\n", err)
		return 1
	}

	return 0
}

func printFeedHelp() {
	fmt.Println(`Usage: spartan feed <command> [options]

Commands:
  add     Create a new RSS/Atom feed monitor
  list    List all feeds
  get     Get a feed by ID
  update  Update a feed
  delete  Delete a feed by ID
  check   Manually check a feed
  items   List seen items for a feed
  start   Start the feed scheduler

Examples:
  # Add an RSS feed
  spartan feed add --url https://news.ycombinator.com/rss --interval 1800

  # Add an Atom feed with auto-scrape disabled
  spartan feed add --url https://example.com/feed.atom --type atom --auto-scrape=false

  # List all feeds
  spartan feed list

  # Check a feed manually
  spartan feed check <feed-id>

  # Start the scheduler
  spartan feed start`)
}

// FeedHelp returns the help text for the feed command.
func FeedHelp() string {
	return `Manage RSS/Atom feed monitoring.

Usage: spartan feed <command> [options]

Commands:
  add     Create a new RSS/Atom feed monitor
  list    List all feeds
  get     Get a feed by ID
  update  Update a feed
  delete  Delete a feed by ID
  check   Manually check a feed
  items   List seen items for a feed
  start   Start the feed scheduler

Use "spartan feed <command> --help" for more information about a command.`
}
