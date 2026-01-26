// Package manage contains template management CLI commands.
//
// It does NOT implement extraction logic; internal/extract does.
package manage

import (
	"context"
	"fmt"
	"os"

	"spartan-scraper/internal/config"
	"spartan-scraper/internal/extract"
)

func RunTemplates(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printTemplatesHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printTemplatesHelp()
		return 0
	}

	switch args[0] {
	case "list":
		names, err := extract.ListTemplateNames(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, name := range names {
			fmt.Println(name)
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		printTemplatesHelp()
		return 1
	}
}

func printTemplatesHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan templates <subcommand> [options]

Subcommands:
  list    List available extraction templates

Examples:
  spartan templates list
`)
}
