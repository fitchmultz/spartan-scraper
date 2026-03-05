// Package manage contains job management CLI command implementations.
//
// This file implements the traffic replay command for replaying captured
// network requests from job results.
package manage

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// RunReplay executes the replay command.
func RunReplay(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)

	var (
		jobID        = fs.String("job-id", "", "Job ID to replay traffic from (required)")
		targetURL    = fs.String("target-url", "", "Target base URL to replay against (required)")
		filterURL    = fs.String("filter-url", "", "URL pattern filter (repeatable, comma-separated)")
		filterMethod = fs.String("filter-method", "", "HTTP method filter (repeatable, comma-separated)")
		compare      = fs.Bool("compare", false, "Enable response comparison")
		outputFormat = fs.String("output", "table", "Output format: table, json")
		timeout      = fs.Int("timeout", 30, "Per-request timeout in seconds")
		addHeader    = fs.String("header", "", "Add/replace header (format: 'Name: Value', repeatable, comma-separated)")
		removeHeader = fs.String("remove-header", "", "Remove header (repeatable, comma-separated)")
	)

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: spartan replay [options]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Replay captured network traffic from a job against a target URL.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  spartan replay --job-id abc123 --target-url https://staging.example.com")
		fmt.Fprintln(os.Stderr, "  spartan replay --job-id abc123 --target-url https://localhost:8080 --compare")
		fmt.Fprintln(os.Stderr, "  spartan replay --job-id abc123 --target-url https://api.example.com --filter-method GET,POST")
		fmt.Fprintln(os.Stderr, "  spartan replay --job-id abc123 --target-url https://api.example.com --header 'Authorization: Bearer token'")
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Validate required flags
	if *jobID == "" {
		fmt.Fprintln(os.Stderr, "Error: --job-id is required")
		fs.Usage()
		return 1
	}

	if *targetURL == "" {
		fmt.Fprintln(os.Stderr, "Error: --target-url is required")
		fs.Usage()
		return 1
	}

	// Build request
	req := api.TrafficReplayRequest{
		JobID:            *jobID,
		TargetBaseURL:    *targetURL,
		CompareResponses: *compare,
		Timeout:          *timeout,
	}

	// Parse filters
	if *filterURL != "" || *filterMethod != "" {
		req.Filter = &api.TrafficReplayFilter{}

		if *filterURL != "" {
			req.Filter.URLPatterns = splitCommaSeparated(*filterURL)
		}

		if *filterMethod != "" {
			req.Filter.Methods = splitCommaSeparated(*filterMethod)
		}
	}

	// Parse modifications
	if *addHeader != "" || *removeHeader != "" {
		req.Modifications = &api.TrafficModifications{
			Headers:       make(map[string]string),
			RemoveHeaders: []string{},
		}

		if *addHeader != "" {
			headers := splitCommaSeparated(*addHeader)
			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) == 2 {
					req.Modifications.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
		}

		if *removeHeader != "" {
			req.Modifications.RemoveHeaders = splitCommaSeparated(*removeHeader)
		}
	}

	// Send request
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/replay/%s", cfg.Port, *jobID)

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal request: %v\n", err)
		return 1
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create request: %v\n", err)
		return 1
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to send request: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			fmt.Fprintf(os.Stderr, "Error: HTTP %d\n", resp.StatusCode)
			return 1
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", errResp.Error)
		return 1
	}

	var result api.TrafficReplayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to decode response: %v\n", err)
		return 1
	}

	// Output results
	switch *outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(result)
	case "table":
		printReplayResults(result)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown output format: %s\n", *outputFormat)
		return 1
	}

	// Return non-zero if any requests failed
	if result.Failed > 0 {
		return 1
	}
	return 0
}

// splitCommaSeparated splits a comma-separated string into a slice.
func splitCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// printReplayResults prints replay results in table format.
func printReplayResults(result api.TrafficReplayResponse) {
	fmt.Printf("\nTraffic Replay Results for Job: %s\n", result.JobID)
	fmt.Printf("Duration: %dms\n\n", result.Duration)

	fmt.Printf("Summary:\n")
	fmt.Printf("  Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("  Successful:     %d\n", result.Successful)
	fmt.Printf("  Failed:         %d\n", result.Failed)

	if result.Comparison != nil {
		fmt.Printf("\nComparison:\n")
		fmt.Printf("  Compared: %d\n", result.Comparison.TotalCompared)
		fmt.Printf("  Matches:  %d\n", result.Comparison.Matches)
		fmt.Printf("  Mismatches: %d\n", result.Comparison.Mismatches)
	}

	fmt.Printf("\nRequest Details:\n")
	fmt.Printf("%-6s %-50s %-10s %-10s %-10s\n", "Method", "URL", "Status", "Size", "Duration")
	fmt.Println(strings.Repeat("-", 90))

	for _, r := range result.Results {
		status := "ERR"
		size := "-"
		duration := fmt.Sprintf("%dms", r.Duration)

		if r.Error == "" {
			status = fmt.Sprintf("%d", r.ReplayedResponse.Status)
			size = formatBytes(int64(r.ReplayedResponse.BodySize))
		} else {
			duration = "ERR"
		}

		url := truncateString(r.ReplayedRequest.URL, 50)
		fmt.Printf("%-6s %-50s %-10s %-10s %-10s\n", r.ReplayedRequest.Method, url, status, size, duration)

		if r.Error != "" {
			fmt.Printf("  Error: %s\n", r.Error)
		}
	}

	// Print differences if comparison was enabled
	if result.Comparison != nil && len(result.Comparison.Differences) > 0 {
		fmt.Printf("\nDifferences:\n")
		for _, diff := range result.Comparison.Differences {
			fmt.Printf("\n  Request: %s\n", diff.RequestID)
			fmt.Printf("  URL: %s\n", diff.URL)

			if diff.StatusDiff != nil {
				fmt.Printf("  Status: %d -> %d\n", diff.StatusDiff.Original, diff.StatusDiff.Replayed)
			}

			if len(diff.HeaderDiffs) > 0 {
				fmt.Printf("  Header Differences:\n")
				for _, hd := range diff.HeaderDiffs {
					fmt.Printf("    %s: '%s' -> '%s'\n", hd.Name, hd.Original, hd.Replayed)
				}
			}

			if diff.BodyDiff != nil {
				fmt.Printf("  Body: %d bytes -> %d bytes\n", diff.BodyDiff.OriginalSize, diff.BodyDiff.ReplayedSize)
				if diff.BodyDiff.Preview != "" {
					fmt.Printf("  Preview: %s\n", diff.BodyDiff.Preview)
				}
			}
		}
	}

	fmt.Println()
}

// truncateString truncates a string to max length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
