// Package manage contains plugin management CLI commands.
//
// It does NOT implement plugin execution; internal/plugins does.
package manage

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/plugins"
)

// RunPlugins handles the plugin management CLI command.
func RunPlugins(_ context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printPluginsHelp()
		return 1
	}

	if isHelpToken(args[0]) {
		printPluginsHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return runPluginList(cfg)
	case "install":
		return runPluginInstall(cfg, args[1:])
	case "uninstall":
		return runPluginUninstall(cfg, args[1:])
	case "enable":
		return runPluginEnable(cfg, args[1:])
	case "disable":
		return runPluginDisable(cfg, args[1:])
	case "configure":
		return runPluginConfigure(cfg, args[1:])
	case "info":
		return runPluginInfo(cfg, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		printPluginsHelp()
		return 1
	}
}

func printPluginsHelp() {
	fmt.Fprint(os.Stderr, `Usage:
  spartan plugin <subcommand> [options]

Subcommands:
  list         List all installed plugins
  install      Install a plugin from a directory
  uninstall    Remove an installed plugin
  enable       Enable a plugin
  disable      Disable a plugin
  configure    Set a plugin configuration value
  info         Show detailed plugin information

Examples:
  spartan plugin list
  spartan plugin install --path ./my-plugin/
  spartan plugin uninstall --name my-plugin
  spartan plugin enable --name my-plugin
  spartan plugin disable --name my-plugin
  spartan plugin configure --name my-plugin --key apiKey --value secret123
  spartan plugin info --name my-plugin
`)
}

func runPluginList(cfg config.Config) int {
	loader := plugins.NewLoader(cfg.DataDir)
	pluginList, err := loader.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if len(pluginList) == 0 {
		fmt.Println("No plugins installed.")
		fmt.Println("Use 'spartan plugin install --path <dir>' to install a plugin.")
		return 0
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tHOOKS\tENABLED\tPRIORITY")
	fmt.Fprintln(w, "----\t-------\t-----\t-------\t--------")

	for _, p := range pluginList {
		hooks := strings.Join(p.Hooks, ", ")
		if len(hooks) > 30 {
			hooks = hooks[:27] + "..."
		}
		enabled := "no"
		if p.Enabled {
			enabled = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", p.Name, p.Version, hooks, enabled, p.Priority)
	}

	w.Flush()
	return 0
}

func runPluginInstall(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	pathFlag := fs.String("path", "", "Path to plugin directory")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *pathFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --path is required")
		fs.Usage()
		return 1
	}

	loader := plugins.NewLoader(cfg.DataDir)
	info, err := loader.Install(*pathFlag)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindValidation) {
			fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	fmt.Printf("Plugin installed successfully:\n")
	fmt.Printf("  Name:        %s\n", info.Name)
	fmt.Printf("  Version:     %s\n", info.Version)
	fmt.Printf("  Description: %s\n", info.Description)
	fmt.Printf("  Author:      %s\n", info.Author)
	fmt.Printf("  Hooks:       %s\n", strings.Join(info.Hooks, ", "))
	fmt.Printf("\nUse 'spartan plugin enable --name %s' to enable the plugin.\n", info.Name)

	return 0
}

func runPluginUninstall(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	nameFlag := fs.String("name", "", "Plugin name")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *nameFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --name is required")
		fs.Usage()
		return 1
	}

	loader := plugins.NewLoader(cfg.DataDir)
	if err := loader.Uninstall(*nameFlag); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	fmt.Printf("Plugin '%s' uninstalled successfully.\n", *nameFlag)
	return 0
}

func runPluginEnable(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("enable", flag.ExitOnError)
	nameFlag := fs.String("name", "", "Plugin name")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *nameFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --name is required")
		fs.Usage()
		return 1
	}

	loader := plugins.NewLoader(cfg.DataDir)
	if err := loader.Enable(*nameFlag); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	fmt.Printf("Plugin '%s' enabled.\n", *nameFlag)
	fmt.Println("Restart the server to load the plugin.")
	return 0
}

func runPluginDisable(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("disable", flag.ExitOnError)
	nameFlag := fs.String("name", "", "Plugin name")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *nameFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --name is required")
		fs.Usage()
		return 1
	}

	loader := plugins.NewLoader(cfg.DataDir)
	if err := loader.Disable(*nameFlag); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	fmt.Printf("Plugin '%s' disabled.\n", *nameFlag)
	fmt.Println("Restart the server to unload the plugin.")
	return 0
}

func runPluginConfigure(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("configure", flag.ExitOnError)
	nameFlag := fs.String("name", "", "Plugin name")
	keyFlag := fs.String("key", "", "Configuration key")
	valueFlag := fs.String("value", "", "Configuration value")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *nameFlag == "" || *keyFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --name and --key are required")
		fs.Usage()
		return 1
	}

	loader := plugins.NewLoader(cfg.DataDir)

	// Try to parse value as different types
	value := parseConfigValue(*valueFlag)

	if err := loader.Configure(*nameFlag, *keyFlag, value); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	fmt.Printf("Plugin '%s' configuration updated: %s = %v\n", *nameFlag, *keyFlag, value)
	return 0
}

func runPluginInfo(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	nameFlag := fs.String("name", "", "Plugin name")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *nameFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --name is required")
		fs.Usage()
		return 1
	}

	loader := plugins.NewLoader(cfg.DataDir)
	manifest, pluginDir, err := loader.LoadPlugin(*nameFlag)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	info, err := manifest.ToInfo(pluginDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Plugin Information:\n")
	fmt.Printf("  Name:        %s\n", info.Name)
	fmt.Printf("  Version:     %s\n", info.Version)
	fmt.Printf("  Description: %s\n", info.Description)
	fmt.Printf("  Author:      %s\n", info.Author)
	fmt.Printf("  Enabled:     %t\n", info.Enabled)
	fmt.Printf("  Priority:    %d\n", info.Priority)
	fmt.Printf("  WASM Size:   %d bytes\n", info.WASMSize)
	fmt.Printf("  Hooks:       %s\n", strings.Join(info.Hooks, ", "))
	fmt.Printf("  Permissions: %s\n", strings.Join(info.Permissions, ", "))

	if len(info.Config) > 0 {
		fmt.Printf("  Config:\n")
		for k, v := range info.Config {
			fmt.Printf("    %s: %v\n", k, v)
		}
	}

	return 0
}

// parseConfigValue attempts to parse a string value into appropriate types.
func parseConfigValue(s string) any {
	// Try bool
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Try int
	var intVal int
	if _, err := fmt.Sscanf(s, "%d", &intVal); err == nil {
		return intVal
	}

	// Try float
	var floatVal float64
	if _, err := fmt.Sscanf(s, "%f", &floatVal); err == nil {
		return floatVal
	}

	// Return as string
	return s
}
