// Package store provides shared SQLite helpers for Spartan Scraper persistence.
//
// Purpose:
// - Centralize repeated SQLite error classification and wrapping logic.
//
// Responsibilities:
// - Normalize `sql.ErrNoRows` detection across store getters.
// - Provide a single helper for mapping scan/query errors into app errors.
//
// Scope:
// - Internal helpers for the store package only.
//
// Usage:
// - Call wrapScanError for row-scan paths that should map missing rows to not-found.
//
// Invariants/Assumptions:
// - Missing-row checks should use `errors.Is` instead of string matching.
package store

import (
	"database/sql"
	"errors"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func isNoRowsError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func wrapScanError(err error, notFoundMessage string, internalMessage string) error {
	if err == nil {
		return nil
	}
	if isNoRowsError(err) {
		return apperrors.NotFound(notFoundMessage)
	}
	return apperrors.Wrap(apperrors.KindInternal, internalMessage, err)
}
