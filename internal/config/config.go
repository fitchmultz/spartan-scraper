// Package config provides application configuration loading from environment variables.
//
// Purpose:
// - Load the application's startup configuration from environment variables and `.env` defaults.
//
// Responsibilities:
// - Parse environment variables into a typed immutable `Config` snapshot.
// - Apply sensible defaults and non-fatal corrections for invalid optional settings.
// - Surface startup notices for configuration issues that should appear inside operator surfaces.
//
// Scope:
// - Configuration loading and validation only; runtime initialization lives elsewhere.
//
// Usage:
// - Call `Load()` once during process startup and pass the returned `Config` by value.
//
// Invariants/Assumptions:
// - `Config` is treated as immutable after loading.
// - `AuthOverrides.Headers` and `AuthOverrides.Cookies` are read-only map fields unless callers deep-copy them first.
//
// # Immutability & thread-safety
//
// This project uses a "load once at startup, then pass by value" configuration pattern:
//
//   - config.Load() is called once at process startup (see internal/cli/cli.go).
//   - Load returns a Config value (not a pointer).
//   - The Config value is passed by value to constructors/handlers, so each component gets
//     its own copy of the struct.
//
// After Load returns, Config is treated as immutable: components must not mutate fields.
// As long as callers follow this rule, Config is safe for concurrent read access.
//
// Note: Config contains AuthOverrides.Headers and AuthOverrides.Cookies map fields.
// Maps are reference types; copying Config copies the map header, not the underlying map.
// Therefore these maps must be treated as read-only. If a component needs to modify them,
// it must make a deep copy first.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/joho/godotenv"
)

// EnvOverrides is an alias for auth.EnvOverrides
type EnvOverrides = auth.EnvOverrides

// WebhookConfig holds global webhook configuration.
type WebhookConfig struct {
	Enabled                 bool
	Secret                  string
	MaxRetries              int
	BaseDelay               time.Duration
	MaxDelay                time.Duration
	Timeout                 time.Duration
	AllowInternal           bool // Allow webhooks to internal/private addresses; bypasses the default dispatch-time private-target guardrail
	MaxConcurrentDispatches int  // Maximum concurrent webhook dispatches (default: 100)
}

// StartupNotice captures non-fatal configuration issues that should surface inside operator surfaces.
type StartupNotice struct {
	ID       string
	Severity string
	Title    string
	Message  string
	Action   string
}

// Config is the application's configuration snapshot.
//
// Config is intended to be immutable after Load() returns. It is passed around by value,
// so each consumer receives its own copy of the struct.
//
// Thread-safety guarantee: Config is safe for concurrent read access as long as callers
// do not mutate it.
//
// WARNING: AuthOverrides.Headers and AuthOverrides.Cookies are maps (reference types).
// Treat them as read-only. If you need to add/remove entries, make a deep copy first.
type Config struct {
	Port     string
	BindAddr string

	// HTTP server hardening timeouts (in seconds). These are applied when constructing
	// API http.Server (see internal/cli/server/server.go).
	ServerReadHeaderTimeoutSecs int
	ServerReadTimeoutSecs       int
	ServerWriteTimeoutSecs      int
	ServerIdleTimeoutSecs       int

	DataDir            string
	UserAgent          string
	MaxConcurrency     int
	RequestTimeoutSecs int
	RateLimitQPS       int
	RateLimitBurst     int
	MaxRetries         int
	RetryBaseMs        int
	MaxResponseBytes   int64
	UsePlaywright      bool
	AuthOverrides      EnvOverrides
	LogLevel           string
	LogFormat          string
	StartupNotices     []StartupNotice

	// Proxy configuration
	ProxyURL      string
	ProxyUsername string
	ProxyPassword string
	ProxyPoolFile string // Path to proxy pool JSON config

	// Webhook configuration
	Webhook WebhookConfig

	// API Authentication
	APIAuthEnabled bool // API_AUTH_ENABLED env var

	// Batch configuration
	MaxBatchSize int // MAX_BATCH_SIZE env var

	// Adaptive rate limiting configuration
	AdaptiveRateLimit        bool    // ADAPTIVE_RATE_LIMIT env var
	AdaptiveMinQPS           float64 // ADAPTIVE_MIN_QPS env var
	AdaptiveMaxQPS           float64 // ADAPTIVE_MAX_QPS env var
	AdaptiveIncreaseQPS      float64 // ADAPTIVE_INCREASE_QPS env var
	AdaptiveDecreaseFactor   float64 // ADAPTIVE_DECREASE_FACTOR env var
	AdaptiveSuccessThreshold int     // ADAPTIVE_SUCCESS_THRESHOLD env var
	AdaptiveCooldownMs       int     // ADAPTIVE_COOLDOWN_MS env var

	// Robots.txt compliance
	RespectRobotsTxt bool // RESPECT_ROBOTS_TXT env var (default: false)

	// Data retention configuration
	RetentionEnabled              bool // RETENTION_ENABLED env var (default: false)
	RetentionJobDays              int  // RETENTION_JOB_DAYS env var (default: 30, 0 = unlimited)
	RetentionCrawlStateDays       int  // RETENTION_CRAWL_STATE_DAYS env var (default: 90, 0 = unlimited)
	RetentionMaxJobs              int  // RETENTION_MAX_JOBS env var (default: 10000, 0 = unlimited)
	RetentionMaxStorageGB         int  // RETENTION_MAX_STORAGE_GB env var (default: 10, 0 = unlimited)
	RetentionCleanupIntervalHours int  // RETENTION_CLEANUP_INTERVAL_HOURS env var (default: 24)
	RetentionDryRunDefault        bool // RETENTION_DRY_RUN_DEFAULT env var (default: false)

	// Circuit breaker configuration
	CircuitBreakerEnabled             bool // CIRCUIT_BREAKER_ENABLED env var (default: true)
	CircuitBreakerFailureThreshold    int  // CIRCUIT_BREAKER_FAILURE_THRESHOLD env var (default: 5)
	CircuitBreakerSuccessThreshold    int  // CIRCUIT_BREAKER_SUCCESS_THRESHOLD env var (default: 3)
	CircuitBreakerResetTimeoutSecs    int  // CIRCUIT_BREAKER_RESET_TIMEOUT_SECONDS env var (default: 30)
	CircuitBreakerHalfOpenMaxRequests int  // CIRCUIT_BREAKER_HALF_OPEN_MAX_REQUESTS env var (default: 3)

	// Enhanced retry configuration
	RetryMaxDelaySecs    int    // RETRY_MAX_DELAY_SECONDS env var (default: 60)
	RetryBackoffStrategy string // RETRY_BACKOFF_STRATEGY env var (default: "exponential_jitter")
	RetryStatusCodes     string // RETRY_STATUS_CODES env var (default: "429,500,502,503,504")

	// AI extraction configuration
	AI AIConfig
}

const (
	DefaultPIMode                  = "sdk"
	DefaultPINodeBin               = "node"
	DefaultPIBridgeScript          = "tools/pi-bridge/dist/main.js"
	DefaultPIStartupTimeoutSecs    = 10
	DefaultPIRequestTimeoutSecs    = 60
	AICapabilityExtractNatural     = "extract.natural_language"
	AICapabilityExtractSchema      = "extract.schema_guided"
	AICapabilityTemplateGeneration = "template.generate"
	AICapabilityRenderProfile      = "render_profile.generate"
	AICapabilityPipelineJS         = "pipeline_js.generate"
	AICapabilityResearchRefine     = "research.refine"
	AICapabilityExportShape        = "export.shape"
	AICapabilityTransformGenerate  = "transform.generate"
)

// AIRoutingConfig maps AI capabilities to ordered provider/model routes.
type AIRoutingConfig struct {
	Routes map[string][]string `json:"routes"`
}

// RoutesFor returns the configured routes for a capability.
func (r AIRoutingConfig) RoutesFor(capability string) []string {
	if len(r.Routes) == 0 {
		return nil
	}
	routes := r.Routes[capability]
	out := make([]string, 0, len(routes))
	for _, route := range routes {
		trimmed := strings.TrimSpace(route)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// RouteFingerprint returns a stable cache fingerprint for the configured route order.
func (r AIRoutingConfig) RouteFingerprint(capability string) string {
	return strings.Join(r.RoutesFor(capability), "->")
}

// DefaultAIRoutingConfig returns the built-in pi routing defaults.
func DefaultAIRoutingConfig() AIRoutingConfig {
	defaultRouteOrder := []string{
		"kimi-coding/k2p5",
		"zai/glm-5",
		"openai-codex/gpt-5.4",
	}
	return AIRoutingConfig{
		Routes: map[string][]string{
			AICapabilityExtractNatural:     append([]string(nil), defaultRouteOrder...),
			AICapabilityExtractSchema:      append([]string(nil), defaultRouteOrder...),
			AICapabilityTemplateGeneration: append([]string(nil), defaultRouteOrder...),
			AICapabilityRenderProfile:      append([]string(nil), defaultRouteOrder...),
			AICapabilityPipelineJS:         append([]string(nil), defaultRouteOrder...),
			AICapabilityResearchRefine:     append([]string(nil), defaultRouteOrder...),
			AICapabilityExportShape:        append([]string(nil), defaultRouteOrder...),
			AICapabilityTransformGenerate:  append([]string(nil), defaultRouteOrder...),
		},
	}
}

// AIConfig holds configuration for AI-powered extraction.
type AIConfig struct {
	Enabled            bool
	ConfigPath         string
	Mode               string
	NodeBin            string
	BridgeScript       string
	StartupTimeoutSecs int
	RequestTimeoutSecs int
	Routing            AIRoutingConfig
}

var (
	startupNoticeMu    sync.Mutex
	startupLoadNotices []StartupNotice
)

func resetStartupNotices() {
	startupNoticeMu.Lock()
	defer startupNoticeMu.Unlock()
	startupLoadNotices = nil
}

func recordStartupNotice(notice StartupNotice) {
	if notice.ID == "" {
		return
	}

	startupNoticeMu.Lock()
	defer startupNoticeMu.Unlock()
	for _, existing := range startupLoadNotices {
		if existing.ID == notice.ID {
			return
		}
	}
	startupLoadNotices = append(startupLoadNotices, notice)
}

func consumeStartupNotices() []StartupNotice {
	startupNoticeMu.Lock()
	defer startupNoticeMu.Unlock()
	out := append([]StartupNotice(nil), startupLoadNotices...)
	startupLoadNotices = nil
	return out
}

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
		Port:     getenv("PORT", "8741"),
		BindAddr: getenv("BIND_ADDR", "127.0.0.1"),

		ServerReadHeaderTimeoutSecs: getenvInt("SERVER_READ_HEADER_TIMEOUT_SECONDS", 10),
		ServerReadTimeoutSecs:       getenvInt("SERVER_READ_TIMEOUT_SECONDS", 30),
		ServerWriteTimeoutSecs:      getenvInt("SERVER_WRITE_TIMEOUT_SECONDS", 60),
		ServerIdleTimeoutSecs:       getenvInt("SERVER_IDLE_TIMEOUT_SECONDS", 120),

		DataDir:            dataDir,
		UserAgent:          getenv("USER_AGENT", "SpartanScraper/0.1 (+https://local)"),
		MaxConcurrency:     getenvInt("MAX_CONCURRENCY", 4),
		RequestTimeoutSecs: getenvInt("REQUEST_TIMEOUT_SECONDS", 30),
		RateLimitQPS:       getenvInt("RATE_LIMIT_QPS", 2),
		RateLimitBurst:     getenvInt("RATE_LIMIT_BURST", 4),
		MaxRetries:         getenvInt("MAX_RETRIES", 2),
		RetryBaseMs:        getenvInt("RETRY_BASE_MS", 400),
		MaxResponseBytes:   getenvInt64("MAX_RESPONSE_BYTES", 10*1024*1024),
		UsePlaywright:      getenvBool("USE_PLAYWRIGHT", false),
		AuthOverrides:      loadAuthOverrides(),
		LogLevel:           getenv("LOG_LEVEL", "info"),
		LogFormat:          getenv("LOG_FORMAT", "text"),

		// Proxy configuration
		ProxyURL:      getenv("PROXY_URL", ""),
		ProxyUsername: getenv("PROXY_USERNAME", ""),
		ProxyPassword: getenv("PROXY_PASSWORD", ""),
		ProxyPoolFile: getenvAllowEmpty("PROXY_POOL_FILE", filepath.Join(dataDir, "proxy_pool.json")),

		// Webhook configuration
		Webhook: WebhookConfig{
			Enabled:                 getenvBool("WEBHOOK_ENABLED", false),
			Secret:                  getenv("WEBHOOK_SECRET", ""),
			MaxRetries:              getenvInt("WEBHOOK_MAX_RETRIES", 3),
			BaseDelay:               time.Duration(getenvInt("WEBHOOK_BASE_DELAY_MS", 1000)) * time.Millisecond,
			MaxDelay:                time.Duration(getenvInt("WEBHOOK_MAX_DELAY_MS", 30000)) * time.Millisecond,
			Timeout:                 time.Duration(getenvInt("WEBHOOK_TIMEOUT_MS", 30000)) * time.Millisecond,
			AllowInternal:           getenvBool("WEBHOOK_ALLOW_INTERNAL", false),
			MaxConcurrentDispatches: getenvInt("WEBHOOK_MAX_CONCURRENT", 100),
		},

		// API Authentication
		APIAuthEnabled: getenvBool("API_AUTH_ENABLED", false),

		// Batch configuration
		MaxBatchSize: getenvInt("MAX_BATCH_SIZE", 100),

		// Adaptive rate limiting configuration
		AdaptiveRateLimit:        getenvBool("ADAPTIVE_RATE_LIMIT", false),
		AdaptiveMinQPS:           getenvFloat64("ADAPTIVE_MIN_QPS", 0.1),
		AdaptiveMaxQPS:           getenvFloat64("ADAPTIVE_MAX_QPS", float64(getenvInt("RATE_LIMIT_QPS", 2))),
		AdaptiveIncreaseQPS:      getenvFloat64("ADAPTIVE_INCREASE_QPS", 0.5),
		AdaptiveDecreaseFactor:   getenvFloat64("ADAPTIVE_DECREASE_FACTOR", 0.5),
		AdaptiveSuccessThreshold: getenvInt("ADAPTIVE_SUCCESS_THRESHOLD", 5),
		AdaptiveCooldownMs:       getenvInt("ADAPTIVE_COOLDOWN_MS", 1000),

		// Robots.txt compliance
		RespectRobotsTxt: getenvBool("RESPECT_ROBOTS_TXT", false),

		// Data retention configuration
		RetentionEnabled:              getenvBool("RETENTION_ENABLED", false),
		RetentionJobDays:              getenvInt("RETENTION_JOB_DAYS", 30),
		RetentionCrawlStateDays:       getenvInt("RETENTION_CRAWL_STATE_DAYS", 90),
		RetentionMaxJobs:              getenvInt("RETENTION_MAX_JOBS", 10000),
		RetentionMaxStorageGB:         getenvInt("RETENTION_MAX_STORAGE_GB", 10),
		RetentionCleanupIntervalHours: getenvInt("RETENTION_CLEANUP_INTERVAL_HOURS", 24),
		RetentionDryRunDefault:        getenvBool("RETENTION_DRY_RUN_DEFAULT", false),

		// Circuit breaker configuration
		CircuitBreakerEnabled:             getenvBool("CIRCUIT_BREAKER_ENABLED", true),
		CircuitBreakerFailureThreshold:    getenvInt("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
		CircuitBreakerSuccessThreshold:    getenvInt("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3),
		CircuitBreakerResetTimeoutSecs:    getenvInt("CIRCUIT_BREAKER_RESET_TIMEOUT_SECONDS", 30),
		CircuitBreakerHalfOpenMaxRequests: getenvInt("CIRCUIT_BREAKER_HALF_OPEN_MAX_REQUESTS", 3),

		// Enhanced retry configuration
		RetryMaxDelaySecs:    getenvInt("RETRY_MAX_DELAY_SECONDS", 60),
		RetryBackoffStrategy: getenv("RETRY_BACKOFF_STRATEGY", "exponential_jitter"),
		RetryStatusCodes:     getenv("RETRY_STATUS_CODES", "429,500,502,503,504"),
	}

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

// validateAndFixAdaptiveConfig ensures adaptive rate limiting configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixAdaptiveConfig(cfg Config) Config {
	if !cfg.AdaptiveRateLimit {
		return cfg
	}

	if cfg.AdaptiveMinQPS > cfg.AdaptiveMaxQPS {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-min-max-swapped",
			Severity: "warning",
			Title:    "Adaptive rate-limit bounds were corrected",
			Message:  fmt.Sprintf("ADAPTIVE_MIN_QPS (%.2f) exceeded ADAPTIVE_MAX_QPS (%.2f), so Spartan swapped them for this session.", cfg.AdaptiveMinQPS, cfg.AdaptiveMaxQPS),
		})
		cfg.AdaptiveMinQPS, cfg.AdaptiveMaxQPS = cfg.AdaptiveMaxQPS, cfg.AdaptiveMinQPS
	}

	if cfg.AdaptiveMinQPS <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-min-invalid",
			Severity: "warning",
			Title:    "Adaptive minimum QPS was reset",
			Message:  "ADAPTIVE_MIN_QPS must be positive, so Spartan is using 0.1 for this session.",
		})
		cfg.AdaptiveMinQPS = 0.1
	}

	if cfg.AdaptiveMaxQPS <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-max-invalid",
			Severity: "warning",
			Title:    "Adaptive maximum QPS was reset",
			Message:  "ADAPTIVE_MAX_QPS must be positive, so Spartan is using RATE_LIMIT_QPS for this session.",
		})
		cfg.AdaptiveMaxQPS = float64(cfg.RateLimitQPS)
	}

	if cfg.AdaptiveDecreaseFactor <= 0 || cfg.AdaptiveDecreaseFactor >= 1 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-decrease-invalid",
			Severity: "warning",
			Title:    "Adaptive decrease factor was reset",
			Message:  "ADAPTIVE_DECREASE_FACTOR must be between 0 and 1, so Spartan is using 0.5 for this session.",
		})
		cfg.AdaptiveDecreaseFactor = 0.5
	}

	if cfg.AdaptiveIncreaseQPS <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-increase-invalid",
			Severity: "warning",
			Title:    "Adaptive increase QPS was reset",
			Message:  "ADAPTIVE_INCREASE_QPS must be positive, so Spartan is using 0.5 for this session.",
		})
		cfg.AdaptiveIncreaseQPS = 0.5
	}

	if cfg.AdaptiveSuccessThreshold <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-success-threshold-invalid",
			Severity: "warning",
			Title:    "Adaptive success threshold was reset",
			Message:  "ADAPTIVE_SUCCESS_THRESHOLD must be positive, so Spartan is using 5 for this session.",
		})
		cfg.AdaptiveSuccessThreshold = 5
	}

	if cfg.AdaptiveCooldownMs < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "adaptive-cooldown-invalid",
			Severity: "warning",
			Title:    "Adaptive cooldown was reset",
			Message:  "ADAPTIVE_COOLDOWN_MS must be non-negative, so Spartan is using 1000ms for this session.",
		})
		cfg.AdaptiveCooldownMs = 1000
	}

	return cfg
}

// validateAndFixRetentionConfig ensures retention configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixRetentionConfig(cfg Config) Config {
	if cfg.RetentionJobDays < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-job-days-invalid",
			Severity: "warning",
			Title:    "Job retention age was reset",
			Message:  "RETENTION_JOB_DAYS must be non-negative, so Spartan is using unlimited retention for this session.",
		})
		cfg.RetentionJobDays = 0
	}
	if cfg.RetentionCrawlStateDays < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-crawl-days-invalid",
			Severity: "warning",
			Title:    "Crawl-state retention age was reset",
			Message:  "RETENTION_CRAWL_STATE_DAYS must be non-negative, so Spartan is using unlimited retention for this session.",
		})
		cfg.RetentionCrawlStateDays = 0
	}
	if cfg.RetentionMaxJobs < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-max-jobs-invalid",
			Severity: "warning",
			Title:    "Retention max jobs was reset",
			Message:  "RETENTION_MAX_JOBS must be non-negative, so Spartan is using unlimited jobs for this session.",
		})
		cfg.RetentionMaxJobs = 0
	}
	if cfg.RetentionMaxStorageGB < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-max-storage-invalid",
			Severity: "warning",
			Title:    "Retention max storage was reset",
			Message:  "RETENTION_MAX_STORAGE_GB must be non-negative, so Spartan is using unlimited storage for this session.",
		})
		cfg.RetentionMaxStorageGB = 0
	}
	if cfg.RetentionCleanupIntervalHours <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-cleanup-interval-invalid",
			Severity: "warning",
			Title:    "Retention cleanup interval was reset",
			Message:  "RETENTION_CLEANUP_INTERVAL_HOURS must be positive, so Spartan is using 24 hours for this session.",
		})
		cfg.RetentionCleanupIntervalHours = 24
	}

	if !cfg.RetentionEnabled {
		hasLimits := cfg.RetentionJobDays > 0 || cfg.RetentionMaxJobs > 0 || cfg.RetentionMaxStorageGB > 0
		if hasLimits && hasExplicitRetentionLimitOverrides() {
			recordStartupNotice(StartupNotice{
				ID:       "retention-disabled-with-limits",
				Severity: "warning",
				Title:    "Retention limits are configured but inactive",
				Message:  "Retention limits are set while RETENTION_ENABLED is false, so automatic cleanup will not run until retention is enabled.",
			})
		}
	}

	return cfg
}

// validateAndFixCircuitBreakerConfig ensures circuit breaker configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixCircuitBreakerConfig(cfg Config) Config {
	if !cfg.CircuitBreakerEnabled {
		return cfg
	}

	if cfg.CircuitBreakerFailureThreshold <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "circuit-breaker-failure-threshold-invalid",
			Severity: "warning",
			Title:    "Circuit-breaker failure threshold was reset",
			Message:  "CIRCUIT_BREAKER_FAILURE_THRESHOLD must be positive, so Spartan is using 5 for this session.",
		})
		cfg.CircuitBreakerFailureThreshold = 5
	}
	if cfg.CircuitBreakerSuccessThreshold <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "circuit-breaker-success-threshold-invalid",
			Severity: "warning",
			Title:    "Circuit-breaker success threshold was reset",
			Message:  "CIRCUIT_BREAKER_SUCCESS_THRESHOLD must be positive, so Spartan is using 3 for this session.",
		})
		cfg.CircuitBreakerSuccessThreshold = 3
	}
	if cfg.CircuitBreakerResetTimeoutSecs <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "circuit-breaker-reset-timeout-invalid",
			Severity: "warning",
			Title:    "Circuit-breaker reset timeout was reset",
			Message:  "CIRCUIT_BREAKER_RESET_TIMEOUT_SECONDS must be positive, so Spartan is using 30 seconds for this session.",
		})
		cfg.CircuitBreakerResetTimeoutSecs = 30
	}
	if cfg.CircuitBreakerHalfOpenMaxRequests <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "circuit-breaker-half-open-invalid",
			Severity: "warning",
			Title:    "Circuit-breaker half-open limit was reset",
			Message:  "CIRCUIT_BREAKER_HALF_OPEN_MAX_REQUESTS must be positive, so Spartan is using 3 for this session.",
		})
		cfg.CircuitBreakerHalfOpenMaxRequests = 3
	}

	return cfg
}

// validateAndFixRetryConfig ensures retry configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixRetryConfig(cfg Config) Config {
	if cfg.RetryMaxDelaySecs < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retry-max-delay-invalid",
			Severity: "warning",
			Title:    "Retry max delay was reset",
			Message:  "RETRY_MAX_DELAY_SECONDS must be non-negative, so Spartan is using 60 seconds for this session.",
		})
		cfg.RetryMaxDelaySecs = 60
	}

	validStrategies := map[string]bool{
		"exponential":        true,
		"exponential_jitter": true,
		"exponential-jitter": true,
		"exponentialjitter":  true,
		"linear":             true,
		"fixed":              true,
	}

	strategyLower := strings.ToLower(cfg.RetryBackoffStrategy)
	if cfg.RetryBackoffStrategy != "" && !validStrategies[strategyLower] {
		recordStartupNotice(StartupNotice{
			ID:       "retry-backoff-strategy-invalid",
			Severity: "warning",
			Title:    "Retry backoff strategy was reset",
			Message:  fmt.Sprintf("RETRY_BACKOFF_STRATEGY %q is unsupported, so Spartan is using exponential_jitter for this session.", cfg.RetryBackoffStrategy),
		})
		cfg.RetryBackoffStrategy = "exponential_jitter"
	}

	if strategyLower == "exponential-jitter" || strategyLower == "exponentialjitter" {
		cfg.RetryBackoffStrategy = "exponential_jitter"
	}

	return cfg
}

func hasExplicitRetentionLimitOverrides() bool {
	defaults := map[string]string{
		"RETENTION_JOB_DAYS":         "30",
		"RETENTION_CRAWL_STATE_DAYS": "90",
		"RETENTION_MAX_JOBS":         "10000",
		"RETENTION_MAX_STORAGE_GB":   "10",
	}

	for key, defaultValue := range defaults {
		value, ok := lookupEnvNormalized(key)
		if !ok {
			continue
		}
		if value != "" && value != defaultValue {
			return true
		}
	}

	return false
}

func validateDataDir(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return apperrors.Wrap(apperrors.KindPermission,
			fmt.Sprintf("failed to create data directory %s", dataDir), err)
	}

	testFile := filepath.Join(dataDir, ".write-test")
	if err := os.WriteFile(testFile, []byte("write test"), 0o600); err != nil {
		return apperrors.Wrap(apperrors.KindPermission,
			fmt.Sprintf("data directory %s is not writable", dataDir), err)
	}

	_ = os.Remove(testFile)

	return nil
}

func loadAuthOverrides() EnvOverrides {
	overrides := EnvOverrides{
		Basic:        os.Getenv("AUTH_BASIC"),
		Bearer:       os.Getenv("AUTH_BEARER"),
		APIKey:       os.Getenv("AUTH_API_KEY"),
		APIKeyHeader: getenv("AUTH_API_KEY_HEADER", getenv("AUTH_TOKEN_API_KEY_HEADER", "")),
		APIKeyQuery:  os.Getenv("AUTH_API_KEY_QUERY"),
		APIKeyCookie: os.Getenv("AUTH_API_KEY_COOKIE"),
		Headers:      map[string]string{},
		Cookies:      map[string]string{},
	}

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		if value == "" {
			continue
		}

		if strings.HasPrefix(key, "AUTH_HEADER_") {
			name := normalizeAuthKeySuffix(strings.TrimPrefix(key, "AUTH_HEADER_"))
			if name != "" {
				overrides.Headers[name] = value
			}
		}
		if strings.HasPrefix(key, "AUTH_COOKIE_") {
			name := normalizeAuthKeySuffix(strings.TrimPrefix(key, "AUTH_COOKIE_"))
			if name != "" {
				overrides.Cookies[name] = value
			}
		}
	}

	if len(overrides.Headers) == 0 {
		overrides.Headers = nil
	}
	if len(overrides.Cookies) == 0 {
		overrides.Cookies = nil
	}
	return overrides
}

func normalizeAuthKeySuffix(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "__", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func recordInvalidEnvValue(key, value, fallback string) {
	recordStartupNotice(StartupNotice{
		ID:       fmt.Sprintf("invalid-env-%s", strings.ToLower(strings.ReplaceAll(key, "_", "-"))),
		Severity: "warning",
		Title:    fmt.Sprintf("%s used a fallback value", key),
		Message:  fmt.Sprintf("%s=%q is invalid, so Spartan is using %s for this session.", key, value, fallback),
	})
}

func getenv(key, fallback string) string {
	value, ok := lookupEnvNormalized(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func getenvAllowEmpty(key, fallback string) string {
	value, ok := lookupEnvNormalized(key)
	if !ok {
		return fallback
	}
	return value
}

func lookupEnvNormalized(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	return normalizeEnvValue(value), true
}

func normalizeEnvValue(value string) string {
	trimmedLeft := strings.TrimLeft(value, " \t")
	if strings.HasPrefix(trimmedLeft, "#") {
		return ""
	}

	if idx := strings.Index(value, " #"); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	if idx := strings.Index(value, "\t#"); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value, ok := lookupEnvNormalized(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		recordInvalidEnvValue(key, value, strconv.Itoa(fallback))
		return fallback
	}
	return parsed
}

func getenvInt64(key string, fallback int64) int64 {
	value, ok := lookupEnvNormalized(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		recordInvalidEnvValue(key, value, strconv.FormatInt(fallback, 10))
		return fallback
	}
	return parsed
}

func getenvBool(key string, fallback bool) bool {
	value, ok := lookupEnvNormalized(key)
	if !ok || value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y":
		return true
	case "0", "false", "no", "n":
		return false
	default:
		recordInvalidEnvValue(key, value, strconv.FormatBool(fallback))
		return fallback
	}
}

func getenvFloat64(key string, fallback float64) float64 {
	value, ok := lookupEnvNormalized(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		recordInvalidEnvValue(key, value, strconv.FormatFloat(fallback, 'f', -1, 64))
		return fallback
	}
	return parsed
}

// validateAndFixAIConfig ensures AI configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixAIConfig(cfg Config) Config {
	ai := AIConfig{
		Enabled:            getenvBool("PI_ENABLED", false),
		ConfigPath:         getenv("PI_CONFIG_PATH", ""),
		Mode:               getenv("PI_MODE", DefaultPIMode),
		NodeBin:            getenv("PI_NODE_BIN", DefaultPINodeBin),
		BridgeScript:       getenv("PI_BRIDGE_SCRIPT", DefaultPIBridgeScript),
		StartupTimeoutSecs: getenvInt("PI_STARTUP_TIMEOUT_SECONDS", DefaultPIStartupTimeoutSecs),
		RequestTimeoutSecs: getenvInt("PI_REQUEST_TIMEOUT_SECONDS", DefaultPIRequestTimeoutSecs),
		Routing:            DefaultAIRoutingConfig(),
	}

	if ai.StartupTimeoutSecs < 1 {
		recordStartupNotice(StartupNotice{
			ID:       "pi-startup-timeout-too-low",
			Severity: "warning",
			Title:    "AI startup timeout was raised",
			Message:  fmt.Sprintf("PI_STARTUP_TIMEOUT_SECONDS was %d, so Spartan raised it to the minimum 1 second.", ai.StartupTimeoutSecs),
		})
		ai.StartupTimeoutSecs = 1
	}
	if ai.StartupTimeoutSecs > 60 {
		recordStartupNotice(StartupNotice{
			ID:       "pi-startup-timeout-too-high",
			Severity: "warning",
			Title:    "AI startup timeout was capped",
			Message:  fmt.Sprintf("PI_STARTUP_TIMEOUT_SECONDS was %d, so Spartan capped it at 60 seconds.", ai.StartupTimeoutSecs),
		})
		ai.StartupTimeoutSecs = 60
	}

	if ai.RequestTimeoutSecs < 5 {
		recordStartupNotice(StartupNotice{
			ID:       "pi-request-timeout-too-low",
			Severity: "warning",
			Title:    "AI request timeout was raised",
			Message:  fmt.Sprintf("PI_REQUEST_TIMEOUT_SECONDS was %d, so Spartan raised it to the minimum 5 seconds.", ai.RequestTimeoutSecs),
		})
		ai.RequestTimeoutSecs = 5
	}
	if ai.RequestTimeoutSecs > 300 {
		recordStartupNotice(StartupNotice{
			ID:       "pi-request-timeout-too-high",
			Severity: "warning",
			Title:    "AI request timeout was capped",
			Message:  fmt.Sprintf("PI_REQUEST_TIMEOUT_SECONDS was %d, so Spartan capped it at 300 seconds.", ai.RequestTimeoutSecs),
		})
		ai.RequestTimeoutSecs = 300
	}

	if ai.Mode == "" {
		ai.Mode = DefaultPIMode
	}
	if ai.NodeBin == "" {
		ai.NodeBin = DefaultPINodeBin
	}
	if ai.BridgeScript == "" {
		ai.BridgeScript = DefaultPIBridgeScript
	}

	if ai.Enabled && ai.ConfigPath != "" {
		loaded, err := loadAIRoutingConfig(ai.ConfigPath)
		if err != nil {
			recordStartupNotice(StartupNotice{
				ID:       "pi-config-path-invalid",
				Severity: "warning",
				Title:    "AI routing config could not be loaded",
				Message:  fmt.Sprintf("PI_CONFIG_PATH %q could not be loaded (%v), so AI features were disabled for this session.", ai.ConfigPath, err),
			})
			ai.Enabled = false
		} else {
			if loaded.Mode != "" {
				ai.Mode = loaded.Mode
			}
			if len(loaded.Routes.Routes) > 0 {
				ai.Routing = loaded.Routes
			}
		}
	}

	cfg.AI = ai
	return cfg
}

func validateNoLegacyAIConfig() error {
	legacyKeys := []string{
		"AI_PROVIDER",
		"AI_API_KEY",
		"AI_MODEL",
		"AI_TIMEOUT_SECONDS",
		"AI_MAX_TOKENS",
		"AI_TEMPERATURE",
		"OLLAMA_URL",
	}

	used := make([]string, 0)
	for _, key := range legacyKeys {
		if strings.TrimSpace(getenv(key, "")) != "" {
			used = append(used, key)
		}
	}
	if len(used) == 0 {
		return nil
	}

	return apperrors.Validation(
		"legacy AI configuration is no longer supported: " + strings.Join(used, ", ") + ". Use PI_ENABLED and related PI_* bridge settings instead.",
	)
}

type aiBridgeConfigFile struct {
	Mode   string              `json:"mode"`
	Routes map[string][]string `json:"routes"`
}

type loadedAIRoutingConfig struct {
	Mode   string
	Routes AIRoutingConfig
}

func loadAIRoutingConfig(path string) (loadedAIRoutingConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return loadedAIRoutingConfig{}, err
	}

	var file aiBridgeConfigFile
	if err := json.Unmarshal(data, &file); err != nil {
		return loadedAIRoutingConfig{}, err
	}

	routing := DefaultAIRoutingConfig()
	for capability, routes := range file.Routes {
		normalized := make([]string, 0, len(routes))
		for _, route := range routes {
			trimmed := strings.TrimSpace(route)
			if trimmed != "" {
				normalized = append(normalized, trimmed)
			}
		}
		if len(normalized) > 0 {
			routing.Routes[capability] = normalized
		}
	}

	return loadedAIRoutingConfig{
		Mode:   strings.TrimSpace(file.Mode),
		Routes: routing,
	}, nil
}
