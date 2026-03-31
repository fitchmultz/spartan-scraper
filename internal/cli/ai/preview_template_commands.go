// Purpose:
// - Hold the preview and template-oriented AI CLI subcommand handlers.
//
// Responsibilities:
// - Parse preview/template flags, resolve optional HTML and image inputs, invoke the shared authoring service, and print JSON results.
//
// Scope:
// - `spartan ai preview`, `template`, and `template-debug` command execution only.
//
// Usage:
// - Called from `RunAI` after top-level subcommand dispatch selects a preview or template workflow.
//
// Invariants/Assumptions:
// - Flag help stays aligned with each bounded authoring workflow.
// - HTML and file-based inputs remain mutually exclusive where required.
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
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

func runPreview(ctx context.Context, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-preview", flag.ContinueOnError)
	url := fs.String("url", "", "Target URL to fetch for preview")
	html := fs.String("html", "", "HTML content to preview directly")
	htmlFile := fs.String("html-file", "", "Path to an HTML file to preview directly")
	mode := fs.String("mode", string(extract.AIModeNaturalLanguage), "natural_language|schema_guided")
	prompt := fs.String("prompt", "", "Instructions for natural-language extraction")
	schemaText := fs.String("schema", "", "Schema-guided JSON example")
	fields := fs.String("fields", "", "Comma-separated fields to focus")
	headless := fs.Bool("headless", false, "Use headless browser when fetching the URL")
	playwright := fs.Bool("playwright", false, "Use Playwright instead of Chromedp when fetching the URL")
	visual := fs.Bool("visual", false, "Capture a screenshot and include visual context when fetching the URL")
	var imageFiles commoncli.StringSliceFlag
	fs.Var(&imageFiles, "image-file", "Reference image file to attach as visual context (repeatable)")
	out := fs.String("out", "", "Write the JSON response to a private file under the current working directory instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai preview [options]

Examples:
  spartan ai preview --url https://example.com --prompt "Extract the title, price, and rating"
  spartan ai preview --url https://example.com --mode schema_guided --schema '{"title":"Example","price":"$19.99"}'
  spartan ai preview --html-file ./fixtures/page.html --prompt "Extract the main facts" --out ./out/preview.json

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	htmlValue, err := resolveHTMLInput(*html, *htmlFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	previewMode := extract.AIExtractionMode(strings.TrimSpace(*mode))
	if previewMode == "" {
		previewMode = extract.AIModeNaturalLanguage
	}
	if previewMode != extract.AIModeNaturalLanguage && previewMode != extract.AIModeSchemaGuided {
		fmt.Fprintf(os.Stderr, "invalid --mode: %s (must be natural_language or schema_guided)\n", *mode)
		return 1
	}

	var schema map[string]interface{}
	if previewMode == extract.AIModeSchemaGuided {
		schema, err = parseJSONObject(*schemaText, "--schema is required when --mode schema_guided")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	images, err := loadAIImageFiles([]string(imageFiles))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.Preview(ctx, aiauthoring.PreviewRequest{
		URL:           strings.TrimSpace(*url),
		HTML:          htmlValue,
		Mode:          previewMode,
		Prompt:        strings.TrimSpace(*prompt),
		Schema:        schema,
		Fields:        splitCSV(*fields),
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

func runTemplateGenerate(ctx context.Context, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-template-generate", flag.ContinueOnError)
	url := fs.String("url", "", "Target URL to fetch for template generation")
	html := fs.String("html", "", "HTML content to generate from directly")
	htmlFile := fs.String("html-file", "", "Path to an HTML file to generate from directly")
	description := fs.String("description", "", "Describe what data the template should extract")
	sampleFields := fs.String("sample-fields", "", "Comma-separated field names to seed the template")
	headless := fs.Bool("headless", false, "Use headless browser when fetching the URL")
	playwright := fs.Bool("playwright", false, "Use Playwright instead of Chromedp when fetching the URL")
	visual := fs.Bool("visual", false, "Capture a screenshot and include visual context when fetching the URL")
	var imageFiles commoncli.StringSliceFlag
	fs.Var(&imageFiles, "image-file", "Reference image file to attach as visual context (repeatable)")
	out := fs.String("out", "", "Write the JSON response to a private file under the current working directory instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai template [options]

Examples:
  spartan ai template --url https://example.com/product --description "Extract product title, price, and availability"
  spartan ai template --html-file ./fixtures/page.html --description "Extract the pricing table" --sample-fields title,price,availability
  spartan ai template --url https://example.com/product --description "Extract product title and price" --headless --playwright --out ./out/template.json

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	htmlValue, err := resolveHTMLInput(*html, *htmlFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	images, err := loadAIImageFiles([]string(imageFiles))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.GenerateTemplate(ctx, aiauthoring.TemplateRequest{
		URL:           strings.TrimSpace(*url),
		HTML:          htmlValue,
		Description:   strings.TrimSpace(*description),
		SampleFields:  splitCSV(*sampleFields),
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

func runTemplateDebug(ctx context.Context, cfg config.Config, runner authoringRunner, args []string) int {
	fs := flag.NewFlagSet("ai-template-debug", flag.ContinueOnError)
	url := fs.String("url", "", "Target URL to fetch for template debugging")
	html := fs.String("html", "", "HTML content to debug directly")
	htmlFile := fs.String("html-file", "", "Path to an HTML file to debug directly")
	templateName := fs.String("template-name", "", "Saved template name to debug")
	templateFile := fs.String("template-file", "", "Path to a template JSON file to debug")
	instructions := fs.String("instructions", "", "Optional repair guidance for the AI")
	headless := fs.Bool("headless", false, "Use headless browser when fetching the URL")
	playwright := fs.Bool("playwright", false, "Use Playwright instead of Chromedp when fetching the URL")
	visual := fs.Bool("visual", false, "Capture a screenshot and include visual context when fetching the URL")
	var imageFiles commoncli.StringSliceFlag
	fs.Var(&imageFiles, "image-file", "Reference image file to attach as visual context (repeatable)")
	out := fs.String("out", "", "Write the JSON response to a private file under the current working directory instead of stdout")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan ai template-debug [options]

Examples:
  spartan ai template-debug --url https://example.com/product --template-name product
  spartan ai template-debug --html-file ./fixtures/page.html --template-file ./fixtures/template.json --instructions "Prefer stable selectors"

Options:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	htmlValue, err := resolveHTMLInput(*html, *htmlFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	template, err := resolveTemplateInput(cfg, *templateName, *templateFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	images, err := loadAIImageFiles([]string(imageFiles))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := runner.DebugTemplate(ctx, aiauthoring.TemplateDebugRequest{
		URL:           strings.TrimSpace(*url),
		HTML:          htmlValue,
		Template:      template,
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
