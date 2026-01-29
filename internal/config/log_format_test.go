package config

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestInitLogger_Formats(t *testing.T) {
	tests := []struct {
		name       string
		logFormat  string
		expectJSON bool
	}{
		{"json", "json", true},
		{"JSON uppercase", "JSON", true},
		{"Json mixed case", "Json", true},
		{"text", "text", false},
		{"TEXT uppercase", "TEXT", false},
		{"Text mixed case", "Text", false},
		{"invalid defaults to text", "invalid", false},
		{"empty defaults to text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				LogLevel:  "info",
				LogFormat: tt.logFormat,
			}

			oldStderr := os.Stderr
			defer func() { os.Stderr = oldStderr }()

			var buf bytes.Buffer
			r, w, _ := os.Pipe()
			os.Stderr = w

			InitLogger(cfg)
			slog.Info("test message")

			w.Close()
			io.Copy(&buf, r)
			r.Close()
			os.Stderr = oldStderr

			output := buf.String()
			if tt.expectJSON && !isJSONOutput(output) {
				t.Errorf("expected JSON output, got: %s", output)
			}
			if !tt.expectJSON && !isTextOutput(output) {
				t.Errorf("expected text output, got: %s", output)
			}
		})
	}
}

func TestInitLogger_FormatSwitch(t *testing.T) {
	tests := []struct {
		name       string
		logFormat  string
		expectJSON bool
	}{
		{"switch to json", "json", true},
		{"switch to text", "text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				LogLevel:  "info",
				LogFormat: tt.logFormat,
			}

			oldStderr := os.Stderr
			defer func() { os.Stderr = oldStderr }()

			var buf bytes.Buffer
			r, w, _ := os.Pipe()
			os.Stderr = w

			InitLogger(cfg)
			slog.Info("test message")

			w.Close()
			io.Copy(&buf, r)
			r.Close()
			os.Stderr = oldStderr

			output := buf.String()
			if tt.expectJSON && !isJSONOutput(output) {
				t.Errorf("expected JSON output, got: %s", output)
			}
			if !tt.expectJSON && !isTextOutput(output) {
				t.Errorf("expected text output, got: %s", output)
			}
		})
	}
}

func TestInitLogger_FormatParseStrings(t *testing.T) {
	tests := []struct {
		input      string
		expectJSON bool
	}{
		{"json", true},
		{"JSON", true},
		{"Json", true},
		{"jSoN", true},
		{"text", false},
		{"TEXT", false},
		{"Text", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg := Config{
				LogLevel:  "info",
				LogFormat: tt.input,
			}

			oldStderr := os.Stderr
			defer func() { os.Stderr = oldStderr }()

			var buf bytes.Buffer
			r, w, _ := os.Pipe()
			os.Stderr = w

			InitLogger(cfg)
			slog.Info("test message")

			w.Close()
			io.Copy(&buf, r)
			r.Close()
			os.Stderr = oldStderr

			output := buf.String()
			if tt.expectJSON && !isJSONOutput(output) {
				t.Errorf("expected JSON output, got: %s", output)
			}
			if !tt.expectJSON && isJSONOutput(output) {
				t.Errorf("expected text output, got JSON: %s", output)
			}
		})
	}
}

func TestInitLogger_UnknownFormatDefaultsToText(t *testing.T) {
	unknownFormats := []string{"xml", "yaml", "csv", "binary"}

	for _, format := range unknownFormats {
		t.Run(format, func(t *testing.T) {
			cfg := Config{
				LogLevel:  "info",
				LogFormat: format,
			}

			oldStderr := os.Stderr
			defer func() { os.Stderr = oldStderr }()

			var buf bytes.Buffer
			r, w, _ := os.Pipe()
			os.Stderr = w

			InitLogger(cfg)
			slog.Info("test message")

			w.Close()
			io.Copy(&buf, r)
			r.Close()
			os.Stderr = oldStderr

			output := buf.String()
			if isJSONOutput(output) {
				t.Errorf("expected text output for unknown format %s, got JSON", format)
			}
		})
	}
}
