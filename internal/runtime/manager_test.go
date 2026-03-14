package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func testRuntimeConfig(dataDir string) config.Config {
	return config.Config{
		DataDir:            dataDir,
		UserAgent:          "SpartanTest/1.0",
		RequestTimeoutSecs: 30,
		MaxConcurrency:     1,
		RateLimitQPS:       10,
		RateLimitBurst:     10,
		MaxRetries:         1,
		RetryBaseMs:        10,
	}
}

func TestInitJobManager_FailsForExplicitMissingProxyPoolFile(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("store.Open() failed: %v", err)
	}
	defer st.Close()

	cfg := testRuntimeConfig(dataDir)
	cfg.ProxyPoolFile = filepath.Join(dataDir, "missing-proxy-pool.json")
	t.Setenv("PROXY_POOL_FILE", cfg.ProxyPoolFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := InitJobManager(ctx, cfg, st)
	if err == nil {
		if manager != nil {
			cancel()
			manager.Wait()
		}
		t.Fatal("expected explicit missing proxy pool file to fail")
	}
}

func TestInitJobManager_LoadsConfiguredProxyPool(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("store.Open() failed: %v", err)
	}
	defer st.Close()

	poolPath := filepath.Join(dataDir, "proxy_pool.json")
	payload, err := json.Marshal(fetch.ProxyPoolConfig{
		DefaultStrategy: "round_robin",
		HealthCheck:     fetch.HealthCheckConfig{Enabled: false},
		Proxies: []fetch.ProxyEntry{{
			ID:  "proxy-1",
			URL: "http://127.0.0.1:8080",
		}},
	})
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}
	if err := os.WriteFile(poolPath, payload, 0o644); err != nil {
		t.Fatalf("os.WriteFile() failed: %v", err)
	}

	cfg := testRuntimeConfig(dataDir)
	cfg.ProxyPoolFile = poolPath
	t.Setenv("PROXY_POOL_FILE", poolPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := InitJobManager(ctx, cfg, st)
	if err != nil {
		t.Fatalf("InitJobManager() failed: %v", err)
	}

	if manager.GetProxyPool() == nil {
		t.Fatal("expected proxy pool to be loaded")
	}

	cancel()
	manager.Wait()
}
