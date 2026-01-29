package scrape

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestLoadExtractOptions_NoConfigPath(t *testing.T) {
	opts, err := loadExtractOptions("", "", false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.Template != "" {
		t.Errorf("expected empty template, got %q", opts.Template)
	}
	if opts.Validate {
		t.Errorf("expected Validate=false, got true")
	}
	if opts.Inline != nil {
		t.Errorf("expected Inline=nil, got %v", opts.Inline)
	}
}

func TestLoadExtractOptions_ValidTemplateNameOnly(t *testing.T) {
	opts, err := loadExtractOptions("default", "", false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.Template != "default" {
		t.Errorf("expected Template=\"default\", got %q", opts.Template)
	}
	if opts.Validate {
		t.Errorf("expected Validate=false, got true")
	}
	if opts.Inline != nil {
		t.Errorf("expected Inline=nil, got %v", opts.Inline)
	}
}

func TestLoadExtractOptions_ValidExtractConfigJSON(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "extract-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	validTemplate := `{"name":"test","selectors":[{"name":"title","selector":"h1"}]}`
	if _, err := tempFile.WriteString(validTemplate); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	opts, err := loadExtractOptions("", tempFile.Name(), true)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.Inline == nil {
		t.Fatal("expected Inline to be set")
	}
	if opts.Inline.Name != "test" {
		t.Errorf("expected Inline.Name=\"test\", got %q", opts.Inline.Name)
	}
	if !opts.Validate {
		t.Errorf("expected Validate=true, got false")
	}
}

func TestLoadExtractOptions_InvalidJSONInConfigFile(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "extract-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	if _, err := tempFile.WriteString("{invalid}"); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	_, err = loadExtractOptions("", tempFile.Name(), false)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if err.Error() != "invalid template JSON: invalid character 'i' looking for beginning of object key string" &&
		err.Error() != "invalid template JSON: invalid character 'i' looking for beginning of object key string" {
		t.Errorf("expected error to contain 'invalid template JSON', got %q", err.Error())
	}
}

func TestLoadExtractOptions_NonExistentConfigFile(t *testing.T) {
	_, err := loadExtractOptions("", "/nonexistent/path/config.json", false)

	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestLoadExtractOptions_MalformedTemplateJSON(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "extract-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	if _, err := tempFile.WriteString("{}"); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	opts, err := loadExtractOptions("", tempFile.Name(), false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.Inline == nil {
		t.Fatal("expected Inline to be set even for empty template")
	}
	if opts.Inline.Name != "" {
		t.Errorf("expected empty name, got %q", opts.Inline.Name)
	}
}

func TestRunScrape_MissingURLFlag(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunScrape(ctx, cfg, []string{})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "--url is required") {
		t.Errorf("expected stderr to contain '--url is required', got %q", stderr)
	}
}

func TestRunScrape_InvalidURLScheme(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunScrape(ctx, cfg, []string{"--url", "ftp://example.com"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "invalid url") {
		t.Errorf("expected stderr to contain validation error about URL, got %q", stderr)
	}
}

func TestRunScrape_InvalidExtractConfigJSON(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	tempFile, err := os.CreateTemp(t.TempDir(), "extract-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	if _, err := tempFile.WriteString("{invalid}"); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunScrape(ctx, cfg, []string{"--url", "https://example.com", "--extract-config", tempFile.Name()})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "invalid template JSON") {
		t.Errorf("expected stderr to contain 'invalid template JSON', got %q", stderr)
	}
}

func TestRunScrape_ValidFlagsCreateJob(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	cfg := config.Config{
		DataDir:            tmpDir,
		UsePlaywright:      false,
		RequestTimeoutSecs: 30,
		MaxConcurrency:     1,
		RateLimitQPS:       10,
		RateLimitBurst:     10,
		MaxRetries:         3,
		RetryBaseMs:        100,
		MaxResponseBytes:   10 * 1024 * 1024,
		UserAgent:          "test-agent",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunScrape(ctx, cfg, []string{"--url", "https://example.com"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "\"kind\": \"scrape\"") {
		t.Errorf("expected output to contain job data, got %q", output)
	}
}

func TestRunScrape_ValidExtractConfigJSON(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	cfg := config.Config{
		DataDir:            tmpDir,
		UsePlaywright:      false,
		RequestTimeoutSecs: 30,
		MaxConcurrency:     1,
		RateLimitQPS:       10,
		RateLimitBurst:     10,
		MaxRetries:         3,
		RetryBaseMs:        100,
		MaxResponseBytes:   10 * 1024 * 1024,
		UserAgent:          "test-agent",
	}

	tempFile, err := os.CreateTemp(t.TempDir(), "extract-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	validTemplate := `{"name":"test","selectors":[{"name":"title","selector":"h1"}]}`
	if _, err := tempFile.WriteString(validTemplate); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunScrape(ctx, cfg, []string{"--url", "https://example.com", "--extract-config", tempFile.Name()})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "\"kind\": \"scrape\"") {
		t.Errorf("expected output to contain job data, got %q", output)
	}
}
