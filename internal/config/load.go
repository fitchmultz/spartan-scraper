// Purpose: Orchestrate startup configuration loading from environment variables into a single immutable Config snapshot.
// Responsibilities:
// - Load .env defaults and compose domain-specific loaders for server, rate-limit, retention, circuit-breaker, and AI settings.
// - Run post-load validation and startup-notice collection before returning Config.
// - Keep the top-level Load flow small and readable while delegating domain parsing elsewhere.
// Scope:
// - Startup config assembly only; env helpers and domain validation live in companion files.
// Usage:
// - Call Load() once during process startup and pass the returned Config by value.
// Invariants/Assumptions:
// - Domain loaders are side-effect free except for startup notice recording performed by validation helpers.
// - Load returns a fully validated Config or an error.
package config

import (
	"time"

	"github.com/joho/godotenv"
)

// Load reads configuration from environment variables (optionally loading defaults from
// a local .env file).
//
// Intended usage: call Load once during application startup, then pass the returned Config
// value into constructors/handlers.
//
// Load does not maintain any singleton/global Config instance; it simply returns a value.
// The returned Config is treated as immutable after loading.
//
// Returns an error if the data directory cannot be created or is not writable.
// Uses apperrors.KindPermission for writability issues.
func Load() (Config, error) {
	resetStartupNotices()
	_ = godotenv.Load()

	dataDir := getenv("DATA_DIR", ".data")
	cfg := Config{
		DataDir:            dataDir,
		UserAgent:          getenv("USER_AGENT", "SpartanScraper/0.1 (+https://local)"),
		MaxConcurrency:     getenvInt("MAX_CONCURRENCY", 4),
		RequestTimeoutSecs: getenvInt("REQUEST_TIMEOUT_SECONDS", 30),
		MaxResponseBytes:   getenvInt64("MAX_RESPONSE_BYTES", 10*1024*1024),
		UsePlaywright:      getenvBool("USE_PLAYWRIGHT", false),
		AuthOverrides:      loadAuthOverrides(),
		LogLevel:           getenv("LOG_LEVEL", "info"),
		LogFormat:          getenv("LOG_FORMAT", "text"),
		APIAuthEnabled:     getenvBool("API_AUTH_ENABLED", false),
		MaxBatchSize:       getenvInt("MAX_BATCH_SIZE", 100),
		RespectRobotsTxt:   getenvBool("RESPECT_ROBOTS_TXT", false),
	}

	cfg = loadServerConfig(cfg)
	cfg = loadProxyConfig(cfg)
	cfg = loadWebhookConfig(cfg)
	cfg = loadRateLimitConfig(cfg)
	cfg = loadRetentionConfig(cfg)
	cfg = loadCircuitBreakerConfig(cfg)
	cfg.AI = loadAIConfig()

	if err := validateDataDir(cfg.DataDir); err != nil {
		return Config{}, err
	}

	cfg = validateAndFixAdaptiveConfig(cfg)
	cfg = validateAndFixRetentionConfig(cfg)
	cfg = validateAndFixCircuitBreakerConfig(cfg)
	cfg = validateAndFixRetryConfig(cfg)
	cfg = validateAndFixAIConfig(cfg)
	if err := validateNoLegacyAIConfig(); err != nil {
		return Config{}, err
	}

	cfg.StartupNotices = consumeStartupNotices()
	return cfg, nil
}

func loadServerConfig(cfg Config) Config {
	cfg.Port = getenv("PORT", "8741")
	cfg.BindAddr = getenv("BIND_ADDR", "127.0.0.1")
	cfg.ServerReadHeaderTimeoutSecs = getenvInt("SERVER_READ_HEADER_TIMEOUT_SECONDS", 10)
	cfg.ServerReadTimeoutSecs = getenvInt("SERVER_READ_TIMEOUT_SECONDS", 30)
	cfg.ServerWriteTimeoutSecs = getenvInt("SERVER_WRITE_TIMEOUT_SECONDS", 60)
	cfg.ServerIdleTimeoutSecs = getenvInt("SERVER_IDLE_TIMEOUT_SECONDS", 120)
	return cfg
}

func loadProxyConfig(cfg Config) Config {
	cfg.ProxyURL = getenv("PROXY_URL", "")
	cfg.ProxyUsername = getenv("PROXY_USERNAME", "")
	cfg.ProxyPassword = getenv("PROXY_PASSWORD", "")
	cfg.ProxyPoolFile = getenvAllowEmpty("PROXY_POOL_FILE", "")
	return cfg
}

func loadWebhookConfig(cfg Config) Config {
	cfg.Webhook = WebhookConfig{
		Enabled:                 getenvBool("WEBHOOK_ENABLED", false),
		Secret:                  getenv("WEBHOOK_SECRET", ""),
		MaxRetries:              getenvInt("WEBHOOK_MAX_RETRIES", 3),
		BaseDelay:               time.Duration(getenvInt("WEBHOOK_BASE_DELAY_MS", 1000)) * time.Millisecond,
		MaxDelay:                time.Duration(getenvInt("WEBHOOK_MAX_DELAY_MS", 30000)) * time.Millisecond,
		Timeout:                 time.Duration(getenvInt("WEBHOOK_TIMEOUT_MS", 30000)) * time.Millisecond,
		AllowInternal:           getenvBool("WEBHOOK_ALLOW_INTERNAL", false),
		MaxConcurrentDispatches: getenvInt("WEBHOOK_MAX_CONCURRENT", 100),
	}
	return cfg
}
