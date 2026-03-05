// Package fetch provides HTTP and headless browser content fetching capabilities.
package fetch

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

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
