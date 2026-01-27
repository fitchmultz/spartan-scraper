package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTemplateRegistryNoFile(t *testing.T) {
	dataDir := t.TempDir()

	registry, err := LoadTemplateRegistry(dataDir)
	if err != nil {
		t.Fatalf("LoadTemplateRegistry failed: %v", err)
	}

	if registry == nil {
		t.Fatal("expected registry to be non-nil")
	}

	if len(registry.Templates) < 3 {
		t.Errorf("expected at least 3 built-in templates, got %d", len(registry.Templates))
	}

	_, hasDefault := registry.Templates["default"]
	if !hasDefault {
		t.Error("expected 'default' template to be present")
	}

	_, hasArticle := registry.Templates["article"]
	if !hasArticle {
		t.Error("expected 'article' template to be present")
	}

	_, hasProduct := registry.Templates["product"]
	if !hasProduct {
		t.Error("expected 'product' template to be present")
	}
}

func TestLoadTemplateRegistryWithFile(t *testing.T) {
	dataDir := t.TempDir()

	customTemplate := Template{
		Name: "custom",
		Selectors: []SelectorRule{
			{Name: "title", Selector: "title", Attr: "text"},
		},
	}

	templateFile := TemplateFile{
		Templates: []Template{customTemplate},
	}

	data, err := json.Marshal(templateFile)
	if err != nil {
		t.Fatalf("failed to marshal template file: %v", err)
	}

	filePath := filepath.Join(dataDir, "extract_templates.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	registry, err := LoadTemplateRegistry(dataDir)
	if err != nil {
		t.Fatalf("LoadTemplateRegistry failed: %v", err)
	}

	_, hasCustom := registry.Templates["custom"]
	if !hasCustom {
		t.Error("expected 'custom' template to be loaded from file")
	}

	_, hasDefault := registry.Templates["default"]
	if !hasDefault {
		t.Error("expected built-in 'default' template to still be present")
	}

	_, hasArticle := registry.Templates["article"]
	if !hasArticle {
		t.Error("expected built-in 'article' template to still be present")
	}
}

func TestLoadTemplateRegistryTemplateMerging(t *testing.T) {
	dataDir := t.TempDir()

	overriddenTemplate := Template{
		Name: "article",
		Selectors: []SelectorRule{
			{Name: "custom_title", Selector: "h1", Attr: "text"},
		},
	}

	templateFile := TemplateFile{
		Templates: []Template{overriddenTemplate},
	}

	data, err := json.Marshal(templateFile)
	if err != nil {
		t.Fatalf("failed to marshal template file: %v", err)
	}

	filePath := filepath.Join(dataDir, "extract_templates.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	registry, err := LoadTemplateRegistry(dataDir)
	if err != nil {
		t.Fatalf("LoadTemplateRegistry failed: %v", err)
	}

	article := registry.Templates["article"]
	if article.Name != "article" {
		t.Errorf("expected template name 'article', got %q", article.Name)
	}

	hasCustomTitle := false
	for _, sel := range article.Selectors {
		if sel.Name == "custom_title" {
			hasCustomTitle = true
			break
		}
	}

	if !hasCustomTitle {
		t.Error("expected custom selector from file template")
	}

	_, hasDefault := registry.Templates["default"]
	if !hasDefault {
		t.Error("expected 'default' template to remain unchanged")
	}

	_, hasProduct := registry.Templates["product"]
	if !hasProduct {
		t.Error("expected 'product' template to remain unchanged")
	}
}

func TestLoadTemplateRegistryInvalidFile(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
	}{
		{
			name:        "malformed JSON",
			fileContent: `{invalid json content}`,
		},
		{
			name:        "wrong schema - templates is not array",
			fileContent: `{"templates": {"name": "template1"}}`,
		},
		{
			name:        "wrong schema - templates has invalid type",
			fileContent: `{"templates": "invalid"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := t.TempDir()

			filePath := filepath.Join(dataDir, "extract_templates.json")
			if err := os.WriteFile(filePath, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("failed to write template file: %v", err)
			}

			_, err := LoadTemplateRegistry(dataDir)
			if err == nil {
				t.Error("expected error for invalid template file, got nil")
			}
		})
	}
}

func TestLoadTemplateRegistryEmptyFile(t *testing.T) {
	dataDir := t.TempDir()

	templateFile := TemplateFile{
		Templates: []Template{},
	}

	data, err := json.Marshal(templateFile)
	if err != nil {
		t.Fatalf("failed to marshal template file: %v", err)
	}

	filePath := filepath.Join(dataDir, "extract_templates.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	registry, err := LoadTemplateRegistry(dataDir)
	if err != nil {
		t.Fatalf("LoadTemplateRegistry failed: %v", err)
	}

	if len(registry.Templates) < 3 {
		t.Errorf("expected at least 3 built-in templates (file has none), got %d", len(registry.Templates))
	}

	_, hasDefault := registry.Templates["default"]
	if !hasDefault {
		t.Error("expected built-in 'default' template to be present")
	}
}

func TestResolveTemplateInline(t *testing.T) {
	inline := &Template{
		Name: "inline_template",
		Selectors: []SelectorRule{
			{Name: "title", Selector: "title", Attr: "text"},
		},
	}

	tests := []struct {
		name     string
		opts     ExtractOptions
		expected string
	}{
		{
			name: "inline template takes precedence",
			opts: ExtractOptions{
				Template: "article",
				Inline:   inline,
			},
			expected: "inline_template",
		},
		{
			name: "inline with empty template name",
			opts: ExtractOptions{
				Template: "",
				Inline:   inline,
			},
			expected: "inline_template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := ResolveTemplate(tt.opts, nil)
			if err != nil {
				t.Fatalf("ResolveTemplate failed: %v", err)
			}

			if tmpl.Name != tt.expected {
				t.Errorf("expected template name %q, got %q", tt.expected, tmpl.Name)
			}
		})
	}
}

func TestResolveTemplateByName(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "resolve article template",
			template: "article",
			expected: "article",
		},
		{
			name:     "resolve product template",
			template: "product",
			expected: "product",
		},
		{
			name:     "resolve default template",
			template: "default",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ExtractOptions{
				Template: tt.template,
			}

			tmpl, err := ResolveTemplate(opts, nil)
			if err != nil {
				t.Fatalf("ResolveTemplate failed: %v", err)
			}

			if tmpl.Name != tt.expected {
				t.Errorf("expected template name %q, got %q", tt.expected, tmpl.Name)
			}
		})
	}
}

func TestResolveTemplateDefaultFallback(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "empty template name falls back to default",
			template: "",
			expected: "default",
		},
		{
			name:     "non-existent template name falls back to default",
			template: "non_existent_template",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ExtractOptions{
				Template: tt.template,
			}

			tmpl, err := ResolveTemplate(opts, nil)
			if err != nil {
				t.Fatalf("ResolveTemplate failed: %v", err)
			}

			if tmpl.Name != tt.expected {
				t.Errorf("expected template name %q, got %q", tt.expected, tmpl.Name)
			}
		})
	}
}

func TestResolveTemplateNilRegistry(t *testing.T) {
	opts := ExtractOptions{
		Template: "article",
	}

	tmpl, err := ResolveTemplate(opts, nil)
	if err != nil {
		t.Fatalf("ResolveTemplate failed: %v", err)
	}

	if tmpl.Name != "article" {
		t.Errorf("expected template name 'article', got %q", tmpl.Name)
	}

	opts.Template = "non_existent"
	tmpl, err = ResolveTemplate(opts, nil)
	if err != nil {
		t.Fatalf("ResolveTemplate failed: %v", err)
	}

	if tmpl.Name != "default" {
		t.Errorf("expected template name 'default' for non-existent, got %q", tmpl.Name)
	}
}

func TestResolveTemplateRegistryMissing(t *testing.T) {
	dataDir := t.TempDir()

	customTemplate := Template{
		Name: "custom1",
		Selectors: []SelectorRule{
			{Name: "title", Selector: "title", Attr: "text"},
		},
	}

	templateFile := TemplateFile{
		Templates: []Template{customTemplate},
	}

	data, err := json.Marshal(templateFile)
	if err != nil {
		t.Fatalf("failed to marshal template file: %v", err)
	}

	filePath := filepath.Join(dataDir, "extract_templates.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	registry, err := LoadTemplateRegistry(dataDir)
	if err != nil {
		t.Fatalf("LoadTemplateRegistry failed: %v", err)
	}

	opts := ExtractOptions{
		Template: "non_existent_in_registry",
	}

	tmpl, err := ResolveTemplate(opts, registry)
	if err != nil {
		t.Fatalf("ResolveTemplate failed: %v", err)
	}

	if tmpl.Name != "default" {
		t.Errorf("expected template name 'default' when registry missing template, got %q", tmpl.Name)
	}
}

func TestBuiltInTemplates(t *testing.T) {
	t.Run("default template structure", func(t *testing.T) {
		tmpl, ok := builtInTemplates["default"]
		if !ok {
			t.Fatal("expected 'default' template in built-ins")
		}

		if tmpl.Name != "default" {
			t.Errorf("expected name 'default', got %q", tmpl.Name)
		}

		if len(tmpl.Selectors) == 0 {
			t.Error("expected selectors in default template")
		}

		hasTitleSelector := false
		for _, sel := range tmpl.Selectors {
			if sel.Name == "title" {
				hasTitleSelector = true
				if sel.Selector != "title" {
					t.Errorf("expected title selector 'title', got %q", sel.Selector)
				}
				if sel.Attr != "text" {
					t.Errorf("expected title attr 'text', got %q", sel.Attr)
				}
				if !sel.Trim {
					t.Error("expected title Trim to be true")
				}
			}
		}

		if !hasTitleSelector {
			t.Error("expected title selector in default template")
		}

		if tmpl.Normalize.TitleField != "title" {
			t.Errorf("expected TitleField 'title', got %q", tmpl.Normalize.TitleField)
		}

		if tmpl.Normalize.DescriptionField != "description" {
			t.Errorf("expected DescriptionField 'description', got %q", tmpl.Normalize.DescriptionField)
		}
	})

	t.Run("article template structure", func(t *testing.T) {
		tmpl, ok := builtInTemplates["article"]
		if !ok {
			t.Fatal("expected 'article' template in built-ins")
		}

		if tmpl.Name != "article" {
			t.Errorf("expected name 'article', got %q", tmpl.Name)
		}

		if len(tmpl.Selectors) == 0 {
			t.Error("expected selectors in article template")
		}

		if len(tmpl.JSONLD) == 0 {
			t.Error("expected JSON-LD rules in article template")
		}

		hasHeadlineRule := false
		for _, rule := range tmpl.JSONLD {
			if rule.Name == "headline" {
				hasHeadlineRule = true
				if rule.Type != "Article" {
					t.Errorf("expected headline rule Type 'Article', got %q", rule.Type)
				}
				if rule.Path != "headline" {
					t.Errorf("expected headline rule Path 'headline', got %q", rule.Path)
				}
			}
		}

		if !hasHeadlineRule {
			t.Error("expected headline JSON-LD rule in article template")
		}

		if tmpl.Normalize.TitleField != "title" {
			t.Errorf("expected TitleField 'title', got %q", tmpl.Normalize.TitleField)
		}

		if tmpl.Normalize.TextField != "content" {
			t.Errorf("expected TextField 'content', got %q", tmpl.Normalize.TextField)
		}

		if len(tmpl.Normalize.MetaFields) == 0 {
			t.Error("expected MetaFields in article template")
		}

		if tmpl.Normalize.MetaFields["author"] != "author" {
			t.Error("expected author meta field mapping")
		}

		if tmpl.Normalize.MetaFields["datePublished"] != "date" {
			t.Error("expected datePublished meta field mapping")
		}
	})

	t.Run("product template structure", func(t *testing.T) {
		tmpl, ok := builtInTemplates["product"]
		if !ok {
			t.Fatal("expected 'product' template in built-ins")
		}

		if tmpl.Name != "product" {
			t.Errorf("expected name 'product', got %q", tmpl.Name)
		}

		if len(tmpl.Selectors) == 0 {
			t.Error("expected selectors in product template")
		}

		if len(tmpl.JSONLD) == 0 {
			t.Error("expected JSON-LD rules in product template")
		}

		hasNameRule := false
		for _, rule := range tmpl.JSONLD {
			if rule.Name == "name" {
				hasNameRule = true
				if rule.Type != "Product" {
					t.Errorf("expected name rule Type 'Product', got %q", rule.Type)
				}
				if rule.Path != "name" {
					t.Errorf("expected name rule Path 'name', got %q", rule.Path)
				}
			}
		}

		if !hasNameRule {
			t.Error("expected name JSON-LD rule in product template")
		}

		if tmpl.Normalize.TitleField != "name" {
			t.Errorf("expected TitleField 'name', got %q", tmpl.Normalize.TitleField)
		}

		if len(tmpl.Normalize.MetaFields) == 0 {
			t.Error("expected MetaFields in product template")
		}

		if tmpl.Normalize.MetaFields["price"] != "price" {
			t.Error("expected price meta field mapping")
		}

		if tmpl.Normalize.MetaFields["currency"] != "currency" {
			t.Error("expected currency meta field mapping")
		}
	})
}
