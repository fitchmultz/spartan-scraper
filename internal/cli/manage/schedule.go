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

		_ = fs.Parse(args[1:])

		if *kind == "" {
			fmt.Fprintln(os.Stderr, "--kind is required")
			return 1
		}

		params := map[string]interface{}{
			"headless":   *bf.Headless,
			"playwright": *bf.Playwright,
			"timeout":    *bf.Timeout,
			"pipeline": pipeline.Options{
				PreProcessors:  []string(pf.PreProcessors),
				PostProcessors: []string(pf.PostProcessors),
				Transformers:   []string(pf.Transformers),
			},
			"incremental": *incremental,
		}

		if *ef.ExtractTemplate != "" {
			params["extractTemplate"] = *ef.ExtractTemplate
		}
		if *ef.ExtractConfig != "" {
			params["extractConfig"] = *ef.ExtractConfig
		}
		if *ef.ExtractValidate {
			params["extractValidate"] = *ef.ExtractValidate
		}

		if *authProfile != "" {
			params["authProfile"] = *authProfile
		}

		if *af.AuthBasic != "" {
			params["authBasic"] = *af.AuthBasic
		}
		if *af.TokenKind != "" {
			params["tokenKind"] = *af.TokenKind
		}
		if *af.TokenHeader != "" {
			params["tokenHeader"] = *af.TokenHeader
		}
		if *af.TokenQuery != "" {
			params["tokenQuery"] = *af.TokenQuery
		}
		if *af.TokenCookie != "" {
			params["tokenCookie"] = *af.TokenCookie
		}
		if len(af.TokenValues) > 0 {
			params["tokens"] = []string(af.TokenValues)
		}
		if len(af.Headers.ToMap()) > 0 {
			params["headers"] = common.ToHeaderKVs(af.Headers.ToMap())
		}
		if len(af.Cookies) > 0 {
			params["cookies"] = common.ToCookies([]string(af.Cookies))
		}

		loginFlow := common.BuildLoginFlow(common.LoginFlowInput{
			URL:            *af.LoginURL,
			UserSelector:   *af.LoginUserSelector,
			PassSelector:   *af.LoginPassSelector,
			SubmitSelector: *af.LoginSubmitSelector,
			Username:       *af.LoginUser,
			Password:       *af.LoginPass,
		})
		if loginFlow != nil {
			params["login"] = loginFlow
		}

		switch *kind {
		case "scrape":
			if *url == "" {
				fmt.Fprintln(os.Stderr, "--url is required for scrape")
				return 1
			}
			params["url"] = *url
		case "crawl":
			if *url == "" {
				fmt.Fprintln(os.Stderr, "--url is required for crawl")
				return 1
			}
			params["url"] = *url
			params["maxDepth"] = *maxDepth
			params["maxPages"] = *maxPages
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
			params["query"] = *query
			params["urls"] = urlList
			params["maxDepth"] = *maxDepth
			params["maxPages"] = *maxPages
		default:
			fmt.Fprintln(os.Stderr, "unknown kind:", *kind)
			return 1
		}

		schedule := scheduler.Schedule{
			Kind:            model.Kind(*kind),
			IntervalSeconds: *interval,
			Params:          params,
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
`)
}
