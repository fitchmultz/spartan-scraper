package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DataDir            string
	UserAgent          string
	MaxConcurrency     int
	RequestTimeoutSecs int
}

func Load() Config {
	_ = godotenv.Load()
	return Config{
		Port:               getenv("PORT", "8741"),
		DataDir:            getenv("DATA_DIR", ".data"),
		UserAgent:          getenv("USER_AGENT", "SpartanScraper/0.1 (+https://local)"),
		MaxConcurrency:     getenvInt("MAX_CONCURRENCY", 4),
		RequestTimeoutSecs: getenvInt("REQUEST_TIMEOUT_SECONDS", 30),
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
