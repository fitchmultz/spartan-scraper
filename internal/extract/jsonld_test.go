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

func TestMatchJSONLDByType(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "Article Headline"},
		{"@type": "Product", "name": "Product Name"},
		{"@type": "Organization", "name": "Org Name"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "Article",
		Path: "headline",
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0] != "Article Headline" {
		t.Errorf("expected match 'Article Headline', got %q", matches[0])
	}
}

func TestMatchJSONLDByTypeArray(t *testing.T) {
	documents := []map[string]any{
		{"@type": []any{"Article", "NewsArticle"}, "headline": "Multiple Types"},
		{"@type": "Product", "name": "Product Name"},
	}

	tests := []struct {
		name     string
		ruleType string
		expected string
	}{
		{
			name:     "match first type in array",
			ruleType: "Article",
			expected: "Multiple Types",
		},
		{
			name:     "match second type in array",
			ruleType: "NewsArticle",
			expected: "Multiple Types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := JSONLDRule{
				Name: "headline",
				Type: tt.ruleType,
				Path: "headline",
			}

			matches := MatchJSONLD(documents, rule)

			if len(matches) != 1 {
				t.Fatalf("expected 1 match, got %d", len(matches))
			}

			if matches[0] != tt.expected {
				t.Errorf("expected match %q, got %q", tt.expected, matches[0])
			}
		})
	}
}

func TestMatchJSONLDByTypeCaseInsensitive(t *testing.T) {
	documents := []map[string]any{
		{"@type": "article", "headline": "lowercase type"},
		{"@type": "ARTICLE", "headline": "uppercase type"},
		{"@type": "ArTiClE", "headline": "mixed case type"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "ARTICLE",
		Path: "headline",
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches (case-insensitive), got %d", len(matches))
	}

	expectedMatches := []string{"lowercase type", "uppercase type", "mixed case type"}
	for i, match := range matches {
		if match != expectedMatches[i] {
			t.Errorf("match %d: expected %q, got %q", i, expectedMatches[i], match)
		}
	}
}

func TestMatchJSONLDNoType(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "Article Headline"},
		{"headline": "No Type Object"},
		{"@type": "Product", "name": "Product Name"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "",
		Path: "headline",
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches (no type filter), got %d", len(matches))
	}

	if matches[0] != "Article Headline" {
		t.Errorf("expected first match 'Article Headline', got %q", matches[0])
	}

	if matches[1] != "No Type Object" {
		t.Errorf("expected second match 'No Type Object', got %q", matches[1])
	}
}

func TestMatchJSONLDTypeNotMatching(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "Article Headline"},
		{"@type": "Product", "name": "Product Name"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "Organization",
		Path: "headline",
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 0 {
		t.Errorf("expected 0 matches (no matching type), got %d", len(matches))
	}
}

func TestMatchJSONLDPathTraversal(t *testing.T) {
	tests := []struct {
		name      string
		documents []map[string]any
		rule      JSONLDRule
		expected  []string
	}{
		{
			name: "simple dot path",
			documents: []map[string]any{
				{"@type": "Article", "headline": "Test Headline"},
			},
			rule: JSONLDRule{
				Name: "headline",
				Type: "Article",
				Path: "headline",
			},
			expected: []string{"Test Headline"},
		},
		{
			name: "nested path author.name",
			documents: []map[string]any{
				{"@type": "Article", "author": map[string]any{"@type": "Person", "name": "John Doe"}},
			},
			rule: JSONLDRule{
				Name: "author",
				Type: "Article",
				Path: "author.name",
			},
			expected: []string{"John Doe"},
		},
		{
			name: "nested path offers.price",
			documents: []map[string]any{
				{"@type": "Product", "offers": map[string]any{"@type": "Offer", "price": "99.99"}},
			},
			rule: JSONLDRule{
				Name: "price",
				Type: "Product",
				Path: "offers.price",
			},
			expected: []string{"99.99"},
		},
		{
			name: "deep nested path",
			documents: []map[string]any{
				{"@type": "Article", "author": map[string]any{"address": map[string]any{"city": "New York"}}},
			},
			rule: JSONLDRule{
				Name: "city",
				Type: "Article",
				Path: "author.address.city",
			},
			expected: []string{"New York"},
		},
		{
			name: "path that doesn't exist",
			documents: []map[string]any{
				{"@type": "Article", "headline": "Headline"},
			},
			rule: JSONLDRule{
				Name: "author",
				Type: "Article",
				Path: "author.name",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := MatchJSONLD(tt.documents, tt.rule)

			if len(matches) != len(tt.expected) {
				t.Fatalf("expected %d matches, got %d", len(tt.expected), len(matches))
			}

			for i, match := range matches {
				if match != tt.expected[i] {
					t.Errorf("match %d: expected %q, got %q", i, tt.expected[i], match)
				}
			}
		})
	}
}

func TestMatchJSONLDAllFlag(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "First Article"},
		{"@type": "Article", "headline": "Second Article"},
		{"@type": "Article", "headline": "Third Article"},
		{"@type": "Product", "name": "Product"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "Article",
		Path: "headline",
		All:  true,
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches (All=true), got %d", len(matches))
	}

	expectedMatches := []string{"First Article", "Second Article", "Third Article"}
	for i, match := range matches {
		if match != expectedMatches[i] {
			t.Errorf("match %d: expected %q, got %q", i, expectedMatches[i], match)
		}
	}
}

func TestMatchJSONLDWithoutAllFlag(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "First Article"},
		{"@type": "Article", "headline": "Second Article"},
		{"@type": "Article", "headline": "Third Article"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "Article",
		Path: "headline",
		All:  false,
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}
}

func TestGetPathSimple(t *testing.T) {
	tests := []struct {
		name     string
		obj      map[string]any
		path     string
		expected any
	}{
		{
			name:     "single part path",
			obj:      map[string]any{"name": "John Doe"},
			path:     "name",
			expected: "John Doe",
		},
		{
			name:     "two part path",
			obj:      map[string]any{"author": map[string]any{"name": "Jane Doe"}},
			path:     "author.name",
			expected: "Jane Doe",
		},
		{
			name:     "three part path",
			obj:      map[string]any{"data": map[string]any{"nested": map[string]any{"value": "deep"}}},
			path:     "data.nested.value",
			expected: "deep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPath(tt.obj, tt.path)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetPathThroughArray(t *testing.T) {
	tests := []struct {
		name     string
		obj      map[string]any
		path     string
		expected any
	}{
		{
			name: "path traversing array",
			obj: map[string]any{
				"authors": []any{
					map[string]any{"name": "Author 1"},
					map[string]any{"name": "Author 2"},
					map[string]any{"name": "Author 3"},
				},
			},
			path:     "authors.name",
			expected: []any{"Author 1", "Author 2", "Author 3"},
		},
		{
			name: "array part not found",
			obj: map[string]any{
				"authors": []any{
					map[string]any{"id": "1"},
					map[string]any{"id": "2"},
				},
			},
			path:     "authors.name",
			expected: nil,
		},
		{
			name: "nested array path",
			obj: map[string]any{
				"articles": []any{
					map[string]any{"author": map[string]any{"name": "John"}},
					map[string]any{"author": map[string]any{"name": "Jane"}},
				},
			},
			path:     "articles.author.name",
			expected: []any{"John", "Jane"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPath(tt.obj, tt.path)

			if result == nil {
				if tt.expected != nil {
					t.Errorf("expected %v, got nil", tt.expected)
				}
				return
			}

			resultSlice, ok := result.([]any)
			if !ok {
				t.Errorf("expected result to be slice, got %T", result)
				return
			}

			expectedSlice, ok := tt.expected.([]any)
			if !ok {
				t.Errorf("expected expected to be slice, got %T", tt.expected)
				return
			}

			if len(resultSlice) != len(expectedSlice) {
				t.Fatalf("expected %d values, got %d", len(expectedSlice), len(resultSlice))
			}

			for i, val := range resultSlice {
				if val != expectedSlice[i] {
					t.Errorf("value %d: expected %v, got %v", i, expectedSlice[i], val)
				}
			}
		})
	}
}

func TestGetPathNotFound(t *testing.T) {
	tests := []struct {
		name string
		obj  map[string]any
		path string
	}{
		{
			name: "key not found",
			obj:  map[string]any{"name": "John"},
			path: "age",
		},
		{
			name: "intermediate key not found",
			obj:  map[string]any{"author": map[string]any{"name": "John"}},
			path: "author.age",
		},
		{
			name: "intermediate nil value",
			obj:  map[string]any{"author": nil},
			path: "author.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPath(tt.obj, tt.path)

			if result != nil {
				t.Errorf("expected nil, got %v", result)
			}
		})
	}
}

func TestGetPathEmptyPath(t *testing.T) {
	obj := map[string]any{
		"name": "John Doe",
		"age":  30,
	}

	result := getPath(obj, "")

	// Empty path returns nil because strings.Split("", ".") = [""]
	// and there's no key "" in the map
	if result != nil {
		t.Errorf("expected nil for empty path, got %v", result)
	}
}

func TestExtractStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "string value",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "float64 value",
			input:    123.45,
			expected: []string{"123.45"},
		},
		{
			name:     "int value",
			input:    42,
			expected: []string{"42"},
		},
		{
			name:     "bool value true",
			input:    true,
			expected: []string{"true"},
		},
		{
			name:     "bool value false",
			input:    false,
			expected: []string{"false"},
		},
		{
			name:     "array of strings",
			input:    []any{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "nested array of strings",
			input:    []any{[]any{"a", "b"}, []any{"c", "d"}},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "mixed nested array",
			input:    []any{"text", 123, true, []any{"nested"}},
			expected: []string{"text", "123", "true", "nested"},
		},
		{
			name:     "empty array",
			input:    []any{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []string
			extractStrings(tt.input, &result)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d strings, got %d", len(tt.expected), len(result))
			}

			for i, str := range result {
				if str != tt.expected[i] {
					t.Errorf("string %d: expected %q, got %q", i, tt.expected[i], str)
				}
			}
		})
	}
}

func TestExtractJSONLDRealWorld(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		expectedType string
		expectedKey  string
		expectedVal  string
	}{
		{
			name: "Article schema",
			html: `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json">
	{
		"@context": "https://schema.org",
		"@type": "NewsArticle",
		"headline": "Breaking News: Major Announcement",
		"datePublished": "2024-01-15T10:00:00Z",
		"author": {
			"@type": "Person",
			"name": "Jane Smith"
		}
	}
	</script>
</head>
<body></body>
</html>`,
			expectedType: "NewsArticle",
			expectedKey:  "headline",
			expectedVal:  "Breaking News: Major Announcement",
		},
		{
			name: "Product schema",
			html: `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json">
	{
		"@context": "https://schema.org",
		"@type": "Product",
		"name": "Premium Widget",
		"description": "A high-quality widget for all your needs",
		"offers": {
			"@type": "Offer",
			"price": "29.99",
			"priceCurrency": "USD",
			"availability": "https://schema.org/InStock"
		}
	}
	</script>
</head>
<body></body>
</html>`,
			expectedType: "Product",
			expectedKey:  "name",
			expectedVal:  "Premium Widget",
		},
		{
			name: "Organization schema",
			html: `<!DOCTYPE html>
<html>
<head>
	<script type="application/ld+json">
	{
		"@context": "https://schema.org",
		"@type": "Organization",
		"name": "Acme Corporation",
		"url": "https://www.example.com",
		"logo": "https://www.example.com/logo.png",
		"contactPoint": {
			"@type": "ContactPoint",
			"telephone": "+1-555-123-4567",
			"contactType": "customer service"
		}
	}
	</script>
</head>
<body></body>
</html>`,
			expectedType: "Organization",
			expectedKey:  "name",
			expectedVal:  "Acme Corporation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExtractJSONLD(tt.html)
			if err != nil {
				t.Fatalf("ExtractJSONLD failed: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("expected at least 1 JSON-LD object")
			}

			if results[0]["@type"] != tt.expectedType {
				t.Errorf("expected @type %q, got %v", tt.expectedType, results[0]["@type"])
			}

			if results[0][tt.expectedKey] != tt.expectedVal {
				t.Errorf("expected %s %q, got %v", tt.expectedKey, tt.expectedVal, results[0][tt.expectedKey])
			}
		})
	}
}
