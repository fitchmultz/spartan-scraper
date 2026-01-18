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
	"spartan-scraper/internal/config"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/mcp"
	"spartan-scraper/internal/model"
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
	out := fs.String("out", "", "Output file (JSON)")
	wait := fs.Bool("wait", false, "Wait for completion and write output")
	timeout := fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds")

	authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
	loginURL := fs.String("login-url", "", "Login URL for headless auth")
	loginUserSelector := fs.String("login-user-selector", "", "CSS selector for username input")
	loginPassSelector := fs.String("login-pass-selector", "", "CSS selector for password input")
	loginSubmitSelector := fs.String("login-submit-selector", "", "CSS selector for submit button")
	loginUser := fs.String("login-user", "", "Username for login")
	loginPass := fs.String("login-pass", "", "Password for login")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan scrape --url <url> [options]

Examples:
  spartan scrape --url https://example.com
  spartan scrape --url https://example.com --headless --wait --out ./out/page.json
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

	manager := jobs.NewManager(store, cfg.DataDir, cfg.UserAgent, time.Duration(cfg.RequestTimeoutSecs)*time.Second, cfg.MaxConcurrency)
	manager.Start(context.Background())
	job, err := manager.CreateScrapeJob(*url, *headless, fetch.AuthOptions{
		Basic:               *authBasic,
		LoginURL:            *loginURL,
		LoginUserSelector:   *loginUserSelector,
		LoginPassSelector:   *loginPassSelector,
		LoginSubmitSelector: *loginSubmitSelector,
		LoginUser:           *loginUser,
		LoginPass:           *loginPass,
	}, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(store, job.ID); err != nil {
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

func waitForJob(store *store.Store, id string) error {
	for {
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
	out := fs.String("out", "", "Output file (JSONL)")
	wait := fs.Bool("wait", false, "Wait for completion and write output")
	timeout := fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan crawl --url <url> [options]

Examples:
  spartan crawl --url https://example.com --max-depth 2 --max-pages 200
  spartan crawl --url https://example.com --headless --wait --out ./out/site.jsonl

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

	manager := jobs.NewManager(store, cfg.DataDir, cfg.UserAgent, time.Duration(cfg.RequestTimeoutSecs)*time.Second, cfg.MaxConcurrency)
	manager.Start(context.Background())
	job, err := manager.CreateCrawlJob(*url, *maxDepth, *maxPages, *headless, fetch.AuthOptions{}, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(store, job.ID); err != nil {
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
	out := fs.String("out", "", "Output file (JSONL)")
	wait := fs.Bool("wait", false, "Wait for completion and write output")
	timeout := fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds")
	authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
	headerList := stringSliceFlag{}
	fs.Var(&headerList, "header", "Extra header (repeatable, Key: Value)")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan research --query <query> --urls <url1,url2,...> [options]

Examples:
  spartan research --query "pricing model" --urls https://example.com,https://example.com/docs
  spartan research --query "login flow" --urls https://example.com --headless --wait --out ./out/research.jsonl

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

	manager := jobs.NewManager(store, cfg.DataDir, cfg.UserAgent, time.Duration(cfg.RequestTimeoutSecs)*time.Second, cfg.MaxConcurrency)
	manager.Start(context.Background())

	job, err := manager.CreateResearchJob(*query, splitCSV(*urls), *maxDepth, *maxPages, *headless, fetch.AuthOptions{
		Basic:   *authBasic,
		Headers: headerList.ToMap(),
	}, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := manager.Enqueue(job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *wait || *out != "" {
		if err := waitForJob(store, job.ID); err != nil {
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

func runServer(ctx context.Context, cfg config.Config) int {
	store, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	manager := jobs.NewManager(store, cfg.DataDir, cfg.UserAgent, time.Duration(cfg.RequestTimeoutSecs)*time.Second, cfg.MaxConcurrency)
	manager.Start(ctx)
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
	store, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer store.Close()
	return tui.Run(store)
}

func printHelp() {
	fmt.Print(`Spartan Scraper

Usage:
  spartan <command> [options]

Commands:
  scrape   Scrape a single page
  crawl    Crawl a website
  research Deep research across multiple sources
  mcp      Run MCP server over stdio
  server   Run API server + workers
  tui      Launch terminal UI

Examples:
  spartan scrape --url https://example.com --out ./out/example.json
  spartan crawl --url https://example.com --max-depth 2 --max-pages 200
  spartan research --query "pricing model" --urls https://example.com,https://example.com/docs
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
