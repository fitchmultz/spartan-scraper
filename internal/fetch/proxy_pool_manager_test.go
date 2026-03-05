// Package fetch provides HTTP and headless browser content fetching capabilities.
package fetch

import (
	"errors"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

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
