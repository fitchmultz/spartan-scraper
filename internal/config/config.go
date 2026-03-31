// Package config provides application configuration loading from environment variables.
//
// Purpose:
// - Define the immutable startup configuration types and AI routing defaults shared across the application.
//
// Responsibilities:
// - Declare the Config snapshot and related configuration domain types.
// - Expose the canonical AI capability list and default routing behavior.
// - Document the package-wide immutability and thread-safety contract.
//
// Scope:
// - Type definitions and static defaults only; loading and validation live in focused companion files.
//
// Usage:
// - Call Load() once during process startup and pass the returned Config by value.
//
// Invariants/Assumptions:
// - Config is treated as immutable after loading.
// - AuthOverrides.Headers and AuthOverrides.Cookies are read-only map fields unless callers deep-copy them first.
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
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
)

// EnvOverrides is an alias for auth.EnvOverrides
// so callers can consume config-level auth overrides without importing internal/auth directly.
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
	ProxyPoolFile string // Explicit path to proxy pool JSON config; empty disables proxy pooling

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

var aiCapabilities = []string{
	AICapabilityExtractNatural,
	AICapabilityExtractSchema,
	AICapabilityTemplateGeneration,
	AICapabilityRenderProfile,
	AICapabilityPipelineJS,
	AICapabilityResearchRefine,
	AICapabilityExportShape,
	AICapabilityTransformGenerate,
}

// AllAICapabilities returns the canonical list of capability keys used across Go and bridge config.
func AllAICapabilities() []string {
	return append([]string(nil), aiCapabilities...)
}

// AIRoutingConfig maps AI capabilities to ordered provider/model routes.
type AIRoutingConfig struct {
	Routes map[string][]string `json:"routes"`
}

func normalizeAIRouteList(routes []string) []string {
	if routes == nil {
		return nil
	}
	out := make([]string, 0, len(routes))
	for _, route := range routes {
		trimmed := strings.TrimSpace(route)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

// RoutesFor returns the configured routes for a capability.
// It returns nil when the capability is absent and an empty slice when explicitly disabled.
func (r AIRoutingConfig) RoutesFor(capability string) []string {
	if len(r.Routes) == 0 {
		return nil
	}
	routes, ok := r.Routes[capability]
	if !ok {
		return nil
	}
	return normalizeAIRouteList(routes)
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
