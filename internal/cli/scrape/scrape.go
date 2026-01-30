// Package scrape contains CLI commands for content extraction workflows (scrape/crawl/research).
//
// It does NOT implement scraping/crawling itself (that lives under internal/jobs, internal/scrape, internal/crawl, etc).
package scrape

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
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

func RunScrape(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("scrape", flag.ExitOnError)
	url := fs.String("url", "", "URL to scrape")
	cf := common.RegisterCommonFlags(fs, cfg)

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan scrape --url <url> [options]

Examples:
  spartan scrape --url https://example.com
  spartan scrape --url https://example.com --headless --wait --out ./out/page.json
  spartan scrape --url https://example.com --headless --playwright --wait --out ./out/page.json
  spartan scrape --url https://example.com/dashboard --headless --login-url https://example.com/login \
    --login-user-selector '#email' --login-pass-selector '#password' --login-submit-selector 'button[type=submit]' \
    --login-user you@example.com --login-pass '***'

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
		Timeout:     *cf.Timeout,
		AuthProfile: *cf.ProfileName,
	}
	if err := validate.ValidateJob(opts, model.KindScrape); err != nil {
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

	// Resolve device preset if specified
	var device *fetch.DeviceEmulation
	if *cf.Device != "" {
		device = fetch.GetDevicePreset(*cf.Device)
		if device == nil {
			fmt.Fprintf(os.Stderr, "Unknown device preset: %s\n", *cf.Device)
			return 1
		}
	}

	spec := jobs.JobSpec{
		Kind:           model.KindScrape,
		URL:            *url,
		Headless:       *cf.Headless,
		UsePlaywright:  *cf.Playwright,
		Auth:           authOptions,
		TimeoutSeconds: *cf.Timeout,
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
		Incremental:    *cf.Incremental,
		Device:         device,
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

func loadExtractOptions(template, configPath string, validateSchema bool) (extract.ExtractOptions, error) {
	opts := extract.ExtractOptions{
		Template: template,
		Validate: validateSchema,
	}
	if configPath == "" {
		return opts, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return extract.ExtractOptions{}, err
	}
	var tmpl extract.Template
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return extract.ExtractOptions{}, fmt.Errorf("invalid template JSON: %w", err)
	}
	opts.Inline = &tmpl
	return opts, nil
}
