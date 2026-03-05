// Package fetch provides tests for proxy-pool persistence helpers.
//
// Purpose:
//   - Validate proxy-pool persistence behavior for strict loads and optional startup loads.
//
// Responsibilities:
//   - Cover missing-file, invalid-JSON, and successful-load cases.
//   - Verify default optional paths stay silent while explicit misconfigurations fail cleanly.
//
// Scope:
//   - Proxy-pool persistence helpers only.
//
// Usage:
//   - Runs as part of go test ./internal/fetch/...
//
// Invariants/Assumptions:
//   - Optional default startup paths do not emit hard errors when absent.
//   - Explicit configured paths still surface not_found errors.
package fetch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestLoadProxyPoolFromFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadProxyPoolFromFile(filepath.Join(tmpDir, "nonexistent.json"))
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		if !apperrors.IsKind(err, apperrors.KindNotFound) {
			t.Errorf("Expected not_found error, got %v", apperrors.KindOf(err))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, err := LoadProxyPoolFromFile(path)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
		if !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Errorf("Expected validation error, got %v", apperrors.KindOf(err))
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		config := ProxyPoolConfig{
			DefaultStrategy: "random",
			HealthCheck: HealthCheckConfig{
				Enabled:         true,
				IntervalSeconds: 30,
			},
			Proxies: []ProxyEntry{
				{ID: "proxy-1", URL: "http://proxy1.example.com:8080", Region: "us-east"},
				{ID: "proxy-2", URL: "http://proxy2.example.com:8080", Region: "us-west", Weight: 5},
			},
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		path := filepath.Join(tmpDir, "proxy_pool.json")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		pool, err := LoadProxyPoolFromFile(path)
		if err != nil {
			t.Errorf("LoadProxyPoolFromFile failed: %v", err)
		}
		if pool == nil {
			t.Fatal("Expected non-nil pool")
		}
		defer pool.Stop()

		if pool.GetStrategy() != RotationRandom {
			t.Errorf("Expected strategy random, got %v", pool.GetStrategy())
		}

		if count := pool.GetTotalProxyCount(); count != 2 {
			t.Errorf("Expected 2 proxies, got %d", count)
		}
	})
}

func TestProxyPoolFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	defaultPath := filepath.Join(tmpDir, "proxy_pool.json")

	t.Run("missing implicit default path is silent", func(t *testing.T) {
		pool, err := ProxyPoolFromConfig(defaultPath, false)
		if err != nil {
			t.Errorf("Expected no error for missing file, got %v", err)
		}
		if pool != nil {
			t.Error("Expected nil pool for missing file")
		}
	})

	t.Run("missing explicit path returns not found", func(t *testing.T) {
		pool, err := ProxyPoolFromConfig(defaultPath, true)
		if err == nil {
			t.Fatal("Expected error for explicit missing file")
		}
		if !apperrors.IsKind(err, apperrors.KindNotFound) {
			t.Fatalf("Expected not_found error, got %v", apperrors.KindOf(err))
		}
		if pool != nil {
			t.Error("Expected nil pool for missing explicit file")
		}
	})

	t.Run("configured file loads", func(t *testing.T) {
		config := ProxyPoolConfig{
			Proxies: []ProxyEntry{
				{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
			},
		}

		data, _ := json.Marshal(config)
		if err := os.WriteFile(defaultPath, data, 0644); err != nil {
			t.Fatalf("Failed to write proxy pool file: %v", err)
		}

		pool, err := ProxyPoolFromConfig(defaultPath, true)
		if err != nil {
			t.Errorf("ProxyPoolFromConfig failed: %v", err)
		}
		if pool == nil {
			t.Error("Expected non-nil pool")
		} else {
			pool.Stop()
		}
	})
}
