// Package manage contains direct export CLI workflows.
//
// Purpose:
// - Run direct exports, inspect persisted export outcomes, and surface guided recovery from the CLI.
//
// Responsibilities:
// - Resolve export config from flags, schedules, shape files, and transform files.
// - Persist direct CLI export outcomes in the shared export history store.
// - Write rendered exports to files and print operator-friendly inspection summaries.
// - Inspect export outcome history by export id, job id, or schedule id.
//
// Scope:
// - CLI command wiring only; export rendering lives in internal/exporter.
//
// Usage:
// - Invoked by `spartan export ...` subcommands and flags.
//
// Invariants/Assumptions:
// - Direct CLI exports default to writing a file under ./exports rather than streaming raw bytes to stdout.
// - JSON mode emits the same canonical export outcome envelopes used by API and MCP.
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
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
	out := fs.String("out", "", "Output file (defaults to ./exports/<job>.<format>)")
	scheduleID := fs.String("schedule-id", "", "Seed format/shape/transform from an existing export schedule")
	shapeFile := fs.String("shape-file", "", "Path to an export shape JSON file")
	transformFile := fs.String("transform-file", "", "Path to a result transform JSON file")
	transformExpression := fs.String("transform-expression", "", "Optional JMESPath/JSONata expression to transform results before export")
	transformLanguage := fs.String("transform-language", "", "Transformation language for --transform-expression: jmespath|jsonata")
	inspectID := fs.String("inspect-id", "", "Inspect a persisted export outcome by export record id")
	historyJobID := fs.String("history-job-id", "", "List persisted export outcomes for a job")
	historyScheduleID := fs.String("history-schedule-id", "", "List persisted export outcomes for an export schedule")
	limit := fs.Int("limit", 20, "Maximum number of history rows to show")
	jsonOut := fs.Bool("json", false, "Emit JSON outcome envelopes instead of human-readable summaries")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan export --job-id <id> [options]
  spartan export --inspect-id <export-id> [--json]
  spartan export --history-job-id <job-id> [--limit <n>] [--json]
  spartan export --history-schedule-id <schedule-id> [--limit <n>] [--json]

Examples:
  spartan export --job-id <id> --format md --out ./out/report.md
  spartan export --job-id <id> --schedule-id <export-schedule-id>
  spartan export --job-id <id> --format csv --transform-language jmespath --transform-expression '{title: title, url: url}'
  spartan export --job-id <id> --format md --shape-file ./shape.json --out ./out/report.md
  spartan export --inspect-id <export-id>
  spartan export --history-job-id <job-id> --limit 5
  spartan export --history-schedule-id <schedule-id> --json

Options:
`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	historyStore := scheduler.NewExportHistoryStore(cfg.DataDir)
	switch {
	case strings.TrimSpace(*inspectID) != "":
		return runExportInspect(historyStore, strings.TrimSpace(*inspectID), *jsonOut)
	case strings.TrimSpace(*historyJobID) != "":
		return runExportHistoryForJob(historyStore, strings.TrimSpace(*historyJobID), *limit, *jsonOut)
	case strings.TrimSpace(*historyScheduleID) != "":
		return runExportHistoryForSchedule(historyStore, strings.TrimSpace(*historyScheduleID), *limit, *jsonOut)
	}

	if strings.TrimSpace(*jobID) == "" {
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
		fmt.Fprintln(os.Stderr, "job has no result file")
		return 1
	}

	exportConfig, err := resolveCLIResultExportConfig(cfg, strings.TrimSpace(*scheduleID), strings.TrimSpace(*format), strings.TrimSpace(*shapeFile), strings.TrimSpace(*transformFile), strings.TrimSpace(*transformExpression), strings.TrimSpace(*transformLanguage))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	outPath := strings.TrimSpace(*out)
	if outPath == "" {
		outPath = filepath.Join("exports", exporter.ResultExportFilename(job, exportConfig))
	}

	record, err := historyStore.CreateRecord(scheduler.CreateRecordInput{
		JobID:       job.ID,
		Trigger:     exporter.OutcomeTriggerCLI,
		Destination: outPath,
		Request:     exportConfig,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	raw, err := os.ReadFile(job.ResultPath)
	if err != nil {
		_ = historyStore.MarkFailed(record.ID, err)
		return printCLIOutcome(historyStore, record.ID, *jsonOut, 1)
	}

	rendered, err := exporter.RenderResultExport(job, raw, exportConfig)
	if err != nil {
		_ = historyStore.MarkFailed(record.ID, err)
		return printCLIOutcome(historyStore, record.ID, *jsonOut, 1)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		_ = historyStore.MarkFailed(record.ID, err)
		return printCLIOutcome(historyStore, record.ID, *jsonOut, 1)
	}
	if err := os.WriteFile(outPath, rendered.Content, 0o644); err != nil {
		_ = historyStore.MarkFailed(record.ID, err)
		return printCLIOutcome(historyStore, record.ID, *jsonOut, 1)
	}
	if err := historyStore.MarkSuccess(record.ID, rendered); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return printCLIOutcome(historyStore, record.ID, *jsonOut, 0)
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

func runExportInspect(historyStore *scheduler.ExportHistoryStore, recordID string, jsonOut bool) int {
	record, err := historyStore.GetByID(recordID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return printOutcomeEnvelope(api.ExportOutcomeResponse{Export: api.BuildExportInspection(*record, nil)}, jsonOut, 0)
}

func runExportHistoryForJob(historyStore *scheduler.ExportHistoryStore, jobID string, limit int, jsonOut bool) int {
	records, total, err := historyStore.GetByJob(jobID, limit, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return printOutcomeList(api.BuildExportOutcomeListResponse(records, total, limit, 0), fmt.Sprintf("job %s", jobID), jsonOut)
}

func runExportHistoryForSchedule(historyStore *scheduler.ExportHistoryStore, scheduleID string, limit int, jsonOut bool) int {
	records, total, err := historyStore.GetBySchedule(scheduleID, limit, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return printOutcomeList(api.BuildExportOutcomeListResponse(records, total, limit, 0), fmt.Sprintf("schedule %s", scheduleID), jsonOut)
}

func printCLIOutcome(historyStore *scheduler.ExportHistoryStore, recordID string, jsonOut bool, exitCode int) int {
	record, err := historyStore.GetByID(recordID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode
	}
	return printOutcomeEnvelope(api.ExportOutcomeResponse{Export: api.BuildExportInspection(*record, nil)}, jsonOut, exitCode)
}

func printOutcomeEnvelope(response api.ExportOutcomeResponse, jsonOut bool, exitCode int) int {
	if jsonOut {
		return printJSON(response, exitCode)
	}
	printSingleOutcome(response.Export)
	return exitCode
}

func printOutcomeList(response api.ExportOutcomeListResponse, scope string, jsonOut bool) int {
	if jsonOut {
		return printJSON(response, 0)
	}
	if len(response.Exports) == 0 {
		fmt.Printf("No export history found for %s.\n", scope)
		return 0
	}
	fmt.Printf("Export history for %s (showing %d of %d):\n\n", scope, len(response.Exports), response.Total)
	for i, outcome := range response.Exports {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 72))
		}
		printSingleOutcome(outcome)
	}
	return 0
}

func printJSON(value any, exitCode int) int {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return exitCode
}

func printSingleOutcome(outcome api.ExportInspection) {
	fmt.Printf("%s\n", strings.ToUpper(outcome.Title))
	fmt.Printf("Export ID: %s\n", outcome.ID)
	fmt.Printf("Job: %s\n", outcome.JobID)
	fmt.Printf("Trigger: %s\n", outcome.Trigger)
	fmt.Printf("Status: %s\n", outcome.Status)
	fmt.Printf("Requested format: %s\n", outcome.Request.Format)
	if outcome.Destination != "" {
		fmt.Printf("Destination: %s\n", outcome.Destination)
	}
	if outcome.Artifact != nil {
		fmt.Printf("Artifact: %s (%s)\n", outcome.Artifact.Filename, outcome.Artifact.ContentType)
		fmt.Printf("Records: %d\n", outcome.Artifact.RecordCount)
		fmt.Printf("Size: %d bytes\n", outcome.Artifact.Size)
	}
	if outcome.Failure != nil {
		fmt.Printf("Failure: %s (%s)\n", outcome.Failure.Summary, outcome.Failure.Category)
	}
	fmt.Printf("Message: %s\n", outcome.Message)
	if len(outcome.Actions) > 0 {
		fmt.Println("Next steps:")
		for _, action := range outcome.Actions {
			fmt.Printf("- %s [%s]: %s\n", action.Label, action.Kind, action.Value)
		}
	}
	fmt.Println()
}
