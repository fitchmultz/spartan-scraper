// Purpose: Provide shared environment parsing, startup-notice bookkeeping, and filesystem validation helpers for config loading.
// Responsibilities:
// - Normalize environment values and parse typed fallbacks consistently.
// - Record deduplicated startup notices for non-fatal configuration problems.
// - Validate writable data directories and load auth override maps from environment variables.
// Scope:
// - Internal config package helpers only; no product-level runtime behavior lives here.
// Usage:
// - Use getenv* helpers inside config loaders and validators.
// Invariants/Assumptions:
// - Startup notices are accumulated only during Load() and consumed before Load() returns.
// - Auth override maps are returned as read-only snapshots.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

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
