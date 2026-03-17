/*
Purpose: Verify runtime job-manager bootstrap handles optional proxy-pool startup correctly.
Responsibilities: Cover missing default proxy-pool tolerance, explicit custom-path failures, and successful proxy-pool attachment.
Scope: Runtime bootstrap behavior only.
Usage: Run with `go test ./internal/runtime`.
Invariants/Assumptions: Missing default proxy-pool files must not block startup, while explicit custom proxy-pool paths still surface configuration mistakes.
*/
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

func TestInitJobManager_IgnoresMissingDefaultProxyPoolFileEvenWhenEnvSet(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("store.Open() failed: %v", err)
	}
	defer st.Close()

	cfg := testRuntimeConfig(dataDir)
	cfg.ProxyPoolFile = filepath.Join(dataDir, "proxy_pool.json")
	t.Setenv("PROXY_POOL_FILE", cfg.ProxyPoolFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := InitJobManager(ctx, cfg, st)
	if err != nil {
		t.Fatalf("InitJobManager() failed: %v", err)
	}
	defer manager.Wait()

	if manager.GetProxyPool() != nil {
		t.Fatal("expected missing default proxy pool file to leave proxy pool disabled")
	}

	cancel()
}

func TestInitJobManager_FailsForExplicitMissingCustomProxyPoolFile(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("store.Open() failed: %v", err)
	}
	defer st.Close()

	cfg := testRuntimeConfig(dataDir)
	cfg.ProxyPoolFile = filepath.Join(dataDir, "missing-proxy-pool-custom.json")
	t.Setenv("PROXY_POOL_FILE", cfg.ProxyPoolFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := InitJobManager(ctx, cfg, st)
	if err == nil {
		if manager != nil {
			cancel()
			manager.Wait()
		}
		t.Fatal("expected explicit missing custom proxy pool file to fail")
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
