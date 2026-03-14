package ai

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

type authoringRunner interface {
	Preview(ctx context.Context, req aiauthoring.PreviewRequest) (aiauthoring.PreviewResult, error)
	GenerateTemplate(ctx context.Context, req aiauthoring.TemplateRequest) (aiauthoring.TemplateResult, error)
	DebugTemplate(ctx context.Context, req aiauthoring.TemplateDebugRequest) (aiauthoring.TemplateDebugResult, error)
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

	switch args[0] {
	case "preview":
		runner, err := newAuthoringRunner(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return runPreview(ctx, runner, args[1:])
	case "template", "template-generate":
		runner, err := newAuthoringRunner(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return runTemplateGenerate(ctx, runner, args[1:])
	case "template-debug":
		runner, err := newAuthoringRunner(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return runTemplateDebug(ctx, cfg, runner, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown ai subcommand: %s\n", args[0])
		printHelp()
		return 1
	}
}

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
	out := fs.String("out", "", "Write the JSON response to a file instead of stdout")
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

	result, err := runner.Preview(ctx, aiauthoring.PreviewRequest{
		URL:           strings.TrimSpace(*url),
		HTML:          htmlValue,
		Mode:          previewMode,
		Prompt:        strings.TrimSpace(*prompt),
		Schema:        schema,
		Fields:        splitCSV(*fields),
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
	out := fs.String("out", "", "Write the JSON response to a file instead of stdout")
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

	result, err := runner.GenerateTemplate(ctx, aiauthoring.TemplateRequest{
		URL:           strings.TrimSpace(*url),
		HTML:          htmlValue,
		Description:   strings.TrimSpace(*description),
		SampleFields:  splitCSV(*sampleFields),
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
	out := fs.String("out", "", "Write the JSON response to a file instead of stdout")
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

	result, err := runner.DebugTemplate(ctx, aiauthoring.TemplateDebugRequest{
		URL:           strings.TrimSpace(*url),
		HTML:          htmlValue,
		Template:      template,
		Instructions:  strings.TrimSpace(*instructions),
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

func printHelp() {
	fmt.Fprint(os.Stderr, `AI authoring utilities.

Usage:
  spartan ai <subcommand> [options]

Subcommands:
  preview             Run AI extraction preview without creating a job
  template            Generate an extraction template without creating a job
  template-debug      Debug and repair an extraction template without creating a job

Examples:
  spartan ai preview --url https://example.com --prompt "Extract the main product facts"
  spartan ai template --url https://example.com --description "Extract product title and price"
  spartan ai template-debug --url https://example.com --template-name product
`)
}

func isHelpToken(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

func resolveHTMLInput(raw string, path string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" && strings.TrimSpace(path) != "" {
		return "", fmt.Errorf("--html and --html-file are mutually exclusive")
	}
	if strings.TrimSpace(path) == "" {
		return trimmed, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read html file: %w", err)
	}
	return string(data), nil
}

func resolveTemplateInput(cfg config.Config, name string, path string) (extract.Template, error) {
	trimmedName := strings.TrimSpace(name)
	trimmedPath := strings.TrimSpace(path)
	if trimmedName == "" && trimmedPath == "" {
		return extract.Template{}, fmt.Errorf("--template-name or --template-file is required")
	}
	if trimmedName != "" && trimmedPath != "" {
		return extract.Template{}, fmt.Errorf("--template-name and --template-file are mutually exclusive")
	}
	if trimmedPath != "" {
		data, err := os.ReadFile(trimmedPath)
		if err != nil {
			return extract.Template{}, fmt.Errorf("read template file: %w", err)
		}
		var template extract.Template
		if err := json.Unmarshal(data, &template); err != nil {
			return extract.Template{}, fmt.Errorf("decode template file: %w", err)
		}
		return template, nil
	}
	registry, err := extract.LoadTemplateRegistry(cfg.DataDir)
	if err != nil {
		return extract.Template{}, fmt.Errorf("load template registry: %w", err)
	}
	return registry.GetTemplate(trimmedName)
}

func parseJSONObject(raw string, emptyMessage string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New(emptyMessage)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid JSON object: %w", err)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("JSON object must not be empty")
	}
	return decoded, nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func writeJSONResult(v interface{}, outPath string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if strings.TrimSpace(outPath) == "" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(outPath, data, 0o644)
}
