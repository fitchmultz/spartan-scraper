// Package extract provides tests for real-world JSON-LD schema parsing.
// Tests cover Article, Product, and Organization schema types with realistic markup.
// Does NOT test malformed JSON or edge cases.
package extract

import (
	"testing"
)

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
