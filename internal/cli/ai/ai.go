// Package ai implements the Spartan CLI subcommands for bounded AI authoring workflows.
//
// Purpose:
// - Expose preview, generation, debugging, and refinement commands for operator-driven AI helpers.
//
// Responsibilities:
// - Route `spartan ai *` subcommands, construct the shared authoring runner, and keep package-level help output aligned with the available commands.
//
// Scope:
// - Top-level AI CLI dispatch and shared runner setup only.
//
// Usage:
// - Invoked by the main CLI entrypoint when operators run `spartan ai ...`.
//
// Invariants/Assumptions:
// - Help text must stay aligned with the real product contract.
// - Commands emit JSON results or actionable errors.
// - Optional guidance flags remain optional where the product can derive sensible defaults.
package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

type authoringRunner interface {
	Preview(ctx context.Context, req aiauthoring.PreviewRequest) (aiauthoring.PreviewResult, error)
	GenerateTemplate(ctx context.Context, req aiauthoring.TemplateRequest) (aiauthoring.TemplateResult, error)
	DebugTemplate(ctx context.Context, req aiauthoring.TemplateDebugRequest) (aiauthoring.TemplateDebugResult, error)
	GenerateRenderProfile(ctx context.Context, req aiauthoring.RenderProfileRequest) (aiauthoring.RenderProfileResult, error)
	DebugRenderProfile(ctx context.Context, req aiauthoring.RenderProfileDebugRequest) (aiauthoring.RenderProfileDebugResult, error)
	GeneratePipelineJS(ctx context.Context, req aiauthoring.PipelineJSRequest) (aiauthoring.PipelineJSResult, error)
	DebugPipelineJS(ctx context.Context, req aiauthoring.PipelineJSDebugRequest) (aiauthoring.PipelineJSDebugResult, error)
	RefineResearch(ctx context.Context, req aiauthoring.ResearchRefineRequest) (aiauthoring.ResearchRefineResult, error)
	GenerateExportShape(ctx context.Context, req aiauthoring.ExportShapeRequest) (aiauthoring.ExportShapeResult, error)
	GenerateTransform(ctx context.Context, req aiauthoring.TransformRequest) (aiauthoring.TransformResult, error)
}

var newAuthoringRunner = func(cfg config.Config) (authoringRunner, error) {
	aiExtractor, err := extract.NewAIExtractor(cfg.AI)
	if err != nil {
		return nil, err
	}
	return aiauthoring.NewService(cfg, aiExtractor, true), nil
}

func RunAI(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		printHelp()
		if len(args) == 0 {
			return 1
		}
		return 0
	}

	withRunner := func(run func(authoringRunner) int) int {
		runner, err := newAuthoringRunner(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return run(runner)
	}

	switch args[0] {
	case "preview":
		return withRunner(func(runner authoringRunner) int {
			return runPreview(ctx, runner, args[1:])
		})
	case "template", "template-generate":
		return withRunner(func(runner authoringRunner) int {
			return runTemplateGenerate(ctx, runner, args[1:])
		})
	case "template-debug":
		return withRunner(func(runner authoringRunner) int {
			return runTemplateDebug(ctx, cfg, runner, args[1:])
		})
	case "render-profile":
		return withRunner(func(runner authoringRunner) int {
			return runRenderProfile(ctx, runner, args[1:])
		})
	case "render-profile-debug":
		return withRunner(func(runner authoringRunner) int {
			return runRenderProfileDebug(ctx, cfg, runner, args[1:])
		})
	case "pipeline-js":
		return withRunner(func(runner authoringRunner) int {
			return runPipelineJS(ctx, runner, args[1:])
		})
	case "pipeline-js-debug":
		return withRunner(func(runner authoringRunner) int {
			return runPipelineJSDebug(ctx, cfg, runner, args[1:])
		})
	case "research-refine":
		return withRunner(func(runner authoringRunner) int {
			return runResearchRefine(ctx, cfg, runner, args[1:])
		})
	case "export-shape":
		return withRunner(func(runner authoringRunner) int {
			return runExportShape(ctx, cfg, runner, args[1:])
		})
	case "transform":
		return withRunner(func(runner authoringRunner) int {
			return runTransform(ctx, cfg, runner, args[1:])
		})
	default:
		fmt.Fprintf(os.Stderr, "unknown ai subcommand: %s\n", args[0])
		printHelp()
		return 1
	}
}

func printHelp() {
	fmt.Fprint(os.Stderr, `AI authoring utilities.

Usage:
  spartan ai <subcommand> [options]

Subcommands:
  preview             Run AI extraction preview without creating a job
  template            Generate an extraction template without creating a job
  template-debug      Debug and repair an extraction template without creating a job
  render-profile      Generate a render profile without creating a job
  render-profile-debug Tune an existing render profile without creating a job
  pipeline-js         Generate a pipeline JS script without creating a job
  pipeline-js-debug   Tune an existing pipeline JS script without creating a job
  research-refine     Refine an existing research result without creating a job
  export-shape        Generate or tune an export shape for recurring exports without creating a job
  transform           Generate or tune a bounded result transform without creating a job

Examples:
  spartan ai preview --url https://example.com --prompt "Extract the main product facts"
  spartan ai template --url https://example.com --description "Extract product title and price"
  spartan ai template-debug --url https://example.com --template-name product
  spartan ai render-profile --url https://example.com/app
  spartan ai render-profile-debug --url https://example.com/app --profile-name example-app
  spartan ai pipeline-js --url https://example.com/app
  spartan ai pipeline-js-debug --url https://example.com/app --script-name example-app
  spartan ai research-refine --job-id <research-job-id>
  spartan ai export-shape --job-id <job-id> --format md
  spartan ai transform --job-id <job-id> --language jmespath
`)
}

func isHelpToken(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}
