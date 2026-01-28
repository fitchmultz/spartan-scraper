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

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/store"
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

		fmt.Println("URL\tDepth\tJobID\tETag\tLast-Modified\tHash\tLast-Scraped")
		for _, state := range states {
			lastScraped := "never"
			if !state.LastScraped.IsZero() {
				lastScraped = state.LastScraped.Format(time.RFC3339)
			}
			fmt.Printf("%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
				state.URL, state.Depth, state.JobID,
				state.ETag, state.LastModified,
				state.ContentHash, lastScraped)
		}
		return 0

	case "delete":
		fs := flag.NewFlagSet("crawl-states delete", flag.ExitOnError)
		url := fs.String("url", "", "URL of the crawl state to delete (required)")
		_ = fs.Parse(args[1:])

		if *url == "" {
			fmt.Fprintln(os.Stderr, "Error: --url is required")
			return 1
		}

		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()

		if err := st.DeleteCrawlState(ctx, *url); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("Deleted crawl state for: %s\n", *url)
		return 0

	case "clear":
		fs := flag.NewFlagSet("crawl-states clear", flag.ExitOnError)
		force := fs.Bool("force", false, "Force clear without confirmation")
		_ = fs.Parse(args[1:])

		if !*force {
			fmt.Print("Are you sure you want to clear ALL crawl states? (y/N): ")
			var response string
			_, err := fmt.Scanln(&response)
			if err != nil || (response != "y" && response != "Y") {
				fmt.Println("Aborted.")
				return 0
			}
		}

		st, err := store.Open(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer st.Close()

		if err := st.DeleteAllCrawlStates(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("Cleared all crawl states.")
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
  delete  Delete a specific crawl state by URL
  clear   Clear all crawl states

Examples:
  spartan crawl-states list
  spartan crawl-states delete --url "https://example.com"
  spartan crawl-states clear --force
`)
}
