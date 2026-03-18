// Package server contains CLI startup preflight helpers for long-running services.
//
// Purpose:
// - Inspect persisted local state before boot and describe guided recovery when the server must start in setup mode.
//
// Responsibilities:
// - Detect legacy or unsupported data directories.
// - Build structured setup metadata for the API health surface.
// - Render operator-facing setup guidance for CLI flows.
//
// Scope:
// - Startup inspection only; server lifecycle and HTTP handling live in sibling files.
//
// Usage:
// - Called by `RunServer` before normal boot and by `RunHealth` when the API is not already running.
//
// Invariants/Assumptions:
// - Legacy or unsupported storage should never be ignored silently.
// - Guided recovery copy must always include a concrete next step.
package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

const defaultResetBackupDir = "output/cutover"

func buildSetupRecoveryActions(commandName string) []api.RecommendedAction {
	resetCommand := fmt.Sprintf("%s reset-data", commandName)
	return []api.RecommendedAction{
		{
			Label: "Archive and recreate the data directory",
			Kind:  api.ActionKindCommand,
			Value: resetCommand,
		},
		{
			Label: "Copy reset command",
			Kind:  api.ActionKindCopy,
			Value: resetCommand,
		},
		{
			Label: "Use a different empty data directory",
			Kind:  api.ActionKindEnv,
			Value: "Set DATA_DIR to a different empty directory before starting the server again.",
		},
		{
			Label: "Copy alternate startup example",
			Kind:  api.ActionKindCopy,
			Value: fmt.Sprintf("DATA_DIR=/path/to/new-empty-dir %s server", commandName),
		},
	}
}

func inspectStartupPreflight(cfg config.Config, commandName string) (*api.SetupStatus, error) {
	inspection, err := store.InspectDataDir(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	actions := buildSetupRecoveryActions(commandName)

	switch inspection.Status {
	case store.DataDirStatusLegacy:
		return &api.SetupStatus{
			Required: true,
			Code:     "legacy_data_dir",
			Title:    "Stored data needs a one-time reset",
			Message: fmt.Sprintf(
				"Detected legacy persisted state at %s. Spartan is running in setup mode so you can recover intentionally instead of failing only in the terminal.",
				cfg.DataDir,
			),
			DataDir: cfg.DataDir,
			Actions: actions,
		}, nil
	case store.DataDirStatusUnsupported:
		return &api.SetupStatus{
			Required:      true,
			Code:          "unsupported_storage_schema",
			Title:         "Stored data uses an unsupported schema",
			Message:       fmt.Sprintf("Detected schema %q in %s. Spartan is running in setup mode so the issue is visible in-product and recoverable.", inspection.SchemaVersion, cfg.DataDir),
			DataDir:       cfg.DataDir,
			SchemaVersion: inspection.SchemaVersion,
			Actions:       actions,
		}, nil
	default:
		return nil, nil
	}
}

func renderSetupStatus(status *api.SetupStatus, commandName string) string {
	if status == nil || !status.Required {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(status.Title)
	builder.WriteString(".\n\n")
	builder.WriteString(status.Message)
	builder.WriteString("\n\nRecovery:\n")
	builder.WriteString(fmt.Sprintf("  1. Run %s reset-data\n", commandName))
	builder.WriteString(fmt.Sprintf("     This archives the current data directory to %s and recreates %s.\n", defaultResetBackupDir, status.DataDir))
	builder.WriteString(fmt.Sprintf("  2. Start the server again with %s server\n", commandName))
	builder.WriteString("\nAlternative:\n")
	builder.WriteString("  Set DATA_DIR to a different empty directory before starting the server.\n")
	return builder.String()
}

func currentCommandName() string {
	command := strings.TrimSpace(os.Args[0])
	if command == "" {
		return "spartan"
	}

	if strings.Contains(command, string(filepath.Separator)+"go-build") {
		return "go run ./cmd/spartan"
	}

	if strings.HasPrefix(command, ".") {
		return command
	}

	if base := filepath.Base(command); base != "" && base != "." {
		return base
	}
	return "spartan"
}
