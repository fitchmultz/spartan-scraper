// Package batch provides CLI commands for batch job operations.
//
// Responsibilities:
// - Submit batch jobs from CSV/JSON input files
// - Check batch status with aggregated statistics
// - Cancel batches and their constituent jobs
//
// Does NOT handle:
// - Individual job operations (see manage/jobs.go)
// - Batch execution logic (see jobs package)
package batch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// RunBatch executes the batch command with the given arguments.
func RunBatch(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printBatchHelp()
		return 1
	}

	switch args[0] {
	case "submit":
		return runBatchSubmit(ctx, cfg, args[1:])
	case "status":
		return runBatchStatus(ctx, cfg, args[1:])
	case "cancel":
		return runBatchCancel(ctx, cfg, args[1:])
	case "help", "--help", "-h":
		printBatchHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown batch subcommand: %s\n", args[0])
		printBatchHelp()
		return 1
	}
}

func printBatchHelp() {
	fmt.Print(`Usage: spartan batch <subcommand> [options]

Subcommands:
  submit       Submit a batch of jobs (scrape, crawl, or research)
  status       Get the status of a batch
  cancel       Cancel a batch and all its jobs

Examples:
  spartan batch submit scrape --file urls.csv --headless
  spartan batch submit crawl --file sites.json --max-depth 2
  spartan batch submit research --urls https://a.com,https://b.com --query "pricing"
  spartan batch status <batch-id> --watch
  spartan batch cancel <batch-id>

Use "spartan batch submit <kind> --help" for more information.
`)
}

// isServerRunning checks if Spartan API server is running by pinging healthz endpoint.
func isServerRunning(ctx context.Context, port string) bool {
	url := fmt.Sprintf("http://localhost:%s/healthz", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
