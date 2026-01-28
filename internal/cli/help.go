// Package cli provides the top-level help text and documentation for Spartan Scraper.
//
// Responsibilities:
// - Print top-level help text to stdout.
// - List available commands and provide usage examples.
//
// Does NOT handle:
// - Command routing or execution logic.
// - Help text for individual subcommands.
//
// Invariants/Assumptions:
// - Help text is static and intended for terminal output.
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
  render-profiles List render profiles
  pipeline-js  List pipeline JavaScript scripts
  templates    List extraction templates
  crawl-states List crawl states (incremental tracking)
  export       Export job results (jsonl, json, md, csv)
  schedule     Manage scheduled jobs
  mcp          Run MCP server over stdio
  server       Run API server + workers
  jobs         Manage jobs (list, get, cancel)
  health       Check system health
  tui          Launch terminal UI
  version      Print version info

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
  spartan render-profiles list
  spartan pipeline-js list
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
