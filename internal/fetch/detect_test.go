// Package fetch provides tests for JavaScript heaviness detection.
// Tests cover detection of JS-heavy pages via SPA indicators, framework markers, and noscript tags.
// Does NOT test actual page rendering or dynamic content evaluation.
package fetch

import (
	"testing"
)

func TestDetectJSHeaviness(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected bool // isHeavy
	}{
		{
			name:     "static content",
			html:     `<html><body><h1>Hello</h1><p>Just some text.</p></body></html>`,
			expected: false,
		},
		{
			name:     "spa root div",
			html:     `<html><body><div id="root"></div></body></html>`,
			expected: true,
		},
		{
			name:     "nextjs data",
			html:     `<html><body><script>window.__NEXT_DATA__ = {}</script></body></html>`,
			expected: true,
		},
		{
			name:     "noscript warning",
			html:     `<html><body><noscript>You need to enable JavaScript to run this app.</noscript></body></html>`,
			expected: true,
		},
		{
			name:     "react root",
			html:     `<html><body><div data-reactroot></div></body></html>`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := DetectJSHeaviness(tt.html)
			if got := IsJSHeavy(h, 0.5); got != tt.expected {
				t.Errorf("IsJSHeavy() = %v, want %v (score: %.2f, reasons: %v)", got, tt.expected, h.Score, h.Reasons)
			}
		})
	}
}
