// Package batch provides CLI commands for batch job operations.
//
// This file contains submission command handlers for batch operations.
//
// Responsibilities:
// - Run batch submit commands (scrape, crawl, research)
// - Parse command-line flags for batch submission
// - Build and validate batch requests
//
// Does NOT handle:
// - File parsing (delegates to parse.go)
// - API calls (delegates to api.go)
// - Direct execution (delegates to direct.go)
package batch

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

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
