// Package manage contains export CLI command wiring.
//
// It does NOT define export formats; internal/exporter does.
package manage

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	commoncli "github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func RunExport(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	jobID := fs.String("job-id", "", "Job id to export")
	format := fs.String("format", "", "Output format: jsonl|json|md|csv|xlsx (defaults to jsonl or seeded schedule format)")
	out := fs.String("out", "", "Output file (defaults to stdout)")
	scheduleID := fs.String("schedule-id", "", "Seed format/shape/transform from an existing export schedule")
	shapeFile := fs.String("shape-file", "", "Path to an export shape JSON file")
	transformFile := fs.String("transform-file", "", "Path to a result transform JSON file")
	transformExpression := fs.String("transform-expression", "", "Optional JMESPath/JSONata expression to transform results before export")
	transformLanguage := fs.String("transform-language", "", "Transformation language for --transform-expression: jmespath|jsonata")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan export --job-id <id> [options]

Examples:
  spartan export --job-id <id> --format md --out ./out/report.md
  spartan export --job-id <id> --schedule-id <export-schedule-id> --out ./out/results.csv
  spartan export --job-id <id> --format csv --transform-language jmespath --transform-expression '{title: title, url: url}'
  spartan export --job-id <id> --format md --shape-file ./shape.json --out ./out/report.md

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

	exportConfig, err := resolveCLIResultExportConfig(cfg, strings.TrimSpace(*scheduleID), strings.TrimSpace(*format), strings.TrimSpace(*shapeFile), strings.TrimSpace(*transformFile), strings.TrimSpace(*transformExpression), strings.TrimSpace(*transformLanguage))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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

	if err := exporter.ExportResultStream(job, f, exportConfig, outWriter); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *out != "" {
		fmt.Println(*out)
	}
	return 0
}

func resolveCLIResultExportConfig(cfg config.Config, scheduleID string, format string, shapeFile string, transformFile string, transformExpression string, transformLanguage string) (exporter.ResultExportConfig, error) {
	config := exporter.ResultExportConfig{}
	if scheduleID != "" {
		schedule, err := scheduler.NewExportStorage(cfg.DataDir).Get(scheduleID)
		if err != nil {
			return exporter.ResultExportConfig{}, fmt.Errorf("load export schedule: %w", err)
		}
		config.Format = schedule.Export.Format
		config.Shape = schedule.Export.Shape
		config.Transform = schedule.Export.Transform
	}
	if format != "" {
		config.Format = format
	}
	if shapeFile != "" {
		shape, err := commoncli.ReadExportShapeFile(shapeFile)
		if err != nil {
			return exporter.ResultExportConfig{}, err
		}
		config.Shape = shape
	}
	if transformFile != "" {
		transform, err := commoncli.ReadTransformConfigFile(transformFile)
		if err != nil {
			return exporter.ResultExportConfig{}, err
		}
		config.Transform = transform
	}
	if transformExpression != "" || transformLanguage != "" {
		config.Transform = exporter.TransformConfig{
			Expression: transformExpression,
			Language:   transformLanguage,
		}
	}
	config = exporter.NormalizeResultExportConfig(config)
	if err := exporter.ValidateResultExportConfig(config); err != nil {
		return exporter.ResultExportConfig{}, err
	}
	return config, nil
}
