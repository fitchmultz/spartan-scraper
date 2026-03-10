// Package manage provides CLI subcommands for managing spartan scraper resources.
//
// This file is responsible for:
// - watch add: Create a new content watch
// - watch list: List all watches
// - watch delete: Delete a watch by ID
// - watch check: Manually check a watch
// - watch start: Start the watch scheduler
//
// This file does NOT handle:
// - Watch scheduling (watch/scheduler.go handles this)
// - Watch execution (watch/watch.go handles this)
// - Diff generation (diff package handles this)
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
	"strconv"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

const watchCommandHelpText = `Watch content for changes.

Usage: spartan watch <command> [options]

Commands:
  add     Create a new content watch
  list    List all watches
  delete  Delete a watch by ID
  check   Manually check a watch
  start   Start the watch scheduler

Examples:
  spartan watch add --url https://example.com --interval 3600
  spartan watch add --url https://example.com --selector "#price" --interval 300
  spartan watch add --url https://example.com --webhook https://hooks.slack.com/... --webhook-secret mysecret
  spartan watch list
  spartan watch check <watch-id>
  spartan watch start

Use "spartan watch <command> --help" for more information about a command.
`

// RunWatch routes watch subcommands.
func RunWatch(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: watch subcommand required (add, list, delete, check, start)")
		printWatchHelp()
		return 1
	}

	switch args[0] {
	case "add":
		return runWatchAdd(cfg, args[1:])
	case "list":
		return runWatchList(cfg, args[1:])
	case "delete":
		return runWatchDelete(cfg, args[1:])
	case "check":
		return runWatchCheck(ctx, cfg, args[1:])
	case "start":
		return runWatchStart(ctx, cfg, args[1:])
	case "help", "--help", "-h":
		printWatchHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown watch subcommand: %s\n", args[0])
		printWatchHelp()
		return 1
	}
}

func runWatchAdd(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("watch add", flag.ExitOnError)
	url := fs.String("url", "", "URL to watch (required)")
	selector := fs.String("selector", "", "CSS selector for targeted monitoring")
	interval := fs.Int("interval", 3600, "Check interval in seconds (min: 60)")
	diffFormat := fs.String("diff-format", "unified", "Diff format: unified, html-side-by-side, html-inline")
	webhookURL := fs.String("webhook", "", "Webhook URL for change notifications")
	webhookSecret := fs.String("webhook-secret", "", "Webhook secret for HMAC signature")
	headless := fs.Bool("headless", false, "Use headless browser")
	playwright := fs.Bool("playwright", false, "Use Playwright instead of chromedp")
	extractMode := fs.String("extract", "html", "Extraction mode: html, text")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *url == "" {
		fmt.Fprintln(os.Stderr, "Error: --url is required")
		return 1
	}

	storage := watch.NewFileStorage(cfg.DataDir)

	w := &watch.Watch{
		URL:             *url,
		Selector:        *selector,
		IntervalSeconds: *interval,
		Enabled:         true,
		CreatedAt:       time.Now(),
		DiffFormat:      *diffFormat,
		NotifyOnChange:  *webhookURL != "",
		Headless:        *headless,
		UsePlaywright:   *playwright,
		ExtractMode:     *extractMode,
	}

	if *webhookURL != "" {
		w.WebhookConfig = &model.WebhookConfig{
			URL:    *webhookURL,
			Events: []string{"content_changed"},
			Secret: *webhookSecret,
		}
	}

	if err := w.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	result, err := storage.Add(w)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to add watch: %v\n", err)
		return 1
	}

	fmt.Printf("Watch created:\n")
	fmt.Printf("  ID:       %s\n", result.ID)
	fmt.Printf("  URL:      %s\n", result.URL)
	fmt.Printf("  Selector: %s\n", result.Selector)
	fmt.Printf("  Interval: %d seconds\n", result.IntervalSeconds)
	fmt.Printf("  Status:   %s\n", result.GetStatus())

	return 0
}

func runWatchList(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("watch list", flag.ExitOnError)
	showAll := fs.Bool("all", false, "Show all watches including disabled")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	storage := watch.NewFileStorage(cfg.DataDir)
	watches, err := storage.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list watches: %v\n", err)
		return 1
	}

	if len(watches) == 0 {
		fmt.Println("No watches configured.")
		return 0
	}

	// Header
	fmt.Printf("%-36s %-40s %-10s %-12s %-20s\n", "ID", "URL", "STATUS", "INTERVAL", "LAST CHECKED")
	fmt.Println(strings.Repeat("-", 120))

	for _, w := range watches {
		if !*showAll && !w.Enabled {
			continue
		}

		status := w.GetStatus()
		url := w.URL
		if len(url) > 38 {
			url = url[:35] + "..."
		}

		lastChecked := "never"
		if !w.LastCheckedAt.IsZero() {
			lastChecked = time.Since(w.LastCheckedAt).Round(time.Second).String() + " ago"
		}

		fmt.Printf("%-36s %-40s %-10s %-12d %-20s\n", w.ID, url, status, w.IntervalSeconds, lastChecked)

		if w.Selector != "" {
			fmt.Printf("  Selector: %s\n", w.Selector)
		}
	}

	return 0
}

func runWatchDelete(cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: watch ID required")
		fmt.Fprintln(os.Stderr, "Usage: spartan watch delete <id>")
		return 1
	}

	id := args[0]
	storage := watch.NewFileStorage(cfg.DataDir)

	if err := storage.Delete(id); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to delete watch: %v\n", err)
		return 1
	}

	fmt.Printf("Watch %s deleted.\n", id)
	return 0
}

func runWatchCheck(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: watch ID required")
		fmt.Fprintln(os.Stderr, "Usage: spartan watch check <id>")
		return 1
	}

	id := args[0]
	storage := watch.NewFileStorage(cfg.DataDir)

	w, err := storage.Get(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get watch: %v\n", err)
		return 1
	}

	// Open store for crawl state
	stateStore, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open store: %v\n", err)
		return 1
	}
	defer stateStore.Close()

	watcher := watch.NewWatcher(storage, stateStore, cfg.DataDir, nil)

	fmt.Printf("Checking watch %s (%s)...\n", w.ID, w.URL)
	result, err := watcher.Check(ctx, w)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: check failed: %v\n", err)
		return 1
	}

	if result.Changed {
		fmt.Println("Content changed!")
		fmt.Printf("  Previous hash: %s\n", result.PreviousHash[:8])
		fmt.Printf("  Current hash:  %s\n", result.CurrentHash[:8])
		if result.DiffText != "" {
			fmt.Println("\nDiff:")
			fmt.Println(result.DiffText)
		}
	} else {
		fmt.Println("No changes detected.")
	}

	return 0
}

func runWatchStart(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("watch start", flag.ExitOnError)
	pollInterval := fs.Duration("poll-interval", 10*time.Second, "How often to check for due watches")
	maxConcurrent := fs.Int("max-concurrent", 5, "Maximum concurrent watch checks")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	storage := watch.NewFileStorage(cfg.DataDir)

	// Open store for crawl state
	stateStore, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open store: %v\n", err)
		return 1
	}
	defer stateStore.Close()

	// Create dispatcher
	dispatcher := webhook.NewDispatcher(webhook.Config{})

	watcher := watch.NewWatcher(storage, stateStore, cfg.DataDir, dispatcher)
	scheduler := watch.NewScheduler(watcher, storage, watch.SchedulerConfig{
		Interval:      *pollInterval,
		MaxConcurrent: *maxConcurrent,
	})

	fmt.Println("Starting watch scheduler...")
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

func printWatchHelp() {
	fmt.Fprint(os.Stderr, watchCommandHelpText)
}

// RunWatchAdd is the entry point for the watch add command (used by CLI router).
func RunWatchAdd(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	result := runWatchAdd(cfg, args)
	if result != 0 {
		return fmt.Errorf("watch add failed with exit code %d", result)
	}
	return nil
}

// RunWatchList is the entry point for the watch list command.
func RunWatchList(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	result := runWatchList(cfg, args)
	if result != 0 {
		return fmt.Errorf("watch list failed with exit code %d", result)
	}
	return nil
}

// RunWatchDelete is the entry point for the watch delete command.
func RunWatchDelete(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	result := runWatchDelete(cfg, args)
	if result != 0 {
		return fmt.Errorf("watch delete failed with exit code %d", result)
	}
	return nil
}

// RunWatchCheck is the entry point for the watch check command.
func RunWatchCheck(ctx context.Context, cfg config.Config, args []string) int {
	return runWatchCheck(ctx, cfg, args)
}

// RunWatchStart is the entry point for the watch start command.
func RunWatchStart(ctx context.Context, cfg config.Config, args []string) int {
	return runWatchStart(ctx, cfg, args)
}

// RunWatchCommand is the main entry point for the watch command.
func RunWatchCommand(ctx context.Context, cfg config.Config, args []string) int {
	return RunWatch(ctx, cfg, args)
}

// WatchHelp returns the help text for the watch command.
func WatchHelp() string {
	return watchCommandHelpText
}

// ParseBool parses a boolean string value.
func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}
