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
)

func InitJobManager(ctx context.Context, cfg config.Config, st *store.Store) *jobs.Manager {
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
