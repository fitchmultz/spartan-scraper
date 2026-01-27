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
	"os"
	"strconv"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/joho/godotenv"
)

// EnvOverrides is an alias for auth.EnvOverrides
type EnvOverrides = auth.EnvOverrides

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
}

// Load reads configuration from environment variables (optionally loading defaults from
// a local .env file).
//
// Intended usage: call Load once during application startup, then pass the returned Config
// value into constructors/handlers.
//
// Load does not maintain any singleton/global Config instance; it simply returns a value.
// The returned Config is treated as immutable after loading.
func Load() Config {
	_ = godotenv.Load()
	return Config{
		Port:     getenv("PORT", "8741"),
		BindAddr: getenv("BIND_ADDR", "127.0.0.1"),

		ServerReadHeaderTimeoutSecs: getenvInt("SERVER_READ_HEADER_TIMEOUT_SECONDS", 10),
		ServerReadTimeoutSecs:       getenvInt("SERVER_READ_TIMEOUT_SECONDS", 30),
		ServerWriteTimeoutSecs:      getenvInt("SERVER_WRITE_TIMEOUT_SECONDS", 60),
		ServerIdleTimeoutSecs:       getenvInt("SERVER_IDLE_TIMEOUT_SECONDS", 120),

		DataDir:            getenv("DATA_DIR", ".data"),
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
	}
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
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvInt64(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y":
		return true
	case "0", "false", "no", "n":
		return false
	default:
		return fallback
	}
}
