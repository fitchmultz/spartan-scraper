// Package diff provides content diffing functionality for change detection.
//
// This file contains tests for the diff package.
package diff

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name       string
		oldText    string
		newText    string
		cfg        Config
		wantChange bool
	}{
		{
			name:       "no changes",
			oldText:    "Hello World",
			newText:    "Hello World",
			cfg:        DefaultConfig(),
			wantChange: false,
		},
		{
			name:       "simple change",
			oldText:    "Hello World",
			newText:    "Hello Universe",
			cfg:        DefaultConfig(),
			wantChange: true,
		},
		{
			name:       "added content",
			oldText:    "Line 1",
			newText:    "Line 1\nLine 2",
			cfg:        DefaultConfig(),
			wantChange: true,
		},
		{
			name:       "removed content",
			oldText:    "Line 1\nLine 2",
			newText:    "Line 1",
			cfg:        DefaultConfig(),
			wantChange: true,
		},
		{
			name:       "ignore whitespace no meaningful change",
			oldText:    "Hello   World",
			newText:    "Hello World",
			cfg:        Config{IgnoreWhitespace: true},
			wantChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Generate(tt.oldText, tt.newText, tt.cfg)

			if result.HasChanges != tt.wantChange {
				t.Errorf("HasChanges = %v, want %v", result.HasChanges, tt.wantChange)
			}

			if tt.wantChange {
				if result.ChangesAdded == 0 && result.ChangesRemoved == 0 {
					t.Error("Expected some changes but got none")
				}
			}
		})
	}
}

func TestUnifiedDiff(t *testing.T) {
	oldText := "Line 1\nLine 2\nLine 3"
	newText := "Line 1\nLine 2 Modified\nLine 3"

	diff := UnifiedDiff(oldText, newText, "test.txt")

	if diff == "" {
		t.Error("Expected non-empty diff")
	}

	// Check that diff contains indicators of change
	if !strings.Contains(diff, "@@") {
		t.Error("Expected unified diff format with @@ markers")
	}
}

func TestHTMLDiff(t *testing.T) {
	oldText := "Hello World"
	newText := "Hello Universe"

	html := HTMLDiff(oldText, newText)

	if html == "" {
		t.Error("Expected non-empty HTML diff")
	}

	// Check for expected HTML structure
	if !strings.Contains(html, "diff-container") {
		t.Error("Expected diff-container class")
	}

	if !strings.Contains(html, "diff-removed") {
		t.Error("Expected diff-removed class")
	}

	if !strings.Contains(html, "diff-added") {
		t.Error("Expected diff-added class")
	}
}

func TestInlineDiff(t *testing.T) {
	oldText := "Hello World"
	newText := "Hello Universe"

	html := InlineDiff(oldText, newText)

	if html == "" {
		t.Error("Expected non-empty inline diff")
	}

	// Check for expected HTML structure
	if !strings.Contains(html, "diff-inline") {
		t.Error("Expected diff-inline class")
	}

	if !strings.Contains(html, "<del") {
		t.Error("Expected <del> element for deletions")
	}

	if !strings.Contains(html, "<ins") {
		t.Error("Expected <ins> element for insertions")
	}
}

func TestHasMeaningfulChanges(t *testing.T) {
	tests := []struct {
		name     string
		oldText  string
		newText  string
		expected bool
	}{
		{
			name:     "identical content",
			oldText:  "Hello World",
			newText:  "Hello World",
			expected: false,
		},
		{
			name:     "whitespace only change",
			oldText:  "Hello   World",
			newText:  "Hello World",
			expected: false,
		},
		{
			name:     "actual content change",
			oldText:  "Hello World",
			newText:  "Hello Universe",
			expected: true,
		},
		{
			name:     "leading/trailing whitespace",
			oldText:  "  Hello World  ",
			newText:  "Hello World",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasMeaningfulChanges(tt.oldText, tt.newText)
			if result != tt.expected {
				t.Errorf("HasMeaningfulChanges() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateStyles(t *testing.T) {
	styles := GenerateStyles()

	if styles == "" {
		t.Error("Expected non-empty styles")
	}

	// Check for expected CSS classes
	expectedClasses := []string{
		".diff-container",
		".diff-side",
		".diff-removed",
		".diff-added",
		".diff-inline",
	}

	for _, class := range expectedClasses {
		if !strings.Contains(styles, class) {
			t.Errorf("Expected styles to contain %s", class)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Format != FormatUnified {
		t.Errorf("Default format = %v, want %v", cfg.Format, FormatUnified)
	}

	if cfg.ContextLines != 3 {
		t.Errorf("Default context lines = %d, want 3", cfg.ContextLines)
	}

	if cfg.IgnoreWhitespace != false {
		t.Error("Default IgnoreWhitespace should be false")
	}
}

func TestResultCounts(t *testing.T) {
	oldText := "Hello World Foo Bar"
	newText := "Hello Universe Foo Baz"

	result := Generate(oldText, newText, DefaultConfig())

	if !result.HasChanges {
		t.Error("Expected changes")
	}

	// Should detect changes in "World" -> "Universe" and "Bar" -> "Baz"
	if result.ChangesAdded == 0 {
		t.Error("Expected some added content")
	}

	if result.ChangesRemoved == 0 {
		t.Error("Expected some removed content")
	}

	// Size tracking
	if result.OldSize != len(oldText) {
		t.Errorf("OldSize = %d, want %d", result.OldSize, len(oldText))
	}

	if result.NewSize != len(newText) {
		t.Errorf("NewSize = %d, want %d", result.NewSize, len(newText))
	}
}

func TestEmptyStrings(t *testing.T) {
	// Test with empty old text
	result := Generate("", "New Content", DefaultConfig())
	if !result.HasChanges {
		t.Error("Expected changes when adding to empty")
	}

	// Test with empty new text
	result = Generate("Old Content", "", DefaultConfig())
	if !result.HasChanges {
		t.Error("Expected changes when removing all")
	}

	// Test with both empty
	result = Generate("", "", DefaultConfig())
	if result.HasChanges {
		t.Error("Expected no changes when both empty")
	}
}

func TestHTMLDiffFormats(t *testing.T) {
	oldText := "Line 1\nLine 2"
	newText := "Line 1\nLine 2 Modified"

	tests := []struct {
		name   string
		format Format
	}{
		{"side-by-side", FormatHTMLSideBySide},
		{"inline", FormatHTMLInline},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Format: tt.format}
			result := Generate(oldText, newText, cfg)

			if result.HTMLDiff == "" {
				t.Error("Expected non-empty HTML diff")
			}
		})
	}
}
