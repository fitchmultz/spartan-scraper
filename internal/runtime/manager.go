// Package runtime provides runtime functionality for Spartan Scraper.
//
// Purpose:
// - Implement manager support for package runtime.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `runtime` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

/*
Purpose: Initialize the fully wired runtime job manager used by local execution surfaces.
Responsibilities: Build fetch/runtime controls, attach optional proxy-pool and AI services, connect store-backed capabilities, and start manager lifecycle processing.
Scope: Runtime bootstrap for CLI, API server, and MCP entrypoints.
Usage: Call `InitJobManager` after loading config and opening the store.
Invariants/Assumptions: Proxy pooling is fully opt-in, explicit proxy-pool paths fail fast when broken, and the returned manager is already started.
*/
package runtime

import (
	"context"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"golang.org/x/time/rate"
)

// InitJobManager builds a fully wired job manager for local runtime surfaces such as
// CLI commands, the API server, and MCP.
func InitJobManager(ctx context.Context, cfg config.Config, st *store.Store) (*jobs.Manager, error) {
	cbConfig := fetch.CircuitBreakerConfig{
		Enabled:             cfg.CircuitBreakerEnabled,
		FailureThreshold:    cfg.CircuitBreakerFailureThreshold,
		SuccessThreshold:    cfg.CircuitBreakerSuccessThreshold,
		ResetTimeout:        time.Duration(cfg.CircuitBreakerResetTimeoutSecs) * time.Second,
		HalfOpenMaxRequests: cfg.CircuitBreakerHalfOpenMaxRequests,
	}

	var adaptiveConfig *fetch.AdaptiveConfig
	if cfg.AdaptiveRateLimit {
		adaptiveConfig = &fetch.AdaptiveConfig{
			Enabled:                true,
			MinQPS:                 rate.Limit(cfg.AdaptiveMinQPS),
			MaxQPS:                 rate.Limit(cfg.AdaptiveMaxQPS),
			AdditiveIncrease:       rate.Limit(cfg.AdaptiveIncreaseQPS),
			MultiplicativeDecrease: cfg.AdaptiveDecreaseFactor,
			SuccessThreshold:       cfg.AdaptiveSuccessThreshold,
			CooldownPeriod:         time.Duration(cfg.AdaptiveCooldownMs) * time.Millisecond,
		}
	}

	manager := jobs.NewManager(
		st,
		cfg.DataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.MaxResponseBytes,
		cfg.UsePlaywright,
		cbConfig,
		adaptiveConfig,
	)

	if cfg.ProxyPoolFile != "" {
		proxyPool, err := fetch.ProxyPoolFromConfig(cfg.ProxyPoolFile, true)
		if err != nil {
			return nil, err
		}
		if proxyPool != nil {
			manager.SetProxyPool(proxyPool)
			slog.Info("proxy pool loaded", "path", cfg.ProxyPoolFile)
		}
	}

	if extract.IsAIEnabled(cfg.AI) {
		aiExtractor, err := extract.NewAIExtractor(cfg.AI)
		if err != nil {
			slog.Warn("failed to initialize AI extractor", "error", err)
		} else if aiExtractor != nil {
			manager.SetAIExtractor(aiExtractor)
			slog.Info("AI extractor initialized", "mode", cfg.AI.Mode, "bridge", cfg.AI.BridgeScript)
		}
	}

	manager.Start(ctx)
	return manager, nil
}
