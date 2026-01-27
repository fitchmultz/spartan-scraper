// Package manage contains CLI commands for configuration/data management (auth/export/templates/states/jobs/schedule).
//
// It does NOT implement auth resolution storage formats; internal/auth owns that.
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func RunAuth(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printAuthHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printAuthHelp()
		return 0
	}

	switch args[0] {
	case "list":
		names, err := auth.ListProfileNames(cfg.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, name := range names {
			fmt.Println(name)
		}
		return 0

	case "set":
		fs := flag.NewFlagSet("auth set", flag.ExitOnError)
		name := fs.String("name", "", "Profile name")

		parentList := common.StringSliceFlag{}
		fs.Var(&parentList, "parent", "Parent profile name (repeatable)")

		authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
		tokenKind := fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key")
		tokenHeader := fs.String("token-header", "", "Token header name (api_key or bearer override)")
		tokenQuery := fs.String("token-query", "", "Token query param name (api_key)")
		tokenCookie := fs.String("token-cookie", "", "Token cookie name (api_key)")
		tokenValues := common.StringSliceFlag{}
		fs.Var(&tokenValues, "token", "Token value (repeatable)")

		presetName := fs.String("preset-name", "", "Create/update a target preset name")
		presetHosts := common.StringSliceFlag{}
		fs.Var(&presetHosts, "preset-host", "Preset host pattern (repeatable)")

		loginURL := fs.String("login-url", "", "Login URL for headless auth")
		loginUserSelector := fs.String("login-user-selector", "", "CSS selector for username input")
		loginPassSelector := fs.String("login-pass-selector", "", "CSS selector for password input")
		loginSubmitSelector := fs.String("login-submit-selector", "", "CSS selector for submit button")
		loginUser := fs.String("login-user", "", "Username for login")
		loginPass := fs.String("login-pass", "", "Password for login")

		headers := common.StringSliceFlag{}
		cookies := common.StringSliceFlag{}
		fs.Var(&headers, "header", "Extra header (repeatable, Key: Value)")
		fs.Var(&cookies, "cookie", "Cookie value (repeatable, name=value)")

		_ = fs.Parse(args[1:])
		if *name == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			return 1
		}

		profile := auth.Profile{
			Name:    *name,
			Parents: []string(parentList),
			Headers: common.ToHeaderKVs(headers.ToMap()),
			Cookies: common.ToCookies([]string(cookies)),
			Tokens:  common.BuildTokens(*authBasic, []string(tokenValues), *tokenKind, *tokenHeader, *tokenQuery, *tokenCookie),
			Login: common.BuildLoginFlow(common.LoginFlowInput{
				URL:            *loginURL,
				UserSelector:   *loginUserSelector,
				PassSelector:   *loginPassSelector,
				SubmitSelector: *loginSubmitSelector,
				Username:       *loginUser,
				Password:       *loginPass,
			}),
		}

		if err := auth.UpsertProfile(cfg.DataDir, profile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		if *presetName != "" {
			preset := auth.TargetPreset{
				Name:         *presetName,
				HostPatterns: []string(presetHosts),
				Profile:      *name,
			}
			if err := auth.UpsertPreset(cfg.DataDir, preset); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}

		fmt.Println("saved", *name)
		return 0

	case "delete":
		fs := flag.NewFlagSet("auth delete", flag.ExitOnError)
		name := fs.String("name", "", "Profile name")
		_ = fs.Parse(args[1:])
		if *name == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			return 1
		}
		if err := auth.DeleteProfile(cfg.DataDir, *name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("deleted", *name)
		return 0

	case "resolve":
		fs := flag.NewFlagSet("auth resolve", flag.ExitOnError)
		url := fs.String("url", "", "Target URL")
		profile := fs.String("profile", "", "Profile name")

		authBasic := fs.String("auth-basic", "", "Basic auth user:pass")
		tokenKind := fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key")
		tokenHeader := fs.String("token-header", "", "Token header name (api_key or bearer override)")
		tokenQuery := fs.String("token-query", "", "Token query param name (api_key)")
		tokenCookie := fs.String("token-cookie", "", "Token cookie name (api_key)")

		tokenValues := common.StringSliceFlag{}
		headers := common.StringSliceFlag{}
		cookies := common.StringSliceFlag{}
		fs.Var(&tokenValues, "token", "Token value (repeatable)")
		fs.Var(&headers, "header", "Extra header (repeatable, Key: Value)")
		fs.Var(&cookies, "cookie", "Cookie value (repeatable, name=value)")

		_ = fs.Parse(args[1:])
		if *url == "" {
			fmt.Fprintln(os.Stderr, "--url is required")
			return 1
		}

		overrides := auth.ResolveInput{
			Headers: common.ToHeaderKVs(headers.ToMap()),
			Cookies: common.ToCookies([]string(cookies)),
			Tokens:  common.BuildTokens(*authBasic, []string(tokenValues), *tokenKind, *tokenHeader, *tokenQuery, *tokenCookie),
		}
		resolved, err := auth.Resolve(cfg.DataDir, common.ResolveInput(cfg, *url, *profile, overrides))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		payload, _ := json.MarshalIndent(resolved, "", "  ")
		fmt.Println(string(payload))
		return 0

	case "vault":
		if len(args) < 2 {
			printAuthVaultHelp()
			return 1
		}
		switch args[1] {
		case "export":
			fs := flag.NewFlagSet("auth vault export", flag.ExitOnError)
			out := fs.String("out", "", "Output path")
			_ = fs.Parse(args[2:])
			if *out == "" {
				fmt.Fprintln(os.Stderr, "--out is required")
				return 1
			}
			if err := auth.ExportVault(cfg.DataDir, *out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(*out)
			return 0

		case "import":
			fs := flag.NewFlagSet("auth vault import", flag.ExitOnError)
			path := fs.String("path", "", "Input path")
			_ = fs.Parse(args[2:])
			if *path == "" {
				fmt.Fprintln(os.Stderr, "--path is required")
				return 1
			}
			if err := auth.ImportVault(cfg.DataDir, *path); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println("imported", *path)
			return 0

		default:
			fmt.Fprintln(os.Stderr, "unknown vault subcommand:", args[1])
			return 1
		}

	default:
		fmt.Fprintln(os.Stderr, "unknown auth subcommand:", args[0])
		return 1
	}
}

func printAuthHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan auth <subcommand> [options]

Subcommands:
  list
  set
  delete
  resolve
  vault

Examples:
  spartan auth list
  spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
  spartan auth set --name acme --parent base --token "token" --token-kind bearer
  spartan auth set --name acme --preset-name acme-site --preset-host "*.acme.com"
  spartan auth resolve --url https://example.com --profile acme
  spartan auth vault export --out ./out/auth_vault.json
  spartan auth vault import --path ./out/auth_vault.json

Use "spartan auth set --help" for full flags.
`)
}

func printAuthVaultHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan auth vault <subcommand> [options]

Subcommands:
  import
  export

Examples:
  spartan auth vault export --out ./out/auth_vault.json
  spartan auth vault import --path ./out/auth_vault.json
`)
}

func isHelpToken(s string) bool {
	return s == "--help" || s == "-h" || s == "help"
}
