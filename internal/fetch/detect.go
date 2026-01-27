// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"strings"
)

// DetectJSHeaviness analyzes HTML content to determine if it requires JavaScript to render meaningful content.
func DetectJSHeaviness(html string) JSHeaviness {
	h := JSHeaviness{
		Reasons: []string{},
	}

	// 1. Check for common SPA root elements
	spaRoots := []string{
		`id="root"`, `id="app"`, `id="__next"`, `id="__nuxt"`,
		`data-reactroot`, `ng-version`,
	}
	for _, root := range spaRoots {
		if strings.Contains(html, root) {
			h.RootDivSignals++
			h.Score += 0.6
			h.Reasons = append(h.Reasons, "found SPA root: "+root)
		}
	}

	// 2. Check for "requires javascript" messages
	noScriptSignals := []string{
		"enable javascript",
		"requires javascript",
		"javascript is disabled",
		"without javascript",
	}
	lowerHTML := strings.ToLower(html)
	for _, sig := range noScriptSignals {
		if strings.Contains(lowerHTML, sig) {
			h.Score += 0.5
			h.Reasons = append(h.Reasons, "found noscript warning: "+sig)
		}
	}

	// 3. Count script tags vs content length
	h.ScriptTagCount = strings.Count(lowerHTML, "<script")
	h.BodyTextLength = len(html)

	// High script count relative to short content usually means data is in JS
	if h.ScriptTagCount > 5 && h.BodyTextLength < 5000 {
		h.Score += 0.3
		h.Reasons = append(h.Reasons, "high script/content ratio")
	}

	// 4. Framework specific signals
	frameworks := []string{
		"window.__INITIAL_STATE__",
		"window.__NEXT_DATA__",
		"window.__NUXT__",
	}
	for _, fw := range frameworks {
		if strings.Contains(html, fw) {
			h.FrameworkSignals++
			h.Score += 0.6
			h.Reasons = append(h.Reasons, "found framework data hydration: "+fw)
		}
	}

	return h
}

// IsJSHeavy determines if the page is JS-heavy based on the score and a threshold.
// Default threshold is usually around 0.5.
func IsJSHeavy(js JSHeaviness, threshold float64) bool {
	if threshold <= 0 {
		threshold = 0.5
	}
	return js.Score >= threshold
}
