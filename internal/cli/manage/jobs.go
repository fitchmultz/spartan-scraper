// Package manage contains jobs CLI command wiring.
//
// Purpose:
// - Expose canonical recent-run inspection and job control commands for terminal operators.
//
// Responsibilities:
// - Parse `spartan jobs` subcommands and flags.
// - Prefer the live API when available, with store-backed direct-mode parity offline.
// - Render recent-run summaries, failed-run summaries, enriched job detail JSON, and cancel results.
//
// Scope:
// - Jobs inspection and cancellation only; job execution lives in internal/jobs and internal/api.
//
// Usage:
// - Run `spartan jobs list`, `spartan jobs failures`, `spartan jobs get <id>`, or `spartan jobs cancel <id>`.
//
// Invariants/Assumptions:
// - Direct-mode output must match the canonical API response envelopes.
// - Status filtering accepts only persisted job lifecycle states.
// - When the server is running, cancellation routes through the API so in-memory workers are updated correctly.
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

const jobsCommandHelpText = `Manage persisted jobs and recent runs.

Usage:
  spartan jobs <subcommand> [options]

Subcommands:
  list       List recent runs with queue/failure context
  failures   List recent failed runs
  get        Get enriched job details
  cancel     Cancel a running or queued job
  help       Show this help text

Examples:
  spartan jobs list
  spartan jobs list --limit 50 --offset 100
  spartan jobs list --status running
  spartan jobs failures --limit 20
  spartan jobs get <job-id>
  spartan jobs cancel <job-id>

Exit codes:
  0  Success
  1  Invalid input, lookup failure, or runtime/API error
`

func RunJobs(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printJobsHelp()
		return 1
	}

	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("jobs list", flag.ContinueOnError)
		limit := fs.Int("limit", 100, "Maximum number of runs to list")
		offset := fs.Int("offset", 0, "Number of runs to skip")
		status := fs.String("status", "", "Filter runs by status (queued|running|succeeded|failed|canceled)")
		fs.SetOutput(ioDiscard{})
		if err := fs.Parse(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		response, err := loadJobList(ctx, cfg, *limit, *offset, *status, false)
		if err != nil {
			fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
			return 1
		}
		printJobList(response, false)
		return 0

	case "failures":
		fs := flag.NewFlagSet("jobs failures", flag.ContinueOnError)
		limit := fs.Int("limit", 50, "Maximum number of failed runs to list")
		offset := fs.Int("offset", 0, "Number of failed runs to skip")
		fs.SetOutput(ioDiscard{})
		if err := fs.Parse(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		response, err := loadJobList(ctx, cfg, *limit, *offset, "", true)
		if err != nil {
			fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
			return 1
		}
		printJobList(response, true)
		return 0

	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job id is required")
			return 1
		}
		response, err := loadJob(ctx, cfg, args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
			return 1
		}
		payload, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to encode job response")
			return 1
		}
		fmt.Println(string(payload))
		return 0

	case "cancel":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job id is required")
			return 1
		}
		id := args[1]

		if isServerRunning(ctx, cfg.Port) {
			if err := cancelJobViaAPI(ctx, cfg.Port, id); err != nil {
				fmt.Fprintf(os.Stderr, "failed to cancel job via API: %s\n", apperrors.SafeMessage(err))
				return 1
			}
			fmt.Println("canceled", id)
			return 0
		}

		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
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
			nil,
		)
		if err := manager.CancelJob(ctx, id); err != nil {
			fmt.Fprintf(os.Stderr, "failed to cancel job: %s\n", apperrors.SafeMessage(err))
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

func loadJobList(ctx context.Context, cfg config.Config, limit, offset int, status string, failuresOnly bool) (*api.JobListResponse, error) {
	limit, offset, normalizedStatus, err := normalizeJobListInputs(limit, offset, status, failuresOnly)
	if err != nil {
		return nil, err
	}

	if isServerRunning(ctx, cfg.Port) {
		return loadJobListViaAPI(ctx, cfg.Port, limit, offset, normalizedStatus, failuresOnly)
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	var (
		jobsList []model.Job
		total    int
	)
	if failuresOnly {
		jobsList, err = st.ListByStatus(ctx, model.StatusFailed, store.ListByStatusOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		total, err = st.CountJobs(ctx, model.StatusFailed)
		if err != nil {
			return nil, err
		}
	} else if normalizedStatus != "" {
		statusValue := model.Status(normalizedStatus)
		jobsList, err = st.ListByStatus(ctx, statusValue, store.ListByStatusOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		total, err = st.CountJobs(ctx, statusValue)
		if err != nil {
			return nil, err
		}
	} else {
		jobsList, err = st.ListOpts(ctx, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		total, err = st.CountJobs(ctx, "")
		if err != nil {
			return nil, err
		}
	}

	response, err := api.BuildStoreBackedJobListResponse(ctx, st, jobsList, total, limit, offset)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func loadJob(ctx context.Context, cfg config.Config, jobID string) (*api.JobResponse, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, apperrors.Validation("job id is required")
	}

	if isServerRunning(ctx, cfg.Port) {
		endpoint := fmt.Sprintf("http://localhost:%s/v1/jobs/%s", cfg.Port, url.PathEscape(jobID))
		return doJSONRequest[api.JobResponse](ctx, http.MethodGet, endpoint)
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	job, err := st.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	response, err := api.BuildStoreBackedJobResponse(ctx, st, job)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func normalizeJobListInputs(limit, offset int, status string, failuresOnly bool) (int, int, string, error) {
	if limit <= 0 {
		limit = 100
	}
	if failuresOnly && limit > 1000 {
		limit = 1000
	}
	if !failuresOnly && limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		return 0, 0, "", apperrors.Validation("offset must be greater than or equal to 0")
	}

	normalizedStatus := strings.TrimSpace(status)
	if failuresOnly {
		return limit, offset, "", nil
	}
	if normalizedStatus == "" {
		return limit, offset, "", nil
	}
	statusValue := model.Status(normalizedStatus)
	if !statusValue.IsValid() {
		return 0, 0, "", apperrors.Validation(fmt.Sprintf("invalid status: %s (must be queued, running, succeeded, failed, or canceled)", normalizedStatus))
	}
	return limit, offset, normalizedStatus, nil
}

func loadJobListViaAPI(ctx context.Context, port string, limit, offset int, status string, failuresOnly bool) (*api.JobListResponse, error) {
	endpoint := fmt.Sprintf("http://localhost:%s/v1/jobs?limit=%d&offset=%d", port, limit, offset)
	if failuresOnly {
		endpoint = fmt.Sprintf("http://localhost:%s/v1/jobs/failures?limit=%d&offset=%d", port, limit, offset)
	} else if status != "" {
		endpoint += "&status=" + url.QueryEscape(status)
	}
	return doJSONRequest[api.JobListResponse](ctx, http.MethodGet, endpoint)
}

func printJobList(response *api.JobListResponse, failuresOnly bool) {
	if response == nil || len(response.Jobs) == 0 {
		if failuresOnly {
			fmt.Println("No failed runs found.")
			return
		}
		fmt.Println("No runs found.")
		return
	}

	label := "Runs"
	if failuresOnly {
		label = "Failed Runs"
	}
	fmt.Printf("%s (showing %d of %d, limit=%d, offset=%d):\n", label, len(response.Jobs), response.Total, response.Limit, response.Offset)
	fmt.Printf("%-36s %-10s %-10s %-8s %-8s %-8s %-18s %s\n", "ID", "KIND", "STATUS", "WAIT", "RUN", "TOTAL", "QUEUE", "FAILURE")
	for _, job := range response.Jobs {
		queue := "-"
		if job.Run.Queue != nil {
			queue = fmt.Sprintf("%d/%d (%d%%)", job.Run.Queue.Index, job.Run.Queue.Total, job.Run.Queue.Percent)
		}
		failure := "-"
		if job.Run.Failure != nil {
			failure = fmt.Sprintf("%s: %s", job.Run.Failure.Category, job.Run.Failure.Summary)
		}
		fmt.Printf("%-36s %-10s %-10s %-8s %-8s %-8s %-18s %s\n",
			job.ID,
			job.Kind,
			job.Status,
			humanDuration(job.Run.WaitMs),
			humanDuration(job.Run.RunMs),
			humanDuration(job.Run.TotalMs),
			queue,
			failure,
		)
	}
}

func humanDuration(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	seconds := float64(ms) / 1000
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	minutes := int(seconds) / 60
	return fmt.Sprintf("%dm%02ds", minutes, int(seconds)%60)
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
	endpoint := fmt.Sprintf("http://localhost:%s/v1/jobs/%s", port, url.PathEscape(jobID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
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
			return apperrors.Validation(fmt.Sprintf("server returned %d: %s", resp.StatusCode, errResp.Error))
		}
		return apperrors.Validation(fmt.Sprintf("server returned %d", resp.StatusCode))
	}

	return nil
}

func doJSONRequest[T any](ctx context.Context, method string, endpoint string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && strings.TrimSpace(errResp.Error) != "" {
			return nil, apperrors.Validation(errResp.Error)
		}
		return nil, apperrors.Validation(fmt.Sprintf("server returned %d", resp.StatusCode))
	}

	var payload T
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
