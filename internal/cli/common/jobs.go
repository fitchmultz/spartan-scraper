// Package common contains CLI helpers for job manager wiring.
//
// Purpose:
//   - Build and configure shared runtime dependencies for job-oriented commands.
//
// Responsibilities:
//   - Construct job managers from loaded config.
//   - Wire optional proxy-pool and AI integrations.
//
// Scope:
//   - CLI-side dependency assembly only.
//
// Usage:
//   - Call InitJobManager from CLI entrypoints after config/store initialization.
//
// Invariants/Assumptions:
//   - Missing optional default proxy-pool files stay silent on startup.
//   - Explicit proxy-pool misconfiguration fails fast instead of silently disabling proxies.
package common

import (
	"context"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	appRuntime "github.com/fitchmultz/spartan-scraper/internal/runtime"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func InitJobManager(ctx context.Context, cfg config.Config, st *store.Store) (*jobs.Manager, error) {
	return appRuntime.InitJobManager(ctx, cfg, st)
}
