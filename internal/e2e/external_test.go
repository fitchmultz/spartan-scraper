// Package e2e provides end-to-end integration tests for authenticated scraping flows.
//
// Purpose:
//   - Validate auth-profile login flows and HTTP basic auth against a deterministic local fixture.
//
// Responsibilities:
//   - Exercise Chromedp form login.
//   - Exercise Playwright form login.
//   - Exercise HTTP basic auth.
//
// Scope:
//   - Authenticated scrape behavior only.
//
// Usage:
//   - Runs as part of go test ./internal/e2e/...
//
// Invariants/Assumptions:
//   - Uses the local testsite fixture instead of third-party websites.
//   - Secure pages include the stable marker text \"Secure Area\" on success.
package e2e

import (
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/testsite"
)

func TestAuthTargets(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	env := baseEnv(dataDir)
	env = append(env,
		"RATE_LIMIT_QPS=2",
		"RATE_LIMIT_BURST=4",
		"USER_AGENT=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	)
	site := testsite.Start(t)

	runOK(t, env, "auth", "set",
		"--name", "chromedp-local",
		"--login-url", site.URL("/login/chromedp"),
		"--login-user-selector", "#username",
		"--login-pass-selector", "#password",
		"--login-submit-selector", "button[type=submit]",
		"--login-user", testsite.ChromedpUsername,
		"--login-pass", testsite.ChromedpPassword,
	)
	chromedpOut := filepath.Join(outDir, "chromedp.json")
	runUntilContains(t, env, chromedpOut, "Secure Area", 3,
		"scrape",
		"--url", site.URL("/secure/chromedp"),
		"--headless",
		"--auth-profile", "chromedp-local",
		"--wait",
		"--wait-timeout", "90",
		"--timeout", "30",
		"--out", chromedpOut,
	)

	runOK(t, env, "auth", "set",
		"--name", "playwright-local",
		"--login-url", site.URL("/login/playwright"),
		"--login-user-selector", "#username",
		"--login-pass-selector", "#password",
		"--login-submit-selector", "#submit-login",
		"--login-user", testsite.PlaywrightUsername,
		"--login-pass", testsite.PlaywrightPassword,
	)
	playwrightOut := filepath.Join(outDir, "playwright.json")
	runUntilContains(t, env, playwrightOut, "Secure Area", 3,
		"scrape",
		"--url", site.URL("/secure/playwright"),
		"--headless",
		"--playwright",
		"--auth-profile", "playwright-local",
		"--wait",
		"--wait-timeout", "90",
		"--timeout", "30",
		"--out", playwrightOut,
	)

	basicOut := filepath.Join(outDir, "basic-auth.json")
	runUntilContains(t, env, basicOut, "authorized", 3,
		"scrape",
		"--url", site.URL("/auth/basic"),
		"--auth-basic", testsite.BasicAuthUsername+":"+testsite.BasicAuthPassword,
		"--wait",
		"--wait-timeout", "60",
		"--timeout", "30",
		"--out", basicOut,
	)
}
