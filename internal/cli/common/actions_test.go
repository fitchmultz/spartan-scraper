// Package common tests shared CLI rendering helpers.
//
// Purpose:
// - Verify CLI recovery actions render in one consistent translated format.
//
// Responsibilities:
// - Assert one-click actions become CLI commands.
// - Assert indentation and bullet formatting stay stable.
// - Prevent output drift across health and management commands.
//
// Scope:
// - Shared CLI action rendering only.
//
// Usage:
// - Run with `go test ./internal/cli/common`.
//
// Invariants/Assumptions:
// - Empty actions should be skipped.
// - Rendered actions should stay concise and operator-readable.
package common

import (
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
)

func TestWriteRecommendedActionsTranslatesAndFormatsBullets(t *testing.T) {
	var builder strings.Builder
	WriteRecommendedActions(&builder, "  ", []api.RecommendedAction{
		{
			Label: "Re-check browser tooling",
			Kind:  api.ActionKindOneClick,
			Value: api.DiagnosticActionPath(api.DiagnosticTargetBrowser),
		},
		{
			Label: "Copy help URL",
			Kind:  api.ActionKindCopy,
			Value: "https://example.com/help",
		},
		{},
	}, "spartan")

	rendered := builder.String()
	if !strings.Contains(rendered, "  - Re-check browser tooling: spartan health --check browser") {
		t.Fatalf("expected translated diagnostic command, got %q", rendered)
	}
	if !strings.Contains(rendered, "  - Copy help URL: https://example.com/help") {
		t.Fatalf("expected copy action bullet, got %q", rendered)
	}
	if strings.Contains(rendered, "Next step:") {
		t.Fatalf("expected unified bullet format without legacy prefix, got %q", rendered)
	}
}
