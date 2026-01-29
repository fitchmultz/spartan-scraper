package config

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestInitLogger_DefaultHandler(t *testing.T) {
	cfg := Config{
		LogLevel:  "info",
		LogFormat: "text",
	}

	oldLogger := slog.Default()
	InitLogger(cfg)

	newLogger := slog.Default()
	if newLogger == nil {
		t.Fatal("slog.Default() returned nil after InitLogger")
	}

	if newLogger == oldLogger {
		t.Error("InitLogger did not update default logger")
	}
}

func TestInitLogger_LogsToStderr(t *testing.T) {
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	var buf bytes.Buffer
	r, w, _ := os.Pipe()
	os.Stderr = w

	cfg := Config{
		LogLevel:  "info",
		LogFormat: "text",
	}

	InitLogger(cfg)
	slog.Info("test message")

	w.Close()
	io.Copy(&buf, r)
	r.Close()
	os.Stderr = oldStderr

	output := buf.String()
	if output == "" {
		t.Error("no output written to stderr")
	}
}

func TestInitLogger_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logFormat string
	}{
		{"all uppercase", "DEBUG", "JSON"},
		{"all lowercase", "debug", "json"},
		{"mixed case", "DeBuG", "JsOn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				LogLevel:  tt.logLevel,
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
			if !isJSONOutput(output) {
				t.Errorf("expected JSON output, got: %s", output)
			}
		})
	}
}

func TestInitLogger_HandlerType(t *testing.T) {
	tests := []struct {
		name       string
		logFormat  string
		expectJSON bool
	}{
		{"json handler", "json", true},
		{"text handler", "text", false},
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
			if !tt.expectJSON && isJSONOutput(output) {
				t.Errorf("expected text output, got JSON: %s", output)
			}
		})
	}
}

func TestInitLogger_MultipleCalls(t *testing.T) {
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	var buf1 bytes.Buffer
	r1, w1, _ := os.Pipe()
	os.Stderr = w1

	cfg1 := Config{
		LogLevel:  "debug",
		LogFormat: "json",
	}

	InitLogger(cfg1)
	slog.Info("message1")

	w1.Close()
	io.Copy(&buf1, r1)
	r1.Close()
	output1 := buf1.String()

	var buf2 bytes.Buffer
	r2, w2, _ := os.Pipe()
	os.Stderr = w2

	cfg2 := Config{
		LogLevel:  "error",
		LogFormat: "text",
	}

	InitLogger(cfg2)
	slog.Info("message2")

	w2.Close()
	io.Copy(&buf2, r2)
	r2.Close()
	output2 := buf2.String()

	os.Stderr = oldStderr

	if isJSONOutput(output1) != true {
		t.Error("first call should produce JSON output")
	}
	if isJSONOutput(output2) != false {
		t.Error("second call should produce text output")
	}
}

func TestInitLogger_StripWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
	}{
		{"debug with spaces", " debug "},
		{"info with tabs", "\tinfo\t"},
		{"warn with newline", "\nwarn\n"},
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
