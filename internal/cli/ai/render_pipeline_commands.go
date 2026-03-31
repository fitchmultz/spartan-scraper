// Purpose:
// - Hold the render-profile and pipeline-JS AI CLI subcommand handlers.
//
// Responsibilities:
// - Parse render/pipeline generation and debug flags, resolve optional saved assets, invoke the shared authoring service, and print JSON results.
//
// Scope:
// - `spartan ai render-profile*` and `pipeline-js*` command execution only.
//
// Usage:
// - Called from `RunAI` after subcommand dispatch selects a render or pipeline workflow.
//
// Invariants/Assumptions:
// - Saved profile/script references remain mutually exclusive with file-based overrides.
// - Visual/debug options keep parity across generation and repair flows.
package ai

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	commoncli "github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func runRenderProfile(ctx context.Context, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-render-profile", flag.ContinueOnError)
	url := fs.String("url", "", "Target URL to inspect for render profile generation")
	name := fs.String("name", "", "Optional render profile name")
	hostPatterns := fs.String("host-patterns", "", "Optional comma-separated host patterns")
	instructions := fs.String("instructions", "", "Optional operator guidance. When omitted, Spartan derives a default objective from the fetched page context")
	headless := fs.Bool("headless", false, "Use headless browser when fetching the URL")
	playwright := fs.Bool("playwright", false, "Use Playwright instead of Chromedp when fetching the URL")
	visual := fs.Bool("visual", false, "Capture a screenshot and include visual context when fetching the URL")
	var imageFiles commoncli.StringSliceFlag
	fs.Var(&imageFiles, "image-file", "Reference image file to attach as visual context (repeatable)")
	out := fs.String("out", "", "Write the JSON response to a private file under the current working directory instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai render-profile [options]

Examples:
  spartan ai render-profile --url https://example.com/app
  spartan ai render-profile --url https://example.com/catalog --name catalog --host-patterns example.com,*.example.com --instructions "Keep static assets light but wait for the product grid" --visual

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}
	images, err := loadAIImageFiles([]string(imageFiles))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.GenerateRenderProfile(ctx, aiauthoring.RenderProfileRequest{
		URL:           strings.TrimSpace(*url),
		Name:          strings.TrimSpace(*name),
		HostPatterns:  splitCSV(*hostPatterns),
		Instructions:  strings.TrimSpace(*instructions),
		Images:        images,
		Headless:      *headless,
		UsePlaywright: *playwright,
		Visual:        *visual,
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

func runPipelineJS(ctx context.Context, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-pipeline-js", flag.ContinueOnError)
	url := fs.String("url", "", "Target URL to inspect for pipeline JS generation")
	name := fs.String("name", "", "Optional pipeline JS script name")
	hostPatterns := fs.String("host-patterns", "", "Optional comma-separated host patterns")
	instructions := fs.String("instructions", "", "Optional operator guidance. When omitted, Spartan derives a default objective from the fetched page context")
	headless := fs.Bool("headless", false, "Use headless browser when fetching the URL")
	playwright := fs.Bool("playwright", false, "Use Playwright instead of Chromedp when fetching the URL")
	visual := fs.Bool("visual", false, "Capture a screenshot and include visual context when fetching the URL")
	var imageFiles commoncli.StringSliceFlag
	fs.Var(&imageFiles, "image-file", "Reference image file to attach as visual context (repeatable)")
	out := fs.String("out", "", "Write the JSON response to a private file under the current working directory instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai pipeline-js [options]

Examples:
  spartan ai pipeline-js --url https://example.com/app
  spartan ai pipeline-js --url https://example.com/catalog --name catalog --host-patterns example.com --instructions "Wait for the product grid and dismiss any cookie banner" --visual

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}
	images, err := loadAIImageFiles([]string(imageFiles))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.GeneratePipelineJS(ctx, aiauthoring.PipelineJSRequest{
		URL:           strings.TrimSpace(*url),
		Name:          strings.TrimSpace(*name),
		HostPatterns:  splitCSV(*hostPatterns),
		Instructions:  strings.TrimSpace(*instructions),
		Images:        images,
		Headless:      *headless,
		UsePlaywright: *playwright,
		Visual:        *visual,
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

func runRenderProfileDebug(ctx context.Context, cfg config.Config, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-render-profile-debug", flag.ContinueOnError)
	url := fs.String("url", "", "Target URL to recheck with the current render profile")
	profileName := fs.String("profile-name", "", "Saved render profile name to debug")
	profileFile := fs.String("profile-file", "", "Path to a render profile JSON file to debug")
	instructions := fs.String("instructions", "", "Optional tuning guidance for the AI")
	headless := fs.Bool("headless", false, "Use headless browser when fetching the baseline page")
	playwright := fs.Bool("playwright", false, "Use Playwright instead of Chromedp when fetching the baseline page")
	visual := fs.Bool("visual", false, "Capture a screenshot and include visual context when fetching the baseline page")
	var imageFiles commoncli.StringSliceFlag
	fs.Var(&imageFiles, "image-file", "Reference image file to attach as visual context (repeatable)")
	out := fs.String("out", "", "Write the JSON response to a private file under the current working directory instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai render-profile-debug [options]

Examples:
  spartan ai render-profile-debug --url https://example.com/app --profile-name example-app
  spartan ai render-profile-debug --url https://example.com/app --profile-file ./fixtures/render-profile.json --instructions "Prefer a selector wait for the dashboard shell" --visual

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	profile, err := resolveRenderProfileInput(cfg, *profileName, *profileFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	images, err := loadAIImageFiles([]string(imageFiles))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.DebugRenderProfile(ctx, aiauthoring.RenderProfileDebugRequest{
		URL:           strings.TrimSpace(*url),
		Profile:       profile,
		Instructions:  strings.TrimSpace(*instructions),
		Images:        images,
		Headless:      *headless,
		UsePlaywright: *playwright,
		Visual:        *visual,
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

func runPipelineJSDebug(ctx context.Context, cfg config.Config, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-pipeline-js-debug", flag.ContinueOnError)
	url := fs.String("url", "", "Target URL to recheck with the current pipeline JS script")
	scriptName := fs.String("script-name", "", "Saved pipeline JS script name to debug")
	scriptFile := fs.String("script-file", "", "Path to a pipeline JS JSON file to debug")
	instructions := fs.String("instructions", "", "Optional tuning guidance for the AI")
	headless := fs.Bool("headless", false, "Use headless browser when fetching the baseline page")
	playwright := fs.Bool("playwright", false, "Use Playwright instead of Chromedp when fetching the baseline page")
	visual := fs.Bool("visual", false, "Capture a screenshot and include visual context when fetching the baseline page")
	var imageFiles commoncli.StringSliceFlag
	fs.Var(&imageFiles, "image-file", "Reference image file to attach as visual context (repeatable)")
	out := fs.String("out", "", "Write the JSON response to a private file under the current working directory instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai pipeline-js-debug [options]

Examples:
  spartan ai pipeline-js-debug --url https://example.com/app --script-name example-app
  spartan ai pipeline-js-debug --url https://example.com/app --script-file ./fixtures/pipeline-js.json --instructions "Prefer selector waits over post-nav JavaScript" --visual

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	script, err := resolvePipelineJSScriptInput(cfg, *scriptName, *scriptFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	images, err := loadAIImageFiles([]string(imageFiles))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.DebugPipelineJS(ctx, aiauthoring.PipelineJSDebugRequest{
		URL:           strings.TrimSpace(*url),
		Script:        script,
		Instructions:  strings.TrimSpace(*instructions),
		Images:        images,
		Headless:      *headless,
		UsePlaywright: *playwright,
		Visual:        *visual,
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
