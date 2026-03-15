// Package scrape contains CLI commands for content extraction workflows (scrape/crawl/research).
//
// It does NOT implement scraping/crawling itself (that lives under internal/jobs, internal/scrape, internal/crawl, etc).
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
	"github.com/fitchmultz/spartan-scraper/internal/extract"
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
  spartan scrape --url https://example.com --ai-extract --ai-prompt "extract all product names and prices"
  spartan scrape --url https://example.com --ai-extract --ai-mode schema_guided --ai-schema '{"title":"Example","price":"$19.99"}' --ai-fields "title,price,rating"

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

	aiExtractOptions, err := common.BuildAIExtractOptions(cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if aiExtractOptions != nil {
		extractOpts.AI = aiExtractOptions
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

	manager, err := common.InitJobManager(ctx, cfg, st)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	authOptions, err := common.ResolveAuthFromCommonFlags(cfg, *url, cf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	device, err := common.ResolveDevicePreset(*cf.Device)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	screenshot, err := common.BuildScreenshotConfig(
		*cf.ScreenshotEnabled,
		*cf.ScreenshotFullPage,
		*cf.ScreenshotFormat,
		*cf.ScreenshotQuality,
		*cf.ScreenshotWidth,
		*cf.ScreenshotHeight,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Parse request body (support @file syntax)
	var body []byte
	if *cf.Body != "" {
		if strings.HasPrefix(*cf.Body, "@") {
			// Read from file
			body, err = os.ReadFile((*cf.Body)[1:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read body file: %v\n", err)
				return 1
			}
		} else {
			body = []byte(*cf.Body)
		}
	}

	networkIntercept := common.BuildNetworkInterceptConfig(
		*cf.InterceptEnabled,
		[]string(cf.InterceptURLPatterns),
		[]string(cf.InterceptResourceTypes),
		*cf.InterceptCaptureRequest,
		*cf.InterceptCaptureResponse,
		*cf.InterceptMaxBodySize,
		*cf.InterceptMaxEntries,
	)

	spec := jobs.JobSpec{
		Kind:             model.KindScrape,
		URL:              *url,
		Method:           *cf.Method,
		Body:             body,
		ContentType:      *cf.ContentType,
		Headless:         *cf.Headless,
		UsePlaywright:    *cf.Playwright,
		AuthProfile:      *cf.ProfileName,
		Auth:             authOptions,
		TimeoutSeconds:   *cf.Timeout,
		Extract:          extractOpts,
		Pipeline:         pipelineOpts,
		Incremental:      *cf.Incremental,
		Screenshot:       screenshot,
		Device:           device,
		NetworkIntercept: networkIntercept,
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
	return common.LoadExtractOptions(template, configPath, validateSchema)
}
