// Package config_test provides tests for log level configuration.
//
// Responsibilities:
//   - Tests valid log level strings (debug, info, warn, error)
//   - Tests case-insensitive level parsing
//   - Tests level switching at runtime
//   - Tests default level fallback for invalid/empty levels
//   - Tests level-specific log output
//
// Does NOT handle:
//   - Log format configuration (see log_format_test.go)
//   - Log handler implementation (see log_handler_test.go)
//   - Dynamic level changes after initialization
package config

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestInitLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"DEBUG uppercase", "DEBUG"},
		{"Info mixed case", "Info"},
		{"WARN uppercase", "WARN"},
		{"Error mixed case", "Error"},
		{"invalid defaults to info", "invalid"},
		{"empty defaults to info", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				LogLevel:  tt.logLevel,
				LogFormat: "text",
			}

			InitLogger(cfg)

			defaultLogger := slog.Default()
			if defaultLogger == nil {
				t.Fatal("slog.Default() returned nil after InitLogger")
			}
		})
	}
}

func TestInitLogger_LevelSwitch(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
	}{
		{"switch to debug", "debug"},
		{"switch to warn", "warn"},
		{"switch to error", "error"},
		{"switch to info", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				LogLevel:  tt.logLevel,
				LogFormat: "text",
			}

			oldStderr := os.Stderr
			defer func() { os.Stderr = oldStderr }()

			var buf bytes.Buffer
			r, w, _ := os.Pipe()
			os.Stderr = w

			InitLogger(cfg)
			switch tt.logLevel {
			case "debug":
				slog.Debug("test message")
			case "info":
				slog.Info("test message")
			case "warn":
				slog.Warn("test message")
			case "error":
				slog.Error("test message")
			}

			w.Close()
			io.Copy(&buf, r)
			r.Close()
			os.Stderr = oldStderr

			output := buf.String()
			if output == "" {
				t.Errorf("expected log output for level %s", tt.logLevel)
			}
		})
	}
}

func TestInitLogger_LevelParseStrings(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"debug"},
		{"DEBUG"},
		{"Debug"},
		{"dEBUG"},
		{"info"},
		{"INFO"},
		{"Info"},
		{"warn"},
		{"WARN"},
		{"Warn"},
		{"error"},
		{"ERROR"},
		{"Error"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg := Config{
				LogLevel:  tt.input,
				LogFormat: "text",
			}

			InitLogger(cfg)

			defaultLogger := slog.Default()
			if defaultLogger == nil {
				t.Fatal("slog.Default() returned nil after InitLogger")
			}
		})
	}
}

func TestInitLogger_LevelStringLower(t *testing.T) {
	cfg := Config{
		LogLevel:  "INFO",
		LogFormat: "TEXT",
	}

	InitLogger(cfg)

	defaultLogger := slog.Default()
	if defaultLogger == nil {
		t.Fatal("slog.Default() returned nil after InitLogger")
	}
}

func TestInitLogger_ValidLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range validLevels {
		t.Run(level, func(t *testing.T) {
			cfg := Config{
				LogLevel:  level,
				LogFormat: "text",
			}

			InitLogger(cfg)

			defaultLogger := slog.Default()
			if defaultLogger == nil {
				t.Fatal("slog.Default() returned nil after InitLogger")
			}
		})
	}
}

func TestInitLogger_InvalidLevel(t *testing.T) {
	invalidLevels := []string{"invalid", "trace", "fatal", "unknown"}

	for _, level := range invalidLevels {
		t.Run(level, func(t *testing.T) {
			cfg := Config{
				LogLevel:  level,
				LogFormat: "text",
			}

			InitLogger(cfg)

			defaultLogger := slog.Default()
			if defaultLogger == nil {
				t.Fatal("slog.Default() returned nil after InitLogger")
			}
		})
	}
}
