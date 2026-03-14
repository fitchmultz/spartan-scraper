package runtime

import (
	"context"
	"log/slog"
	"os"
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
func InitJobManager(ctx context.Context, cfg config.Config, st *store.Store) *jobs.Manager {
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
		_, proxyPoolExplicit := os.LookupEnv("PROXY_POOL_FILE")
		proxyPool, err := fetch.ProxyPoolFromConfig(cfg.ProxyPoolFile, proxyPoolExplicit)
		if err != nil {
			slog.Warn("failed to load proxy pool", "path", cfg.ProxyPoolFile, "error", err)
		} else if proxyPool != nil {
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

	contentIndex := st.GetContentIndex()
	if contentIndex != nil {
		manager.SetContentIndex(contentIndex)
		slog.Info("content index initialized for cross-job deduplication")
	} else {
		slog.Warn("content index not initialized, cross-job deduplication disabled")
	}

	manager.Start(ctx)
	return manager
}
