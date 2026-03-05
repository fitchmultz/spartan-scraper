// Package manage provides CLI subcommands for managing pipeline JavaScript scripts.
// This file implements the `spartan pipeline-js` command with full CRUD operations.
//
// Responsibilities:
// - Loading, listing, creating, updating, and deleting pipeline JS scripts
// - Providing help text with examples for all subcommands
// - Validating script configuration before saving
//
// This file does NOT:
// - Execute JavaScript code
// - Handle script matching at runtime
//
// Invariants:
// - Scripts are stored in DATA_DIR/pipeline_js.json
// - All write operations use atomic file writes
// - Subcommands return exit codes: 0 for success, 1 for errors
// - Help is displayed for unknown subcommands or when explicitly requested

package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// RunPipelineJS handles pipeline-js management subcommands.
func RunPipelineJS(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printPipelineJSHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printPipelineJSHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return runPipelineJSList(cfg)
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: script name required")
			fmt.Fprintln(os.Stderr, "Usage: spartan pipeline-js get <name>")
			return 1
		}
		return runPipelineJSGet(cfg, args[1])
	case "create":
		return runPipelineJSCreate(cfg, args[1:])
	case "update":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: script name required")
			fmt.Fprintln(os.Stderr, "Usage: spartan pipeline-js update <name> [flags]")
			return 1
		}
		return runPipelineJSUpdate(cfg, args[1], args[2:])
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: script name required")
			fmt.Fprintln(os.Stderr, "Usage: spartan pipeline-js delete <name>")
			return 1
		}
		return runPipelineJSDelete(cfg, args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown pipeline-js subcommand: %s\n", args[0])
		printPipelineJSHelp()
		return 1
	}
}

func runPipelineJSList(cfg config.Config) int {
	names, err := pipeline.ListJSScriptNames(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading pipeline JS registry: %v\n", err)
		return 1
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return 0
}

func runPipelineJSGet(cfg config.Config, name string) int {
	script, found, err := pipeline.GetJSScript(cfg.DataDir, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading script: %v\n", err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Script not found: %s\n", name)
		return 1
	}

	data, err := json.MarshalIndent(script, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting script: %v\n", err)
		return 1
	}
	fmt.Println(string(data))
	return 0
}

func runPipelineJSCreate(cfg config.Config, args []string) int {
	fs := newFlagSet("pipeline-js create", "Create a new pipeline JS script")
	name := fs.String("name", "", "Script name (required)")
	hostPatterns := fs.String("host-patterns", "", "Comma-separated host patterns (required)")
	engine := fs.String("engine", "", "Engine: chromedp, playwright")
	preNav := fs.String("pre-nav", "", "Pre-navigation JavaScript code")
	postNav := fs.String("post-nav", "", "Post-navigation JavaScript code")
	selectors := fs.String("selectors", "", "Comma-separated wait selectors")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		return 1
	}

	if *name == "" {
		fmt.Fprintln(os.Stderr, "Error: --name is required")
		return 1
	}
	if *hostPatterns == "" {
		fmt.Fprintln(os.Stderr, "Error: --host-patterns is required")
		return 1
	}

	patterns := parseCommaSeparated(*hostPatterns)
	selectorList := parseCommaSeparated(*selectors)

	script := pipeline.JSTargetScript{
		Name:         *name,
		HostPatterns: patterns,
		Engine:       *engine,
		PreNav:       *preNav,
		PostNav:      *postNav,
		Selectors:    selectorList,
	}

	if err := pipeline.UpsertJSScript(cfg.DataDir, script); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating script: %v\n", err)
		return 1
	}

	fmt.Printf("Created pipeline JS script: %s\n", *name)
	return 0
}

func runPipelineJSUpdate(cfg config.Config, name string, args []string) int {
	fs := newFlagSet("pipeline-js update", "Update an existing pipeline JS script")
	hostPatterns := fs.String("host-patterns", "", "Comma-separated host patterns")
	engine := fs.String("engine", "", "Engine: chromedp, playwright")
	clearEngine := fs.Bool("clear-engine", false, "Clear engine setting")
	preNav := fs.String("pre-nav", "", "Pre-navigation JavaScript code")
	clearPreNav := fs.Bool("clear-pre-nav", false, "Clear pre-nav script")
	postNav := fs.String("post-nav", "", "Post-navigation JavaScript code")
	clearPostNav := fs.Bool("clear-post-nav", false, "Clear post-nav script")
	selectors := fs.String("selectors", "", "Comma-separated wait selectors")
	clearSelectors := fs.Bool("clear-selectors", false, "Clear selectors")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		return 1
	}

	script, found, err := pipeline.GetJSScript(cfg.DataDir, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading script: %v\n", err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Script not found: %s\n", name)
		return 1
	}

	if *hostPatterns != "" {
		script.HostPatterns = parseCommaSeparated(*hostPatterns)
	}
	if *clearEngine {
		script.Engine = ""
	} else if *engine != "" {
		script.Engine = *engine
	}
	if *clearPreNav {
		script.PreNav = ""
	} else if *preNav != "" {
		script.PreNav = *preNav
	}
	if *clearPostNav {
		script.PostNav = ""
	} else if *postNav != "" {
		script.PostNav = *postNav
	}
	if *clearSelectors {
		script.Selectors = nil
	} else if *selectors != "" {
		script.Selectors = parseCommaSeparated(*selectors)
	}

	if err := pipeline.UpsertJSScript(cfg.DataDir, script); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating script: %v\n", err)
		return 1
	}

	fmt.Printf("Updated pipeline JS script: %s\n", name)
	return 0
}

func runPipelineJSDelete(cfg config.Config, name string) int {
	if err := pipeline.DeleteJSScript(cfg.DataDir, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting script: %v\n", err)
		return 1
	}
	fmt.Printf("Deleted pipeline JS script: %s\n", name)
	return 0
}

func printPipelineJSHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan pipeline-js <subcommand> [options]

Subcommands:
  list              List all configured pipeline JavaScript scripts
  get <name>        Show a specific script as JSON
  create            Create a new pipeline JS script
  update <name>     Update an existing pipeline JS script
  delete <name>     Delete a pipeline JS script

create flags:
  --name string           Script name (required)
  --host-patterns string  Comma-separated host patterns (required)
  --engine string         Engine: chromedp, playwright
  --pre-nav string        Pre-navigation JavaScript code
  --post-nav string       Post-navigation JavaScript code
  --selectors string      Comma-separated wait selectors

update flags:
  --host-patterns string  Comma-separated host patterns
  --engine string         Engine: chromedp, playwright
  --clear-engine          Clear engine setting
  --pre-nav string        Pre-navigation JavaScript code
  --clear-pre-nav         Clear pre-nav script
  --post-nav string       Post-navigation JavaScript code
  --clear-post-nav        Clear post-nav script
  --selectors string      Comma-separated wait selectors
  --clear-selectors       Clear selectors

Examples:
  spartan pipeline-js list
  spartan pipeline-js get example-script
  spartan pipeline-js create --name "example" --host-patterns "example.com" --pre-nav "window.scrollTo(0, document.body.scrollHeight);"
  spartan pipeline-js update example --selectors "#content,.article"
  spartan pipeline-js delete example
`)
}
