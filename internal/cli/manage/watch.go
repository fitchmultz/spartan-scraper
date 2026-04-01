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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/runtime"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

const watchCommandHelpText = `Watch content for changes.

Usage: spartan watch <command> [options]

Commands:
  add      Create a new content watch
  list     List all watches
  delete   Delete a watch by ID
  check    Manually check a watch
  history  Inspect persisted watch check history
  start    Start the watch scheduler

Examples:
  spartan watch add --url https://example.com --interval 3600
  spartan watch add --url https://example.com --selector "#price" --interval 300
  spartan watch add --url https://example.com --webhook https://hooks.slack.com/... --webhook-secret mysecret
  spartan watch add --url https://example.com/pricing --trigger-kind scrape --trigger-request-file ./pricing-job.json
  spartan watch list
  spartan watch check <watch-id>
  spartan watch history <watch-id>
  spartan watch history <watch-id> --check-id <check-id>
  spartan watch start

Use "spartan watch <command> --help" for more information about a command.
`

// RunWatch routes watch subcommands.
func RunWatch(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: watch subcommand required (add, list, delete, check, history, start)")
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
	case "history":
		return runWatchHistory(cfg, args[1:])
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

func loadWatchJobTrigger(cfg config.Config, kind string, requestFile string, requestJSON string) (*watch.JobTrigger, error) {
	trimmedKind := strings.TrimSpace(kind)
	if trimmedKind == "" && strings.TrimSpace(requestFile) == "" && strings.TrimSpace(requestJSON) == "" {
		return nil, nil
	}
	if trimmedKind == "" {
		return nil, fmt.Errorf("--trigger-kind is required when providing a watch trigger request")
	}
	if strings.TrimSpace(requestFile) != "" && strings.TrimSpace(requestJSON) != "" {
		return nil, fmt.Errorf("--trigger-request-file and --trigger-request-json are mutually exclusive")
	}
	if strings.TrimSpace(requestFile) == "" && strings.TrimSpace(requestJSON) == "" {
		return nil, fmt.Errorf("--trigger-request-file or --trigger-request-json is required when --trigger-kind is set")
	}
	var raw []byte
	if strings.TrimSpace(requestFile) != "" {
		data, err := os.ReadFile(requestFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read trigger request file: %w", err)
		}
		raw = data
	} else {
		raw = []byte(requestJSON)
	}
	normalizedRequest, err := submission.NormalizeRawRequest(model.Kind(trimmedKind), raw)
	if err != nil {
		return nil, err
	}
	if _, _, err := submission.JobSpecFromRawRequest(cfg, submission.Defaults{
		DefaultTimeoutSeconds: cfg.RequestTimeoutSecs,
		DefaultUsePlaywright:  cfg.UsePlaywright,
		ResolveAuth:           false,
	}, model.Kind(trimmedKind), normalizedRequest); err != nil {
		return nil, err
	}
	return &watch.JobTrigger{Kind: model.Kind(trimmedKind), Request: normalizedRequest}, nil
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
	triggerKind := fs.String("trigger-kind", "", "Job kind to submit on detected change (scrape|crawl|research)")
	triggerRequestFile := fs.String("trigger-request-file", "", "Path to JSON file containing the operator-facing request payload to submit on change")
	triggerRequestJSON := fs.String("trigger-request-json", "", "Inline JSON operator-facing request payload to submit on change")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *url == "" {
		fmt.Fprintln(os.Stderr, "Error: --url is required")
		return 1
	}

	trigger, err := loadWatchJobTrigger(cfg, *triggerKind, *triggerRequestFile, *triggerRequestJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid watch trigger: %v\n", err)
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
		JobTrigger:      trigger,
	}

	if *webhookURL != "" {
		w.WebhookConfig = &model.WebhookSpec{
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
	if result.JobTrigger != nil {
		fmt.Printf("  Trigger:  %s\n", result.JobTrigger.Kind)
	}

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
		if w.JobTrigger != nil {
			fmt.Printf("  Trigger: %s\n", w.JobTrigger.Kind)
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

	// Open store for crawl state and optional triggered job creation.
	stateStore, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open store: %v\n", err)
		return 1
	}
	defer stateStore.Close()

	manager, err := runtime.InitJobManager(ctx, cfg, stateStore)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize job manager: %v\n", err)
		return 1
	}

	watcher := watch.NewWatcher(storage, stateStore, cfg.DataDir, nil, &watch.TriggerRuntime{
		Config:  cfg,
		Manager: manager,
	})

	fmt.Printf("Checking watch %s (%s)...\n", w.ID, w.URL)
	result, err := watcher.Check(ctx, w)
	if result == nil {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: check failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(os.Stderr, "Error: watch check returned no result")
		return 1
	}

	inspection := api.BuildWatchCheckInspection(watch.RecordFromCheckResult(result))
	printWatchInspection(inspection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Check completed with errors: %v\n", err)
		return 1
	}
	return 0
}

func runWatchHistory(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("watch history", flag.ExitOnError)
	checkID := fs.String("check-id", "", "Inspect a single persisted watch check by check id")
	limit := fs.Int("limit", 10, "Maximum number of history rows to show")
	offset := fs.Int("offset", 0, "History row offset for pagination")
	jsonOut := fs.Bool("json", false, "Emit JSON watch history envelopes instead of guided text")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan watch history <watch-id> [--limit <n>] [--offset <n>] [--json]
  spartan watch history <watch-id> --check-id <check-id> [--json]

Examples:
  spartan watch history <watch-id>
  spartan watch history <watch-id> --limit 5
  spartan watch history <watch-id> --check-id <check-id>
  spartan watch history <watch-id> --json

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: watch ID required")
		fs.Usage()
		return 1
	}

	watchID := strings.TrimSpace(fs.Arg(0))
	historyStore := watch.NewWatchHistoryStore(cfg.DataDir)
	if strings.TrimSpace(*checkID) != "" {
		record, err := historyStore.GetByID(watchID, strings.TrimSpace(*checkID))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to load watch check: %v\n", err)
			return 1
		}
		return printWatchHistoryEnvelope(api.WatchCheckInspectionResponse{Check: api.BuildWatchCheckInspection(*record)}, *jsonOut, 0)
	}

	records, total, err := historyStore.GetByWatch(watchID, *limit, *offset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load watch history: %v\n", err)
		return 1
	}
	return printWatchHistoryList(api.BuildWatchCheckHistoryResponse(records, total, *limit, *offset), watchID, *jsonOut)
}

func printWatchHistoryEnvelope(response api.WatchCheckInspectionResponse, jsonOut bool, exitCode int) int {
	if jsonOut {
		return printWatchJSON(response, exitCode)
	}
	printWatchInspection(response.Check)
	return exitCode
}

func printWatchHistoryList(response api.WatchCheckHistoryResponse, watchID string, jsonOut bool) int {
	if jsonOut {
		return printWatchJSON(response, 0)
	}
	if len(response.Checks) == 0 {
		fmt.Printf("No watch history found for %s.\n", watchID)
		return 0
	}
	fmt.Printf("Watch history for %s (showing %d of %d):\n\n", watchID, len(response.Checks), response.Total)
	for i, check := range response.Checks {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 72))
		}
		printWatchInspection(check)
	}
	return 0
}

func printWatchJSON(value any, exitCode int) int {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return exitCode
}

func printWatchInspection(inspection api.WatchCheckInspection) {
	fmt.Printf("%s\n", strings.ToUpper(inspection.Title))
	fmt.Printf("Check ID: %s\n", inspection.ID)
	fmt.Printf("Watch: %s\n", inspection.WatchID)
	fmt.Printf("Status: %s\n", inspection.Status)
	fmt.Printf("Checked: %s\n", inspection.CheckedAt.Format(time.RFC3339))
	if inspection.CurrentHash != "" {
		fmt.Printf("Current hash: %s\n", inspection.CurrentHash)
	}
	if inspection.PreviousHash != "" {
		fmt.Printf("Previous hash: %s\n", inspection.PreviousHash)
	}
	if inspection.Baseline {
		fmt.Println("Baseline: this check established the first comparison snapshot")
	}
	if len(inspection.TriggeredJobs) > 0 {
		fmt.Printf("Triggered jobs: %s\n", strings.Join(inspection.TriggeredJobs, ", "))
	}
	if len(inspection.Artifacts) > 0 {
		fmt.Println("Artifacts:")
		for _, artifact := range inspection.Artifacts {
			fmt.Printf("- %s (%s) [%s]\n", artifact.Filename, artifact.ContentType, artifact.DownloadURL)
		}
	}
	if inspection.Error != "" {
		fmt.Printf("Error: %s\n", inspection.Error)
	}
	fmt.Printf("Message: %s\n", inspection.Message)
	if inspection.DiffText != "" {
		fmt.Println("Diff:")
		fmt.Println(inspection.DiffText)
	}
	if len(inspection.Actions) > 0 {
		fmt.Println("Next steps:")
		for _, action := range inspection.Actions {
			fmt.Printf("- %s [%s]: %s\n", action.Label, action.Kind, action.Value)
		}
	}
	fmt.Println()
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

	manager, err := runtime.InitJobManager(ctx, cfg, stateStore)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize job manager: %v\n", err)
		return 1
	}

	// Create dispatcher
	dispatcher := webhook.NewDispatcher(webhook.Config{})
	defer dispatcher.Close()

	watcher := watch.NewWatcher(storage, stateStore, cfg.DataDir, dispatcher, &watch.TriggerRuntime{
		Config:  cfg,
		Manager: manager,
	})
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
