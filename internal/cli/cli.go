package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"spartan-scraper/internal/api"
	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/config"
	"spartan-scraper/internal/exporter"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/mcp"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/scheduler"
	"spartan-scraper/internal/store"
	"spartan-scraper/internal/ui/tui"
)

func Run(ctx context.Context) int {
	cfg := config.Load()
	config.InitLogger(cfg)
	if len(os.Args) < 2 {
		printHelp()
		return 1
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch os.Args[1] {
	case "scrape":
		return runScrape(ctx, cfg)
	case "crawl":
		return runCrawl(ctx, cfg)
	case "research":
		return runResearch(ctx, cfg)
	case "auth":
		return runAuth(ctx, cfg)
	case "export":
		return runExport(ctx, cfg)
	case "schedule":
		return runSchedule(ctx, cfg)
	case "mcp":
		return runMCP(ctx, cfg)
	case "server":
		return runServer(ctx, cfg)
	case "jobs":
		return runJobs(ctx, cfg)
	case "tui":
		return runTUI(ctx, cfg)
	case "help", "--help", "-h":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printHelp()
		return 1
	}
}

func runScrape(ctx context.Context, cfg config.Config) int {
	fs := flag.NewFlagSet("scrape", flag.ExitOnError)
	url := fs.String("url", "", "URL to scrape")
	headless := fs.Bool("headless", false, "Use headless browser")
	playwright := fs.Bool("playwright", cfg.UsePlaywright, "Use Playwright for headless pages")
	out := fs.String("out", "", "Output file (JSON)")
	wait := fs.Bool("wait", false, "Wait for completion and write output")
	waitTimeout := fs.Int("wait-timeout", 0, "Max wait time in seconds (0 = no timeout)")
	timeout := fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds")
	profileName := fs.String("auth-profile", "", "Auth profile name")
	incremental := fs.Bool("incremental", false, "Use incremental crawling (ETag/Hash)")

	extractTemplate := fs.String("extract-template", "", "Extraction template name")
	extractConfig := fs.String("extract-config", "", "Path to inline template JSON")
	extractValidate := fs.Bool("extract-validate", false, "Validate extraction against schema")

	preProcessors := stringSliceFlag{}
	postProcessors := stringSliceFlag{}
	transformers := stringSliceFlag{}
	fs.Var(&preProcessors, "pre-processor", "Pipeline pre-processor plugin name (repeatable)")
	fs.Var(&postProcessors, "post-processor", "Pipeline post-processor plugin name (repeatable)")
	fs.Var(&transformers, "transformer", "Output transformer name (repeatable)")

	authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
	tokenKind := fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key")
	tokenHeader := fs.String("token-header", "", "Token header name (api_key or bearer override)")
	tokenQuery := fs.String("token-query", "", "Token query param name (api_key)")
	tokenCookie := fs.String("token-cookie", "", "Token cookie name (api_key)")
	tokenValues := stringSliceFlag{}
	fs.Var(&tokenValues, "token", "Token value (repeatable)")
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

	extractOpts, err := loadExtractOptions(*extractTemplate, *extractConfig, *extractValidate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pipelineOpts := pipeline.Options{
		PreProcessors:  []string(preProcessors),
		PostProcessors: []string(postProcessors),
		Transformers:   []string(transformers),
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
	manager.Start(ctx)
	authOverrides := auth.ResolveInput{
		Headers: toHeaderKVs(headers.ToMap()),
		Cookies: toCookies([]string(cookies)),
		Tokens:  buildTokens(*authBasic, []string(tokenValues), *tokenKind, *tokenHeader, *tokenQuery, *tokenCookie),
		Login: buildLoginFlow(loginFlowInput{
			URL:            *loginURL,
			UserSelector:   *loginUserSelector,
			PassSelector:   *loginPassSelector,
			SubmitSelector: *loginSubmitSelector,
			Username:       *loginUser,
			Password:       *loginPass,
		}),
	}
	authOptions, err := resolveAuthForRequest(cfg, *url, *profileName, authOverrides)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	job, err := manager.CreateScrapeJob(ctx, *url, *headless, *playwright, authOptions, *timeout, extractOpts, pipelineOpts, *incremental)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(ctx, store, job.ID, time.Duration(*waitTimeout)*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *out != "" {
			if err := copyResults(ctx, store, job.ID, *out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(job.ID)
			return 0
		}
		if err := printResults(ctx, store, job.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	payload, _ := json.MarshalIndent(job, "", "  ")
	fmt.Println(string(payload))
	return 0
}

func waitForJob(ctx context.Context, store *store.Store, id string, timeout time.Duration) error {
	start := time.Now()
	for {
		if timeout > 0 && time.Since(start) > timeout {
			return fmt.Errorf("wait timeout after %s", timeout)
		}
		job, err := store.Get(ctx, id)
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func copyResults(ctx context.Context, store *store.Store, id, outPath string) error {
	job, err := store.Get(ctx, id)
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

func printResults(ctx context.Context, store *store.Store, id string) error {
	job, err := store.Get(ctx, id)
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

func runCrawl(ctx context.Context, cfg config.Config) int {
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
	incremental := fs.Bool("incremental", false, "Use incremental crawling (ETag/Hash)")

	extractTemplate := fs.String("extract-template", "", "Extraction template name")
	extractConfig := fs.String("extract-config", "", "Path to inline template JSON")
	extractValidate := fs.Bool("extract-validate", false, "Validate extraction against schema")

	preProcessors := stringSliceFlag{}
	postProcessors := stringSliceFlag{}
	transformers := stringSliceFlag{}
	fs.Var(&preProcessors, "pre-processor", "Pipeline pre-processor plugin name (repeatable)")
	fs.Var(&postProcessors, "post-processor", "Pipeline post-processor plugin name (repeatable)")
	fs.Var(&transformers, "transformer", "Output transformer name (repeatable)")

	authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
	tokenKind := fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key")
	tokenHeader := fs.String("token-header", "", "Token header name (api_key or bearer override)")
	tokenQuery := fs.String("token-query", "", "Token query param name (api_key)")
	tokenCookie := fs.String("token-cookie", "", "Token cookie name (api_key)")
	tokenValues := stringSliceFlag{}
	headers := stringSliceFlag{}
	cookies := stringSliceFlag{}
	fs.Var(&tokenValues, "token", "Token value (repeatable)")
	fs.Var(&headers, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&cookies, "cookie", "Cookie value (repeatable, name=value)")
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
	_ = fs.Parse(os.Args[2:])
	if *url == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		return 1
	}

	extractOpts, err := loadExtractOptions(*extractTemplate, *extractConfig, *extractValidate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pipelineOpts := pipeline.Options{
		PreProcessors:  []string(preProcessors),
		PostProcessors: []string(postProcessors),
		Transformers:   []string(transformers),
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
	manager.Start(ctx)
	authOverrides := auth.ResolveInput{
		Headers: toHeaderKVs(headers.ToMap()),
		Cookies: toCookies([]string(cookies)),
		Tokens:  buildTokens(*authBasic, []string(tokenValues), *tokenKind, *tokenHeader, *tokenQuery, *tokenCookie),
	}
	authOptions, err := resolveAuthForRequest(cfg, *url, *profileName, authOverrides)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	job, err := manager.CreateCrawlJob(ctx, *url, *maxDepth, *maxPages, *headless, *playwright, authOptions, *timeout, extractOpts, pipelineOpts, *incremental)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(ctx, store, job.ID, time.Duration(*waitTimeout)*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *out != "" {
			if err := copyResults(ctx, store, job.ID, *out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(job.ID)
			return 0
		}
		if err := printResults(ctx, store, job.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	payload, _ := json.MarshalIndent(job, "", "  ")
	fmt.Println(string(payload))
	return 0
}

func runResearch(ctx context.Context, cfg config.Config) int {
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
	incremental := fs.Bool("incremental", false, "Use incremental crawling (ETag/Hash)")

	extractTemplate := fs.String("extract-template", "", "Extraction template name")
	extractConfig := fs.String("extract-config", "", "Path to inline template JSON")
	extractValidate := fs.Bool("extract-validate", false, "Validate extraction against schema")

	preProcessors := stringSliceFlag{}
	postProcessors := stringSliceFlag{}
	transformers := stringSliceFlag{}
	fs.Var(&preProcessors, "pre-processor", "Pipeline pre-processor plugin name (repeatable)")
	fs.Var(&postProcessors, "post-processor", "Pipeline post-processor plugin name (repeatable)")
	fs.Var(&transformers, "transformer", "Output transformer name (repeatable)")

	tokenKind := fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key")
	tokenHeader := fs.String("token-header", "", "Token header name (api_key or bearer override)")
	tokenQuery := fs.String("token-query", "", "Token query param name (api_key)")
	tokenCookie := fs.String("token-cookie", "", "Token cookie name (api_key)")
	tokenValues := stringSliceFlag{}
	headerList := stringSliceFlag{}
	cookieList := stringSliceFlag{}
	fs.Var(&tokenValues, "token", "Token value (repeatable)")
	fs.Var(&headerList, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&cookieList, "cookie", "Cookie value (repeatable, name=value)")
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
	_ = fs.Parse(os.Args[2:])
	if *query == "" || *urls == "" {
		fmt.Fprintln(os.Stderr, "--query and --urls are required")
		return 1
	}

	extractOpts, err := loadExtractOptions(*extractTemplate, *extractConfig, *extractValidate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pipelineOpts := pipeline.Options{
		PreProcessors:  []string(preProcessors),
		PostProcessors: []string(postProcessors),
		Transformers:   []string(transformers),
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
	manager.Start(ctx)

	authOverrides := auth.ResolveInput{
		Headers: toHeaderKVs(headerList.ToMap()),
		Cookies: toCookies([]string(cookieList)),
		Tokens:  buildTokens(*authBasic, []string(tokenValues), *tokenKind, *tokenHeader, *tokenQuery, *tokenCookie),
	}
	authOptions, err := resolveAuthForRequest(cfg, splitCSV(*urls)[0], *profileName, authOverrides)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	job, err := manager.CreateResearchJob(ctx, *query, splitCSV(*urls), *maxDepth, *maxPages, *headless, *playwright, authOptions, *timeout, extractOpts, pipelineOpts, *incremental)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(ctx, store, job.ID, time.Duration(*waitTimeout)*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *out != "" {
			if err := copyResults(ctx, store, job.ID, *out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(job.ID)
			return 0
		}
		if err := printResults(ctx, store, job.ID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	payload, _ := json.MarshalIndent(job, "", "  ")
	fmt.Println(string(payload))
	return 0
}

func runAuth(ctx context.Context, cfg config.Config) int {
	if len(os.Args) < 3 {
		fmt.Fprint(os.Stderr, `Usage:
  spartan auth <subcommand> [options]

Subcommands:
  list
  set
  delete
	resolve
	vault

Examples:
  spartan auth list
  spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
	spartan auth set --name acme --parent base --token "token" --token-kind bearer
	spartan auth set --name acme --preset-name acme-site --preset-host "*.acme.com"
	spartan auth resolve --url https://example.com --profile acme
	spartan auth vault export --out ./out/auth_vault.json
	spartan auth vault import --path ./out/auth_vault.json

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
	resolve
	vault

Examples:
	spartan auth list
	spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
	spartan auth set --name acme --parent base --token "token" --token-kind bearer
	spartan auth set --name acme --preset-name acme-site --preset-host "*.acme.com"
	spartan auth resolve --url https://example.com --profile acme
	spartan auth vault export --out ./out/auth_vault.json
	spartan auth vault import --path ./out/auth_vault.json

Use "spartan auth set --help" for full flags.
`)
		return 0
	}
	switch os.Args[2] {
	case "list":
		names, err := auth.ListProfileNames(cfg.DataDir)
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
		parentList := stringSliceFlag{}
		fs.Var(&parentList, "parent", "Parent profile name (repeatable)")
		authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
		tokenKind := fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key")
		tokenHeader := fs.String("token-header", "", "Token header name (api_key or bearer override)")
		tokenQuery := fs.String("token-query", "", "Token query param name (api_key)")
		tokenCookie := fs.String("token-cookie", "", "Token cookie name (api_key)")
		tokenValues := stringSliceFlag{}
		fs.Var(&tokenValues, "token", "Token value (repeatable)")
		presetName := fs.String("preset-name", "", "Create/update a target preset name")
		presetHosts := stringSliceFlag{}
		fs.Var(&presetHosts, "preset-host", "Preset host pattern (repeatable)")
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
			Name:    *name,
			Parents: []string(parentList),
			Headers: toHeaderKVs(headers.ToMap()),
			Cookies: toCookies([]string(cookies)),
			Tokens:  buildTokens(*authBasic, []string(tokenValues), *tokenKind, *tokenHeader, *tokenQuery, *tokenCookie),
			Login: buildLoginFlow(loginFlowInput{
				URL:            *loginURL,
				UserSelector:   *loginUserSelector,
				PassSelector:   *loginPassSelector,
				SubmitSelector: *loginSubmitSelector,
				Username:       *loginUser,
				Password:       *loginPass,
			}),
		}
		if err := auth.UpsertProfile(cfg.DataDir, profile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *presetName != "" {
			preset := auth.TargetPreset{
				Name:         *presetName,
				HostPatterns: []string(presetHosts),
				Profile:      *name,
			}
			if err := auth.UpsertPreset(cfg.DataDir, preset); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
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
		if err := auth.DeleteProfile(cfg.DataDir, *name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("deleted", *name)
		return 0
	case "resolve":
		fs := flag.NewFlagSet("auth resolve", flag.ExitOnError)
		url := fs.String("url", "", "Target URL")
		profile := fs.String("profile", "", "Profile name")
		authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
		tokenKind := fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key")
		tokenHeader := fs.String("token-header", "", "Token header name (api_key or bearer override)")
		tokenQuery := fs.String("token-query", "", "Token query param name (api_key)")
		tokenCookie := fs.String("token-cookie", "", "Token cookie name (api_key)")
		tokenValues := stringSliceFlag{}
		headers := stringSliceFlag{}
		cookies := stringSliceFlag{}
		fs.Var(&tokenValues, "token", "Token value (repeatable)")
		fs.Var(&headers, "header", "Extra header (repeatable, Key: Value)")
		fs.Var(&cookies, "cookie", "Cookie value (repeatable, name=value)")
		_ = fs.Parse(os.Args[3:])
		if *url == "" {
			fmt.Fprintln(os.Stderr, "--url is required")
			return 1
		}
		overrides := auth.ResolveInput{
			Headers: toHeaderKVs(headers.ToMap()),
			Cookies: toCookies([]string(cookies)),
			Tokens:  buildTokens(*authBasic, []string(tokenValues), *tokenKind, *tokenHeader, *tokenQuery, *tokenCookie),
		}
		resolved, err := auth.Resolve(cfg.DataDir, resolveInput(cfg, *url, *profile, overrides))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		payload, _ := json.MarshalIndent(resolved, "", "  ")
		fmt.Println(string(payload))
		return 0
	case "vault":
		if len(os.Args) < 4 {
			fmt.Fprint(os.Stderr, `Usage:
	spartan auth vault <subcommand> [options]

Subcommands:
	import
	export

Examples:
	spartan auth vault export --out ./out/auth_vault.json
	spartan auth vault import --path ./out/auth_vault.json
`)
			return 1
		}
		switch os.Args[3] {
		case "export":
			fs := flag.NewFlagSet("auth vault export", flag.ExitOnError)
			out := fs.String("out", "", "Output path")
			_ = fs.Parse(os.Args[4:])
			if *out == "" {
				fmt.Fprintln(os.Stderr, "--out is required")
				return 1
			}
			if err := auth.ExportVault(cfg.DataDir, *out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(*out)
			return 0
		case "import":
			fs := flag.NewFlagSet("auth vault import", flag.ExitOnError)
			path := fs.String("path", "", "Input path")
			_ = fs.Parse(os.Args[4:])
			if *path == "" {
				fmt.Fprintln(os.Stderr, "--path is required")
				return 1
			}
			if err := auth.ImportVault(cfg.DataDir, *path); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println("imported", *path)
			return 0
		default:
			fmt.Fprintln(os.Stderr, "unknown vault subcommand:", os.Args[3])
			return 1
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown auth subcommand:", os.Args[2])
		return 1
	}
}

func runExport(ctx context.Context, cfg config.Config) int {
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

	job, err := st.Get(ctx, *jobID)
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

func runSchedule(ctx context.Context, cfg config.Config) int {
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

		preProcessors := stringSliceFlag{}
		postProcessors := stringSliceFlag{}
		transformers := stringSliceFlag{}
		fs.Var(&preProcessors, "pre-processor", "Pipeline pre-processor plugin name (repeatable)")
		fs.Var(&postProcessors, "post-processor", "Pipeline post-processor plugin name (repeatable)")
		fs.Var(&transformers, "transformer", "Output transformer name (repeatable)")

		_ = fs.Parse(os.Args[3:])

		if *kind == "" {
			fmt.Fprintln(os.Stderr, "--kind is required")
			return 1
		}

		params := map[string]interface{}{
			"headless":   *headless,
			"playwright": *playwright,
			"timeout":    *timeout,
			"pipeline": pipeline.Options{
				PreProcessors:  []string(preProcessors),
				PostProcessors: []string(postProcessors),
				Transformers:   []string(transformers),
			},
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

	serverCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := store.Open(cfg.DataDir)
	if err != nil {
		slog.Error("failed to open store", "error", err)
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

	manager.Start(serverCtx)
	go func() {
		if err := scheduler.Run(serverCtx, cfg.DataDir, manager); err != nil {
			slog.Error("scheduler error", "error", err)
		}
	}()

	apiServer := api.NewServer(manager, store, cfg)
	httpServer := &http.Server{
		Addr:    "0.0.0.0:" + cfg.Port,
		Handler: apiServer.Routes(),
	}

	go func() {
		slog.Info("Spartan server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
			stop()
		}
	}()

	<-serverCtx.Done()
	slog.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	slog.Info("waiting for job workers to finish...")
	waitCh := make(chan struct{})
	go func() {
		manager.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		slog.Info("all workers finished")
	case <-shutdownCtx.Done():
		slog.Warn("timed out waiting for workers to finish")
	}
	slog.Info("shutdown complete")

	return 0
}

func runMCP(ctx context.Context, cfg config.Config) int {
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

	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runJobs(ctx context.Context, cfg config.Config) int {
	if len(os.Args) < 3 {
		printJobsHelp()
		return 1
	}

	switch os.Args[2] {
	case "list":
		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()
		jobsList, err := st.List(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, job := range jobsList {
			fmt.Printf("%s\t%s\t%s\t%s\n", job.ID, job.Kind, job.Status, job.CreatedAt.Format(time.RFC3339))
		}
		return 0
	case "get":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "job id is required")
			return 1
		}
		id := os.Args[3]
		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()
		job, err := st.Get(ctx, id)
		if err != nil {
			fmt.Fprintln(os.Stderr, "job not found")
			return 1
		}
		payload, _ := json.MarshalIndent(job, "", "  ")
		fmt.Println(string(payload))
		return 0
	case "cancel":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "job id is required")
			return 1
		}
		id := os.Args[3]
		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()

		manager := jobs.NewManager(
			st,
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

		if err := manager.CancelJob(ctx, id); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("canceled", id)
		return 0
	case "help", "--help", "-h":
		printJobsHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown jobs subcommand: %s\n", os.Args[2])
		printJobsHelp()
		return 1
	}
}

func printJobsHelp() {
	fmt.Print(`Usage:
  spartan jobs <subcommand> [options]

Subcommands:
  list    List all jobs
  get     Get job details
  cancel  Cancel a running or queued job

Examples:
  spartan jobs list
  spartan jobs get <job-id>
  spartan jobs cancel <job-id>
`)
}

func runTUI(ctx context.Context, cfg config.Config) int {
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
	return tui.RunWithOptions(ctx, store, tui.Options{Smoke: *smoke})
}

func printHelp() {
	fmt.Print(`Spartan Scraper

Usage:
  spartan <command> [options]

Commands:
  scrape   Scrape a single page
  crawl    Crawl a website
  research Deep research across multiple sources
	auth     Manage auth vault and profiles
  export   Export job results (jsonl, json, md, csv)
  schedule Manage scheduled jobs
  mcp      Run MCP server over stdio
  server   Run API server + workers
  jobs     Manage jobs (list, get, cancel)
  tui      Launch terminal UI

Examples:
  spartan scrape --url https://example.com --out ./out/example.json
  spartan crawl --url https://example.com --max-depth 2 --max-pages 200
  spartan research --query "pricing model" --urls https://example.com,https://example.com/docs
  spartan auth list
  spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
	spartan auth set --name acme --parent base --token "token" --token-kind bearer
	spartan auth set --name acme --preset-name acme-site --preset-host "*.acme.com"
	spartan auth resolve --url https://example.com --profile acme
	spartan auth vault export --out ./out/auth_vault.json
	spartan auth vault import --path ./out/auth_vault.json
  spartan export --job-id <id> --format md --out ./out/report.md
  spartan schedule add --kind scrape --interval 3600 --url https://example.com
  spartan schedule list
  spartan schedule delete --id <id>
  spartan jobs list
  spartan jobs cancel <id>
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

type loginFlowInput struct {
	URL            string
	UserSelector   string
	PassSelector   string
	SubmitSelector string
	Username       string
	Password       string
}

func buildLoginFlow(input loginFlowInput) *auth.LoginFlow {
	if input.URL == "" && input.UserSelector == "" && input.PassSelector == "" && input.SubmitSelector == "" && input.Username == "" && input.Password == "" {
		return nil
	}
	return &auth.LoginFlow{
		URL:            input.URL,
		UserSelector:   input.UserSelector,
		PassSelector:   input.PassSelector,
		SubmitSelector: input.SubmitSelector,
		Username:       input.Username,
		Password:       input.Password,
	}
}

func buildTokens(basic string, tokens []string, kind string, header string, query string, cookie string) []auth.Token {
	out := make([]auth.Token, 0, len(tokens)+1)
	if strings.TrimSpace(basic) != "" {
		out = append(out, auth.Token{Kind: auth.TokenBasic, Value: basic})
	}
	tokenKind := parseTokenKind(kind)
	for _, value := range tokens {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, auth.Token{
			Kind:   tokenKind,
			Value:  value,
			Header: header,
			Query:  query,
			Cookie: cookie,
		})
	}
	return out
}

func parseTokenKind(kind string) auth.TokenKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "basic":
		return auth.TokenBasic
	case "api_key", "api-key", "apikey":
		return auth.TokenApiKey
	default:
		return auth.TokenBearer
	}
}

func toHeaderKVs(headers map[string]string) []auth.HeaderKV {
	if len(headers) == 0 {
		return nil
	}
	out := make([]auth.HeaderKV, 0, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out = append(out, auth.HeaderKV{Key: key, Value: value})
	}
	return out
}

func toCookies(cookies []string) []auth.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]auth.Cookie, 0, len(cookies))
	for _, raw := range cookies {
		parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}
		out = append(out, auth.Cookie{Name: name, Value: value})
	}
	return out
}

func resolveInput(cfg config.Config, url string, profile string, overrides auth.ResolveInput) auth.ResolveInput {
	overrides.ProfileName = profile
	overrides.URL = url
	overrides.Env = &cfg.AuthOverrides
	return overrides
}

func resolveAuthForRequest(cfg config.Config, url string, profile string, overrides auth.ResolveInput) (fetch.AuthOptions, error) {
	input := resolveInput(cfg, url, profile, overrides)
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	return auth.ToFetchOptions(resolved), nil
}

func loadExtractOptions(template, configPath string, validate bool) (extract.ExtractOptions, error) {
	opts := extract.ExtractOptions{
		Template: template,
		Validate: validate,
	}
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return extract.ExtractOptions{}, err
		}
		var tmpl extract.Template
		if err := json.Unmarshal(data, &tmpl); err != nil {
			return extract.ExtractOptions{}, fmt.Errorf("invalid template JSON: %w", err)
		}
		opts.Inline = &tmpl
	}
	return opts, nil
}
