// Package scrape contains crawl CLI command wiring.
//
// It does NOT implement crawling itself; it only translates CLI flags into jobs.
package scrape

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func RunCrawl(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("crawl", flag.ExitOnError)
	url := fs.String("url", "", "Root URL to crawl")
	maxDepth := fs.Int("max-depth", 2, "Max crawl depth")
	maxPages := fs.Int("max-pages", 200, "Max pages to crawl")
	sitemapURL := fs.String("sitemap-url", "", "Optional URL to sitemap.xml for URL discovery")
	sitemapOnly := fs.Bool("sitemap-only", false, "Only crawl URLs from sitemap, not the root URL")
	include := fs.String("include", "", "Comma-separated URL path patterns to include (glob syntax, e.g., /blog/**,/products/*)")
	exclude := fs.String("exclude", "", "Comma-separated URL path patterns to exclude (glob syntax, e.g., /admin/*,/api/**)")
	cf := common.RegisterCommonFlags(fs, cfg)

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan crawl --url <url> [options]

Examples:
  spartan crawl --url https://example.com --max-depth 2 --max-pages 200
  spartan crawl --url https://example.com --headless --wait --out ./out/site.jsonl
  spartan crawl --url https://example.com --pre-processor redact --transformer json-clean
  spartan crawl --url https://example.com --sitemap-url https://example.com/sitemap.xml
  spartan crawl --url https://example.com --sitemap-url https://example.com/sitemap.xml --sitemap-only
  spartan crawl --url https://example.com --include "/blog/**,/products/*"
  spartan crawl --url https://example.com --exclude "/admin/*,/api/**"

Options:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if *url == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		return 1
	}

	opts := validate.JobValidationOpts{
		URL:         *url,
		MaxDepth:    *maxDepth,
		MaxPages:    *maxPages,
		Timeout:     *cf.Timeout,
		AuthProfile: *cf.ProfileName,
	}
	if err := validate.ValidateJob(opts, model.KindCrawl); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	extractOpts, err := loadExtractOptions(*cf.ExtractTemplate, *cf.ExtractConfig, *cf.ExtractValidate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pipelineOpts := pipeline.Options{
		PreProcessors:  []string(cf.PreProcessors),
		PostProcessors: []string(cf.PostProcessors),
		Transformers:   []string(cf.Transformers),
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)

	authOptions, err := common.ResolveAuthFromCommonFlags(cfg, *url, cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	spec := jobs.JobSpec{
		Kind:            model.KindCrawl,
		URL:             *url,
		MaxDepth:        *maxDepth,
		MaxPages:        *maxPages,
		Headless:        *cf.Headless,
		UsePlaywright:   *cf.Playwright,
		Auth:            authOptions,
		TimeoutSeconds:  *cf.Timeout,
		Extract:         extractOpts,
		Pipeline:        pipelineOpts,
		Incremental:     *cf.Incremental,
		SitemapURL:      *sitemapURL,
		SitemapOnly:     *sitemapOnly,
		IncludePatterns: parsePatternList(*include),
		ExcludePatterns: parsePatternList(*exclude),
	}
	job, err := manager.CreateJob(ctx, spec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	waitTimeout := time.Duration(*cf.WaitTimeout) * time.Second
	wait := *cf.Wait || *cf.Out != ""
	return common.HandleJobResult(ctx, st, job, wait, waitTimeout, *cf.Out)
}

// parsePatternList parses a comma-separated string into a slice of trimmed strings.
// Returns nil if the input is empty.
func parsePatternList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
