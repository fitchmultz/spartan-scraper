// Purpose:
// - Hold the research-refinement, export-shape, and transform AI CLI subcommand handlers.
//
// Responsibilities:
// - Parse result-oriented authoring flags, resolve existing job/schedule inputs, invoke the shared authoring service, and print JSON results.
//
// Scope:
// - `spartan ai research-refine`, `export-shape`, and `transform` command execution only.
//
// Usage:
// - Called from `RunAI` after subcommand dispatch selects a result-shaping workflow.
//
// Invariants/Assumptions:
// - Existing job/result inputs remain mutually exclusive where required.
// - Commands continue to emit stable JSON payloads for downstream automation.
package ai

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func runResearchRefine(ctx context.Context, cfg config.Config, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-research-refine", flag.ContinueOnError)
	jobID := fs.String("job-id", "", "Research job ID to refine from the local data directory")
	resultFile := fs.String("result-file", "", "Path to a research result JSON, single-item JSON array, or single-result JSONL file")
	instructions := fs.String("instructions", "", "Optional rewrite guidance for the refinement")
	out := fs.String("out", "", "Write the JSON response to a file instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai research-refine [options]

Examples:
  spartan ai research-refine --job-id <research-job-id>
  spartan ai research-refine --result-file ./out/research-result.json --instructions "Condense this into an operator brief"

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	result, err := resolveResearchResultInput(cfg, *jobID, *resultFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	refined, err := runner.RefineResearch(ctx, aiauthoring.ResearchRefineRequest{
		Result:       result,
		Instructions: strings.TrimSpace(*instructions),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := writeJSONResult(refined, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runExportShape(ctx context.Context, cfg config.Config, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-export-shape", flag.ContinueOnError)
	jobID := fs.String("job-id", "", "Job ID to use as the representative export sample")
	resultFile := fs.String("result-file", "", "Path to a result file when no local job ID is available")
	kind := fs.String("kind", "", "Optional job kind for --result-file: scrape|crawl|research")
	format := fs.String("format", "", "Target export format: md,csv,xlsx")
	scheduleID := fs.String("schedule-id", "", "Existing export schedule ID to tune")
	shapeFile := fs.String("shape-file", "", "Path to an existing export shape JSON file")
	instructions := fs.String("instructions", "", "Optional guidance for the export shape")
	out := fs.String("out", "", "Write the JSON response to a file instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai export-shape [options]

Examples:
  spartan ai export-shape --job-id <job-id> --format md
  spartan ai export-shape --schedule-id <schedule-id> --job-id <job-id>
  spartan ai export-shape --result-file ./out/crawl.jsonl --kind crawl --format csv --shape-file ./shape.json

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	jobKind, rawResult, resolvedFormat, currentShape, err := resolveExportShapeInput(cfg, strings.TrimSpace(*jobID), strings.TrimSpace(*resultFile), strings.TrimSpace(*kind), strings.TrimSpace(*format), strings.TrimSpace(*scheduleID), strings.TrimSpace(*shapeFile))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.GenerateExportShape(ctx, aiauthoring.ExportShapeRequest{
		JobKind:      jobKind,
		Format:       resolvedFormat,
		RawResult:    rawResult,
		CurrentShape: currentShape,
		Instructions: strings.TrimSpace(*instructions),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := writeJSONResult(result, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runTransform(ctx context.Context, cfg config.Config, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-transform", flag.ContinueOnError)
	jobID := fs.String("job-id", "", "Job ID whose saved results should seed the transform")
	resultFile := fs.String("result-file", "", "Path to a result file when no local job ID is available")
	scheduleID := fs.String("schedule-id", "", "Existing export schedule ID to tune")
	transformFile := fs.String("transform-file", "", "Path to an existing transform JSON file")
	language := fs.String("language", "", "Preferred transform language: jmespath|jsonata")
	expression := fs.String("expression", "", "Current transform expression to tune")
	instructions := fs.String("instructions", "", "Optional guidance for the transform")
	out := fs.String("out", "", "Write the JSON response to a file instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai transform [options]

Examples:
  spartan ai transform --job-id <job-id> --language jmespath
  spartan ai transform --schedule-id <schedule-id> --job-id <job-id>
  spartan ai transform --job-id <job-id> --language jsonata --expression '$.{"url": url}'
  spartan ai transform --result-file ./out/crawl.jsonl --transform-file ./transform.json --instructions "Project the URL and title for export"

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	rawResult, currentTransform, preferredLanguage, err := resolveTransformRequestInput(
		cfg,
		strings.TrimSpace(*jobID),
		strings.TrimSpace(*resultFile),
		strings.TrimSpace(*scheduleID),
		strings.TrimSpace(*transformFile),
		strings.TrimSpace(*language),
		strings.TrimSpace(*expression),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.GenerateTransform(ctx, aiauthoring.TransformRequest{
		RawResult:         rawResult,
		CurrentTransform:  currentTransform,
		PreferredLanguage: preferredLanguage,
		Instructions:      strings.TrimSpace(*instructions),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := writeJSONResult(result, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
