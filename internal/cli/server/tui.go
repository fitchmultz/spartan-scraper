// Package server contains TUI launcher CLI wiring.
//
// It does NOT implement UI; internal/ui/tui does.
package server

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	uitui "github.com/fitchmultz/spartan-scraper/internal/ui/tui"
)

func RunTUI(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
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
	_ = fs.Parse(args)

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
		cfg.MaxResponseBytes,
		cfg.UsePlaywright,
	)

	return uitui.RunWithOptions(ctx, st, manager, uitui.Options{Smoke: *smoke})
}
