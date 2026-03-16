// Package batch provides CLI commands for batch job operations.
//
// Purpose:
// - Expose paginated batch summary inspection for terminal operators.
//
// Responsibilities:
// - Parse `spartan batch list` flags.
// - Prefer the local API when available, with direct-store fallback offline.
// - Render aggregate batch summaries with pagination metadata.
//
// Scope:
// - Batch listing only; batch submission, detail, and cancel flows live elsewhere.
//
// Usage:
// - Run `spartan batch list [--limit N] [--offset N]`.
//
// Invariants/Assumptions:
// - Limit defaults to 100 and is capped at 1000.
// - Offset must be greater than or equal to 0.
// - Batch rows show aggregate stats only; use `spartan batch status <id>` for jobs.
package batch

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func runBatchList(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("batch-list", flag.ContinueOnError)
	limit := fs.Int("limit", 100, "Maximum number of batches to show")
	offset := fs.Int("offset", 0, "Number of batches to skip")
	fs.Usage = func() {
		fmt.Print(`Usage: spartan batch list [options]

Options:
  --limit int   Maximum number of batches to show (default: 100)
  --offset int  Number of batches to skip (default: 0)

Examples:
  spartan batch list
  spartan batch list --limit 25 --offset 50
`)
	}
	_ = fs.Parse(args)

	normalizedLimit, normalizedOffset, err := normalizeBatchPage(*limit, *offset)
	if err != nil {
		fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
		return 1
	}

	result, err := listBatches(ctx, cfg, normalizedLimit, normalizedOffset)
	if err != nil {
		fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
		return 1
	}

	printBatchList(result)
	return 0
}

func normalizeBatchPage(limit, offset int) (int, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		return 0, 0, apperrors.Validation("offset must be greater than or equal to 0")
	}
	return limit, offset, nil
}

func listBatches(ctx context.Context, cfg config.Config, limit, offset int) (*BatchListResponse, error) {
	if isServerRunning(ctx, cfg.Port) {
		return listBatchesViaAPI(ctx, cfg.Port, limit, offset)
	}
	return listBatchesDirect(ctx, cfg, limit, offset)
}

func printBatchList(result *BatchListResponse) {
	if result == nil || len(result.Batches) == 0 {
		fmt.Println("No batches found.")
		return
	}

	fmt.Printf("Batches (showing %d of %d, limit=%d, offset=%d):\n", len(result.Batches), result.Total, result.Limit, result.Offset)
	fmt.Printf("%-36s %-10s %-12s %-10s %-6s %-7s %-8s %-9s %-10s %-10s\n", "ID", "KIND", "STATUS", "PROGRESS", "JOBS", "QUEUED", "RUNNING", "SUCCESS", "FAILED", "CANCELED")
	fmt.Println(strings.Repeat("-", 138))
	for _, batch := range result.Batches {
		progress := fmt.Sprintf("%d%%", batch.Progress.Percent)
		fmt.Printf("%-36s %-10s %-12s %-10s %-6d %-7d %-8d %-9d %-10d %-10d\n",
			batch.ID,
			batch.Kind,
			batch.Status,
			progress,
			batch.JobCount,
			batch.Stats.Queued,
			batch.Stats.Running,
			batch.Stats.Succeeded,
			batch.Stats.Failed,
			batch.Stats.Canceled,
		)
	}
}
