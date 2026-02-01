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
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

const maxBatchSize = 100

// BatchJobRequest represents a single job in a batch.
type BatchJobRequest struct {
	URL         string            `json:"url"`
	Method      string            `json:"method,omitempty"`
	Body        string            `json:"body,omitempty"`
	ContentType string            `json:"contentType,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// BatchScrapeRequest represents a batch scrape request.
type BatchScrapeRequest struct {
	Jobs           []BatchJobRequest       `json:"jobs"`
	Headless       bool                    `json:"headless,omitempty"`
	Playwright     *bool                   `json:"playwright,omitempty"`
	TimeoutSeconds int                     `json:"timeoutSeconds,omitempty"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth,omitempty"`
	Extract        *extract.ExtractOptions `json:"extract,omitempty"`
	Pipeline       *pipeline.Options       `json:"pipeline,omitempty"`
	Incremental    *bool                   `json:"incremental,omitempty"`
	Webhook        *model.WebhookConfig    `json:"webhook,omitempty"`
}

// BatchCrawlRequest represents a batch crawl request.
type BatchCrawlRequest struct {
	Jobs           []BatchJobRequest       `json:"jobs"`
	MaxDepth       int                     `json:"maxDepth,omitempty"`
	MaxPages       int                     `json:"maxPages,omitempty"`
	Headless       bool                    `json:"headless,omitempty"`
	Playwright     *bool                   `json:"playwright,omitempty"`
	TimeoutSeconds int                     `json:"timeoutSeconds,omitempty"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth,omitempty"`
	Extract        *extract.ExtractOptions `json:"extract,omitempty"`
	Pipeline       *pipeline.Options       `json:"pipeline,omitempty"`
	Incremental    *bool                   `json:"incremental,omitempty"`
	SitemapURL     string                  `json:"sitemapURL,omitempty"`
	SitemapOnly    *bool                   `json:"sitemapOnly,omitempty"`
	Webhook        *model.WebhookConfig    `json:"webhook,omitempty"`
}

// BatchResearchRequest represents a batch research request.
type BatchResearchRequest struct {
	Jobs           []BatchJobRequest       `json:"jobs"`
	Query          string                  `json:"query"`
	MaxDepth       int                     `json:"maxDepth,omitempty"`
	MaxPages       int                     `json:"maxPages,omitempty"`
	Headless       bool                    `json:"headless,omitempty"`
	Playwright     *bool                   `json:"playwright,omitempty"`
	TimeoutSeconds int                     `json:"timeoutSeconds,omitempty"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth,omitempty"`
	Extract        *extract.ExtractOptions `json:"extract,omitempty"`
	Pipeline       *pipeline.Options       `json:"pipeline,omitempty"`
	Webhook        *model.WebhookConfig    `json:"webhook,omitempty"`
}

// BatchResponse represents a batch creation response.
type BatchResponse struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	JobCount  int       `json:"jobCount"`
	Jobs      []JobInfo `json:"jobs,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// BatchStatusResponse represents batch status with statistics.
type BatchStatusResponse struct {
	ID        string              `json:"id"`
	Kind      string              `json:"kind"`
	Status    string              `json:"status"`
	JobCount  int                 `json:"jobCount"`
	Stats     model.BatchJobStats `json:"stats"`
	Jobs      []JobInfo           `json:"jobs,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

// JobInfo represents a job in the batch response.
type JobInfo struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

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

func runBatchSubmit(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printBatchSubmitHelp()
		return 1
	}

	kind := args[0]
	switch kind {
	case "scrape":
		return runBatchSubmitScrape(ctx, cfg, args[1:])
	case "crawl":
		return runBatchSubmitCrawl(ctx, cfg, args[1:])
	case "research":
		return runBatchSubmitResearch(ctx, cfg, args[1:])
	case "help", "--help", "-h":
		printBatchSubmitHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown batch submit kind: %s\n", kind)
		printBatchSubmitHelp()
		return 1
	}
}

func printBatchSubmitHelp() {
	fmt.Print(`Usage: spartan batch submit <kind> [options]

Kinds:
  scrape       Submit a batch of scrape jobs
  crawl        Submit a batch of crawl jobs
  research     Submit a batch of research jobs

Common Options:
  --file string             Path to CSV or JSON file containing URLs
  --urls string             Comma-separated list of URLs
  --headless                Use headless browser
  --playwright              Use Playwright instead of Chromedp
  --timeout int             Request timeout in seconds (default: from config)
  --auth-profile string     Named auth profile to use
  --wait-completion         Wait for batch completion
  --wait-timeout-secs int   Max wait time in seconds (0 = no timeout)

Scrape-specific Options:
  --extract-template string    Extraction template name
  --extract-validate           Validate extraction against schema
  --method string              HTTP method (GET, POST, PUT, etc.)
  --body string                Request body (use @file to read from file)
  --content-type string        Content-Type header

Crawl-specific Options:
  --max-depth int        Maximum crawl depth (default: 3)
  --max-pages int        Maximum pages to crawl (default: 100)
  --sitemap-url string   Sitemap URL for seed URLs
  --sitemap-only         Only crawl sitemap URLs
  --incremental          Use incremental crawling

Research-specific Options:
  --query string         Research query (required)
  --max-depth int        Maximum research depth (default: 3)
  --max-pages int        Maximum pages to research (default: 100)

File Formats:
  CSV:  url,method,body,contentType (headers optional)
  JSON: [{"url": "...", "method": "GET", ...}]

Examples:
  spartan batch submit scrape --file urls.csv --headless
  spartan batch submit scrape --urls https://a.com,https://b.com --extract-template article
  spartan batch submit crawl --file sites.csv --max-depth 2 --max-pages 50
  spartan batch submit research --file sources.json --query "pricing model" --max-depth 2
`)
}

func runBatchSubmitScrape(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("batch-submit-scrape", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to CSV or JSON file containing URLs")
	urlsList := fs.String("urls", "", "Comma-separated list of URLs")
	waitFlag := fs.Bool("wait-completion", false, "Wait for batch completion")
	waitTimeout := fs.Int("wait-timeout-secs", 0, "Max wait time in seconds (0 = no timeout)")
	cf := common.RegisterCommonFlags(fs, cfg)

	fs.Usage = func() {
		printBatchSubmitHelp()
	}
	_ = fs.Parse(args)

	// Parse jobs from file or URLs
	jobReqs, err := parseBatchJobs(*filePath, *urlsList, *cf.Method, *cf.Body, *cf.ContentType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing batch jobs: %v\n", err)
		return 1
	}

	if len(jobReqs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no URLs provided (use --file or --urls)")
		return 1
	}

	if len(jobReqs) > maxBatchSize {
		fmt.Fprintf(os.Stderr, "Error: batch size %d exceeds maximum of %d\n", len(jobReqs), maxBatchSize)
		return 1
	}

	// Validate URLs
	for i, job := range jobReqs {
		if err := validate.ValidateURL(job.URL); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid URL at index %d: %v\n", i, err)
			return 1
		}
	}

	// Build auth options
	authOptions, err := common.ResolveAuthFromCommonFlags(cfg, jobReqs[0].URL, cf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving auth: %v\n", err)
		return 1
	}

	// Build extract options
	extractOpts := &extract.ExtractOptions{}
	if *cf.ExtractTemplate != "" {
		extractOpts.Template = *cf.ExtractTemplate
	}
	extractOpts.Validate = *cf.ExtractValidate

	// Build pipeline options
	pipelineOpts := &pipeline.Options{
		PreProcessors:  []string(cf.PreProcessors),
		PostProcessors: []string(cf.PostProcessors),
		Transformers:   []string(cf.Transformers),
	}

	// Build request
	req := BatchScrapeRequest{
		Jobs:           jobReqs,
		Headless:       *cf.Headless,
		TimeoutSeconds: *cf.Timeout,
		AuthProfile:    *cf.ProfileName,
		Auth:           &authOptions,
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
	}

	if *cf.Playwright != cfg.UsePlaywright {
		req.Playwright = cf.Playwright
	}
	if *cf.Incremental {
		req.Incremental = cf.Incremental
	}

	// Submit batch
	var resp *BatchResponse
	if isServerRunning(ctx, cfg.Port) {
		resp, err = submitBatchScrapeViaAPI(ctx, cfg.Port, req)
	} else {
		resp, err = submitBatchScrapeDirect(ctx, cfg, req)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting batch: %v\n", err)
		return 1
	}

	fmt.Printf("Batch submitted: %s\n", resp.ID)
	fmt.Printf("Kind: %s, Jobs: %d, Status: %s\n", resp.Kind, resp.JobCount, resp.Status)

	// Wait for completion if requested
	if *waitFlag {
		timeout := time.Duration(*waitTimeout) * time.Second
		return waitForBatch(ctx, cfg, resp.ID, timeout)
	}

	return 0
}

func runBatchSubmitCrawl(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("batch-submit-crawl", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to CSV or JSON file containing URLs")
	urlsList := fs.String("urls", "", "Comma-separated list of URLs")
	maxDepth := fs.Int("max-depth", 3, "Maximum crawl depth")
	maxPages := fs.Int("max-pages", 100, "Maximum pages to crawl")
	sitemapURL := fs.String("sitemap-url", "", "Sitemap URL for seed URLs")
	sitemapOnly := fs.Bool("sitemap-only", false, "Only crawl sitemap URLs")
	waitFlag := fs.Bool("wait-completion", false, "Wait for batch completion")
	waitTimeout := fs.Int("wait-timeout-secs", 0, "Max wait time in seconds (0 = no timeout)")
	cf := common.RegisterCommonFlags(fs, cfg)

	fs.Usage = func() {
		printBatchSubmitHelp()
	}
	_ = fs.Parse(args)

	// Parse jobs from file or URLs
	jobReqs, err := parseBatchJobs(*filePath, *urlsList, "GET", "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing batch jobs: %v\n", err)
		return 1
	}

	if len(jobReqs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no URLs provided (use --file or --urls)")
		return 1
	}

	if len(jobReqs) > maxBatchSize {
		fmt.Fprintf(os.Stderr, "Error: batch size %d exceeds maximum of %d\n", len(jobReqs), maxBatchSize)
		return 1
	}

	// Validate URLs
	for i, job := range jobReqs {
		if err := validate.ValidateURL(job.URL); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid URL at index %d: %v\n", i, err)
			return 1
		}
	}

	// Build auth options
	authOptions, err := common.ResolveAuthFromCommonFlags(cfg, jobReqs[0].URL, cf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving auth: %v\n", err)
		return 1
	}

	// Build extract options
	extractOpts := &extract.ExtractOptions{}
	if *cf.ExtractTemplate != "" {
		extractOpts.Template = *cf.ExtractTemplate
	}
	extractOpts.Validate = *cf.ExtractValidate

	// Build pipeline options
	pipelineOpts := &pipeline.Options{
		PreProcessors:  []string(cf.PreProcessors),
		PostProcessors: []string(cf.PostProcessors),
		Transformers:   []string(cf.Transformers),
	}

	// Build request
	req := BatchCrawlRequest{
		Jobs:           jobReqs,
		MaxDepth:       *maxDepth,
		MaxPages:       *maxPages,
		Headless:       *cf.Headless,
		TimeoutSeconds: *cf.Timeout,
		AuthProfile:    *cf.ProfileName,
		SitemapURL:     *sitemapURL,
		Auth:           &authOptions,
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
	}

	if *cf.Playwright != cfg.UsePlaywright {
		req.Playwright = cf.Playwright
	}
	if *cf.Incremental {
		req.Incremental = cf.Incremental
	}
	if *sitemapOnly {
		req.SitemapOnly = sitemapOnly
	}

	// Submit batch
	var resp *BatchResponse
	if isServerRunning(ctx, cfg.Port) {
		resp, err = submitBatchCrawlViaAPI(ctx, cfg.Port, req)
	} else {
		resp, err = submitBatchCrawlDirect(ctx, cfg, req)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting batch: %v\n", err)
		return 1
	}

	fmt.Printf("Batch submitted: %s\n", resp.ID)
	fmt.Printf("Kind: %s, Jobs: %d, Status: %s\n", resp.Kind, resp.JobCount, resp.Status)

	// Wait for completion if requested
	if *waitFlag {
		timeout := time.Duration(*waitTimeout) * time.Second
		return waitForBatch(ctx, cfg, resp.ID, timeout)
	}

	return 0
}

func runBatchSubmitResearch(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("batch-submit-research", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to CSV or JSON file containing URLs")
	urlsList := fs.String("urls", "", "Comma-separated list of URLs")
	query := fs.String("query", "", "Research query (required)")
	maxDepth := fs.Int("max-depth", 3, "Maximum research depth")
	maxPages := fs.Int("max-pages", 100, "Maximum pages to research")
	waitFlag := fs.Bool("wait-completion", false, "Wait for batch completion")
	waitTimeout := fs.Int("wait-timeout-secs", 0, "Max wait time in seconds (0 = no timeout)")
	cf := common.RegisterCommonFlags(fs, cfg)

	fs.Usage = func() {
		printBatchSubmitHelp()
	}
	_ = fs.Parse(args)

	if *query == "" {
		fmt.Fprintln(os.Stderr, "Error: --query is required for research jobs")
		return 1
	}

	// Parse jobs from file or URLs
	jobReqs, err := parseBatchJobs(*filePath, *urlsList, "GET", "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing batch jobs: %v\n", err)
		return 1
	}

	if len(jobReqs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no URLs provided (use --file or --urls)")
		return 1
	}

	if len(jobReqs) > maxBatchSize {
		fmt.Fprintf(os.Stderr, "Error: batch size %d exceeds maximum of %d\n", len(jobReqs), maxBatchSize)
		return 1
	}

	// Validate URLs
	for i, job := range jobReqs {
		if err := validate.ValidateURL(job.URL); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid URL at index %d: %v\n", i, err)
			return 1
		}
	}

	// Build auth options
	authOptions, err := common.ResolveAuthFromCommonFlags(cfg, jobReqs[0].URL, cf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving auth: %v\n", err)
		return 1
	}

	// Build extract options
	extractOpts := &extract.ExtractOptions{}
	if *cf.ExtractTemplate != "" {
		extractOpts.Template = *cf.ExtractTemplate
	}
	extractOpts.Validate = *cf.ExtractValidate

	// Build pipeline options
	pipelineOpts := &pipeline.Options{
		PreProcessors:  []string(cf.PreProcessors),
		PostProcessors: []string(cf.PostProcessors),
		Transformers:   []string(cf.Transformers),
	}

	// Build request
	req := BatchResearchRequest{
		Jobs:           jobReqs,
		Query:          *query,
		MaxDepth:       *maxDepth,
		MaxPages:       *maxPages,
		Headless:       *cf.Headless,
		TimeoutSeconds: *cf.Timeout,
		AuthProfile:    *cf.ProfileName,
		Auth:           &authOptions,
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
	}

	if *cf.Playwright != cfg.UsePlaywright {
		req.Playwright = cf.Playwright
	}

	// Submit batch
	var resp *BatchResponse
	if isServerRunning(ctx, cfg.Port) {
		resp, err = submitBatchResearchViaAPI(ctx, cfg.Port, req)
	} else {
		resp, err = submitBatchResearchDirect(ctx, cfg, req)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting batch: %v\n", err)
		return 1
	}

	fmt.Printf("Batch submitted: %s\n", resp.ID)
	fmt.Printf("Kind: %s, Jobs: %d, Status: %s\n", resp.Kind, resp.JobCount, resp.Status)

	// Wait for completion if requested
	if *waitFlag {
		timeout := time.Duration(*waitTimeout) * time.Second
		return waitForBatch(ctx, cfg, resp.ID, timeout)
	}

	return 0
}

func parseBatchJobs(filePath, urlsList, method, body, contentType string) ([]BatchJobRequest, error) {
	if filePath != "" {
		return parseBatchJobsFromFile(filePath)
	}

	if urlsList != "" {
		urls := strings.Split(urlsList, ",")
		jobs := make([]BatchJobRequest, 0, len(urls))
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url != "" {
				m := method
				if m == "" {
					m = "GET"
				}
				jobs = append(jobs, BatchJobRequest{
					URL:         url,
					Method:      m,
					Body:        body,
					ContentType: contentType,
				})
			}
		}
		return jobs, nil
	}

	return nil, nil
}

func parseBatchJobsFromFile(filePath string) ([]BatchJobRequest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try JSON first
	if strings.HasSuffix(filePath, ".json") || looksLikeJSON(data) {
		var jobs []BatchJobRequest
		if err := json.Unmarshal(data, &jobs); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		return jobs, nil
	}

	// Try CSV
	if strings.HasSuffix(filePath, ".csv") {
		return parseBatchJobsFromCSV(data)
	}

	return nil, fmt.Errorf("unsupported file format (use .json or .csv)")
}

func looksLikeJSON(data []byte) bool {
	data = []byte(strings.TrimSpace(string(data)))
	return len(data) > 0 && (data[0] == '[' || data[0] == '{')
}

func parseBatchJobsFromCSV(data []byte) ([]BatchJobRequest, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Detect if first row is headers
	var urlIdx, methodIdx, bodyIdx, contentTypeIdx int = -1, -1, -1, -1
	firstRow := records[0]

	// Check if first row looks like headers
	hasHeaders := false
	for _, col := range firstRow {
		lower := strings.ToLower(strings.TrimSpace(col))
		if lower == "url" || lower == "method" || lower == "body" || lower == "contenttype" || lower == "content-type" {
			hasHeaders = true
			break
		}
	}

	if hasHeaders {
		for i, col := range firstRow {
			switch strings.ToLower(strings.TrimSpace(col)) {
			case "url":
				urlIdx = i
			case "method":
				methodIdx = i
			case "body":
				bodyIdx = i
			case "contenttype", "content-type":
				contentTypeIdx = i
			}
		}
		records = records[1:]
	} else {
		// Default column order: url, method, body, contentType
		urlIdx = 0
		if len(firstRow) > 1 {
			methodIdx = 1
		}
		if len(firstRow) > 2 {
			bodyIdx = 2
		}
		if len(firstRow) > 3 {
			contentTypeIdx = 3
		}
	}

	if urlIdx < 0 {
		return nil, fmt.Errorf("CSV must have a 'url' column")
	}

	jobs := make([]BatchJobRequest, 0, len(records))
	for _, row := range records {
		if len(row) <= urlIdx {
			continue
		}
		url := strings.TrimSpace(row[urlIdx])
		if url == "" {
			continue
		}

		job := BatchJobRequest{URL: url, Method: "GET"}

		if methodIdx >= 0 && methodIdx < len(row) {
			if m := strings.TrimSpace(row[methodIdx]); m != "" {
				job.Method = strings.ToUpper(m)
			}
		}
		if bodyIdx >= 0 && bodyIdx < len(row) {
			job.Body = strings.TrimSpace(row[bodyIdx])
		}
		if contentTypeIdx >= 0 && contentTypeIdx < len(row) {
			job.ContentType = strings.TrimSpace(row[contentTypeIdx])
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

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

	if err := cancelBatch(ctx, cfg, batchID); err != nil {
		fmt.Fprintf(os.Stderr, "Error canceling batch: %v\n", err)
		return 1
	}

	fmt.Printf("Batch %s canceled successfully\n", batchID)
	return 0
}

func printBatchStatus(status *BatchStatusResponse) {
	fmt.Printf("Batch: %s\n", status.ID)
	fmt.Printf("Kind: %s\n", status.Kind)
	fmt.Printf("Status: %s\n", status.Status)
	fmt.Printf("Jobs: %d total\n", status.JobCount)
	fmt.Printf("Stats: %d queued, %d running, %d succeeded, %d failed, %d canceled\n",
		status.Stats.Queued, status.Stats.Running, status.Stats.Succeeded,
		status.Stats.Failed, status.Stats.Canceled)

	if len(status.Jobs) > 0 {
		fmt.Println("\nJobs:")
		for _, job := range status.Jobs {
			fmt.Printf("  %s (%s): %s\n", job.ID, job.Kind, job.Status)
		}
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
		completed := status.Stats.Succeeded + status.Stats.Failed + status.Stats.Canceled
		fmt.Printf("\rStatus: %s | Progress: %d/%d (%d%%)",
			status.Status,
			completed,
			status.JobCount,
			(completed*100)/status.JobCount,
		)

		// Check if batch is in terminal state
		if isTerminalStatus(status.Status) {
			fmt.Println() // New line after progress
			fmt.Printf("\nBatch %s finished with status: %s\n", batchID, status.Status)
			fmt.Printf("Final stats: %d succeeded, %d failed, %d canceled\n",
				status.Stats.Succeeded, status.Stats.Failed, status.Stats.Canceled)
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
		completed := status.Stats.Succeeded + status.Stats.Failed + status.Stats.Canceled
		fmt.Printf("\rProgress: %d/%d jobs complete (%d%%) - Status: %s",
			completed, status.JobCount, (completed*100)/status.JobCount, status.Status)

		// Check if batch is in terminal state
		if isTerminalStatus(status.Status) {
			fmt.Println() // New line after progress
			fmt.Printf("\nBatch %s finished with status: %s\n", batchID, status.Status)
			fmt.Printf("Results: %d succeeded, %d failed, %d canceled\n",
				status.Stats.Succeeded, status.Stats.Failed, status.Stats.Canceled)
			if status.Status == "completed" {
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

// Direct submission functions (when server is not running)

func submitBatchScrapeDirect(ctx context.Context, cfg config.Config, req BatchScrapeRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)

	// Build job specs
	specs := make([]jobs.JobSpec, len(req.Jobs))
	for i, job := range req.Jobs {
		specs[i] = jobs.JobSpec{
			Kind:           model.KindScrape,
			URL:            job.URL,
			Method:         job.Method,
			Body:           []byte(job.Body),
			ContentType:    job.ContentType,
			Headless:       req.Headless,
			UsePlaywright:  req.Playwright != nil && *req.Playwright,
			TimeoutSeconds: req.TimeoutSeconds,
			Auth:           *req.Auth,
			Extract:        *req.Extract,
			Pipeline:       *req.Pipeline,
			Incremental:    req.Incremental != nil && *req.Incremental,
		}
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindScrape, specs, batchID)
	if err != nil {
		return nil, err
	}

	// Enqueue all jobs
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	return &BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindScrape),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		CreatedAt: time.Now(),
	}, nil
}

func submitBatchCrawlDirect(ctx context.Context, cfg config.Config, req BatchCrawlRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)

	// Build job specs
	specs := make([]jobs.JobSpec, len(req.Jobs))
	for i, job := range req.Jobs {
		specs[i] = jobs.JobSpec{
			Kind:           model.KindCrawl,
			URL:            job.URL,
			MaxDepth:       req.MaxDepth,
			MaxPages:       req.MaxPages,
			Headless:       req.Headless,
			UsePlaywright:  req.Playwright != nil && *req.Playwright,
			TimeoutSeconds: req.TimeoutSeconds,
			SitemapURL:     req.SitemapURL,
			SitemapOnly:    req.SitemapOnly != nil && *req.SitemapOnly,
			Auth:           *req.Auth,
			Extract:        *req.Extract,
			Pipeline:       *req.Pipeline,
			Incremental:    req.Incremental != nil && *req.Incremental,
		}
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindCrawl, specs, batchID)
	if err != nil {
		return nil, err
	}

	// Enqueue all jobs
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	return &BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindCrawl),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		CreatedAt: time.Now(),
	}, nil
}

func submitBatchResearchDirect(ctx context.Context, cfg config.Config, req BatchResearchRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)

	// Collect URLs from jobs
	urls := make([]string, len(req.Jobs))
	for i, job := range req.Jobs {
		urls[i] = job.URL
	}

	// Research jobs only need one job with all URLs
	spec := jobs.JobSpec{
		Kind:           model.KindResearch,
		Query:          req.Query,
		URLs:           urls,
		MaxDepth:       req.MaxDepth,
		MaxPages:       req.MaxPages,
		Headless:       req.Headless,
		UsePlaywright:  req.Playwright != nil && *req.Playwright,
		TimeoutSeconds: req.TimeoutSeconds,
		Auth:           *req.Auth,
		Extract:        *req.Extract,
		Pipeline:       *req.Pipeline,
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindResearch, []jobs.JobSpec{spec}, batchID)
	if err != nil {
		return nil, err
	}

	// Enqueue all jobs
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	return &BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindResearch),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		CreatedAt: time.Now(),
	}, nil
}

// API submission functions (when server is running)

func submitBatchScrapeViaAPI(ctx context.Context, port string, req BatchScrapeRequest) (*BatchResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/scrape", port)
	return submitBatchViaAPI(ctx, url, req)
}

func submitBatchCrawlViaAPI(ctx context.Context, port string, req BatchCrawlRequest) (*BatchResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/crawl", port)
	return submitBatchViaAPI(ctx, url, req)
}

func submitBatchResearchViaAPI(ctx context.Context, port string, req BatchResearchRequest) (*BatchResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/research", port)
	return submitBatchViaAPI(ctx, url, req)
}

func submitBatchViaAPI(ctx context.Context, url string, req interface{}) (*BatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result BatchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func getBatchStatus(ctx context.Context, cfg config.Config, batchID string, includeJobs bool) (*BatchStatusResponse, error) {
	if isServerRunning(ctx, cfg.Port) {
		return getBatchStatusViaAPI(ctx, cfg.Port, batchID, includeJobs)
	}
	return getBatchStatusDirect(ctx, cfg, batchID)
}

func getBatchStatusDirect(ctx context.Context, cfg config.Config, batchID string) (*BatchStatusResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	batch, err := st.GetBatch(ctx, batchID)
	if err != nil {
		return nil, err
	}

	stats, err := st.CountJobsByBatchAndStatus(ctx, batchID)
	if err != nil {
		return nil, err
	}

	// Calculate current batch status
	batch.Status = model.CalculateBatchStatus(stats, batch.JobCount)

	return &BatchStatusResponse{
		ID:        batch.ID,
		Kind:      string(batch.Kind),
		Status:    string(batch.Status),
		JobCount:  batch.JobCount,
		Stats:     stats,
		CreatedAt: batch.CreatedAt,
		UpdatedAt: batch.UpdatedAt,
	}, nil
}

func getBatchStatusViaAPI(ctx context.Context, port, batchID string, includeJobs bool) (*BatchStatusResponse, error) {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/%s", port, batchID)
	if includeJobs {
		url += "?include_jobs=true"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("batch %s not found", batchID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result BatchStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func cancelBatch(ctx context.Context, cfg config.Config, batchID string) error {
	if isServerRunning(ctx, cfg.Port) {
		return cancelBatchViaAPI(ctx, cfg.Port, batchID)
	}
	return cancelBatchDirect(ctx, cfg, batchID)
}

func cancelBatchDirect(ctx context.Context, cfg config.Config, batchID string) error {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)
	_, err = manager.CancelBatch(ctx, batchID)
	return err
}

func cancelBatchViaAPI(ctx context.Context, port, batchID string) error {
	url := fmt.Sprintf("http://localhost:%s/v1/jobs/batch/%s", port, batchID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("batch %s not found", batchID)
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}
