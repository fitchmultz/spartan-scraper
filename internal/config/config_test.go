// Package config_test provides tests for the config package.
//
// Responsibilities:
//   - Tests configuration loading from environment variables
//   - Tests helper functions (getenvBool, getenvInt, getenvInt64)
//   - Tests auth key normalization
//   - Tests concurrent read safety of Config (race detector validation)
//   - Tests data directory validation and creation
//
// Does NOT handle:
//   - Production configuration loading (see config.go)
//   - Logger initialization tests (see log_*_test.go files)
//   - Cross-package integration tests
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestGetenvInt_InvalidValue(t *testing.T) {
	os.Setenv("TEST_INT_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_INT_INVALID")

	result := getenvInt("TEST_INT_INVALID", 10)

	if result != 10 {
		t.Errorf("getenvInt with invalid value should return fallback 10, got %d", result)
	}
}

func TestGetenvInt_ValidValue(t *testing.T) {
	os.Setenv("TEST_INT_VALID", "123")
	defer os.Unsetenv("TEST_INT_VALID")

	result := getenvInt("TEST_INT_VALID", 10)

	if result != 123 {
		t.Errorf("getenvInt with valid value should return 123, got %d", result)
	}
}

func TestGetenvInt64_InvalidValue(t *testing.T) {
	os.Setenv("TEST_INT64_INVALID", "xyz")
	defer os.Unsetenv("TEST_INT64_INVALID")

	result := getenvInt64("TEST_INT64_INVALID", 42)

	if result != 42 {
		t.Errorf("getenvInt64 with invalid value should return fallback 42, got %d", result)
	}
}

func TestGetenvInt64_ValidValue(t *testing.T) {
	os.Setenv("TEST_INT64_VALID", "9876543210")
	defer os.Unsetenv("TEST_INT64_VALID")

	result := getenvInt64("TEST_INT64_VALID", 0)

	if result != 9876543210 {
		t.Errorf("getenvInt64 with valid value should return 9876543210, got %d", result)
	}
}

func TestGetenvBool_InvalidValue(t *testing.T) {
	os.Setenv("TEST_BOOL_INVALID", "maybe")
	defer os.Unsetenv("TEST_BOOL_INVALID")

	result := getenvBool("TEST_BOOL_INVALID", true)

	if result != true {
		t.Errorf("getenvBool with invalid value should return fallback true, got %t", result)
	}
}

func TestGetenvBool_ValidValues(t *testing.T) {
	trueValues := []string{"1", "true", "yes", "y", "TRUE", "YES", "Y"}
	for _, val := range trueValues {
		os.Setenv("TEST_BOOL_TRUE", val)
		result := getenvBool("TEST_BOOL_TRUE", false)
		if result != true {
			t.Errorf("getenvBool(%q) should return true, got %t", val, result)
		}
	}

	falseValues := []string{"0", "false", "no", "n", "FALSE", "NO", "N"}
	for _, val := range falseValues {
		os.Setenv("TEST_BOOL_FALSE", val)
		result := getenvBool("TEST_BOOL_FALSE", true)
		if result != false {
			t.Errorf("getenvBool(%q) should return false, got %t", val, result)
		}
	}
	os.Unsetenv("TEST_BOOL_TRUE")
	os.Unsetenv("TEST_BOOL_FALSE")
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

func TestHasExplicitRetentionLimitOverrides(t *testing.T) {
	keys := []string{
		"RETENTION_JOB_DAYS",
		"RETENTION_CRAWL_STATE_DAYS",
		"RETENTION_MAX_JOBS",
		"RETENTION_MAX_STORAGE_GB",
	}

	for _, key := range keys {
		k := key
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("failed to unset %s: %v", k, err)
		}
		t.Cleanup(func() {
			_ = os.Unsetenv(k)
		})
	}

	if hasExplicitRetentionLimitOverrides() {
		t.Fatal("expected no explicit retention overrides when env vars are unset")
	}

	if err := os.Setenv("RETENTION_MAX_JOBS", "250"); err != nil {
		t.Fatalf("failed to set RETENTION_MAX_JOBS: %v", err)
	}

	if !hasExplicitRetentionLimitOverrides() {
		t.Fatal("expected explicit retention override detection when retention env var is set")
	}
}

func TestHasExplicitRetentionLimitOverrides_IgnoresDefaultValues(t *testing.T) {
	t.Setenv("RETENTION_JOB_DAYS", "30")
	t.Setenv("RETENTION_CRAWL_STATE_DAYS", "90")
	t.Setenv("RETENTION_MAX_JOBS", "10000")
	t.Setenv("RETENTION_MAX_STORAGE_GB", "10")

	if hasExplicitRetentionLimitOverrides() {
		t.Fatal("expected default retention values to not count as explicit overrides")
	}
}

func TestGetenv_NormalizesInlineComments(t *testing.T) {
	t.Setenv("TEST_VALUE_EMPTY_COMMENT", "   # comment only")
	t.Setenv("TEST_VALUE_TRAILING_COMMENT", "60   # trailing comment")
	t.Setenv("TEST_VALUE_HASH_LITERAL", "abc#123")

	if got := getenv("TEST_VALUE_EMPTY_COMMENT", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback for comment-only value, got %q", got)
	}
	if got := getenvInt("TEST_VALUE_TRAILING_COMMENT", 10); got != 60 {
		t.Fatalf("expected trailing comment to be stripped before int parse, got %d", got)
	}
	if got := getenv("TEST_VALUE_HASH_LITERAL", "fallback"); got != "abc#123" {
		t.Fatalf("expected literal hash value to be preserved, got %q", got)
	}
}

func TestLoad_IgnoresLegacyAIProviderEnv(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("AI_PROVIDER", "openai")
	t.Setenv("AI_API_KEY", "sk-test")

	_, err := Load()
	if err == nil {
		t.Fatal("expected legacy AI_* env vars to hard fail")
	}
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestDefaultAIRoutingConfig_UsesPreferredRouteOrder(t *testing.T) {
	routing := DefaultAIRoutingConfig()
	want := []string{"kimi-coding/k2p5", "zai/glm-5", "openai-codex/gpt-5.4"}

	for _, capability := range []string{
		AICapabilityExtractNatural,
		AICapabilityExtractSchema,
		AICapabilityTemplateGeneration,
		AICapabilityRenderProfile,
		AICapabilityPipelineJS,
		AICapabilityResearchRefine,
		AICapabilityExportShape,
		AICapabilityTransformGenerate,
	} {
		if got := routing.RoutesFor(capability); !reflect.DeepEqual(got, want) {
			t.Fatalf("RoutesFor(%q) = %#v, want %#v", capability, got, want)
		}
	}
}

func TestLoad_LoadsPIConfigPathOverrides(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("PI_ENABLED", "true")
	configPath := filepath.Join(dataDir, "pi-config.json")
	configFile := map[string]any{
		"mode": "fixture",
		"routes": map[string][]string{
			AICapabilityTemplateGeneration: {"kimi-coding/k2p5", "zai/glm-5"},
		},
	}
	data, err := json.Marshal(configFile)
	if err != nil {
		t.Fatalf("failed to marshal pi config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("failed to write pi config: %v", err)
	}
	t.Setenv("PI_CONFIG_PATH", configPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.AI.Enabled {
		t.Fatal("expected PI_ENABLED=true to enable AI")
	}
	if cfg.AI.Mode != "fixture" {
		t.Fatalf("expected mode override from PI_CONFIG_PATH, got %q", cfg.AI.Mode)
	}
	if got := cfg.AI.Routing.RoutesFor(AICapabilityTemplateGeneration); len(got) != 2 || got[0] != "kimi-coding/k2p5" {
		t.Fatalf("unexpected template routes: %#v", got)
	}
}

func TestLoad_AllowsEmptyProxyPoolFileOverride(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("PROXY_POOL_FILE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.ProxyPoolFile != "" {
		t.Fatalf("expected explicit empty PROXY_POOL_FILE to disable proxy pool loading, got %q", cfg.ProxyPoolFile)
	}
}

func TestLoad_CollectsStartupNotices(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("RETENTION_ENABLED", "false")
	t.Setenv("RETENTION_MAX_JOBS", "250")
	t.Setenv("ADAPTIVE_RATE_LIMIT", "true")
	t.Setenv("ADAPTIVE_MIN_QPS", "not-a-number")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(cfg.StartupNotices) == 0 {
		t.Fatal("expected startup notices to be collected")
	}

	ids := make(map[string]bool, len(cfg.StartupNotices))
	for _, notice := range cfg.StartupNotices {
		ids[notice.ID] = true
	}
	if !ids["invalid-env-adaptive-min-qps"] {
		t.Fatalf("expected invalid adaptive env notice, got %#v", cfg.StartupNotices)
	}
	if !ids["retention-disabled-with-limits"] {
		t.Fatalf("expected retention warning notice, got %#v", cfg.StartupNotices)
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
