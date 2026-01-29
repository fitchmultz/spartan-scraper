package extract

import (
	"testing"
)

func TestExtractJSONLDSingleObject(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json">
	{
		"@context": "https://schema.org",
		"@type": "Article",
		"headline": "Test Article",
		"author": {"@type": "Person", "name": "John Doe"}
	}
	</script>
</head>
<body></body>
</html>`

	results, err := ExtractJSONLD(html)
	if err != nil {
		t.Fatalf("ExtractJSONLD failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 JSON-LD object, got %d", len(results))
	}

	if results[0]["@type"] != "Article" {
		t.Errorf("expected @type 'Article', got %v", results[0]["@type"])
	}

	if results[0]["headline"] != "Test Article" {
		t.Errorf("expected headline 'Test Article', got %v", results[0]["headline"])
	}
}

func TestExtractJSONLDArray(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json">
	[
		{
			"@context": "https://schema.org",
			"@type": "Product",
			"name": "Product 1"
		},
		{
			"@context": "https://schema.org",
			"@type": "Product",
			"name": "Product 2"
		}
	]
	</script>
</head>
<body></body>
</html>`

	results, err := ExtractJSONLD(html)
	if err != nil {
		t.Fatalf("ExtractJSONLD failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 JSON-LD objects, got %d", len(results))
	}

	if results[0]["name"] != "Product 1" {
		t.Errorf("expected name 'Product 1', got %v", results[0]["name"])
	}

	if results[1]["name"] != "Product 2" {
		t.Errorf("expected name 'Product 2', got %v", results[1]["name"])
	}
}

func TestExtractJSONLDNoScript(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
</head>
<body>
	<h1>Test</h1>
</body>
</html>`

	results, err := ExtractJSONLD(html)
	if err != nil {
		t.Fatalf("ExtractJSONLD failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 JSON-LD objects, got %d", len(results))
	}
}

func TestExtractJSONLDInvalidScript(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		expectLen int
	}{
		{
			name:      "invalid JSON",
			html:      `<!DOCTYPE html><html><head><script type="application/ld+json">{invalid json}</script></head><body></body></html>`,
			expectLen: 0,
		},
		{
			name:      "non-JSON content",
			html:      `<!DOCTYPE html><html><head><script type="application/ld+json">This is not JSON</script></head><body></body></html>`,
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExtractJSONLD(tt.html)
			if err != nil {
				t.Fatalf("ExtractJSONLD failed: %v", err)
			}

			if len(results) != tt.expectLen {
				t.Errorf("expected %d JSON-LD objects, got %d", tt.expectLen, len(results))
			}
		})
	}
}

func TestExtractJSONLDGraphContainer(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json">
	{
		"@context": "https://schema.org",
		"@graph": [
			{
				"@type": "Article",
				"headline": "Article 1"
			},
			{
				"@type": "Article",
				"headline": "Article 2"
			}
		]
	}
	</script>
</head>
<body></body>
</html>`

	results, err := ExtractJSONLD(html)
	if err != nil {
		t.Fatalf("ExtractJSONLD failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 JSON-LD objects from @graph, got %d", len(results))
	}

	if results[0]["headline"] != "Article 1" {
		t.Errorf("expected headline 'Article 1', got %v", results[0]["headline"])
	}

	if results[1]["headline"] != "Article 2" {
		t.Errorf("expected headline 'Article 2', got %v", results[1]["headline"])
	}
}

func TestExtractJSONLDMultipleScripts(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json">
	{
		"@context": "https://schema.org",
		"@type": "Article",
		"headline": "Article from first script"
	}
	</script>
	<script type="application/ld+json">
	{
		"@context": "https://schema.org",
		"@type": "Product",
		"name": "Product from second script"
	}
	</script>
</head>
<body></body>
</html>`

	results, err := ExtractJSONLD(html)
	if err != nil {
		t.Fatalf("ExtractJSONLD failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 JSON-LD objects, got %d", len(results))
	}

	articleFound := false
	productFound := false

	for _, result := range results {
		if result["@type"] == "Article" && result["headline"] == "Article from first script" {
			articleFound = true
		}
		if result["@type"] == "Product" && result["name"] == "Product from second script" {
			productFound = true
		}
	}

	if !articleFound {
		t.Error("Article from first script not found")
	}

	if !productFound {
		t.Error("Product from second script not found")
	}
}

func TestExtractJSONLDWhitespace(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json">
		
		
	</script>
</head>
<body></body>
</html>`

	results, err := ExtractJSONLD(html)
	if err != nil {
		t.Fatalf("ExtractJSONLD failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 JSON-LD objects (whitespace should be skipped), got %d", len(results))
	}
}

func TestExtractJSONLDEmptyScript(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json"></script>
</head>
<body></body>
</html>`

	results, err := ExtractJSONLD(html)
	if err != nil {
		t.Fatalf("ExtractJSONLD failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 JSON-LD objects (empty script should be skipped), got %d", len(results))
	}
}
