// Package batch provides CLI commands for batch job operations.
//
// This file contains status checking, display, watching, and cancellation logic.
//
// Responsibilities:
// - Run batch status and cancel commands
// - Display batch status to stdout
// - Watch batch progress until completion
// - Wait for batch completion with timeout
//
// Does NOT handle:
// - Batch submission
// - File parsing
// - Direct API client calls (uses api.go and direct.go)
package batch

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func runBatchStatus(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("batch-status", flag.ContinueOnError)
	watch := fs.Bool("watch", false, "Poll status until batch is complete")
	includeJobs := fs.Bool("include-jobs", false, "Include individual jobs in output")
	pollInterval := fs.Int("poll-interval", 2, "Polling interval in seconds (used with --watch)")

	fs.Usage = func() {
		fmt.Print(`Usage: spartan batch status <batch-id> [options]

Options:
  --watch              Poll status until batch is complete
  --poll-interval int  Polling interval in seconds (default: 2)
  --include-jobs       Include individual jobs in output

Examples:
  spartan batch status abc-123
  spartan batch status abc-123 --watch
  spartan batch status abc-123 --include-jobs
`)
	}
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: batch ID required")
		fs.Usage()
		return 1
	}

	batchID := fs.Arg(0)

	if *watch {
		return watchBatchStatus(ctx, cfg, batchID, time.Duration(*pollInterval)*time.Second)
	}

	status, err := getBatchStatus(ctx, cfg, batchID, *includeJobs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting batch status: %v\n", err)
		return 1
	}

	printBatchStatus(status)
	return 0
}

func runBatchCancel(ctx context.Context, cfg config.Config, args []string) int {
	_ = ctx // Context not used in cancel, but kept for interface consistency
	fs := flag.NewFlagSet("batch-cancel", flag.ContinueOnError)

	fs.Usage = func() {
		fmt.Print(`Usage: spartan batch cancel <batch-id>

Examples:
  spartan batch cancel abc-123
`)
	}
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: batch ID required")
		fs.Usage()
		return 1
	}

	batchID := fs.Arg(0)

	status, err := cancelBatch(ctx, cfg, batchID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error canceling batch: %v\n", err)
		return 1
	}

	printBatchStatus(status)
	return 0
}

func printBatchStatus(status *BatchStatusResponse) {
	fmt.Printf("Batch: %s\n", status.Batch.ID)
	fmt.Printf("Kind: %s\n", status.Batch.Kind)
	fmt.Printf("Status: %s\n", status.Batch.Status)
	fmt.Printf("Jobs: %d total\n", status.Batch.JobCount)
	fmt.Printf("Stats: %d queued, %d running, %d succeeded, %d failed, %d canceled\n",
		status.Batch.Stats.Queued, status.Batch.Stats.Running, status.Batch.Stats.Succeeded,
		status.Batch.Stats.Failed, status.Batch.Stats.Canceled)

	if len(status.Jobs) > 0 {
		fmt.Println("\nJobs:")
		for _, job := range status.Jobs {
			fmt.Printf("  %s (%s): %s\n", job.ID, job.Kind, job.Status)
		}
		return
	}
	if status.Total > 0 && status.Limit == 0 {
		fmt.Println("\nJobs: omitted (re-run with --include-jobs to load individual job entries)")
	}
}

func watchBatchStatus(ctx context.Context, cfg config.Config, batchID string, interval time.Duration) int {
	fmt.Printf("Watching batch %s (press Ctrl+C to stop)...\n", batchID)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		status, err := getBatchStatus(ctx, cfg, batchID, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError getting batch status: %v\n", err)
			return 1
		}

		// Clear previous line and print status
		completed := status.Batch.Stats.Succeeded + status.Batch.Stats.Failed + status.Batch.Stats.Canceled
		fmt.Printf("\rStatus: %s | Progress: %d/%d (%d%%)",
			status.Batch.Status,
			completed,
			status.Batch.JobCount,
			(completed*100)/status.Batch.JobCount,
		)

		// Check if batch is in terminal state
		if isTerminalStatus(status.Batch.Status) {
			fmt.Println() // New line after progress
			fmt.Printf("\nBatch %s finished with status: %s\n", batchID, status.Batch.Status)
			fmt.Printf("Final stats: %d succeeded, %d failed, %d canceled\n",
				status.Batch.Stats.Succeeded, status.Batch.Stats.Failed, status.Batch.Stats.Canceled)
			return 0
		}

		select {
		case <-ctx.Done():
			fmt.Println("\n\nStopped watching")
			return 0
		case <-ticker.C:
			// Continue to next poll
		}
	}
}

func isTerminalStatus(status string) bool {
	switch status {
	case "completed", "failed", "partial", "canceled":
		return true
	default:
		return false
	}
}

func waitForBatch(ctx context.Context, cfg config.Config, batchID string, timeout time.Duration) int {
	fmt.Printf("Waiting for batch %s...\n", batchID)

	start := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		status, err := getBatchStatus(ctx, cfg, batchID, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError getting batch status: %v\n", err)
			return 1
		}

		// Check timeout
		if timeout > 0 && time.Since(start) > timeout {
			fmt.Printf("\nTimeout waiting for batch %s\n", batchID)
			return 1
		}

		// Print progress
		completed := status.Batch.Stats.Succeeded + status.Batch.Stats.Failed + status.Batch.Stats.Canceled
		fmt.Printf("\rProgress: %d/%d jobs complete (%d%%) - Status: %s",
			completed, status.Batch.JobCount, (completed*100)/status.Batch.JobCount, status.Batch.Status)

		// Check if batch is in terminal state
		if isTerminalStatus(status.Batch.Status) {
			fmt.Println() // New line after progress
			fmt.Printf("\nBatch %s finished with status: %s\n", batchID, status.Batch.Status)
			fmt.Printf("Results: %d succeeded, %d failed, %d canceled\n",
				status.Batch.Stats.Succeeded, status.Batch.Stats.Failed, status.Batch.Stats.Canceled)
			if status.Batch.Status == "completed" {
				return 0
			}
			return 1
		}

		select {
		case <-ctx.Done():
			fmt.Println("\n\nCanceled")
			return 1
		case <-ticker.C:
			// Continue to next poll
		}
	}
}

func getBatchStatus(ctx context.Context, cfg config.Config, batchID string, includeJobs bool) (*BatchStatusResponse, error) {
	if isServerRunning(ctx, cfg.Port) {
		return getBatchStatusViaAPI(ctx, cfg.Port, batchID, includeJobs)
	}
	return getBatchStatusDirect(ctx, cfg, batchID, includeJobs)
}

func cancelBatch(ctx context.Context, cfg config.Config, batchID string) (*BatchResponse, error) {
	if isServerRunning(ctx, cfg.Port) {
		return cancelBatchViaAPI(ctx, cfg.Port, batchID)
	}
	return cancelBatchDirect(ctx, cfg, batchID)
}
