// Package manage contains crawl-states CLI command wiring.
//
// It does NOT define crawl state persistence; internal/store does.
package manage

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"spartan-scraper/internal/config"
	"spartan-scraper/internal/store"
)

func RunCrawlStates(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printCrawlStatesHelp()
		return 1
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printCrawlStatesHelp()
		return 0
	}

	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("crawl-states list", flag.ExitOnError)
		limit := fs.Int("limit", 100, "Maximum number of crawl states to list")
		offset := fs.Int("offset", 0, "Number of crawl states to skip")
		_ = fs.Parse(args[1:])

		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()

		opts := store.ListCrawlStatesOptions{Limit: *limit, Offset: *offset}
		states, err := st.ListCrawlStates(ctx, opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		if len(states) == 0 {
			fmt.Println("No crawl states found.")
			return 0
		}

		fmt.Println("URL\tETag\tLast-Modified\tHash\tLast-Scraped")
		for _, state := range states {
			lastScraped := "never"
			if !state.LastScraped.IsZero() {
				lastScraped = state.LastScraped.Format(time.RFC3339)
			}
			fmt.Printf("%s\t%s\t%s\t%s\t%s\n",
				state.URL, state.ETag, state.LastModified,
				state.ContentHash, lastScraped)
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		return 1
	}
}

func printCrawlStatesHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan crawl-states <subcommand> [options]

Subcommands:
  list    List crawl states (incremental tracking)

Examples:
  spartan crawl-states list
  spartan crawl-states list --limit 50
`)
}
