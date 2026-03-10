// Package extract provides tests for the core extraction engine.
// Tests cover template application, built-in template listing, and execution with custom registries.
// Does NOT test JSON-LD parsing, normalization, or schema validation.
package extract

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyTemplate(t *testing.T) {
	html := `
		<html>
			<head>
				<title>Test Page</title>
				<meta name="description" content="A test page description">
				<script type="application/ld+json">
				{
					"@context": "https://schema.org",
					"@type": "Product",
					"name": "Test Product",
					"offers": {
						"@type": "Offer",
						"price": "99.99"
					}
				}
				</script>
			</head>
			<body>
				<h1>Main Header</h1>
				<div class="content">Some content here.</div>
				<a href="/link1">Link 1</a>
				<span id="secret">12345</span>
			</body>
		</html>
	`

	tmpl := Template{
		Name: "test",
		Selectors: []SelectorRule{
			{Name: "header", Selector: "h1", Attr: "text", Trim: true},
			{Name: "content", Selector: ".content", Attr: "text", Trim: true},
		},
		JSONLD: []JSONLDRule{
			{Name: "productName", Type: "Product", Path: "name"},
			{Name: "productPrice", Type: "Product", Path: "offers.price"},
		},
		Regex: []RegexRule{
			{Name: "secretCode", Pattern: `\d+`, Source: RegexSourceText},
		},
	}

	extracted, err := ApplyTemplate("http://example.com", html, tmpl)
	if err != nil {
		t.Fatalf("ApplyTemplate failed: %v", err)
	}

	checkField := func(name, expected string) {
		t.Helper()
		f, ok := extracted.Fields[name]
		if !ok || len(f.Values) == 0 {
			t.Errorf("expected %s %q, got empty/missing", name, expected)
			return
		}
		if val := f.Values[0]; val != expected {
			t.Errorf("expected %s %q, got %q", name, expected, val)
		}
	}

	checkField("header", "Main Header")
	checkField("content", "Some content here.")
	checkField("productName", "Test Product")
	checkField("productPrice", "99.99")
	// Base extraction check
	if extracted.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", extracted.Title)
	}
	if len(extracted.Links) != 1 || extracted.Links[0] != "/link1" {
		t.Errorf("expected 1 link '/link1', got %v", extracted.Links)
	}
}

func TestListTemplateNames(t *testing.T) {
	dataDir := t.TempDir()

	// Test with no custom templates (should return built-ins)
	names, err := ListTemplateNames(dataDir)
	if err != nil {
		t.Fatalf("ListTemplateNames failed: %v", err)
	}

	// Should have at least the 3 built-in templates
	if len(names) < 3 {
		t.Errorf("expected at least 3 templates, got %d", len(names))
	}

	// Check for specific built-in templates
	hasDefault := false
	hasArticle := false
	hasProduct := false
	for _, name := range names {
		if name == "default" {
			hasDefault = true
		}
		if name == "article" {
			hasArticle = true
		}
		if name == "product" {
			hasProduct = true
		}
	}

	if !hasDefault {
		t.Error("expected 'default' template in list")
	}
	if !hasArticle {
		t.Error("expected 'article' template in list")
	}
	if !hasProduct {
		t.Error("expected 'product' template in list")
	}

	// Verify sorting (should be alphabetical)
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("templates not sorted: %s > %s", names[i-1], names[i])
		}
	}
}

func TestExecuteWithRegistry(t *testing.T) {
	registry := &TemplateRegistry{
		Templates: map[string]Template{
			"custom": {
				Name: "custom",
				Selectors: []SelectorRule{
					{Name: "custom_field", Selector: "body", Attr: "text", Trim: true},
				},
			},
		},
	}

	html := "<html><body>Hello World</body></html>"
	input := ExecuteInput{
		URL:  "http://example.com",
		HTML: html,
		Options: ExtractOptions{
			Template: "custom",
		},
		Registry: registry,
	}

	output, err := Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.Extracted.Template != "custom" {
		t.Errorf("expected template 'custom', got %q", output.Extracted.Template)
	}

	f, ok := output.Extracted.Fields["custom_field"]
	if !ok || len(f.Values) == 0 || f.Values[0] != "Hello World" {
		t.Errorf("expected field 'custom_field' to be 'Hello World', got %v", f.Values)
	}
}

func TestExecuteReturnsRegistryLoadError(t *testing.T) {
	dataDir := t.TempDir()
	path := filepath.Join(dataDir, "extract_templates.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("failed to seed invalid template registry: %v", err)
	}

	_, err := Execute(ExecuteInput{
		URL:     "https://example.com",
		HTML:    "<html><title>Example</title></html>",
		DataDir: dataDir,
		Options: ExtractOptions{Template: "default"},
	})
	if err == nil {
		t.Fatal("expected Execute to return template registry load error")
	}
}

// mockLLMProvider is a test stub that captures context for verification
type mockLLMProvider struct {
	capturedContext context.Context
	extractCalled   bool
	result          AIExtractResult
	err             error
	delay           time.Duration
}

func (m *mockLLMProvider) Extract(ctx context.Context, req AIExtractRequest) (AIExtractResult, error) {
	m.capturedContext = ctx
	m.extractCalled = true

	// Simulate work that respects context cancellation
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
			return m.result, m.err
		case <-ctx.Done():
			return AIExtractResult{}, ctx.Err()
		}
	}

	return m.result, m.err
}

func (m *mockLLMProvider) HealthCheck(ctx context.Context) error {
	return nil
}

// TestExecute_ContextPropagation verifies that context is propagated to AIExtractor
func TestExecute_ContextPropagation(t *testing.T) {
	mock := &mockLLMProvider{
		result: AIExtractResult{
			Fields: map[string]FieldValue{
				"test_field": {Values: []string{"test_value"}, Source: FieldSourceDerived},
			},
			Confidence: 0.95,
		},
	}

	// Create AIExtractor with mock provider
	extractor := &AIExtractor{
		provider: mock,
		cache:    &mockAICache{},
	}

	// Create a context with a value to verify propagation
	type contextKey string
	ctxKey := contextKey("test_key")
	ctx := context.WithValue(context.Background(), ctxKey, "test_value")

	html := "<html><body>Test content</body></html>"
	input := ExecuteInput{
		URL:  "http://example.com",
		HTML: html,
		Options: ExtractOptions{
			AI: &AIExtractOptions{
				Enabled: true,
				Mode:    AIModeNaturalLanguage,
				Fields:  []string{"test_field"},
			},
		},
		AIExtractor: extractor,
		Context:     ctx,
	}

	_, err := Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !mock.extractCalled {
		t.Fatal("AIExtractor.Extract was not called")
	}

	if mock.capturedContext == nil {
		t.Fatal("Context was not propagated to AIExtractor")
	}

	// Verify the context value was propagated
	if val := mock.capturedContext.Value(ctxKey); val != "test_value" {
		t.Errorf("Expected context value 'test_value', got %v", val)
	}
}

// TestExecute_ContextCancellation verifies that context cancellation is respected
func TestExecute_ContextCancellation(t *testing.T) {
	mock := &mockLLMProvider{
		result: AIExtractResult{
			Fields: map[string]FieldValue{
				"test_field": {Values: []string{"test_value"}, Source: FieldSourceDerived},
			},
		},
		delay: 100 * time.Millisecond, // Simulate slow extraction
	}

	extractor := &AIExtractor{
		provider: mock,
		cache:    &mockAICache{},
	}

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	html := "<html><body>Test content</body></html>"
	input := ExecuteInput{
		URL:  "http://example.com",
		HTML: html,
		Options: ExtractOptions{
			AI: &AIExtractOptions{
				Enabled: true,
				Mode:    AIModeNaturalLanguage,
				Fields:  []string{"test_field"},
			},
		},
		AIExtractor: extractor,
		Context:     ctx,
	}

	// Execute should complete but AI extraction may fail due to cancelled context
	// The error is logged but not returned (AI errors don't fail extraction)
	_, err := Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify the cancelled context was passed
	if !mock.extractCalled {
		t.Fatal("AIExtractor.Extract was not called")
	}

	if mock.capturedContext.Err() != context.Canceled {
		t.Errorf("Expected cancelled context, got: %v", mock.capturedContext.Err())
	}
}

// TestExecute_NilContextFallback verifies that nil context falls back to background
func TestExecute_NilContextFallback(t *testing.T) {
	mock := &mockLLMProvider{
		result: AIExtractResult{
			Fields: map[string]FieldValue{
				"test_field": {Values: []string{"test_value"}, Source: FieldSourceDerived},
			},
			Confidence: 0.95,
		},
	}

	extractor := &AIExtractor{
		provider: mock,
		cache:    &mockAICache{},
	}

	html := "<html><body>Test content</body></html>"
	input := ExecuteInput{
		URL:  "http://example.com",
		HTML: html,
		Options: ExtractOptions{
			AI: &AIExtractOptions{
				Enabled: true,
				Mode:    AIModeNaturalLanguage,
				Fields:  []string{"test_field"},
			},
		},
		AIExtractor: extractor,
		Context:     nil, // Explicitly nil
	}

	_, err := Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !mock.extractCalled {
		t.Fatal("AIExtractor.Extract was not called")
	}

	if mock.capturedContext == nil {
		t.Fatal("Context should not be nil (should fallback to background)")
	}

	// Verify it's a background context (no error)
	if mock.capturedContext.Err() != nil {
		t.Errorf("Expected background context (no error), got: %v", mock.capturedContext.Err())
	}
}

// mockAICache is a simple in-memory cache for testing
type mockAICache struct {
	data map[string]*AIExtractResult
}

func (m *mockAICache) Get(key string) (*AIExtractResult, bool) {
	if m.data == nil {
		return nil, false
	}
	val, ok := m.data[key]
	return val, ok
}

func (m *mockAICache) Set(key string, result *AIExtractResult) {
	if m.data == nil {
		m.data = make(map[string]*AIExtractResult)
	}
	m.data[key] = result
}
