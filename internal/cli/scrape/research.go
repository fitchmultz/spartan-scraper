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

	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func RunResearch(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("research", flag.ExitOnError)
	query := fs.String("query", "", "Research query")
	urls := fs.String("urls", "", "Comma-separated list of URLs to research")
	maxDepth := fs.Int("max-depth", 2, "Max crawl depth per URL (0 for single-page scrape)")
	maxPages := fs.Int("max-pages", 200, "Max pages to crawl per URL")
	cf := common.RegisterCommonFlags(fs, cfg)
	common.RegisterResearchAgenticFlags(fs, cf)

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan research --query <query> --urls <url1,url2,...> [options]

Examples:
  spartan research --query "pricing model" --urls https://example.com,https://example.com/docs
  spartan research --query "login flow" --urls https://example.com --headless --wait --out ./out/research.jsonl
  spartan research --query "pricing model" --urls https://example.com --transformer json-clean
  spartan research --query "pricing model" --urls https://example.com --ai-extract --ai-prompt "Extract the pricing model and support terms" --ai-fields "pricing_model,support_terms"
  spartan research --query "pricing model" --urls https://example.com,https://example.com/docs --agentic --agentic-instructions "Prioritize pricing and support commitments"

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

	opts := validate.JobValidationOpts{
		Query:       *query,
		URLs:        urlList,
		MaxDepth:    *maxDepth,
		MaxPages:    *maxPages,
		Timeout:     *cf.Timeout,
		AuthProfile: *cf.ProfileName,
	}
	if err := validate.ValidateJob(opts, model.KindResearch); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	agenticConfig := common.BuildResearchAgenticConfig(cf)
	if err := model.ValidateResearchAgenticConfig(agenticConfig); err != nil {
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

	// Resolve auth using first URL as base (validator ensures urlList non-empty).
	authOptions, err := common.ResolveAuthFromCommonFlags(cfg, urlList[0], cf)
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

	// Build network intercept config if enabled
	var networkIntercept *fetch.NetworkInterceptConfig
	if *cf.InterceptEnabled {
		resourceTypes := make([]fetch.InterceptedResourceType, 0, len(cf.InterceptResourceTypes))
		for _, rt := range cf.InterceptResourceTypes {
			resourceTypes = append(resourceTypes, fetch.InterceptedResourceType(rt))
		}
		networkIntercept = &fetch.NetworkInterceptConfig{
			Enabled:             true,
			URLPatterns:         []string(cf.InterceptURLPatterns),
			ResourceTypes:       resourceTypes,
			CaptureRequestBody:  *cf.InterceptCaptureRequest,
			CaptureResponseBody: *cf.InterceptCaptureResponse,
			MaxBodySize:         int64(*cf.InterceptMaxBodySize),
			MaxEntries:          *cf.InterceptMaxEntries,
		}
	}

	spec := jobs.JobSpec{
		Kind:             model.KindResearch,
		Query:            *query,
		URLs:             urlList,
		MaxDepth:         *maxDepth,
		MaxPages:         *maxPages,
		Headless:         *cf.Headless,
		UsePlaywright:    *cf.Playwright,
		AuthProfile:      *cf.ProfileName,
		Auth:             authOptions,
		TimeoutSeconds:   *cf.Timeout,
		Extract:          extractOpts,
		Pipeline:         pipelineOpts,
		Incremental:      *cf.Incremental,
		Device:           device,
		NetworkIntercept: networkIntercept,
		Agentic:          agenticConfig,
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
