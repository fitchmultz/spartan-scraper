package main

import (
	"context"
	"os"

	"spartan-scraper/internal/cli"
)

func main() {
	os.Exit(cli.Run(context.Background()))
}
