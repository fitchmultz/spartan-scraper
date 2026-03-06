// Package config provides application configuration loading from environment variables.
// It handles loading defaults from .env files and parsing environment variables.
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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	AllowInternal           bool // Allow webhooks to internal/private addresses (default: false for security)
	MaxConcurrentDispatches int  // Maximum concurrent webhook dispatches (default: 100)
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
	ReplayTimeoutSecs  int // REPLAY_REQUEST_TIMEOUT_SECONDS env var (default: 30)
	RateLimitQPS       int
	RateLimitBurst     int
	MaxRetries         int
	RetryBaseMs        int
	MaxResponseBytes   int64
	UsePlaywright      bool
	AuthOverrides      EnvOverrides
	LogLevel           string
	LogFormat          string

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

	// Queue backend configuration
	QueueBackend string // QUEUE_BACKEND env var (default: "memory", options: "memory", "redis")

	// Redis configuration (used when QueueBackend="redis")
	RedisAddr      string // REDIS_ADDR env var (default: "localhost:6379")
	RedisPassword  string // REDIS_PASSWORD env var
	RedisDB        int    // REDIS_DB env var (default: 0)
	RedisKeyPrefix string // REDIS_KEY_PREFIX env var (default: "spartan:")

	// Distributed state configuration
	CrawlStateBackend string // CRAWL_STATE_BACKEND env var (default: "sqlite", options: "sqlite", "redis")

	// AI extraction configuration
	AI AIConfig
}

// AIProvider identifies the LLM provider for AI-powered extraction.
type AIProvider string

const (
	AIProviderOpenAI    AIProvider = "openai"
	AIProviderAnthropic AIProvider = "anthropic"
	AIProviderOllama    AIProvider = "ollama"
)

// AIConfig holds configuration for AI-powered extraction.
type AIConfig struct {
	Provider    AIProvider
	APIKey      string
	Model       string
	TimeoutSecs int
	MaxTokens   int
	Temperature float64
	OllamaURL   string
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
		ReplayTimeoutSecs:  getenvInt("REPLAY_REQUEST_TIMEOUT_SECONDS", 30),
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
		ProxyPoolFile: getenv("PROXY_POOL_FILE", filepath.Join(dataDir, "proxy_pool.json")),

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

		// Queue backend configuration
		QueueBackend:   getenv("QUEUE_BACKEND", "memory"),
		RedisAddr:      getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getenv("REDIS_PASSWORD", ""),
		RedisDB:        getenvInt("REDIS_DB", 0),
		RedisKeyPrefix: getenv("REDIS_KEY_PREFIX", "spartan:"),

		// Distributed state configuration
		CrawlStateBackend: getenv("CRAWL_STATE_BACKEND", "sqlite"),
	}

	if err := validateDataDir(cfg.DataDir); err != nil {
		return Config{}, err
	}

	cfg = validateAndFixAdaptiveConfig(cfg)
	cfg = validateAndFixRetentionConfig(cfg)
	cfg = validateAndFixCircuitBreakerConfig(cfg)
	cfg = validateAndFixRetryConfig(cfg)
	cfg = validateAndFixQueueConfig(cfg)
	cfg = validateAndFixAIConfig(cfg)

	return cfg, nil
}

// validateAndFixAdaptiveConfig ensures adaptive rate limiting configuration invariants.
// It logs warnings and applies sensible defaults for invalid configurations.
func validateAndFixAdaptiveConfig(cfg Config) Config {
	if !cfg.AdaptiveRateLimit {
		return cfg
	}

	// Ensure MinQPS <= MaxQPS
	if cfg.AdaptiveMinQPS > cfg.AdaptiveMaxQPS {
		fmt.Fprintf(os.Stderr, "[WARN] ADAPTIVE_MIN_QPS (%.2f) > ADAPTIVE_MAX_QPS (%.2f), swapping values\n",
			cfg.AdaptiveMinQPS, cfg.AdaptiveMaxQPS)
		cfg.AdaptiveMinQPS, cfg.AdaptiveMaxQPS = cfg.AdaptiveMaxQPS, cfg.AdaptiveMinQPS
	}

	// Ensure MinQPS is positive and finite
	if cfg.AdaptiveMinQPS <= 0 {
		fmt.Fprintf(os.Stderr, "[WARN] ADAPTIVE_MIN_QPS must be positive, using default 0.1\n")
		cfg.AdaptiveMinQPS = 0.1
	}

	// Ensure MaxQPS is positive and finite
	if cfg.AdaptiveMaxQPS <= 0 {
		fmt.Fprintf(os.Stderr, "[WARN] ADAPTIVE_MAX_QPS must be positive, using RATE_LIMIT_QPS\n")
		cfg.AdaptiveMaxQPS = float64(cfg.RateLimitQPS)
	}

	// Ensure decrease factor is in valid range (0, 1)
	if cfg.AdaptiveDecreaseFactor <= 0 || cfg.AdaptiveDecreaseFactor >= 1 {
		fmt.Fprintf(os.Stderr, "[WARN] ADAPTIVE_DECREASE_FACTOR must be in (0, 1), using default 0.5\n")
		cfg.AdaptiveDecreaseFactor = 0.5
	}

	// Ensure increase amount is positive
	if cfg.AdaptiveIncreaseQPS <= 0 {
		fmt.Fprintf(os.Stderr, "[WARN] ADAPTIVE_INCREASE_QPS must be positive, using default 0.5\n")
		cfg.AdaptiveIncreaseQPS = 0.5
	}

	// Ensure success threshold is positive
	if cfg.AdaptiveSuccessThreshold <= 0 {
		cfg.AdaptiveSuccessThreshold = 5
	}

	// Ensure cooldown is non-negative
	if cfg.AdaptiveCooldownMs < 0 {
		cfg.AdaptiveCooldownMs = 1000
	}

	return cfg
}

// validateAndFixRetentionConfig ensures retention configuration invariants.
// It logs warnings and applies sensible defaults for invalid configurations.
func validateAndFixRetentionConfig(cfg Config) Config {
	// Ensure non-negative values
	if cfg.RetentionJobDays < 0 {
		fmt.Fprintf(os.Stderr, "[WARN] RETENTION_JOB_DAYS must be non-negative, using 0 (unlimited)\n")
		cfg.RetentionJobDays = 0
	}
	if cfg.RetentionCrawlStateDays < 0 {
		fmt.Fprintf(os.Stderr, "[WARN] RETENTION_CRAWL_STATE_DAYS must be non-negative, using 0 (unlimited)\n")
		cfg.RetentionCrawlStateDays = 0
	}
	if cfg.RetentionMaxJobs < 0 {
		fmt.Fprintf(os.Stderr, "[WARN] RETENTION_MAX_JOBS must be non-negative, using 0 (unlimited)\n")
		cfg.RetentionMaxJobs = 0
	}
	if cfg.RetentionMaxStorageGB < 0 {
		fmt.Fprintf(os.Stderr, "[WARN] RETENTION_MAX_STORAGE_GB must be non-negative, using 0 (unlimited)\n")
		cfg.RetentionMaxStorageGB = 0
	}
	if cfg.RetentionCleanupIntervalHours <= 0 {
		fmt.Fprintf(os.Stderr, "[WARN] RETENTION_CLEANUP_INTERVAL_HOURS must be positive, using default 24\n")
		cfg.RetentionCleanupIntervalHours = 24
	}

	// Log warning if retention is disabled but custom limits are explicitly set.
	// Defaults in code are intentionally non-zero and should not warn on first run.
	if !cfg.RetentionEnabled {
		hasLimits := cfg.RetentionJobDays > 0 || cfg.RetentionMaxJobs > 0 || cfg.RetentionMaxStorageGB > 0
		if hasLimits && hasExplicitRetentionLimitOverrides() {
			fmt.Fprintf(os.Stderr, "[WARN] Retention limits are set but RETENTION_ENABLED is false; cleanup will not run automatically\n")
		}
	}

	return cfg
}

// validateAndFixCircuitBreakerConfig ensures circuit breaker configuration invariants.
// It logs warnings and applies sensible defaults for invalid configurations.
func validateAndFixCircuitBreakerConfig(cfg Config) Config {
	if !cfg.CircuitBreakerEnabled {
		return cfg
	}

	// Ensure positive thresholds
	if cfg.CircuitBreakerFailureThreshold <= 0 {
		fmt.Fprintf(os.Stderr, "[WARN] CIRCUIT_BREAKER_FAILURE_THRESHOLD must be positive, using default 5\n")
		cfg.CircuitBreakerFailureThreshold = 5
	}
	if cfg.CircuitBreakerSuccessThreshold <= 0 {
		fmt.Fprintf(os.Stderr, "[WARN] CIRCUIT_BREAKER_SUCCESS_THRESHOLD must be positive, using default 3\n")
		cfg.CircuitBreakerSuccessThreshold = 3
	}
	if cfg.CircuitBreakerResetTimeoutSecs <= 0 {
		fmt.Fprintf(os.Stderr, "[WARN] CIRCUIT_BREAKER_RESET_TIMEOUT_SECONDS must be positive, using default 30\n")
		cfg.CircuitBreakerResetTimeoutSecs = 30
	}
	if cfg.CircuitBreakerHalfOpenMaxRequests <= 0 {
		fmt.Fprintf(os.Stderr, "[WARN] CIRCUIT_BREAKER_HALF_OPEN_MAX_REQUESTS must be positive, using default 3\n")
		cfg.CircuitBreakerHalfOpenMaxRequests = 3
	}

	return cfg
}

// validateAndFixRetryConfig ensures retry configuration invariants.
// It logs warnings and applies sensible defaults for invalid configurations.
func validateAndFixRetryConfig(cfg Config) Config {
	// Validate retry max delay
	if cfg.RetryMaxDelaySecs < 0 {
		fmt.Fprintf(os.Stderr, "[WARN] RETRY_MAX_DELAY_SECONDS must be non-negative, using default 60\n")
		cfg.RetryMaxDelaySecs = 60
	}

	// Validate backoff strategy
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
		fmt.Fprintf(os.Stderr, "[WARN] Invalid RETRY_BACKOFF_STRATEGY: %q, using default 'exponential_jitter'\n", cfg.RetryBackoffStrategy)
		cfg.RetryBackoffStrategy = "exponential_jitter"
	}

	// Normalize to canonical form
	if strategyLower == "exponential-jitter" || strategyLower == "exponentialjitter" {
		cfg.RetryBackoffStrategy = "exponential_jitter"
	}

	return cfg
}

// validateAndFixQueueConfig ensures queue backend configuration invariants.
// It logs warnings and applies sensible defaults for invalid configurations.
func validateAndFixQueueConfig(cfg Config) Config {
	// Validate queue backend
	validBackends := map[string]bool{
		"memory": true,
		"redis":  true,
	}
	if !validBackends[cfg.QueueBackend] {
		fmt.Fprintf(os.Stderr, "[WARN] Invalid QUEUE_BACKEND: %q, using default 'memory'\n", cfg.QueueBackend)
		cfg.QueueBackend = "memory"
	}

	// Validate crawl state backend
	validStateBackends := map[string]bool{
		"sqlite": true,
		"redis":  true,
	}
	if !validStateBackends[cfg.CrawlStateBackend] {
		fmt.Fprintf(os.Stderr, "[WARN] Invalid CRAWL_STATE_BACKEND: %q, using default 'sqlite'\n", cfg.CrawlStateBackend)
		cfg.CrawlStateBackend = "sqlite"
	}

	// If queue backend is redis but crawl state is not explicitly set, use redis for state too
	if cfg.QueueBackend == "redis" && cfg.CrawlStateBackend == "sqlite" {
		// Check if CRAWL_STATE_BACKEND was explicitly set
		if os.Getenv("CRAWL_STATE_BACKEND") == "" {
			cfg.CrawlStateBackend = "redis"
		}
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

func getenv(key, fallback string) string {
	value, ok := lookupEnvNormalized(key)
	if !ok || value == "" {
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
		fmt.Fprintf(os.Stderr, "[WARN] Invalid value for %s: %q (using default: %d)\n", key, value, fallback)
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
		fmt.Fprintf(os.Stderr, "[WARN] Invalid value for %s: %q (using default: %d)\n", key, value, fallback)
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
		fmt.Fprintf(os.Stderr, "[WARN] Invalid value for %s: %q (using default: %t)\n", key, value, fallback)
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
		fmt.Fprintf(os.Stderr, "[WARN] Invalid value for %s: %q (using default: %f)\n", key, value, fallback)
		return fallback
	}
	return parsed
}

// validateAndFixAIConfig ensures AI configuration invariants.
// It logs warnings and applies sensible defaults for invalid configurations.
func validateAndFixAIConfig(cfg Config) Config {
	ai := AIConfig{
		Provider:    AIProvider(getenv("AI_PROVIDER", "")),
		APIKey:      getenv("AI_API_KEY", ""),
		Model:       getenv("AI_MODEL", ""),
		TimeoutSecs: getenvInt("AI_TIMEOUT_SECONDS", 60),
		MaxTokens:   getenvInt("AI_MAX_TOKENS", 4096),
		Temperature: getenvFloat64("AI_TEMPERATURE", 0.1),
		OllamaURL:   getenv("OLLAMA_URL", "http://localhost:11434"),
	}

	// If AI provider is set, validate it
	if ai.Provider != "" {
		validProviders := map[AIProvider]bool{
			AIProviderOpenAI:    true,
			AIProviderAnthropic: true,
			AIProviderOllama:    true,
		}
		if !validProviders[ai.Provider] {
			fmt.Fprintf(os.Stderr, "[WARN] Invalid AI_PROVIDER: %q, AI features disabled\n", ai.Provider)
			ai.Provider = ""
		}

		// Validate timeout bounds
		if ai.TimeoutSecs < 5 {
			fmt.Fprintf(os.Stderr, "[WARN] AI_TIMEOUT_SECONDS too low (%d), using minimum 5\n", ai.TimeoutSecs)
			ai.TimeoutSecs = 5
		}
		if ai.TimeoutSecs > 300 {
			fmt.Fprintf(os.Stderr, "[WARN] AI_TIMEOUT_SECONDS too high (%d), using maximum 300\n", ai.TimeoutSecs)
			ai.TimeoutSecs = 300
		}

		// Validate temperature bounds
		if ai.Temperature < 0 || ai.Temperature > 1.0 {
			fmt.Fprintf(os.Stderr, "[WARN] AI_TEMPERATURE out of range (%f), using 0.1\n", ai.Temperature)
			ai.Temperature = 0.1
		}

		// Set default models if not specified
		if ai.Model == "" {
			switch ai.Provider {
			case AIProviderOpenAI:
				ai.Model = "gpt-4o-mini"
			case AIProviderAnthropic:
				ai.Model = "claude-3-haiku-20240307"
			case AIProviderOllama:
				ai.Model = "llama3.1"
			}
		}

		// Validate API key is present for cloud providers
		if ai.Provider == AIProviderOpenAI || ai.Provider == AIProviderAnthropic {
			if ai.APIKey == "" {
				fmt.Fprintf(os.Stderr, "[WARN] AI_PROVIDER is %s but AI_API_KEY is not set, AI features disabled\n", ai.Provider)
				ai.Provider = ""
			}
		}
	}

	cfg.AI = ai
	return cfg
}
