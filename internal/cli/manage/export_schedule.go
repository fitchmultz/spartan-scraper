// Package manage contains export schedule management CLI commands.
//
// It does NOT implement export scheduling logic; internal/scheduler does.
package manage

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

func RunExportSchedule(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printExportScheduleHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printExportScheduleHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return listExportSchedules(cfg)
	case "add":
		return addExportSchedule(cfg, args[1:])
	case "get":
		return getExportSchedule(cfg, args[1:])
	case "delete":
		return deleteExportSchedule(cfg, args[1:])
	case "enable":
		return setExportScheduleEnabled(cfg, args[1:], true)
	case "disable":
		return setExportScheduleEnabled(cfg, args[1:], false)
	case "history":
		return getExportScheduleHistory(cfg, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown export-schedule subcommand: %s\n", args[0])
		printExportScheduleHelp()
		return 1
	}
}

func listExportSchedules(cfg config.Config) int {
	store := scheduler.NewExportStorage(cfg.DataDir)
	schedules, err := store.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if len(schedules) == 0 {
		fmt.Println("No export schedules found.")
		return 0
	}

	fmt.Printf("%-36s %-20s %-8s %-15s %-20s\n", "ID", "NAME", "ENABLED", "DESTINATION", "FORMAT")
	fmt.Println(strings.Repeat("-", 105))
	for _, s := range schedules {
		enabled := "no"
		if s.Enabled {
			enabled = "yes"
		}
		format := s.Export.Format
		if exporter.HasMeaningfulTransform(s.Export.Transform) {
			format += "+transform"
		} else if exporter.HasMeaningfulShape(s.Export.Shape) {
			format += "+shape"
		}
		fmt.Printf("%-36s %-20s %-8s %-15s %-20s\n",
			s.ID,
			truncate(s.Name, 20),
			enabled,
			s.Export.DestinationType,
			format)
	}
	return 0
}

func addExportSchedule(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("export-schedule add", flag.ExitOnError)
	name := fs.String("name", "", "Schedule name (required)")
	filterKinds := fs.String("filter-kinds", "", "Comma-separated job kinds (scrape,crawl,research)")
	filterStatus := fs.String("filter-status", "completed", "Comma-separated statuses (completed,failed,succeeded,canceled)")
	filterHasResults := fs.Bool("filter-has-results", false, "Only export jobs with non-empty results")
	format := fs.String("format", "", "Export format: json,jsonl,md,csv,xlsx (required)")
	destination := fs.String("destination", "", "Destination type: local,webhook (required)")
	localPath := fs.String("local-path", "", "Local file path template within DATA_DIR/exports (defaults to exports/{kind}/{job_id}.{format})")
	webhookURL := fs.String("webhook-url", "", "Webhook URL (for webhook destination)")
	transformExpression := fs.String("transform-expression", "", "Optional JMESPath/JSONata expression to transform results before export")
	transformLanguage := fs.String("transform-language", "", "Transformation language for --transform-expression: jmespath|jsonata")
	maxRetries := fs.Int("max-retries", 3, "Maximum retry attempts")
	baseDelayMs := fs.Int("base-delay-ms", 1000, "Base retry delay in milliseconds")

	_ = fs.Parse(args)

	if *name == "" {
		fmt.Fprintln(os.Stderr, "--name is required")
		return 1
	}
	if *format == "" {
		fmt.Fprintln(os.Stderr, "--format is required")
		return 1
	}
	if *destination == "" {
		fmt.Fprintln(os.Stderr, "--destination is required")
		return 1
	}

	// Build filters
	filters := scheduler.ExportFilters{
		HasResults: *filterHasResults,
	}
	if *filterKinds != "" {
		filters.JobKinds = splitAndTrim(*filterKinds, ",")
	}
	if *filterStatus != "" {
		filters.JobStatus = splitAndTrim(*filterStatus, ",")
	}

	// Build export config
	exportConfig := scheduler.ExportConfig{
		Format:          *format,
		DestinationType: *destination,
	}

	// Set destination-specific config
	switch *destination {
	case "local":
		if strings.TrimSpace(*localPath) != "" {
			exportConfig.LocalPath = *localPath
			exportConfig.PathTemplate = *localPath
		}
	case "webhook":
		if *webhookURL == "" {
			fmt.Fprintln(os.Stderr, "--webhook-url is required for webhook destination")
			return 1
		}
		exportConfig.WebhookURL = *webhookURL
	}

	if strings.TrimSpace(*transformExpression) != "" || strings.TrimSpace(*transformLanguage) != "" {
		exportConfig.Transform = exporter.TransformConfig{
			Expression: strings.TrimSpace(*transformExpression),
			Language:   strings.TrimSpace(*transformLanguage),
		}
	}

	// Build retry config
	retryConfig := scheduler.ExportRetryConfig{
		MaxRetries:  *maxRetries,
		BaseDelayMs: *baseDelayMs,
	}

	schedule := scheduler.ExportSchedule{
		Name:    *name,
		Enabled: true,
		Filters: filters,
		Export:  exportConfig,
		Retry:   retryConfig,
	}

	store := scheduler.NewExportStorage(cfg.DataDir)
	created, err := store.Add(schedule)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("Created export schedule: %s\n", created.ID)
	fmt.Printf("  Name: %s\n", created.Name)
	fmt.Printf("  Enabled: %v\n", created.Enabled)
	fmt.Printf("  Format: %s\n", created.Export.Format)
	fmt.Printf("  Destination: %s\n", created.Export.DestinationType)
	return 0
}

func getExportSchedule(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("export-schedule get", flag.ExitOnError)
	id := fs.String("id", "", "Schedule ID (required)")
	_ = fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		return 1
	}

	store := scheduler.NewExportStorage(cfg.DataDir)
	schedule, err := store.Get(*id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("ID: %s\n", schedule.ID)
	fmt.Printf("Name: %s\n", schedule.Name)
	fmt.Printf("Enabled: %v\n", schedule.Enabled)
	fmt.Printf("Created: %s\n", schedule.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", schedule.UpdatedAt.Format(time.RFC3339))
	fmt.Println("\nFilters:")
	if len(schedule.Filters.JobKinds) > 0 {
		fmt.Printf("  Job Kinds: %s\n", strings.Join(schedule.Filters.JobKinds, ", "))
	}
	if len(schedule.Filters.JobStatus) > 0 {
		fmt.Printf("  Job Status: %s\n", strings.Join(schedule.Filters.JobStatus, ", "))
	}
	if schedule.Filters.HasResults {
		fmt.Println("  Has Results: yes")
	}
	fmt.Println("\nExport:")
	fmt.Printf("  Format: %s\n", schedule.Export.Format)
	fmt.Printf("  Destination: %s\n", schedule.Export.DestinationType)
	if schedule.Export.LocalPath != "" {
		fmt.Printf("  Local Path: %s\n", schedule.Export.LocalPath)
	}
	if schedule.Export.WebhookURL != "" {
		fmt.Printf("  Webhook URL: %s\n", schedule.Export.WebhookURL)
	}
	if exporter.HasMeaningfulTransform(schedule.Export.Transform) {
		fmt.Println("  Transform:")
		fmt.Printf("    Language: %s\n", schedule.Export.Transform.Language)
		fmt.Printf("    Expression: %s\n", schedule.Export.Transform.Expression)
	}
	fmt.Println("\nRetry:")
	fmt.Printf("  Max Retries: %d\n", schedule.Retry.MaxRetries)
	fmt.Printf("  Base Delay: %dms\n", schedule.Retry.BaseDelayMs)

	return 0
}

func deleteExportSchedule(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("export-schedule delete", flag.ExitOnError)
	id := fs.String("id", "", "Schedule ID (required)")
	_ = fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		return 1
	}

	store := scheduler.NewExportStorage(cfg.DataDir)
	if err := store.Delete(*id); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("Deleted export schedule: %s\n", *id)
	return 0
}

func setExportScheduleEnabled(cfg config.Config, args []string, enabled bool) int {
	fs := flag.NewFlagSet("export-schedule enable/disable", flag.ExitOnError)
	id := fs.String("id", "", "Schedule ID (required)")
	_ = fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		return 1
	}

	store := scheduler.NewExportStorage(cfg.DataDir)
	schedule, err := store.Get(*id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	schedule.Enabled = enabled
	_, err = store.Update(*schedule)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	status := "enabled"
	if !enabled {
		status = "disabled"
	}
	fmt.Printf("Export schedule %s: %s\n", status, *id)
	return 0
}

func getExportScheduleHistory(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("export-schedule history", flag.ExitOnError)
	id := fs.String("id", "", "Schedule ID (required)")
	limit := fs.Int("limit", 50, "Maximum number of records to show")
	jsonOut := fs.Bool("json", false, "Emit JSON export outcome history instead of the guided text view")
	_ = fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		return 1
	}

	historyStore := scheduler.NewExportHistoryStore(cfg.DataDir)
	records, total, err := historyStore.GetBySchedule(*id, *limit, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return printOutcomeList(api.BuildExportOutcomeListResponse(records, total, *limit, 0), fmt.Sprintf("schedule %s", *id), *jsonOut)
}

func printExportScheduleHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan export-schedule <subcommand> [options]

Subcommands:
  list      List all export schedules
  add       Add a new export schedule
  get       Get details of an export schedule
  delete    Delete an export schedule
  enable    Enable an export schedule
  disable   Disable an export schedule
  history   View export history for a schedule

Examples:
  # Create a schedule to export all jobs to local files
  spartan export-schedule add \
    --name "Local Exports" \
    --format json \
    --destination local \
    --local-path "exports/{kind}/{job_id}.json"

  # Create a schedule to export failed jobs via webhook
  spartan export-schedule add \
    --name "Failed Job Alerts" \
    --filter-status failed \
    --format json \
    --destination webhook \
    --webhook-url https://example.com/webhook

  # Create a schedule that transforms results before export
  spartan export-schedule add \
    --name "Projected CSV" \
    --filter-kinds scrape \
    --format csv \
    --destination local \
    --transform-language jmespath \
    --transform-expression '{title: title, url: url}'

  # List all schedules
  spartan export-schedule list

  # Get schedule details
  spartan export-schedule get --id <schedule-id>

  # View export history
  spartan export-schedule history --id <schedule-id>

  # Enable/disable a schedule
  spartan export-schedule enable --id <schedule-id>
  spartan export-schedule disable --id <schedule-id>

  # Delete a schedule
  spartan export-schedule delete --id <schedule-id>
`)
}

// splitAndTrim splits a string by separator and trims whitespace from each part.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// truncate truncates a string to max length.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
