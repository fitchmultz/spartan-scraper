// Package fetch provides HTTP and headless browser content fetching capabilities.
package fetch

import (
	"context"
	"testing"
	"time"
)

// mockHealthChecker is a mock implementation of HealthChecker for testing.
type mockHealthChecker struct {
	latencyMs int64
	err       error
}

func (m *mockHealthChecker) Check(ctx context.Context, proxy ProxyEntry) (int64, error) {
	return m.latencyMs, m.err
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
