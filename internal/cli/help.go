// Package cli provides top-level help text for Spartan Scraper.
//
// It does NOT implement command routing (that's in cli.Run) or command
// handlers (those live in internal/cli/* domain packages).
package cli

import "fmt"

func printHelp() {
	fmt.Print(`Spartan Scraper

Usage:
  spartan <command> [options]

Commands:
  scrape       Scrape a single page
  crawl        Crawl a website
  research     Deep research across multiple sources
  auth         Manage auth vault and profiles
  templates    List extraction templates
  crawl-states List crawl states (incremental tracking)
  export       Export job results (jsonl, json, md, csv)
  schedule     Manage scheduled jobs
  mcp          Run MCP server over stdio
  server       Run API server + workers
  jobs         Manage jobs (list, get, cancel)
  health       Check system health
  tui          Launch terminal UI

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
  spartan templates list
  spartan crawl-states list
  spartan crawl-states list --limit 10
   spartan export --job-id <id> --format md --out ./out/report.md
   spartan schedule add --kind scrape --interval 3600 --url https://example.com
   spartan schedule list
   spartan schedule delete --id <id>
   spartan jobs list
   spartan jobs cancel <id>
   spartan health
   spartan mcp
   spartan server
   spartan tui

Use "spartan <command> --help" for command-specific flags.
`)
}
