package config

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"
)

type testHandler struct {
	buf    *bytes.Buffer
	inner  slog.Handler
	output string
}

func (h *testHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *testHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}
	h.output = h.buf.String()
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &testHandler{
		buf:    h.buf,
		inner:  h.inner.WithAttrs(attrs),
		output: h.output,
	}
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	return &testHandler{
		buf:    h.buf,
		inner:  h.inner.WithGroup(name),
		output: h.output,
	}
}

func isJSONOutput(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func isTextOutput(s string) bool {
	return !isJSONOutput(s)
}

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
