// File log.go configures the default slog logger based on Config.
//
// It does NOT implement configuration loading (see config.go) and it does NOT implement
// runtime log-level switching; it is intended to run once at startup.
package config

import (
	"log/slog"
	"os"
	"strings"
)

func InitLogger(cfg Config) {
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if strings.ToLower(cfg.LogFormat) == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
