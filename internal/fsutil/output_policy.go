// Package fsutil provides secure filesystem helpers for operator-generated artifacts.
//
// Purpose:
// - Resolve operator-provided output paths against explicit allowed roots and write private files safely.
//
// Responsibilities:
// - Normalize requested output paths against a caller-supplied root.
// - Reject path traversal or absolute destinations that escape the allowed root.
// - Persist operator-generated files with private permissions via atomic writes.
//
// Scope:
// - Output-path validation and private file writes only; higher-level export/AI workflows supply policy roots.
//
// Usage:
// - Used by CLI export flows, AI authoring output helpers, and other operator-facing artifact writers.
//
// Invariants/Assumptions:
// - Allowed roots are trusted, caller-controlled directories.
// - Returned paths are absolute, cleaned filesystem paths.
// - Escaping the allowed root is treated as a permission error.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// ResolvePathWithinRoot resolves a requested output path against an allowed root.
func ResolvePathWithinRoot(root string, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", apperrors.Validation("output path is required")
	}

	resolvedRoot, err := resolvePathForContainment(root)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to resolve output root", err)
	}

	candidate := requested
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(resolvedRoot, candidate)
	}
	resolvedCandidate, err := resolvePathForContainment(candidate)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to resolve output path", err)
	}

	rel, err := filepath.Rel(resolvedRoot, resolvedCandidate)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to validate output path", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", apperrors.Permission(fmt.Sprintf("output path must stay within %s", resolvedRoot))
	}

	return resolvedCandidate, nil
}

func resolvePathForContainment(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	missing := make([]string, 0, 4)
	current := absPath
	for {
		_, err := os.Lstat(current)
		if err == nil {
			resolvedCurrent, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolvedCurrent = filepath.Join(resolvedCurrent, missing[i])
			}
			return resolvedCurrent, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return absPath, nil
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

// WritePrivateFileWithinRoot writes a private file after validating the destination root.
func WritePrivateFileWithinRoot(root string, requested string, data []byte) (string, error) {
	resolved, err := ResolvePathWithinRoot(root, requested)
	if err != nil {
		return "", err
	}
	if err := WriteFileAtomic(resolved, data, FileMode); err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to write output file", err)
	}
	return resolved, nil
}
