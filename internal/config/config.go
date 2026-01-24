package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"spartan-scraper/internal/auth"
)

// EnvOverrides is an alias for auth.EnvOverrides
type EnvOverrides = auth.EnvOverrides

type Config struct {
	Port               string
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

func Load() Config {
	_ = godotenv.Load()
	return Config{
		Port:               getenv("PORT", "8741"),
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
