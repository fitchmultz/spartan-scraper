// Package scrape contains research CLI command wiring.
//
// It does NOT implement research logic; it only translates CLI flags into jobs.
package scrape

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"spartan-scraper/internal/cli/common"
	"spartan-scraper/internal/config"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/store"
	"spartan-scraper/internal/validate"
)

func RunResearch(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("research", flag.ExitOnError)
	query := fs.String("query", "", "Research query")
	urls := fs.String("urls", "", "Comma-separated list of URLs to research")
	maxDepth := fs.Int("max-depth", 2, "Max crawl depth per URL (0 for single-page scrape)")
	maxPages := fs.Int("max-pages", 200, "Max pages to crawl per URL")
	cf := common.RegisterCommonFlags(fs, cfg)

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan research --query <query> --urls <url1,url2,...> [options]

Examples:
  spartan research --query "pricing model" --urls https://example.com,https://example.com/docs
  spartan research --query "login flow" --urls https://example.com --headless --wait --out ./out/research.jsonl
  spartan research --query "pricing model" --urls https://example.com --transformer json-clean

Options:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if *query == "" || *urls == "" {
		fmt.Fprintln(os.Stderr, "--query and --urls are required")
		return 1
	}

	urlList := common.SplitCSV(*urls)

	validator := validate.ResearchRequestValidator{
		Query:       *query,
		URLs:        urlList,
		MaxDepth:    *maxDepth,
		MaxPages:    *maxPages,
		Timeout:     *cf.Timeout,
		AuthProfile: *cf.ProfileName,
	}
	if err := validator.Validate(); err != nil {
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

	// Resolve auth using first URL as base (validator ensures urlList non-empty).
	authOptions, err := common.ResolveAuthFromCommonFlags(cfg, urlList[0], cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	spec := jobs.JobSpec{
		Kind:           model.KindResearch,
		Query:          *query,
		URLs:           urlList,
		MaxDepth:       *maxDepth,
		MaxPages:       *maxPages,
		Headless:       *cf.Headless,
		UsePlaywright:  *cf.Playwright,
		Auth:           authOptions,
		TimeoutSeconds: *cf.Timeout,
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
		Incremental:    *cf.Incremental,
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
