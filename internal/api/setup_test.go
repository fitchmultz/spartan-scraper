// Package api provides shared test infrastructure for API integration tests.
// Contains setupTestServer helper used across all test files.
// Does NOT contain tests itself (tests are in other _test.go files).
package api

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	dataDir := t.TempDir()
	cfg := config.Config{
		DataDir:            dataDir,
		RequestTimeoutSecs: 30,
		MaxConcurrency:     4,
		RateLimitQPS:       10,
		RateLimitBurst:     20,
		MaxRetries:         3,
		RetryBaseMs:        100,
		UserAgent:          "SpartanTest/1.0",
		Port:               "0",
		BindAddr:           "127.0.0.1",
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	manager := jobs.NewManager(
		st,
		dataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.MaxResponseBytes,
		false,
		fetch.DefaultCircuitBreakerConfig(),
		nil, // no adaptive rate limiting in tests
	)
	ctx, cancel := context.WithCancel(context.Background())
	manager.Start(ctx)

	srv := NewServer(manager, st, cfg)

	cleanup := func() {
		cancel()
		manager.Wait()
		st.Close()
	}

	return srv, cleanup
}
