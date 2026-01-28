package manage

import (
	"context"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// RunPipelineJS handles pipeline-js management subcommands.
func RunPipelineJS(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printPipelineJSHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printPipelineJSHelp()
		return 0
	}

	switch args[0] {
	case "list":
		registry, err := pipeline.LoadJSRegistry(cfg.DataDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading pipeline JS registry: %v\n", err)
			return 1
		}
		for _, s := range registry.Scripts {
			fmt.Println(s.Name)
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown pipeline-js subcommand: %s\n", args[0])
		printPipelineJSHelp()
		return 1
	}
}

func printPipelineJSHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan pipeline-js <subcommand> [options]

Subcommands:
  list    List all configured pipeline JavaScript scripts

Examples:
  spartan pipeline-js list
`)
}
