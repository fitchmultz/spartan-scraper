// Package scheduler enforces automated export destination policy.
//
// Purpose:
// - Keep recurring local export destinations constrained to the repo-managed DATA_DIR/exports root.
//
// Responsibilities:
// - Reject absolute or escaping local destination templates during schedule authoring.
// - Resolve rendered local export paths for runtime writes under the same policy.
// - Centralize the operator-facing policy message shared by validation and execution.
//
// Scope:
// - Automated export-schedule local destination policy only.
//
// Usage:
// - Called by export schedule storage validation and the live export trigger.
//
// Invariants/Assumptions:
// - Persisted local export templates are relative to DATA_DIR.
// - Valid automated local destinations always stay within DATA_DIR/exports.
// - Policy failures use a stable operator-facing message without leaking host paths.
package scheduler

import (
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

const automatedLocalExportRootPolicy = "DATA_DIR/exports"

func resolveAutomatedLocalExportDestination(dataDir string, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", apperrors.Validation("local_path or path_template is required for local destination")
	}
	if filepath.IsAbs(requested) {
		return "", apperrors.Permission("local destination must stay within " + automatedLocalExportRootPolicy)
	}

	resolvedPath, err := fsutil.ResolvePathWithinRoot(dataDir, requested)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindPermission) {
			return "", apperrors.Permission("local destination must stay within " + automatedLocalExportRootPolicy)
		}
		return "", err
	}

	exportsRoot, err := fsutil.ResolvePathWithinRoot(dataDir, "exports")
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to resolve automated export root", err)
	}

	rel, err := filepath.Rel(exportsRoot, resolvedPath)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to validate automated export destination", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", apperrors.Permission("local destination must stay within " + automatedLocalExportRootPolicy)
	}

	return resolvedPath, nil
}

func validateAutomatedLocalExportDestination(dataDir string, requested string) error {
	_, err := resolveAutomatedLocalExportDestination(dataDir, requested)
	if err == nil {
		return nil
	}
	if apperrors.IsKind(err, apperrors.KindPermission) {
		return apperrors.Validation(apperrors.SafeMessage(err))
	}
	return err
}

func validateStoredExportSchedule(dataDir string, schedule ExportSchedule) error {
	if err := ValidateExportSchedule(schedule); err != nil {
		return err
	}
	if schedule.Export.DestinationType != "local" {
		return nil
	}
	if strings.TrimSpace(schedule.Export.LocalPath) != "" {
		if err := validateAutomatedLocalExportDestination(dataDir, schedule.Export.LocalPath); err != nil {
			return err
		}
	}
	if strings.TrimSpace(schedule.Export.PathTemplate) != "" {
		if err := validateAutomatedLocalExportDestination(dataDir, schedule.Export.PathTemplate); err != nil {
			return err
		}
	}
	return nil
}
