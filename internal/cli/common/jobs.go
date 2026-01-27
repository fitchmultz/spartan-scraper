// Package common contains CLI helpers for job manager wiring.
//
// It does NOT define job execution behavior; internal/jobs does that.
package common

import (
	"context"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
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
	manager.Start(ctx)
	return manager
}
