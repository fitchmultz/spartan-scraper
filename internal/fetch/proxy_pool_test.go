// Package fetch provides HTTP and headless browser content fetching capabilities.
package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// mockHealthChecker is a mock implementation of HealthChecker for testing.
type mockHealthChecker struct {
	latencyMs int64
	err       error
}

func (m *mockHealthChecker) Check(ctx context.Context, proxy ProxyEntry) (int64, error) {
	return m.latencyMs, m.err
}

func TestProxyEntry_ToProxyConfig(t *testing.T) {
	entry := ProxyEntry{
		ID:       "test-proxy",
		URL:      "http://proxy.example.com:8080",
		Username: "user",
		Password: "pass",
		Region:   "us-east",
		Tags:     []string{"datacenter"},
		Weight:   10,
	}

	config := entry.ToProxyConfig()

	if config.URL != entry.URL {
		t.Errorf("URL mismatch: got %q, want %q", config.URL, entry.URL)
	}
	if config.Username != entry.Username {
		t.Errorf("Username mismatch: got %q, want %q", config.Username, entry.Username)
	}
	if config.Password != entry.Password {
		t.Errorf("Password mismatch: got %q, want %q", config.Password, entry.Password)
	}
}

func TestProxyStats_SuccessRate(t *testing.T) {
	tests := []struct {
		name      string
		stats     ProxyStats
		want      float64
		tolerance float64
	}{
		{
			name:      "no requests",
			stats:     ProxyStats{},
			want:      100.0,
			tolerance: 0.01,
		},
		{
			name: "all success",
			stats: ProxyStats{
				SuccessCount: 10,
				FailureCount: 0,
			},
			want:      100.0,
			tolerance: 0.01,
		},
		{
			name: "all failure",
			stats: ProxyStats{
				SuccessCount: 0,
				FailureCount: 10,
			},
			want:      0.0,
			tolerance: 0.01,
		},
		{
			name: "mixed",
			stats: ProxyStats{
				SuccessCount: 75,
				FailureCount: 25,
			},
			want:      75.0,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stats.SuccessRate()
			if got < tt.want-tt.tolerance || got > tt.want+tt.tolerance {
				t.Errorf("SuccessRate() = %v, want %v (tolerance %v)", got, tt.want, tt.tolerance)
			}
		})
	}
}

func TestRotationStrategy_String(t *testing.T) {
	tests := []struct {
		strategy RotationStrategy
		want     string
	}{
		{RotationRoundRobin, "round_robin"},
		{RotationRandom, "random"},
		{RotationLeastUsed, "least_used"},
		{RotationWeighted, "weighted"},
		{RotationLeastLatency, "least_latency"},
		{RotationStrategy(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.strategy.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRotationStrategy(t *testing.T) {
	tests := []struct {
		input string
		want  RotationStrategy
	}{
		{"round_robin", RotationRoundRobin},
		{"random", RotationRandom},
		{"least_used", RotationLeastUsed},
		{"weighted", RotationWeighted},
		{"least_latency", RotationLeastLatency},
		{"unknown", RotationRoundRobin},
		{"", RotationRoundRobin},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseRotationStrategy(tt.input)
			if got != tt.want {
				t.Errorf("ParseRotationStrategy(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultHealthCheckConfig(t *testing.T) {
	cfg := DefaultHealthCheckConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.IntervalSeconds != 60 {
		t.Errorf("Expected IntervalSeconds = 60, got %d", cfg.IntervalSeconds)
	}
	if cfg.TimeoutSeconds != 10 {
		t.Errorf("Expected TimeoutSeconds = 10, got %d", cfg.TimeoutSeconds)
	}
	if cfg.MaxConsecutiveFails != 3 {
		t.Errorf("Expected MaxConsecutiveFails = 3, got %d", cfg.MaxConsecutiveFails)
	}
	if cfg.RecoveryAfterSeconds != 300 {
		t.Errorf("Expected RecoveryAfterSeconds = 300, got %d", cfg.RecoveryAfterSeconds)
	}
	if cfg.TestURL != "http://httpbin.org/ip" {
		t.Errorf("Expected TestURL = http://httpbin.org/ip, got %q", cfg.TestURL)
	}
}

func TestNewProxyPool_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  ProxyPoolConfig
		wantErr bool
		errKind apperrors.Kind
	}{
		{
			name:    "empty proxies",
			config:  ProxyPoolConfig{Proxies: []ProxyEntry{}},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "missing proxy ID",
			config: ProxyPoolConfig{
				Proxies: []ProxyEntry{
					{ID: "", URL: "http://proxy.example.com:8080"},
				},
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "missing proxy URL",
			config: ProxyPoolConfig{
				Proxies: []ProxyEntry{
					{ID: "proxy-1", URL: ""},
				},
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "duplicate proxy ID",
			config: ProxyPoolConfig{
				Proxies: []ProxyEntry{
					{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
					{ID: "proxy-1", URL: "http://proxy2.example.com:8080"},
				},
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "valid config",
			config: ProxyPoolConfig{
				Proxies: []ProxyEntry{
					{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
					{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := NewProxyPool(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errKind != "" && !apperrors.IsKind(err, tt.errKind) {
					t.Errorf("Expected error kind %v, got %v", tt.errKind, apperrors.KindOf(err))
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if pool != nil {
					pool.Stop()
				}
			}
		})
	}
}

func TestNewProxyPool_Defaults(t *testing.T) {
	config := ProxyPoolConfig{
		DefaultStrategy: "round_robin",
		HealthCheck: HealthCheckConfig{
			Enabled: true,
			// Leave other fields at zero to test defaults
		},
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	// Check that defaults were applied
	if pool.strategy != RotationRoundRobin {
		t.Errorf("Expected strategy round_robin, got %v", pool.strategy)
	}

	// Check stats were initialized
	stats := pool.GetStats()
	if len(stats) != 1 {
		t.Errorf("Expected 1 proxy stats, got %d", len(stats))
	}

	if _, ok := stats["proxy-1"]; !ok {
		t.Error("Expected stats for proxy-1")
	}
}

func TestProxyPool_Select(t *testing.T) {
	config := ProxyPoolConfig{
		DefaultStrategy: "round_robin",
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080", Region: "us-east", Tags: []string{"datacenter"}},
			{ID: "proxy-2", URL: "http://proxy2.example.com:8080", Region: "us-west", Tags: []string{"residential"}},
			{ID: "proxy-3", URL: "http://proxy3.example.com:8080", Region: "us-east", Tags: []string{"datacenter", "premium"}},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	t.Run("select without hints", func(t *testing.T) {
		proxy, err := pool.Select(ProxySelectionHints{})
		if err != nil {
			t.Errorf("Select failed: %v", err)
		}
		if proxy.ID == "" {
			t.Error("Expected non-empty proxy ID")
		}
	})

	t.Run("select with region filter", func(t *testing.T) {
		hints := ProxySelectionHints{PreferredRegion: "us-west"}
		proxy, err := pool.Select(hints)
		if err != nil {
			t.Errorf("Select failed: %v", err)
		}
		if proxy.Region != "us-west" {
			t.Errorf("Expected region us-west, got %s", proxy.Region)
		}
	})

	t.Run("select with tags filter", func(t *testing.T) {
		hints := ProxySelectionHints{RequiredTags: []string{"premium"}}
		proxy, err := pool.Select(hints)
		if err != nil {
			t.Errorf("Select failed: %v", err)
		}
		if !containsString(proxy.Tags, "premium") {
			t.Errorf("Expected proxy with premium tag, got %v", proxy.Tags)
		}
	})

	t.Run("select with exclude filter", func(t *testing.T) {
		hints := ProxySelectionHints{ExcludeProxyIDs: []string{"proxy-1", "proxy-2"}}
		proxy, err := pool.Select(hints)
		if err != nil {
			t.Errorf("Select failed: %v", err)
		}
		if proxy.ID != "proxy-3" {
			t.Errorf("Expected proxy-3, got %s", proxy.ID)
		}
	})

	t.Run("select with no matching proxies", func(t *testing.T) {
		hints := ProxySelectionHints{PreferredRegion: "eu-west"}
		_, err := pool.Select(hints)
		if err == nil {
			t.Error("Expected error for no matching proxies")
		}
		if !apperrors.IsKind(err, apperrors.KindNotFound) {
			t.Errorf("Expected not_found error, got %v", apperrors.KindOf(err))
		}
	})
}

func TestProxyPool_Select_Unhealthy(t *testing.T) {
	config := ProxyPoolConfig{
		DefaultStrategy: "round_robin",
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
			{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	// Manually mark proxy-1 as unhealthy by accessing internal stats
	// (In production, this would be done by the health checker)
	pool.mu.Lock()
	pool.stats["proxy-1"].IsHealthy = false
	pool.mu.Unlock()

	// Should only select proxy-2 since proxy-1 is unhealthy
	for i := 0; i < 5; i++ {
		proxy, err := pool.Select(ProxySelectionHints{})
		if err != nil {
			t.Errorf("Select failed: %v", err)
			continue
		}
		if proxy.ID != "proxy-2" {
			t.Errorf("Expected proxy-2 (healthy), got %s", proxy.ID)
		}
	}
}

func TestProxyPool_RecordSuccess(t *testing.T) {
	config := ProxyPoolConfig{
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	// Record some successes
	pool.RecordSuccess("proxy-1", 100)
	pool.RecordSuccess("proxy-1", 200)
	pool.RecordSuccess("proxy-1", 150)

	stats, ok := pool.GetProxyStats("proxy-1")
	if !ok {
		t.Fatal("Expected stats for proxy-1")
	}

	if stats.RequestCount != 3 {
		t.Errorf("Expected RequestCount = 3, got %d", stats.RequestCount)
	}
	if stats.SuccessCount != 3 {
		t.Errorf("Expected SuccessCount = 3, got %d", stats.SuccessCount)
	}
	if stats.FailureCount != 0 {
		t.Errorf("Expected FailureCount = 0, got %d", stats.FailureCount)
	}
	if stats.ConsecutiveFails != 0 {
		t.Errorf("Expected ConsecutiveFails = 0, got %d", stats.ConsecutiveFails)
	}
	if stats.AvgLatencyMs == 0 {
		t.Error("Expected non-zero AvgLatencyMs")
	}
}

func TestProxyPool_RecordFailure(t *testing.T) {
	config := ProxyPoolConfig{
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	// Record some failures
	testErr := errors.New("connection refused")
	pool.RecordFailure("proxy-1", testErr)
	pool.RecordFailure("proxy-1", testErr)

	stats, ok := pool.GetProxyStats("proxy-1")
	if !ok {
		t.Fatal("Expected stats for proxy-1")
	}

	if stats.RequestCount != 2 {
		t.Errorf("Expected RequestCount = 2, got %d", stats.RequestCount)
	}
	if stats.SuccessCount != 0 {
		t.Errorf("Expected SuccessCount = 0, got %d", stats.SuccessCount)
	}
	if stats.FailureCount != 2 {
		t.Errorf("Expected FailureCount = 2, got %d", stats.FailureCount)
	}
	if stats.ConsecutiveFails != 2 {
		t.Errorf("Expected ConsecutiveFails = 2, got %d", stats.ConsecutiveFails)
	}
}

func TestProxyPool_GetStats(t *testing.T) {
	config := ProxyPoolConfig{
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
			{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	stats := pool.GetStats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 stats entries, got %d", len(stats))
	}

	// Verify the copy is independent by modifying the returned map
	// Note: We can't modify the struct value in the map directly, but we can verify
	// that getting stats again returns the original values
	freshStats, _ := pool.GetProxyStats("proxy-1")
	if freshStats.RequestCount != stats["proxy-1"].RequestCount {
		t.Error("Expected stats to be consistent")
	}
}

func TestProxyPool_GetHealthyProxyCount(t *testing.T) {
	config := ProxyPoolConfig{
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
			{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
			{ID: "proxy-3", URL: "http://proxy3.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	// Initially all healthy
	if count := pool.GetHealthyProxyCount(); count != 3 {
		t.Errorf("Expected 3 healthy proxies, got %d", count)
	}

	// Manually mark one as unhealthy
	pool.mu.Lock()
	pool.stats["proxy-1"].IsHealthy = false
	pool.mu.Unlock()

	if count := pool.GetHealthyProxyCount(); count != 2 {
		t.Errorf("Expected 2 healthy proxies, got %d", count)
	}
}

func TestProxyPool_SetStrategy(t *testing.T) {
	config := ProxyPoolConfig{
		DefaultStrategy: "round_robin",
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	if pool.GetStrategy() != RotationRoundRobin {
		t.Error("Expected initial strategy to be round_robin")
	}

	pool.SetStrategy(RotationRandom)
	if pool.GetStrategy() != RotationRandom {
		t.Error("Expected strategy to be random after SetStrategy")
	}
}

func TestProxyPool_GetEntries(t *testing.T) {
	config := ProxyPoolConfig{
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080", Region: "us-east"},
			{ID: "proxy-2", URL: "http://proxy2.example.com:8080", Region: "us-west"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	entries := pool.GetEntries()
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	// Verify it's a copy
	entries[0].Region = "modified"
	freshEntries := pool.GetEntries()
	if freshEntries[0].Region == "modified" {
		t.Error("Expected entries to be a copy, not a reference")
	}
}

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

	t.Run("file does not exist", func(t *testing.T) {
		pool, err := ProxyPoolFromConfig(tmpDir)
		if err != nil {
			t.Errorf("Expected no error for missing file, got %v", err)
		}
		if pool != nil {
			t.Error("Expected nil pool for missing file")
		}
	})

	t.Run("empty dataDir defaults to .data", func(t *testing.T) {
		// Create .data directory with proxy pool file
		dataDir := filepath.Join(tmpDir, ".data")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatalf("Failed to create .data directory: %v", err)
		}

		config := ProxyPoolConfig{
			Proxies: []ProxyEntry{
				{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
			},
		}

		data, _ := json.Marshal(config)
		if err := os.WriteFile(filepath.Join(dataDir, "proxy_pool.json"), data, 0644); err != nil {
			t.Fatalf("Failed to write proxy pool file: %v", err)
		}

		// Change to tmpDir so .data is relative
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		pool, err := ProxyPoolFromConfig("")
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

func TestProxyPool_Select_Strategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy RotationStrategy
		proxies  []ProxyEntry
	}{
		{
			name:     "round robin",
			strategy: RotationRoundRobin,
			proxies: []ProxyEntry{
				{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
				{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
			},
		},
		{
			name:     "random",
			strategy: RotationRandom,
			proxies: []ProxyEntry{
				{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
				{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
			},
		},
		{
			name:     "least used",
			strategy: RotationLeastUsed,
			proxies: []ProxyEntry{
				{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
				{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
			},
		},
		{
			name:     "weighted",
			strategy: RotationWeighted,
			proxies: []ProxyEntry{
				{ID: "proxy-1", URL: "http://proxy1.example.com:8080", Weight: 1},
				{ID: "proxy-2", URL: "http://proxy2.example.com:8080", Weight: 10},
			},
		},
		{
			name:     "least latency",
			strategy: RotationLeastLatency,
			proxies: []ProxyEntry{
				{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
				{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProxyPoolConfig{
				Proxies: tt.proxies,
			}

			pool, err := NewProxyPool(config)
			if err != nil {
				t.Fatalf("NewProxyPool failed: %v", err)
			}
			defer pool.Stop()

			pool.SetStrategy(tt.strategy)

			// Select multiple times to ensure it works
			for i := 0; i < 5; i++ {
				proxy, err := pool.Select(ProxySelectionHints{})
				if err != nil {
					t.Errorf("Select failed: %v", err)
				}
				if proxy.ID == "" {
					t.Error("Expected non-empty proxy ID")
				}
			}
		})
	}
}

func TestProxyPool_Select_LeastUsed(t *testing.T) {
	config := ProxyPoolConfig{
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
			{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	pool.SetStrategy(RotationLeastUsed)

	// Record some usage on proxy-1
	pool.RecordSuccess("proxy-1", 100)
	pool.RecordSuccess("proxy-1", 100)

	// Should select proxy-2 since it has fewer requests
	for i := 0; i < 5; i++ {
		proxy, err := pool.Select(ProxySelectionHints{})
		if err != nil {
			t.Errorf("Select failed: %v", err)
			continue
		}
		if proxy.ID != "proxy-2" {
			t.Errorf("Expected proxy-2 (least used), got %s", proxy.ID)
		}
	}
}

func TestProxyPool_Select_LeastLatency(t *testing.T) {
	config := ProxyPoolConfig{
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
			{ID: "proxy-2", URL: "http://proxy2.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	pool.SetStrategy(RotationLeastLatency)

	// Record high latency on proxy-1
	pool.RecordSuccess("proxy-1", 500)
	pool.RecordSuccess("proxy-1", 500)

	// Record low latency on proxy-2
	pool.RecordSuccess("proxy-2", 50)
	pool.RecordSuccess("proxy-2", 50)

	// Should select proxy-2 since it has lower latency
	for range 5 {
		proxy, err := pool.Select(ProxySelectionHints{})
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		if proxy.ID != "proxy-2" {
			t.Errorf("Expected proxy-2 (least latency), got %s", proxy.ID)
		}
	}
}

func TestProxyPool_HealthCheck_MarkUnhealthy(t *testing.T) {
	config := ProxyPoolConfig{
		Proxies: []ProxyEntry{
			{ID: "proxy-1", URL: "http://proxy1.example.com:8080"},
		},
	}

	pool, err := NewProxyPool(config)
	if err != nil {
		t.Fatalf("NewProxyPool failed: %v", err)
	}
	defer pool.Stop()

	// Manually mark proxy as unhealthy (in production, this would be done by health checker)
	pool.mu.Lock()
	pool.stats["proxy-1"].IsHealthy = false
	pool.mu.Unlock()

	stats, _ := pool.GetProxyStats("proxy-1")
	if stats.IsHealthy {
		t.Error("Expected proxy to be unhealthy")
	}
}

func TestDefaultHealthChecker_Check(t *testing.T) {
	checker := &DefaultHealthChecker{
		TestURL: "http://httpbin.org/ip",
		Timeout: 5 * time.Second,
	}

	// This test may fail if httpbin.org is unavailable
	// We're testing the structure, not the actual network call
	t.Run("invalid proxy URL", func(t *testing.T) {
		proxy := ProxyEntry{ID: "bad-proxy", URL: "://invalid-url"}
		_, err := checker.Check(context.Background(), proxy)
		if err == nil {
			t.Error("Expected error for invalid proxy URL")
		}
	})
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		slice []string
		s     string
		want  bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"a"}, "a", true},
	}

	for _, tt := range tests {
		got := containsString(tt.slice, tt.s)
		if got != tt.want {
			t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.s, got, tt.want)
		}
	}
}

func TestHasAllTags(t *testing.T) {
	tests := []struct {
		proxyTags    []string
		requiredTags []string
		want         bool
	}{
		{[]string{"a", "b", "c"}, []string{"b"}, true},
		{[]string{"a", "b", "c"}, []string{"b", "c"}, true},
		{[]string{"a", "b", "c"}, []string{"d"}, false},
		{[]string{"a", "b", "c"}, []string{"a", "d"}, false},
		{[]string{}, []string{"a"}, false},
		{[]string{"a"}, []string{}, true},
	}

	for _, tt := range tests {
		got := hasAllTags(tt.proxyTags, tt.requiredTags)
		if got != tt.want {
			t.Errorf("hasAllTags(%v, %v) = %v, want %v", tt.proxyTags, tt.requiredTags, got, tt.want)
		}
	}
}
