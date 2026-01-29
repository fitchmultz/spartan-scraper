package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestLoad(t *testing.T) {
	dataDir := t.TempDir()
	os.Unsetenv("PORT")
	os.Setenv("DATA_DIR", dataDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Port != "8741" {
		t.Errorf("expected default port 8741, got %s", cfg.Port)
	}

	os.Setenv("PORT", "9999")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
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

	// Force AuthOverrides to have non-nil maps so that the test exercises map reads too.
	setEnv("AUTH_HEADER_X__API__KEY", "abc123")
	setEnv("AUTH_COOKIE_SESSION", "sess123")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

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

func TestValidateDataDir_Success(t *testing.T) {
	dataDir := t.TempDir()
	err := validateDataDir(dataDir)
	if err != nil {
		t.Fatalf("validateDataDir(%q) failed: %v", dataDir, err)
	}
}

func TestValidateDataDir_ReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping read-only directory test on Windows")
	}

	dataDir := t.TempDir()

	// Make directory read-only
	if err := os.Chmod(dataDir, 0o444); err != nil {
		t.Fatalf("failed to make directory read-only: %v", err)
	}

	// Restore permissions in cleanup
	t.Cleanup(func() {
		_ = os.Chmod(dataDir, 0o755)
	})

	err := validateDataDir(dataDir)
	if err == nil {
		t.Fatal("expected error for read-only directory, got nil")
	}

	if !apperrors.IsKind(err, apperrors.KindPermission) {
		t.Errorf("expected KindPermission error, got: %v", err)
	}
}

func TestValidateDataDir_NonExistentPath(t *testing.T) {
	// validateDataDir creates the directory if it doesn't exist,
	// so this test verifies that it successfully creates nested paths.
	nonExistentPath := filepath.Join(t.TempDir(), "nonexistent", "nested", "path")

	err := validateDataDir(nonExistentPath)
	if err != nil {
		t.Fatalf("validateDataDir(%q) failed: %v", nonExistentPath, err)
	}

	// Verify the directory was created
	if _, err := os.Stat(nonExistentPath); err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
}

func TestLoad_WithInvalidDataDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping read-only directory test on Windows")
	}

	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	// Make directory read-only
	if err := os.Chmod(dataDir, 0o444); err != nil {
		t.Fatalf("failed to make directory read-only: %v", err)
	}

	// Restore permissions in cleanup
	t.Cleanup(func() {
		_ = os.Chmod(dataDir, 0o755)
	})

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for read-only DATA_DIR, got nil")
	}

	if !apperrors.IsKind(err, apperrors.KindPermission) {
		t.Errorf("expected KindPermission error, got: %v", err)
	}
}

func TestLoad_WithValidDataDir(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.DataDir != dataDir {
		t.Errorf("expected DataDir %q, got %q", dataDir, cfg.DataDir)
	}
}

func TestLoad_CreatesDirectoryIfNeeded(t *testing.T) {
	parentDir := t.TempDir()
	nonExistentDataDir := filepath.Join(parentDir, "newdata", "nested")

	t.Setenv("DATA_DIR", nonExistentDataDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.DataDir != nonExistentDataDir {
		t.Errorf("expected DataDir %q, got %q", nonExistentDataDir, cfg.DataDir)
	}

	// Verify directory was created
	if _, err := os.Stat(nonExistentDataDir); err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
}
