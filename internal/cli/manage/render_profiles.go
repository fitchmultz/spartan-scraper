package manage

import (
	"context"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// RunRenderProfiles handles render-profiles management subcommands.
func RunRenderProfiles(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printRenderProfilesHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printRenderProfilesHelp()
		return 0
	}

	switch args[0] {
	case "list":
		store := fetch.NewRenderProfileStore(cfg.DataDir)
		profiles := store.Profiles()
		for _, p := range profiles {
			fmt.Println(p.Name)
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown render-profiles subcommand: %s\n", args[0])
		printRenderProfilesHelp()
		return 1
	}
}

func printRenderProfilesHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan render-profiles <subcommand> [options]

Subcommands:
  list    List all configured render profiles

Examples:
  spartan render-profiles list
`)
}
