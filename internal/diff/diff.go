// Package diff provides content diffing functionality for change detection.
//
// This package is responsible for:
// - Generating unified diffs (text format)
// - Generating HTML diffs (side-by-side and inline)
// - Detecting meaningful changes (ignoring whitespace)
//
// This file does NOT handle:
// - Content fetching (fetch package handles this)
// - Storage of diffs (store package handles this)
// - Webhook delivery (webhook package handles this)
//
// Invariants:
// - All functions handle empty strings gracefully
// - HTML output is properly escaped
// - Unified diff format follows standard unified diff conventions
package diff

import (
	"html"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Format represents the output format for diffs.
type Format string

const (
	FormatUnified        Format = "unified"
	FormatHTMLSideBySide Format = "html-side-by-side"
	FormatHTMLInline     Format = "html-inline"
)

// Config holds configuration for diff generation.
type Config struct {
	Format           Format
	ContextLines     int
	IgnoreWhitespace bool
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Format:           FormatUnified,
		ContextLines:     3,
		IgnoreWhitespace: false,
	}
}

// Result contains the diff output and metadata.
type Result struct {
	HasChanges     bool
	UnifiedDiff    string
	HTMLDiff       string
	OldSize        int
	NewSize        int
	ChangesAdded   int
	ChangesRemoved int
}

// Generate creates a diff between old and new content.
func Generate(oldText, newText string, cfg Config) Result {
	oldSize := len(oldText)
	newSize := len(newText)

	// Normalize whitespace if requested
	if cfg.IgnoreWhitespace {
		oldText = normalizeWhitespace(oldText)
		newText = normalizeWhitespace(newText)
	}

	// Check for changes
	hasChanges := oldText != newText

	result := Result{
		HasChanges: hasChanges,
		OldSize:    oldSize,
		NewSize:    newSize,
	}

	if !hasChanges {
		return result
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, false)

	// Count changes
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			result.ChangesAdded += len(diff.Text)
		case diffmatchpatch.DiffDelete:
			result.ChangesRemoved += len(diff.Text)
		}
	}

	// Generate unified diff
	switch cfg.Format {
	case FormatUnified, "":
		result.UnifiedDiff = generateUnifiedDiff(dmp, diffs)
	}

	// Generate HTML diff
	switch cfg.Format {
	case FormatHTMLSideBySide:
		result.HTMLDiff = generateHTMLSideBySideDiff(diffs)
	case FormatHTMLInline:
		result.HTMLDiff = generateHTMLInlineDiff(diffs)
	}

	return result
}

// UnifiedDiff generates a unified text diff between old and new content.
func UnifiedDiff(oldText, newText, filename string) string {
	cfg := DefaultConfig()
	cfg.Format = FormatUnified
	result := Generate(oldText, newText, cfg)
	return result.UnifiedDiff
}

// HTMLDiff generates an HTML side-by-side diff.
func HTMLDiff(oldText, newText string) string {
	cfg := DefaultConfig()
	cfg.Format = FormatHTMLSideBySide
	result := Generate(oldText, newText, cfg)
	return result.HTMLDiff
}

// InlineDiff generates an inline HTML diff with highlights.
func InlineDiff(oldText, newText string) string {
	cfg := DefaultConfig()
	cfg.Format = FormatHTMLInline
	result := Generate(oldText, newText, cfg)
	return result.HTMLDiff
}

// HasMeaningfulChanges checks if content has meaningful changes (ignores whitespace-only changes).
func HasMeaningfulChanges(oldText, newText string) bool {
	cfg := Config{IgnoreWhitespace: true}
	result := Generate(oldText, newText, cfg)
	return result.HasChanges
}

// normalizeWhitespace removes extra whitespace for comparison.
func normalizeWhitespace(s string) string {
	// Replace all whitespace sequences with a single space
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// generateUnifiedDiff creates a unified diff format output.
func generateUnifiedDiff(dmp *diffmatchpatch.DiffMatchPatch, diffs []diffmatchpatch.Diff) string {
	if len(diffs) == 0 {
		return ""
	}

	var result strings.Builder
	patches := dmp.PatchMake(diffs)

	for _, patch := range patches {
		result.WriteString(patch.String())
	}

	return result.String()
}

// generateHTMLSideBySideDiff creates a side-by-side HTML diff.
func generateHTMLSideBySideDiff(diffs []diffmatchpatch.Diff) string {
	var left, right strings.Builder

	left.WriteString(`<div class="diff-side diff-old"><h3>Previous</h3><pre>`)
	right.WriteString(`<div class="diff-side diff-new"><h3>Current</h3><pre>`)

	for _, diff := range diffs {
		text := html.EscapeString(diff.Text)
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			left.WriteString(text)
			right.WriteString(text)
		case diffmatchpatch.DiffDelete:
			left.WriteString(`<span class="diff-removed">`)
			left.WriteString(text)
			left.WriteString(`</span>`)
		case diffmatchpatch.DiffInsert:
			right.WriteString(`<span class="diff-added">`)
			right.WriteString(text)
			right.WriteString(`</span>`)
		}
	}

	left.WriteString(`</pre></div>`)
	right.WriteString(`</pre></div>`)

	return `<div class="diff-container">` + left.String() + right.String() + `</div>`
}

// generateHTMLInlineDiff creates an inline HTML diff with highlights.
func generateHTMLInlineDiff(diffs []diffmatchpatch.Diff) string {
	var result strings.Builder

	result.WriteString(`<div class="diff-inline"><pre>`)

	for _, diff := range diffs {
		text := html.EscapeString(diff.Text)
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			result.WriteString(text)
		case diffmatchpatch.DiffDelete:
			result.WriteString(`<del class="diff-removed">`)
			result.WriteString(text)
			result.WriteString(`</del>`)
		case diffmatchpatch.DiffInsert:
			result.WriteString(`<ins class="diff-added">`)
			result.WriteString(text)
			result.WriteString(`</ins>`)
		}
	}

	result.WriteString(`</pre></div>`)

	return result.String()
}

// GenerateStyles returns CSS styles for HTML diffs.
func GenerateStyles() string {
	return `
<style>
.diff-container {
	display: flex;
	gap: 20px;
	font-family: monospace;
}
.diff-side {
	flex: 1;
	border: 1px solid #ccc;
	border-radius: 4px;
}
.diff-side h3 {
	margin: 0;
	padding: 10px;
	background: #f5f5f5;
	border-bottom: 1px solid #ccc;
}
.diff-side pre {
	margin: 0;
	padding: 10px;
	white-space: pre-wrap;
	word-wrap: break-word;
}
.diff-removed {
	background-color: #fee;
	text-decoration: line-through;
}
.diff-added {
	background-color: #efe;
}
.diff-inline {
	font-family: monospace;
	border: 1px solid #ccc;
	border-radius: 4px;
	padding: 10px;
}
.diff-inline .diff-removed {
	background-color: #fee;
	text-decoration: line-through;
}
.diff-inline .diff-added {
	background-color: #efe;
	text-decoration: underline;
}
</style>
`
}
