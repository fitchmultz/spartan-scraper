// Package manage provides CLI subcommands for managing render profiles.
// This file implements the `spartan render-profiles` command with full CRUD operations.
//
// Responsibilities:
// - Loading, listing, creating, updating, and deleting render profiles
// - Providing help text with examples for all subcommands
// - Validating render profile configuration before saving
//
// This file does NOT:
// - Execute fetches or apply profiles to requests
// - Handle runtime profile matching
//
// Invariants:
// - Profiles are stored in DATA_DIR/render_profiles.json
// - All write operations use atomic file writes
// - Subcommands return exit codes: 0 for success, 1 for errors
// - Help is displayed for unknown subcommands or when explicitly requested

package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// RunRenderProfiles handles render-profiles management subcommands.
func RunRenderProfiles(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printRenderProfilesHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printRenderProfilesHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return runRenderProfilesList(cfg)
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: profile name required")
			fmt.Fprintln(os.Stderr, "Usage: spartan render-profiles get <name>")
			return 1
		}
		return runRenderProfilesGet(cfg, args[1])
	case "create":
		return runRenderProfilesCreate(cfg, args[1:])
	case "update":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: profile name required")
			fmt.Fprintln(os.Stderr, "Usage: spartan render-profiles update <name> [flags]")
			return 1
		}
		return runRenderProfilesUpdate(cfg, args[1], args[2:])
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: profile name required")
			fmt.Fprintln(os.Stderr, "Usage: spartan render-profiles delete <name>")
			return 1
		}
		return runRenderProfilesDelete(cfg, args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown render-profiles subcommand: %s\n", args[0])
		printRenderProfilesHelp()
		return 1
	}
}

func runRenderProfilesList(cfg config.Config) int {
	store := fetch.NewRenderProfileStore(cfg.DataDir)
	profiles := store.Profiles()
	for _, p := range profiles {
		fmt.Println(p.Name)
	}
	return 0
}

func runRenderProfilesGet(cfg config.Config, name string) int {
	profile, found, err := fetch.GetRenderProfile(cfg.DataDir, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading profile: %v\n", err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Profile not found: %s\n", name)
		return 1
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting profile: %v\n", err)
		return 1
	}
	fmt.Println(string(data))
	return 0
}

func runRenderProfilesCreate(cfg config.Config, args []string) int {
	fs := newFlagSet("render-profiles create", "Create a new render profile")
	name := fs.String("name", "", "Profile name (required)")
	hostPatterns := fs.String("host-patterns", "", "Comma-separated host patterns (required)")
	forceEngine := fs.String("engine", "", "Force engine: http, chromedp, playwright")
	preferHeadless := fs.Bool("prefer-headless", false, "Skip HTTP probe and use headless")
	neverHeadless := fs.Bool("never-headless", false, "Force HTTP, never use headless")
	assumeJSHeavy := fs.Bool("assume-js-heavy", false, "Treat all pages as JS-heavy")
	jsThreshold := fs.Float64("js-threshold", 0, "JS-heavy threshold (0-1, 0=use default)")
	qps := fs.Int("rate-limit-qps", 0, "Rate limit QPS (0=use default)")
	burst := fs.Int("rate-limit-burst", 0, "Rate limit burst (0=use default)")

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

	profile := fetch.RenderProfile{
		Name:             *name,
		HostPatterns:     patterns,
		PreferHeadless:   *preferHeadless,
		NeverHeadless:    *neverHeadless,
		AssumeJSHeavy:    *assumeJSHeavy,
		JSHeavyThreshold: *jsThreshold,
		RateLimitQPS:     *qps,
		RateLimitBurst:   *burst,
	}

	if *forceEngine != "" {
		profile.ForceEngine = fetch.RenderEngine(*forceEngine)
	}

	if err := fetch.UpsertRenderProfile(cfg.DataDir, profile); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating profile: %v\n", err)
		return 1
	}

	fmt.Printf("Created render profile: %s\n", *name)
	return 0
}

func runRenderProfilesUpdate(cfg config.Config, name string, args []string) int {
	fs := newFlagSet("render-profiles update", "Update an existing render profile")
	hostPatterns := fs.String("host-patterns", "", "Comma-separated host patterns")
	forceEngine := fs.String("engine", "", "Force engine: http, chromedp, playwright")
	preferHeadless := fs.Bool("prefer-headless", false, "Skip HTTP probe and use headless")
	neverHeadless := fs.Bool("never-headless", false, "Force HTTP, never use headless")
	assumeJSHeavy := fs.Bool("assume-js-heavy", false, "Treat all pages as JS-heavy")
	jsThreshold := fs.Float64("js-threshold", -1, "JS-heavy threshold (0-1, -1=unchanged)")
	qps := fs.Int("rate-limit-qps", -1, "Rate limit QPS (-1=unchanged)")
	burst := fs.Int("rate-limit-burst", -1, "Rate limit burst (-1=unchanged)")
	clearEngine := fs.Bool("clear-engine", false, "Clear forced engine")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		return 1
	}

	profile, found, err := fetch.GetRenderProfile(cfg.DataDir, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading profile: %v\n", err)
		return 1
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Profile not found: %s\n", name)
		return 1
	}

	if *hostPatterns != "" {
		profile.HostPatterns = parseCommaSeparated(*hostPatterns)
	}
	if *clearEngine {
		profile.ForceEngine = ""
	} else if *forceEngine != "" {
		profile.ForceEngine = fetch.RenderEngine(*forceEngine)
	}
	profile.PreferHeadless = *preferHeadless
	profile.NeverHeadless = *neverHeadless
	profile.AssumeJSHeavy = *assumeJSHeavy
	if *jsThreshold >= 0 {
		profile.JSHeavyThreshold = *jsThreshold
	}
	if *qps >= 0 {
		profile.RateLimitQPS = *qps
	}
	if *burst >= 0 {
		profile.RateLimitBurst = *burst
	}

	if err := fetch.UpsertRenderProfile(cfg.DataDir, profile); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating profile: %v\n", err)
		return 1
	}

	fmt.Printf("Updated render profile: %s\n", name)
	return 0
}

func runRenderProfilesDelete(cfg config.Config, name string) int {
	if err := fetch.DeleteRenderProfile(cfg.DataDir, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting profile: %v\n", err)
		return 1
	}
	fmt.Printf("Deleted render profile: %s\n", name)
	return 0
}

func printRenderProfilesHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan render-profiles <subcommand> [options]

Subcommands:
  list              List all configured render profiles
  get <name>        Show a specific profile as JSON
  create            Create a new render profile
  update <name>     Update an existing render profile
  delete <name>     Delete a render profile

create flags:
  --name string           Profile name (required)
  --host-patterns string  Comma-separated host patterns (required)
  --engine string         Force engine: http, chromedp, playwright
  --prefer-headless       Skip HTTP probe and use headless
  --never-headless        Force HTTP, never use headless
  --assume-js-heavy       Treat all pages as JS-heavy
  --js-threshold float    JS-heavy threshold (0-1)
  --rate-limit-qps int    Rate limit QPS
  --rate-limit-burst int  Rate limit burst

update flags:
  --host-patterns string  Comma-separated host patterns
  --engine string         Force engine: http, chromedp, playwright
  --clear-engine          Clear forced engine
  --prefer-headless       Skip HTTP probe and use headless
  --never-headless        Force HTTP, never use headless
  --assume-js-heavy       Treat all pages as JS-heavy
  --js-threshold float    JS-heavy threshold (0-1, -1=unchanged)
  --rate-limit-qps int    Rate limit QPS (-1=unchanged)
  --rate-limit-burst int  Rate limit burst (-1=unchanged)

Examples:
  spartan render-profiles list
  spartan render-profiles get example-profile
  spartan render-profiles create --name "example" --host-patterns "example.com,*.example.com" --engine chromedp
  spartan render-profiles update example --host-patterns "example.com"
  spartan render-profiles delete example
`)
}

func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// newFlagSet creates a new flag set with error handling disabled
// to allow custom error messages.
func newFlagSet(name, usage string) *flagSet {
	return &flagSet{
		name:  name,
		usage: usage,
		flags: make(map[string]*flagValue),
	}
}

type flagSet struct {
	name  string
	usage string
	flags map[string]*flagValue
	args  []string
	err   error
}

type flagValue struct {
	value      interface{}
	set        bool
	name       string
	usage      string
	defaultVal interface{}
}

func (fs *flagSet) String(name, def, usage string) *string {
	v := &flagValue{
		value:      &def,
		name:       name,
		usage:      usage,
		defaultVal: def,
	}
	fs.flags[name] = v
	return v.value.(*string)
}

func (fs *flagSet) Bool(name string, def bool, usage string) *bool {
	v := &flagValue{
		value:      &def,
		name:       name,
		usage:      usage,
		defaultVal: def,
	}
	fs.flags[name] = v
	return v.value.(*bool)
}

func (fs *flagSet) Float64(name string, def float64, usage string) *float64 {
	v := &flagValue{
		value:      &def,
		name:       name,
		usage:      usage,
		defaultVal: def,
	}
	fs.flags[name] = v
	return v.value.(*float64)
}

func (fs *flagSet) Int(name string, def int, usage string) *int {
	v := &flagValue{
		value:      &def,
		name:       name,
		usage:      usage,
		defaultVal: def,
	}
	fs.flags[name] = v
	return v.value.(*int)
}

func (fs *flagSet) Parse(args []string) error {
	fs.args = args
	i := 0
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			i++
			continue
		}

		name := strings.TrimPrefix(arg, "--")
		if idx := strings.Index(name, "="); idx >= 0 {
			// --flag=value format
			value := name[idx+1:]
			name = name[:idx]
			if flag, ok := fs.flags[name]; ok {
				if err := fs.setFlag(flag, value); err != nil {
					return fmt.Errorf("invalid value for --%s: %w", name, err)
				}
				flag.set = true
			}
			i++
			continue
		}

		// --flag value format or --bool-flag
		if flag, ok := fs.flags[name]; ok {
			if _, isBool := flag.value.(*bool); isBool {
				*flag.value.(*bool) = true
				flag.set = true
				i++
			} else {
				if i+1 >= len(args) {
					return fmt.Errorf("flag --%s requires a value", name)
				}
				if err := fs.setFlag(flag, args[i+1]); err != nil {
					return fmt.Errorf("invalid value for --%s: %w", name, err)
				}
				flag.set = true
				i += 2
			}
		} else {
			i++
		}
	}
	return nil
}

func (fs *flagSet) setFlag(fv *flagValue, value string) error {
	switch v := fv.value.(type) {
	case *string:
		*v = value
	case *bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		*v = b
	case *float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		*v = f
	case *int:
		n, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		*v = n
	}
	return nil
}
