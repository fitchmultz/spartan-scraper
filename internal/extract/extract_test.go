package extract

import (
	"testing"
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
