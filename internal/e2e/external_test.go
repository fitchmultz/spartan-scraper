// Package e2e provides end-to-end integration tests for external authentication mechanisms.
// Tests cover form-based login with Chromedp and Playwright, plus HTTP basic authentication.
// Does NOT test internal job management, API workflows, or web frontend.
package e2e

import (
	"path/filepath"
	"testing"
)

func TestExternalAuthTargets(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	env := baseEnv(dataDir)
	env = append(env,
		"RATE_LIMIT_QPS=2",
		"RATE_LIMIT_BURST=4",
		"USER_AGENT=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	)

	runOK(t, env, "auth", "set",
		"--name", "herokuapp",
		"--login-url", "https://the-internet.herokuapp.com/login",
		"--login-user-selector", "#username",
		"--login-pass-selector", "#password",
		"--login-submit-selector", "button[type=submit]",
		"--login-user", "tomsmith",
		"--login-pass", "SuperSecretPassword!",
	)
	herokuOut := filepath.Join(outDir, "herokuapp.json")
	runUntilContains(t, env, herokuOut, "Secure Area", 3,
		"scrape",
		"--url", "https://the-internet.herokuapp.com/secure",
		"--headless",
		"--auth-profile", "herokuapp",
		"--wait",
		"--wait-timeout", "90",
		"--timeout", "30",
		"--out", herokuOut,
	)

	runOK(t, env, "auth", "set",
		"--name", "expandtesting",
		"--login-url", "https://practice.expandtesting.com/login",
		"--login-user-selector", "#username",
		"--login-pass-selector", "#password",
		"--login-submit-selector", "#submit-login",
		"--login-user", "practice",
		"--login-pass", "SuperSecretPassword!",
	)
	expandOut := filepath.Join(outDir, "expandtesting.json")
	runUntilContains(t, env, expandOut, "Secure Area", 3,
		"scrape",
		"--url", "https://practice.expandtesting.com/secure",
		"--headless",
		"--playwright",
		"--auth-profile", "expandtesting",
		"--wait",
		"--wait-timeout", "90",
		"--timeout", "30",
		"--out", expandOut,
	)

	httpbinOut := filepath.Join(outDir, "httpbin-basic.json")
	runUntilContains(t, env, httpbinOut, "authorized", 3,
		"scrape",
		"--url", "https://httpbin.dev/basic-auth/user/passwd",
		"--auth-basic", "user:passwd",
		"--wait",
		"--wait-timeout", "60",
		"--timeout", "30",
		"--out", httpbinOut,
	)
}
