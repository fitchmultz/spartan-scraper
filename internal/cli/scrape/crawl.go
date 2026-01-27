// Package scrape contains crawl CLI command wiring.
//
// It does NOT implement crawling itself; it only translates CLI flags into jobs.
package scrape

import (
	"context"
	"flag"
	"fmt"
	"os"
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
	cf := common.RegisterCommonFlags(fs, cfg)

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan crawl --url <url> [options]

Examples:
  spartan crawl --url https://example.com --max-depth 2 --max-pages 200
  spartan crawl --url https://example.com --headless --wait --out ./out/site.jsonl
  spartan crawl --url https://example.com --pre-processor redact --transformer json-clean

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
		Kind:           model.KindCrawl,
		URL:            *url,
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
