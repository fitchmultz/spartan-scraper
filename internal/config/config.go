package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

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
	UsePlaywright      bool
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
		UsePlaywright:      getenvBool("USE_PLAYWRIGHT", false),
	}
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
