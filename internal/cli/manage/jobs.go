// Package manage contains jobs CLI command wiring.
//
// It does NOT define job execution; internal/jobs and internal/api do.
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

const jobsCommandHelpText = `Manage persisted jobs.

Usage:
  spartan jobs <subcommand> [options]

Subcommands:
  list    List jobs (with pagination)
  get     Get job details
  cancel  Cancel a running or queued job

Examples:
  spartan jobs list
  spartan jobs list --limit 50
  spartan jobs list --offset 100
  spartan jobs list --status running
  spartan jobs list --status failed --limit 20
  spartan jobs get <job-id>
  spartan jobs cancel <job-id>
`

func RunJobs(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printJobsHelp()
		return 1
	}

	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("jobs list", flag.ExitOnError)
		limit := fs.Int("limit", 100, "Maximum number of jobs to list")
		offset := fs.Int("offset", 0, "Number of jobs to skip")
		status := fs.String("status", "", "Filter jobs by status (queued|running|succeeded|failed|canceled)")
		_ = fs.Parse(args[1:])

		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()

		var jobsList []model.Job
		if *status != "" {
			statusVal := model.Status(*status)
			if !statusVal.IsValid() {
				fmt.Fprintf(os.Stderr, "invalid status: %s (must be queued, running, succeeded, failed, or canceled)\n", *status)
				return 1
			}
			opts := store.ListByStatusOptions{Limit: *limit, Offset: *offset}
			jobsList, err = st.ListByStatus(ctx, statusVal, opts)
		} else {
			opts := store.ListOptions{Limit: *limit, Offset: *offset}
			jobsList, err = st.ListOpts(ctx, opts)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		for _, job := range jobsList {
			fmt.Printf("%s\t%s\t%s\t%s\n", job.ID, job.Kind, job.Status, job.CreatedAt.Format(time.RFC3339))
		}
		return 0

	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job id is required")
			return 1
		}
		id := args[1]
		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()
		job, err := st.Get(ctx, id)
		if err != nil {
			fmt.Fprintln(os.Stderr, "job not found")
			return 1
		}
		payload, _ := json.MarshalIndent(job, "", "  ")
		fmt.Println(string(payload))
		return 0

	case "cancel":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job id is required")
			return 1
		}
		id := args[1]

		// Check if server is running first
		if isServerRunning(ctx, cfg.Port) {
			// Server owns the job - use API to properly cancel active job
			if err := cancelJobViaAPI(ctx, cfg.Port, id); err != nil {
				fmt.Fprintf(os.Stderr, "failed to cancel job via API: %v\n", err)
				return 1
			}
			fmt.Println("canceled", id)
			return 0
		}

		// Server not running - use manager's cancel logic for consistency
		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()

		manager := jobs.NewManager(
			st,
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
			fetch.DefaultCircuitBreakerConfig(),
			nil, // no adaptive rate limiting for cancel operations
		)
		if err := manager.CancelJob(ctx, id); err != nil {
			fmt.Fprintf(os.Stderr, "failed to cancel job: %v\n", err)
			return 1
		}
		fmt.Println("canceled", id)
		return 0

	case "help", "--help", "-h":
		printJobsHelp()
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown jobs subcommand: %s\n", args[0])
		printJobsHelp()
		return 1
	}
}

func printJobsHelp() {
	fmt.Fprint(os.Stderr, jobsCommandHelpText)
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

// cancelJobViaAPI cancels a job by calling server's DELETE /v1/jobs/{id} endpoint.
func cancelJobViaAPI(ctx context.Context, port, jobID string) error {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/%s", port, jobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}
