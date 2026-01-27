package config

import (
	"os"
	"sync"
	"testing"
)

func TestLoad(t *testing.T) {
	// Clear env vars that might interfere
	os.Unsetenv("PORT")
	os.Unsetenv("DATA_DIR")

	cfg := Load()
	if cfg.Port != "8741" {
		t.Errorf("expected default port 8741, got %s", cfg.Port)
	}

	os.Setenv("PORT", "9999")
	cfg = Load()
	if cfg.Port != "9999" {
		t.Errorf("expected port 9999, got %s", cfg.Port)
	}
}

func TestGetenvBool(t *testing.T) {
	tests := []struct {
		val      string
		expected bool
	}{
		{"1", true},
		{"true", true},
		{"yes", true},
		{"y", true},
		{"0", false},
		{"false", false},
		{"", false}, // fallback
	}

	for _, tt := range tests {
		os.Setenv("TEST_BOOL", tt.val)
		got := getenvBool("TEST_BOOL", false)
		if got != tt.expected {
			t.Errorf("getenvBool(%q) = %v; want %v", tt.val, got, tt.expected)
		}
	}
}

func TestNormalizeAuthKeySuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"USER_AGENT", "USER-AGENT"},
		{"X__API__KEY", "X-API-KEY"},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeAuthKeySuffix(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeAuthKeySuffix(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestConfig_ConcurrentReadIsSafe(t *testing.T) {
	// This test is primarily validated by running with the race detector:
	//   go test -race ./...
	//
	// The key invariant: config.Config is treated as immutable after Load()
	// and is therefore safe for concurrent read access.

	// Ensure stable inputs regardless of local .env contents.
	dataDir := t.TempDir()
	setEnv := func(k, v string) {
		t.Helper()
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("failed to set env %s: %v", k, err)
		}
		t.Cleanup(func() { _ = os.Unsetenv(k) })
	}

	setEnv("PORT", "8741")
	setEnv("DATA_DIR", dataDir)

	// Force AuthOverrides to have non-nil maps so the test exercises map reads too.
	setEnv("AUTH_HEADER_X__API__KEY", "abc123")
	setEnv("AUTH_COOKIE_SESSION", "sess123")

	cfg := Load()

	const goroutines = 64
	const iterations = 2000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = cfg.Port
				_ = cfg.DataDir
				_ = cfg.UserAgent
				_ = cfg.MaxConcurrency
				_ = cfg.RequestTimeoutSecs
				_ = cfg.RateLimitQPS
				_ = cfg.RateLimitBurst
				_ = cfg.MaxRetries
				_ = cfg.RetryBaseMs
				_ = cfg.MaxResponseBytes
				_ = cfg.UsePlaywright
				_ = cfg.LogLevel
				_ = cfg.LogFormat

				// Map reads must remain read-only; concurrent reads are safe if there are no writes.
				if cfg.AuthOverrides.Headers != nil {
					_ = cfg.AuthOverrides.Headers["X-API-KEY"]
					_ = len(cfg.AuthOverrides.Headers)
				}
				if cfg.AuthOverrides.Cookies != nil {
					_ = cfg.AuthOverrides.Cookies["SESSION"]
					_ = len(cfg.AuthOverrides.Cookies)
				}
			}
		}()
	}

	wg.Wait()
}
