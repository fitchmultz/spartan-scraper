package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spartan-scraper/internal/api"
	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/config"
	"spartan-scraper/internal/exporter"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/mcp"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/scheduler"
	"spartan-scraper/internal/store"
	"spartan-scraper/internal/ui/tui"
)

func Run(ctx context.Context) int {
	cfg := config.Load()
	if len(os.Args) < 2 {
		printHelp()
		return 1
	}

	switch os.Args[1] {
	case "scrape":
		return runScrape(cfg)
	case "crawl":
		return runCrawl(cfg)
	case "research":
		return runResearch(cfg)
	case "auth":
		return runAuth(cfg)
	case "export":
		return runExport(cfg)
	case "schedule":
		return runSchedule(cfg)
	case "mcp":
		return runMCP(cfg)
	case "server":
		return runServer(ctx, cfg)
	case "tui":
		return runTUI(cfg)
	case "help", "--help", "-h":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printHelp()
		return 1
	}
}

func runScrape(cfg config.Config) int {
	fs := flag.NewFlagSet("scrape", flag.ExitOnError)
	url := fs.String("url", "", "URL to scrape")
	headless := fs.Bool("headless", false, "Use headless browser")
	playwright := fs.Bool("playwright", cfg.UsePlaywright, "Use Playwright for headless pages")
	out := fs.String("out", "", "Output file (JSON)")
	wait := fs.Bool("wait", false, "Wait for completion and write output")
	waitTimeout := fs.Int("wait-timeout", 0, "Max wait time in seconds (0 = no timeout)")
	timeout := fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds")
	profileName := fs.String("auth-profile", "", "Auth profile name")

	authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
	loginURL := fs.String("login-url", "", "Login URL for headless auth")
	loginUserSelector := fs.String("login-user-selector", "", "CSS selector for username input")
	loginPassSelector := fs.String("login-pass-selector", "", "CSS selector for password input")
	loginSubmitSelector := fs.String("login-submit-selector", "", "CSS selector for submit button")
	loginUser := fs.String("login-user", "", "Username for login")
	loginPass := fs.String("login-pass", "", "Password for login")
	headers := stringSliceFlag{}
	cookies := stringSliceFlag{}
	fs.Var(&headers, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&cookies, "cookie", "Cookie value (repeatable, name=value)")
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

	_ = fs.Parse(os.Args[2:])
	if *url == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		return 1
	}

	store, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer store.Close()

	manager := jobs.NewManager(
		store,
		cfg.DataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.UsePlaywright,
	)
	manager.Start(context.Background())
	authOptions := fetch.AuthOptions{
		Basic:               *authBasic,
		Headers:             headers.ToMap(),
		Cookies:             []string(cookies),
		LoginURL:            *loginURL,
		LoginUserSelector:   *loginUserSelector,
		LoginPassSelector:   *loginPassSelector,
		LoginSubmitSelector: *loginSubmitSelector,
		LoginUser:           *loginUser,
		LoginPass:           *loginPass,
	}
	if merged, err := mergeAuthProfile(cfg, *profileName, authOptions); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	} else {
		authOptions = merged
	}

	job, err := manager.CreateScrapeJob(*url, *headless, *playwright, authOptions, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(store, job.ID, time.Duration(*waitTimeout)*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *out != "" {
			if err := copyResults(store, job.ID, *out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(job.ID)
			return 0
		}
		if err := printResults(store, job.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	payload, _ := json.MarshalIndent(job, "", "  ")
	fmt.Println(string(payload))
	return 0
}

func waitForJob(store *store.Store, id string, timeout time.Duration) error {
	start := time.Now()
	for {
		if timeout > 0 && time.Since(start) > timeout {
			return fmt.Errorf("wait timeout after %s", timeout)
		}
		job, err := store.Get(id)
		if err != nil {
			return err
		}
		switch job.Status {
		case model.StatusSucceeded:
			return nil
		case model.StatusFailed:
			if job.Error != "" {
				return fmt.Errorf("job failed: %s", job.Error)
			}
			return fmt.Errorf("job failed")
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func copyResults(store *store.Store, id, outPath string) error {
	job, err := store.Get(id)
	if err != nil {
		return err
	}
	if job.ResultPath == "" {
		return fmt.Errorf("no result path for job")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	src, err := os.Open(job.ResultPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func printResults(store *store.Store, id string) error {
	job, err := store.Get(id)
	if err != nil {
		return err
	}
	if job.ResultPath == "" {
		return fmt.Errorf("no result path for job")
	}
	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func runCrawl(cfg config.Config) int {
	fs := flag.NewFlagSet("crawl", flag.ExitOnError)
	url := fs.String("url", "", "Root URL to crawl")
	maxDepth := fs.Int("max-depth", 2, "Max crawl depth")
	maxPages := fs.Int("max-pages", 200, "Max pages to crawl")
	headless := fs.Bool("headless", false, "Use headless browser")
	playwright := fs.Bool("playwright", cfg.UsePlaywright, "Use Playwright for headless pages")
	out := fs.String("out", "", "Output file (JSONL)")
	wait := fs.Bool("wait", false, "Wait for completion and write output")
	waitTimeout := fs.Int("wait-timeout", 0, "Max wait time in seconds (0 = no timeout)")
	timeout := fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds")
	profileName := fs.String("auth-profile", "", "Auth profile name")
	headers := stringSliceFlag{}
	cookies := stringSliceFlag{}
	fs.Var(&headers, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&cookies, "cookie", "Cookie value (repeatable, name=value)")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan crawl --url <url> [options]

Examples:
  spartan crawl --url https://example.com --max-depth 2 --max-pages 200
  spartan crawl --url https://example.com --headless --wait --out ./out/site.jsonl
  spartan crawl --url https://example.com --headless --playwright --wait --out ./out/site.jsonl

Options:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[2:])
	if *url == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		return 1
	}

	store, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer store.Close()

	manager := jobs.NewManager(
		store,
		cfg.DataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.UsePlaywright,
	)
	manager.Start(context.Background())
	authOptions := fetch.AuthOptions{
		Headers: headers.ToMap(),
		Cookies: []string(cookies),
	}
	if merged, err := mergeAuthProfile(cfg, *profileName, authOptions); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	} else {
		authOptions = merged
	}

	job, err := manager.CreateCrawlJob(*url, *maxDepth, *maxPages, *headless, *playwright, authOptions, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(store, job.ID, time.Duration(*waitTimeout)*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *out != "" {
			if err := copyResults(store, job.ID, *out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(job.ID)
			return 0
		}
		if err := printResults(store, job.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	payload, _ := json.MarshalIndent(job, "", "  ")
	fmt.Println(string(payload))
	return 0
}

func runResearch(cfg config.Config) int {
	fs := flag.NewFlagSet("research", flag.ExitOnError)
	query := fs.String("query", "", "Research query")
	urls := fs.String("urls", "", "Comma-separated list of URLs to research")
	maxDepth := fs.Int("max-depth", 2, "Max crawl depth per URL (0 for single-page scrape)")
	maxPages := fs.Int("max-pages", 200, "Max pages to crawl per URL")
	headless := fs.Bool("headless", false, "Use headless browser")
	playwright := fs.Bool("playwright", cfg.UsePlaywright, "Use Playwright for headless pages")
	out := fs.String("out", "", "Output file (JSONL)")
	wait := fs.Bool("wait", false, "Wait for completion and write output")
	waitTimeout := fs.Int("wait-timeout", 0, "Max wait time in seconds (0 = no timeout)")
	timeout := fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds")
	authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
	profileName := fs.String("auth-profile", "", "Auth profile name")
	headerList := stringSliceFlag{}
	cookieList := stringSliceFlag{}
	fs.Var(&headerList, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&cookieList, "cookie", "Cookie value (repeatable, name=value)")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan research --query <query> --urls <url1,url2,...> [options]

Examples:
  spartan research --query "pricing model" --urls https://example.com,https://example.com/docs
  spartan research --query "login flow" --urls https://example.com --headless --wait --out ./out/research.jsonl
  spartan research --query "login flow" --urls https://example.com --headless --playwright --wait --out ./out/research.jsonl

Options:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[2:])
	if *query == "" || *urls == "" {
		fmt.Fprintln(os.Stderr, "--query and --urls are required")
		return 1
	}

	store, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer store.Close()

	manager := jobs.NewManager(
		store,
		cfg.DataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.UsePlaywright,
	)
	manager.Start(context.Background())

	authOptions := fetch.AuthOptions{
		Basic:   *authBasic,
		Headers: headerList.ToMap(),
		Cookies: []string(cookieList),
	}
	if merged, err := mergeAuthProfile(cfg, *profileName, authOptions); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	} else {
		authOptions = merged
	}

	job, err := manager.CreateResearchJob(*query, splitCSV(*urls), *maxDepth, *maxPages, *headless, *playwright, authOptions, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(store, job.ID, time.Duration(*waitTimeout)*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *out != "" {
			if err := copyResults(store, job.ID, *out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(job.ID)
			return 0
		}
		if err := printResults(store, job.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	payload, _ := json.MarshalIndent(job, "", "  ")
	fmt.Println(string(payload))
	return 0
}

func runAuth(cfg config.Config) int {
	if len(os.Args) < 3 {
		fmt.Fprint(os.Stderr, `Usage:
  spartan auth <subcommand> [options]

Subcommands:
  list
  set
  delete

Examples:
  spartan auth list
  spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
  spartan auth set --name acme --login-url https://example.com/login \
    --login-user-selector '#email' --login-pass-selector '#password' --login-submit-selector 'button[type=submit]' \
    --login-user you@example.com --login-pass '***'
  spartan auth delete --name acme

Use "spartan auth set --help" for full flags.
`)
		return 1
	}
	if os.Args[2] == "--help" || os.Args[2] == "-h" || os.Args[2] == "help" {
		fmt.Fprint(os.Stderr, `Usage:
  spartan auth <subcommand> [options]

Subcommands:
  list
  set
  delete

Examples:
  spartan auth list
  spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
  spartan auth set --name acme --login-url https://example.com/login \
    --login-user-selector '#email' --login-pass-selector '#password' --login-submit-selector 'button[type=submit]' \
    --login-user you@example.com --login-pass '***'
  spartan auth delete --name acme

Use "spartan auth set --help" for full flags.
`)
		return 0
	}

	switch os.Args[2] {
	case "list":
		names, err := auth.ListNames(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, name := range names {
			fmt.Println(name)
		}
		return 0
	case "set":
		fs := flag.NewFlagSet("auth set", flag.ExitOnError)
		name := fs.String("name", "", "Profile name")
		authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
		loginURL := fs.String("login-url", "", "Login URL for headless auth")
		loginUserSelector := fs.String("login-user-selector", "", "CSS selector for username input")
		loginPassSelector := fs.String("login-pass-selector", "", "CSS selector for password input")
		loginSubmitSelector := fs.String("login-submit-selector", "", "CSS selector for submit button")
		loginUser := fs.String("login-user", "", "Username for login")
		loginPass := fs.String("login-pass", "", "Password for login")
		headers := stringSliceFlag{}
		cookies := stringSliceFlag{}
		fs.Var(&headers, "header", "Extra header (repeatable, Key: Value)")
		fs.Var(&cookies, "cookie", "Cookie value (repeatable, name=value)")
		_ = fs.Parse(os.Args[3:])
		if *name == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			return 1
		}

		profile := auth.Profile{
			Name: *name,
			Auth: fetch.AuthOptions{
				Basic:               *authBasic,
				Headers:             headers.ToMap(),
				Cookies:             []string(cookies),
				LoginURL:            *loginURL,
				LoginUserSelector:   *loginUserSelector,
				LoginPassSelector:   *loginPassSelector,
				LoginSubmitSelector: *loginSubmitSelector,
				LoginUser:           *loginUser,
				LoginPass:           *loginPass,
			},
		}
		if err := auth.Upsert(cfg.DataDir, profile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("saved", *name)
		return 0
	case "delete":
		fs := flag.NewFlagSet("auth delete", flag.ExitOnError)
		name := fs.String("name", "", "Profile name")
		_ = fs.Parse(os.Args[3:])
		if *name == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			return 1
		}
		if err := auth.Delete(cfg.DataDir, *name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("deleted", *name)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown auth subcommand:", os.Args[2])
		return 1
	}
}

func runExport(cfg config.Config) int {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	jobID := fs.String("job-id", "", "Job id to export")
	format := fs.String("format", "jsonl", "Output format: jsonl|json|md|csv")
	out := fs.String("out", "", "Output file (defaults to stdout)")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan export --job-id <id> [options]

Examples:
  spartan export --job-id <id> --format md --out ./out/report.md
  spartan export --job-id <id> --format csv

Options:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[2:])
	if *jobID == "" {
		fmt.Fprintln(os.Stderr, "--job-id is required")
		return 1
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()

	job, err := st.Get(*jobID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if job.ResultPath == "" {
		fmt.Fprintln(os.Stderr, "no result path for job")
		return 1
	}
	raw, err := os.ReadFile(job.ResultPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	payload, err := exporter.Export(job, raw, *format)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *out == "" {
		fmt.Print(payload)
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.WriteFile(*out, []byte(payload), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(*out)
	return 0
}

func runSchedule(cfg config.Config) int {
	if len(os.Args) < 3 {
		fmt.Fprint(os.Stderr, `Usage:
  spartan schedule <subcommand> [options]

Subcommands:
  list
  add
  delete

Examples:
  spartan schedule add --kind scrape --interval 3600 --url https://example.com
  spartan schedule add --kind crawl --interval 7200 --url https://example.com --max-depth 2 --max-pages 200
  spartan schedule add --kind research --interval 86400 --query "pricing" --urls https://example.com,https://example.com/docs
  spartan schedule list
  spartan schedule delete --id <schedule-id>
`)
		return 1
	}
	if os.Args[2] == "--help" || os.Args[2] == "-h" || os.Args[2] == "help" {
		fmt.Fprint(os.Stderr, `Usage:
  spartan schedule <subcommand> [options]

Subcommands:
  list
  add
  delete

Examples:
  spartan schedule add --kind scrape --interval 3600 --url https://example.com
  spartan schedule add --kind crawl --interval 7200 --url https://example.com --max-depth 2 --max-pages 200
  spartan schedule add --kind research --interval 86400 --query "pricing" --urls https://example.com,https://example.com/docs
  spartan schedule list
  spartan schedule delete --id <schedule-id>
`)
		return 0
	}

	switch os.Args[2] {
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
		headless := fs.Bool("headless", false, "Use headless browser")
		playwright := fs.Bool("playwright", cfg.UsePlaywright, "Use Playwright for headless pages")
		timeout := fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds")
		authProfile := fs.String("auth-profile", "", "Auth profile name")
		_ = fs.Parse(os.Args[3:])

		if *kind == "" {
			fmt.Fprintln(os.Stderr, "--kind is required")
			return 1
		}

		params := map[string]interface{}{
			"headless":   *headless,
			"playwright": *playwright,
			"timeout":    *timeout,
		}
		if *authProfile != "" {
			params["authProfile"] = *authProfile
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
			params["query"] = *query
			params["urls"] = splitCSV(*urls)
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
		if err := scheduler.Add(cfg.DataDir, schedule); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("scheduled", *kind)
		return 0
	case "delete":
		fs := flag.NewFlagSet("schedule delete", flag.ExitOnError)
		id := fs.String("id", "", "Schedule id")
		_ = fs.Parse(os.Args[3:])
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
		fmt.Fprintln(os.Stderr, "unknown schedule subcommand:", os.Args[2])
		return 1
	}
}

func runServer(ctx context.Context, cfg config.Config) int {
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h" || os.Args[2] == "help") {
		fmt.Fprint(os.Stderr, `Usage:
  spartan server

Notes:
  Starts the API server, job workers, and scheduler.
`)
		return 0
	}
	store, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	manager := jobs.NewManager(
		store,
		cfg.DataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.UsePlaywright,
	)
	manager.Start(ctx)
	go func() {
		_ = scheduler.Run(ctx, cfg.DataDir, manager)
	}()
	server := api.NewServer(manager, store)
	fmt.Printf("Spartan server listening on :%s\n", cfg.Port)
	return httpListenAndServe("0.0.0.0:"+cfg.Port, server.Routes())
}

func runMCP(cfg config.Config) int {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan mcp

Examples:
  spartan mcp

Notes:
  MCP uses stdio. One JSON-RPC request per line.
`)
	}
	_ = fs.Parse(os.Args[2:])
	if fs.NArg() > 0 && (fs.Arg(0) == "--help" || fs.Arg(0) == "-h" || fs.Arg(0) == "help") {
		fs.Usage()
		return 0
	}

	server, err := mcp.NewServer(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer server.Close()

	if err := server.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runTUI(cfg config.Config) int {
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h" || os.Args[2] == "help") {
		fmt.Fprint(os.Stderr, `Usage:
  spartan tui [--smoke]

Examples:
  spartan tui
  spartan tui --smoke

Notes:
  Terminal UI for browsing jobs and statuses.
  --smoke renders a single frame and exits (CI smoke test).
`)
		return 0
	}
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	smoke := fs.Bool("smoke", false, "Render a single frame and exit (CI smoke test)")
	_ = fs.Parse(os.Args[2:])
	store, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer store.Close()
	return tui.RunWithOptions(store, tui.Options{Smoke: *smoke})
}

func printHelp() {
	fmt.Print(`Spartan Scraper

Usage:
  spartan <command> [options]

Commands:
  scrape   Scrape a single page
  crawl    Crawl a website
  research Deep research across multiple sources
  auth     Manage auth profiles
  export   Export job results (jsonl, json, md, csv)
  schedule Manage scheduled jobs
  mcp      Run MCP server over stdio
  server   Run API server + workers
  tui      Launch terminal UI

Examples:
  spartan scrape --url https://example.com --out ./out/example.json
  spartan crawl --url https://example.com --max-depth 2 --max-pages 200
  spartan research --query "pricing model" --urls https://example.com,https://example.com/docs
  spartan auth list
  spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
  spartan auth delete --name acme
  spartan export --job-id <id> --format md --out ./out/report.md
  spartan schedule add --kind scrape --interval 3600 --url https://example.com
  spartan schedule list
  spartan schedule delete --id <id>
  spartan mcp
  spartan server
  spartan tui

Use "spartan <command> --help" for command-specific flags.
`)
}

func httpListenAndServe(addr string, handler http.Handler) int {
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *stringSliceFlag) ToMap() map[string]string {
	if len(*s) == 0 {
		return nil
	}
	out := map[string]string{}
	for _, item := range *s {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := strings.TrimSpace(part)
		if trim != "" {
			out = append(out, trim)
		}
	}
	return out
}

func mergeAuthProfile(cfg config.Config, profileName string, override fetch.AuthOptions) (fetch.AuthOptions, error) {
	if profileName == "" {
		return override, nil
	}
	profile, found, err := auth.Get(cfg.DataDir, profileName)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	if !found {
		return fetch.AuthOptions{}, fmt.Errorf("auth profile not found: %s", profileName)
	}

	merged := profile.Auth
	if override.Basic != "" {
		merged.Basic = override.Basic
	}
	if override.Headers != nil {
		if merged.Headers == nil {
			merged.Headers = map[string]string{}
		}
		for k, v := range override.Headers {
			merged.Headers[k] = v
		}
	}
	if len(override.Cookies) > 0 {
		merged.Cookies = override.Cookies
	}
	if override.LoginURL != "" {
		merged.LoginURL = override.LoginURL
	}
	if override.LoginUserSelector != "" {
		merged.LoginUserSelector = override.LoginUserSelector
	}
	if override.LoginPassSelector != "" {
		merged.LoginPassSelector = override.LoginPassSelector
	}
	if override.LoginSubmitSelector != "" {
		merged.LoginSubmitSelector = override.LoginSubmitSelector
	}
	if override.LoginUser != "" {
		merged.LoginUser = override.LoginUser
	}
	if override.LoginPass != "" {
		merged.LoginPass = override.LoginPass
	}
	return merged, nil
}
