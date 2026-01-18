package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	runOK(t, env, "scrape",
		"--url", "https://the-internet.herokuapp.com/secure",
		"--headless",
		"--auth-profile", "herokuapp",
		"--wait",
		"--wait-timeout", "90",
		"--timeout", "30",
		"--out", herokuOut,
	)
	assertJSONContains(t, herokuOut, "Secure Area")

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
	runOK(t, env, "scrape",
		"--url", "https://practice.expandtesting.com/secure",
		"--headless",
		"--playwright",
		"--auth-profile", "expandtesting",
		"--wait",
		"--wait-timeout", "90",
		"--timeout", "30",
		"--out", expandOut,
	)
	assertJSONContains(t, expandOut, "Secure Area")

	httpbinOut := filepath.Join(outDir, "httpbin-basic.json")
	runScrapeUntilContains(t, env, httpbinOut, "authorized", 3,
		"scrape",
		"--url", "https://httpbin.dev/basic-auth/user/passwd",
		"--auth-basic", "user:passwd",
		"--wait",
		"--wait-timeout", "60",
		"--timeout", "30",
		"--out", httpbinOut,
	)
}

func runScrapeUntilContains(t *testing.T, env []string, outPath string, needle string, attempts int, args ...string) {
	t.Helper()
	var lastErr error
	for i := 0; i < attempts; i++ {
		runOK(t, env, args...)
		data, err := os.ReadFile(outPath)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		text := string(data)
		if strings.Contains(text, needle) {
			return
		}
		lastErr = fmt.Errorf("missing %q in %s", needle, strings.TrimSpace(text))
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("scrape verification failed: %v", lastErr)
	}
	t.Fatalf("scrape verification failed")
}
