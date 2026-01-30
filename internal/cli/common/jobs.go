// Package common contains CLI helpers for job manager wiring.
//
// It does NOT define job execution behavior; internal/jobs does that.
package common

import (
	"context"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"golang.org/x/time/rate"
)

func InitJobManager(ctx context.Context, cfg config.Config, st *store.Store) *jobs.Manager {
	// Build circuit breaker config from env
	cbConfig := fetch.CircuitBreakerConfig{
		Enabled:             cfg.CircuitBreakerEnabled,
		FailureThreshold:    cfg.CircuitBreakerFailureThreshold,
		SuccessThreshold:    cfg.CircuitBreakerSuccessThreshold,
		ResetTimeout:        time.Duration(cfg.CircuitBreakerResetTimeoutSecs) * time.Second,
		HalfOpenMaxRequests: cfg.CircuitBreakerHalfOpenMaxRequests,
	}

	// Build adaptive config from env (if enabled)
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

	// Initialize proxy pool if configured
	if cfg.ProxyPoolFile != "" {
		proxyPool, err := fetch.LoadProxyPoolFromFile(cfg.ProxyPoolFile)
		if err != nil {
			// Log warning but don't fail - proxy pool is optional
			slog.Warn("failed to load proxy pool", "path", cfg.ProxyPoolFile, "error", err)
		} else if proxyPool != nil {
			manager.SetProxyPool(proxyPool)
			slog.Info("proxy pool loaded", "path", cfg.ProxyPoolFile)
		}
	}

	manager.Start(ctx)
	return manager
}
