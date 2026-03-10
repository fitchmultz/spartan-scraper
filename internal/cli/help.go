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

import (
	"fmt"

	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
)

func printHelp() {
	fmt.Printf("Spartan Scraper v%s\n\nUsage:\n", buildinfo.Version)
	fmt.Print(`  spartan <command> [options]

Commands:
  scrape       Scrape a single page
  crawl        Crawl a website
  research     Deep research across multiple sources
  auth         Manage auth vault and profiles
  form         Detect, fill, and submit forms
  batch        Submit and manage batch jobs
  chains       Manage job chains (create/list/get/submit/delete)
  watch        Watch content for changes
  feed         Monitor RSS/Atom feeds
  retention    Manage data retention and cleanup
  export       Export job results (jsonl, json, md, csv)
  export-schedule Manage automated export schedules
  render-profiles List render profiles
  pipeline-js  List pipeline JavaScript scripts
  templates    List extraction templates
  crawl-states List crawl states (incremental tracking)
  replay       Replay captured network traffic
  backup       Create and manage backups
  restore      Restore from a backup archive
  plugin       Manage plugins
  schedule     Manage scheduled jobs
  jobs         Manage jobs (list, get, cancel)
  server       Run API server + workers
  mcp          Run MCP server over stdio
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
  spartan form detect --url https://example.com/contact
  spartan render-profiles list
  spartan pipeline-js list
  spartan templates list
  spartan crawl-states list
  spartan crawl-states list --limit 10
  spartan replay --job-id <id> --target-url https://staging.example.com
  spartan replay --job-id <id> --target-url https://localhost:8080 --compare
  spartan retention status
  spartan retention cleanup --dry-run
  spartan backup create
  spartan backup create -o /backups
  spartan backup create --exclude-jobs
  spartan backup list
  spartan restore --from spartan-backup-20240115-120000.tar.gz
  spartan restore --from backup.tar.gz --dry-run
  spartan plugin list
  spartan watch add --url https://example.com --interval 3600
  spartan watch add --url https://example.com --selector "#price" --webhook https://hooks.slack.com/...
  spartan watch list
  spartan watch check <id>
  spartan watch start
  spartan export --job-id <id> --format md --out ./out/report.md
  spartan schedule add --kind scrape --interval 3600 --url https://example.com
  spartan schedule list
  spartan schedule delete --id <id>
  spartan batch submit scrape --file urls.csv --headless
  spartan batch submit crawl --file sites.json --max-depth 2
  spartan batch submit research --urls https://a.com,https://b.com --query "pricing"
  spartan batch status <batch-id> --watch
  spartan batch cancel <batch-id>
  spartan jobs list
  spartan jobs cancel <id>
  spartan chains list
  spartan chains get <chain-id>
  spartan chains create --file ./my-chain.json
  spartan chains submit <chain-id>
  spartan chains submit <chain-id> --overrides ./overrides.json
  spartan chains delete <chain-id>
  spartan feed add --url https://example.com/rss --interval 1800
  spartan feed list
  spartan feed check <feed-id>
  spartan feed start
  spartan export-schedule list
  spartan export-schedule add --name "Daily Exports" --filter-kinds crawl --format jsonl --destination local --local-path "exports/{kind}/{job_id}.jsonl"
  spartan health
  spartan mcp
  spartan server
  spartan tui

Use "spartan <command> --help" for command-specific flags.
`)
}
