// Package manage contains export CLI command wiring.
//
// It does NOT define export formats; internal/exporter does that.
package manage

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"spartan-scraper/internal/config"
	"spartan-scraper/internal/exporter"
	"spartan-scraper/internal/store"
)

func RunExport(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	jobID := fs.String("job-id", "", "Job id to export")
	format := fs.String("format", "jsonl", "Output format: jsonl|json|md|csv")
	out := fs.String("out", "", "Output file (defaults to stdout)")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan export --job-id <id> [options]

Examples:
  spartan export --job-id <id> --format md --out ./out/report.md
  spartan export --job-id <id> --format csv

Options:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if *jobID == "" {
		fmt.Fprintln(os.Stderr, "--job-id is required")
		return 1
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()

	job, err := st.Get(ctx, *jobID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if job.ResultPath == "" {
		fmt.Fprintln(os.Stderr, "no result path for job")
		return 1
	}

	f, err := os.Open(job.ResultPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer f.Close()

	var outWriter io.Writer
	if *out == "" {
		outWriter = os.Stdout
	} else {
		if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		outFile, err := os.Create(*out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer outFile.Close()
		outWriter = outFile
	}

	if err := exporter.ExportStream(job, f, *format, outWriter); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *out != "" {
		fmt.Println(*out)
	}
	return 0
}
