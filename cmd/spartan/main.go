// Package main provides the command-line entry point for Spartan Scraper.
// It initializes the CLI runner and exits with the appropriate status code.
package main

import (
	"context"
	"os"

	"spartan-scraper/internal/cli"
)

func main() {
	os.Exit(cli.Run(context.Background()))
}
