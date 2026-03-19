// Package common provides shared CLI helpers used across command modules.
//
// Purpose:
// - Render operator-facing recovery actions consistently across CLI surfaces.
//
// Responsibilities:
// - Translate API recommended actions into CLI-friendly commands.
// - Format action lists with predictable indentation and bullets.
// - Keep health, proxy-pool, and retention guidance visually aligned.
//
// Scope:
// - Shared CLI action rendering only.
//
// Usage:
// - Call `WriteRecommendedActions` with a strings.Builder or other io.StringWriter.
//
// Invariants/Assumptions:
// - Empty actions should render nothing.
// - One-click API actions must be translated through CLIRecommendedActions before printing.
package common

import (
	"fmt"
	"io"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
)

// WriteRecommendedActions writes translated CLI actions as an indented bullet list.
func WriteRecommendedActions(
	writer io.StringWriter,
	indent string,
	actions []api.RecommendedAction,
	commandName string,
) {
	translated := api.CLIRecommendedActions(actions, commandName)
	for _, action := range translated {
		label := strings.TrimSpace(action.Label)
		value := strings.TrimSpace(action.Value)
		switch {
		case label == "" && value == "":
			continue
		case value == "":
			_, _ = writer.WriteString(fmt.Sprintf("%s- %s\n", indent, label))
		case label == "":
			_, _ = writer.WriteString(fmt.Sprintf("%s- %s\n", indent, value))
		default:
			_, _ = writer.WriteString(fmt.Sprintf("%s- %s: %s\n", indent, label, value))
		}
	}
}
