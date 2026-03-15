// Package manage contains schedule management CLI commands.
//
// It does NOT implement scheduler logic; internal/scheduler does.
package manage

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

func RunSchedule(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printScheduleHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printScheduleHelp()
		return 0
	}

	switch args[0] {
	case "list":
		items, err := scheduler.List(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, item := range items {
			fmt.Printf("%s\t%s\tnext=%s\tinterval=%ds\n", item.ID, item.Kind, item.NextRun.Format(time.RFC3339), item.IntervalSeconds)
		}
		return 0
	case "add":
		fs := flag.NewFlagSet("schedule add", flag.ExitOnError)
		kind := fs.String("kind", "", "Job kind: scrape|crawl|research")
		interval := fs.Int("interval", 3600, "Interval in seconds")
		url := fs.String("url", "", "Target URL")
		query := fs.String("query", "", "Research query")
		urls := fs.String("urls", "", "Comma-separated URLs for research")
		maxDepth := fs.Int("max-depth", 2, "Max crawl depth")
		maxPages := fs.Int("max-pages", 200, "Max pages to crawl")
		authProfile := fs.String("auth-profile", "", "Auth profile name")
		incremental := fs.Bool("incremental", false, "Use incremental crawling (ETag/Hash)")

		bf := common.RegisterBrowserFlags(fs, cfg)
		pf := common.RegisterPipelineFlags(fs)
		ef := common.RegisterExtractFlags(fs)
		af := common.RegisterAuthFlags(fs)
		agentic := fs.Bool("agentic", false, "Enable bounded pi-powered follow-up and synthesis for research schedules")
		agenticInstructions := fs.String("agentic-instructions", "", "Additional instructions for agentic research")
		agenticMaxRounds := fs.Int("agentic-max-rounds", 1, "Maximum bounded follow-up rounds for agentic research (1-3)")
		agenticMaxFollowUps := fs.Int("agentic-max-follow-up-urls", 3, "Maximum follow-up URLs selected per agentic research round (1-10)")

		_ = fs.Parse(args[1:])

		if *kind == "" {
			fmt.Fprintln(os.Stderr, "--kind is required")
			return 1
		}

		extractOpts, err := common.LoadExtractOptions(*ef.ExtractTemplate, *ef.ExtractConfig, *ef.ExtractValidate)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		device, err := common.ResolveDevicePreset(*bf.Device)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		screenshot, err := common.BuildScreenshotConfig(
			*bf.ScreenshotEnabled,
			*bf.ScreenshotFullPage,
			*bf.ScreenshotFormat,
			*bf.ScreenshotQuality,
			*bf.ScreenshotWidth,
			*bf.ScreenshotHeight,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		networkIntercept := common.BuildNetworkInterceptConfig(
			*bf.InterceptEnabled,
			[]string(bf.InterceptURLPatterns),
			[]string(bf.InterceptResourceTypes),
			*bf.InterceptCaptureRequest,
			*bf.InterceptCaptureResponse,
			*bf.InterceptMaxBodySize,
			*bf.InterceptMaxEntries,
		)

		exec := model.ExecutionSpec{
			Headless:         *bf.Headless,
			UsePlaywright:    *bf.Playwright,
			TimeoutSeconds:   *bf.Timeout,
			AuthProfile:      *authProfile,
			Screenshot:       screenshot,
			NetworkIntercept: networkIntercept,
			Device:           device,
			Auth: fetch.AuthOptions{
				Basic:   *af.AuthBasic,
				Headers: af.Headers.ToMap(),
				Cookies: []string(af.Cookies),
			},
			Extract: extractOpts,
			Pipeline: pipeline.Options{
				PreProcessors:  []string(pf.PreProcessors),
				PostProcessors: []string(pf.PostProcessors),
				Transformers:   []string(pf.Transformers),
			},
		}
		if len(af.TokenValues) > 0 {
			exec.Auth.Query = nil
		}
		loginFlow := common.BuildLoginFlow(common.LoginFlowInput{
			URL:            *af.LoginURL,
			UserSelector:   *af.LoginUserSelector,
			PassSelector:   *af.LoginPassSelector,
			SubmitSelector: *af.LoginSubmitSelector,
			Username:       *af.LoginUser,
			Password:       *af.LoginPass,
			AutoDetect:     *af.LoginAutoDetect,
		})
		if loginFlow != nil {
			exec.Auth.LoginURL = loginFlow.URL
			exec.Auth.LoginUserSelector = loginFlow.UserSelector
			exec.Auth.LoginPassSelector = loginFlow.PassSelector
			exec.Auth.LoginSubmitSelector = loginFlow.SubmitSelector
			exec.Auth.LoginUser = loginFlow.Username
			exec.Auth.LoginPass = loginFlow.Password
			exec.Auth.LoginAutoDetect = loginFlow.AutoDetect
		}
		tokens := common.BuildTokens(*af.AuthBasic, []string(af.TokenValues), *af.TokenKind, *af.TokenHeader, *af.TokenQuery, *af.TokenCookie)
		for _, token := range tokens {
			switch token.Kind {
			case "basic":
				exec.Auth.Basic = token.Value
			case "api_key", "bearer":
				if exec.Auth.Headers == nil {
					exec.Auth.Headers = map[string]string{}
				}
				if token.Header != "" {
					exec.Auth.Headers[token.Header] = token.Value
				}
				if token.Cookie != "" {
					exec.Auth.Cookies = append(exec.Auth.Cookies, token.Cookie+"="+token.Value)
				}
				if token.Query != "" {
					if exec.Auth.Query == nil {
						exec.Auth.Query = map[string]string{}
					}
					exec.Auth.Query[token.Query] = token.Value
				}
			}
		}
		if err := common.ApplyProxyOverrides(&exec.Auth, common.ProxyFlagConfig{
			ProxyURL:        *af.ProxyURL,
			ProxyUsername:   *af.ProxyUsername,
			ProxyPassword:   *af.ProxyPassword,
			PreferredRegion: *af.ProxyRegion,
			RequiredTags:    []string(af.ProxyRequiredTags),
			ExcludeProxyIDs: []string(af.ProxyExcludeProxyID),
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		var spec any

		switch *kind {
		case "scrape":
			if *url == "" {
				fmt.Fprintln(os.Stderr, "--url is required for scrape")
				return 1
			}
			spec = model.ScrapeSpecV1{
				Version:     model.JobSpecVersion1,
				URL:         *url,
				Incremental: *incremental,
				Execution:   exec,
			}
		case "crawl":
			if *url == "" {
				fmt.Fprintln(os.Stderr, "--url is required for crawl")
				return 1
			}
			spec = model.CrawlSpecV1{
				Version:     model.JobSpecVersion1,
				URL:         *url,
				MaxDepth:    *maxDepth,
				MaxPages:    *maxPages,
				Incremental: *incremental,
				Execution:   exec,
			}
		case "research":
			if *query == "" || *urls == "" {
				fmt.Fprintln(os.Stderr, "--query and --urls are required for research")
				return 1
			}
			urlList := common.SplitCSV(*urls)
			if len(urlList) == 0 {
				fmt.Fprintln(os.Stderr, "--urls must contain at least one valid URL")
				return 1
			}
			agenticConfig := &model.ResearchAgenticConfig{
				Enabled:         *agentic,
				Instructions:    *agenticInstructions,
				MaxRounds:       *agenticMaxRounds,
				MaxFollowUpURLs: *agenticMaxFollowUps,
			}
			if !agenticConfig.Enabled {
				agenticConfig = nil
			}
			if err := model.ValidateResearchAgenticConfig(agenticConfig); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			spec = model.ResearchSpecV1{
				Version:   model.JobSpecVersion1,
				Query:     *query,
				URLs:      urlList,
				MaxDepth:  *maxDepth,
				MaxPages:  *maxPages,
				Agentic:   agenticConfig,
				Execution: exec,
			}
		default:
			fmt.Fprintln(os.Stderr, "unknown kind:", *kind)
			return 1
		}

		schedule := scheduler.Schedule{
			Kind:            model.Kind(*kind),
			IntervalSeconds: *interval,
			SpecVersion:     model.JobSpecVersion1,
			Spec:            spec,
		}
		if _, err := scheduler.Add(cfg.DataDir, schedule); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("scheduled", *kind)
		return 0
	case "delete":
		fs := flag.NewFlagSet("schedule delete", flag.ExitOnError)
		id := fs.String("id", "", "Schedule id")
		_ = fs.Parse(args[1:])
		if *id == "" {
			fmt.Fprintln(os.Stderr, "--id is required")
			return 1
		}
		if err := scheduler.Delete(cfg.DataDir, *id); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("deleted", *id)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown schedule subcommand: %s\n", args[0])
		printScheduleHelp()
		return 1
	}
}

func printScheduleHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan schedule <subcommand> [options]

Subcommands:
  list    List all schedules
  add     Add a new schedule
  delete  Delete a schedule

Examples:
  spartan schedule add --kind scrape --interval 3600 --url https://example.com
  spartan schedule add --kind crawl --interval 7200 --url https://example.com --max-depth 2 --max-pages 200
  spartan schedule add --kind research --interval 86400 --query "pricing" --urls https://example.com,https://example.com/docs
  spartan schedule list
  spartan schedule delete --id <schedule-id>

Notes:
  Use --auth-profile to attach stored auth.
  Use --extract-template / --extract-validate / pipeline flags on add to keep scheduled jobs aligned with direct runs.
`)
}
