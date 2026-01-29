package extract

import (
	"encoding/json"
	"testing"
)

func TestExtractOptionsStruct(t *testing.T) {
	t.Run("with template name", func(t *testing.T) {
		opts := ExtractOptions{
			Template: "article",
			Validate: false,
		}

		if opts.Template != "article" {
			t.Errorf("expected template 'article', got %q", opts.Template)
		}
	})

	t.Run("with inline template", func(t *testing.T) {
		inline := &Template{
			Name: "custom",
			Selectors: []SelectorRule{
				{Name: "title", Selector: "title", Attr: "text"},
			},
		}
		opts := ExtractOptions{
			Inline:   inline,
			Validate: true,
		}

		if opts.Inline == nil {
			t.Error("expected Inline to be set")
		}
		if opts.Inline.Name != "custom" {
			t.Errorf("expected name 'custom', got %q", opts.Inline.Name)
		}
		if !opts.Validate {
			t.Error("expected Validate to be true")
		}
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		opts := ExtractOptions{
			Template: "product",
			Validate: true,
		}

		data, err := json.Marshal(opts)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled ExtractOptions
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Template != "product" {
			t.Errorf("template not preserved after marshaling")
		}
		if !unmarshaled.Validate {
			t.Error("validate flag not preserved after marshaling")
		}
	})
}

func TestTemplateStruct(t *testing.T) {
	t.Run("with selectors", func(t *testing.T) {
		tmpl := Template{
			Name: "test",
			Selectors: []SelectorRule{
				{Name: "title", Selector: "title", Attr: "text"},
				{Name: "desc", Selector: "meta[name=description]", Attr: "content"},
			},
		}

		if tmpl.Name != "test" {
			t.Errorf("expected name 'test', got %q", tmpl.Name)
		}
		if len(tmpl.Selectors) != 2 {
			t.Errorf("expected 2 selectors, got %d", len(tmpl.Selectors))
		}
	})

	t.Run("with JSON-LD rules", func(t *testing.T) {
		tmpl := Template{
			Name: "test",
			JSONLD: []JSONLDRule{
				{Name: "headline", Type: "Article", Path: "headline"},
				{Name: "author", Type: "Article", Path: "author.name"},
			},
		}

		if len(tmpl.JSONLD) != 2 {
			t.Errorf("expected 2 JSON-LD rules, got %d", len(tmpl.JSONLD))
		}
	})

	t.Run("with regex rules", func(t *testing.T) {
		tmpl := Template{
			Name: "test",
			Regex: []RegexRule{
				{Name: "email", Pattern: `\S+@\S+\.\S+`},
				{Name: "phone", Pattern: `\d{3}-\d{3}-\d{4}`},
			},
		}

		if len(tmpl.Regex) != 2 {
			t.Errorf("expected 2 regex rules, got %d", len(tmpl.Regex))
		}
	})

	t.Run("with schema", func(t *testing.T) {
		schema := &Schema{
			Type:     SchemaObject,
			Required: []string{"title", "content"},
			Properties: map[string]*Schema{
				"title":   {Type: SchemaString},
				"content": {Type: SchemaString},
			},
		}
		tmpl := Template{
			Name:   "test",
			Schema: schema,
		}

		if tmpl.Schema == nil {
			t.Error("expected Schema to be set")
		}
		if tmpl.Schema.Type != SchemaObject {
			t.Errorf("expected type SchemaObject, got %v", tmpl.Schema.Type)
		}
	})

	t.Run("with normalize spec", func(t *testing.T) {
		tmpl := Template{
			Name: "test",
			Normalize: NormalizeSpec{
				TitleField:       "headline",
				DescriptionField: "summary",
				TextField:        "body",
			},
		}

		if tmpl.Normalize.TitleField != "headline" {
			t.Errorf("expected TitleField 'headline', got %q", tmpl.Normalize.TitleField)
		}
	})
}

func TestNormalizeSpecStruct(t *testing.T) {
	t.Run("title field mapping", func(t *testing.T) {
		spec := NormalizeSpec{
			TitleField: "headline",
		}

		if spec.TitleField != "headline" {
			t.Errorf("expected TitleField 'headline', got %q", spec.TitleField)
		}
	})

	t.Run("description field mapping", func(t *testing.T) {
		spec := NormalizeSpec{
			DescriptionField: "summary",
		}

		if spec.DescriptionField != "summary" {
			t.Errorf("expected DescriptionField 'summary', got %q", spec.DescriptionField)
		}
	})

	t.Run("text field mapping", func(t *testing.T) {
		spec := NormalizeSpec{
			TextField: "body",
		}

		if spec.TextField != "body" {
			t.Errorf("expected TextField 'body', got %q", spec.TextField)
		}
	})

	t.Run("single meta field mapping", func(t *testing.T) {
		spec := NormalizeSpec{
			MetaFields: map[string]string{
				"author": "author_name",
			},
		}

		if spec.MetaFields["author"] != "author_name" {
			t.Errorf("expected author mapping to author_name")
		}
	})

	t.Run("multiple meta field mappings", func(t *testing.T) {
		spec := NormalizeSpec{
			MetaFields: map[string]string{
				"author":        "author_name",
				"datePublished": "published_date",
				"category":      "article_category",
			},
		}

		if len(spec.MetaFields) != 3 {
			t.Errorf("expected 3 meta field mappings, got %d", len(spec.MetaFields))
		}
		if spec.MetaFields["author"] != "author_name" {
			t.Errorf("author mapping incorrect")
		}
		if spec.MetaFields["datePublished"] != "published_date" {
			t.Errorf("datePublished mapping incorrect")
		}
		if spec.MetaFields["category"] != "article_category" {
			t.Errorf("category mapping incorrect")
		}
	})

	t.Run("empty spec", func(t *testing.T) {
		spec := NormalizeSpec{}

		if spec.TitleField != "" {
			t.Errorf("expected empty TitleField")
		}
		if spec.DescriptionField != "" {
			t.Errorf("expected empty DescriptionField")
		}
		if spec.TextField != "" {
			t.Errorf("expected empty TextField")
		}
		if spec.MetaFields != nil {
			t.Errorf("expected nil MetaFields")
		}
	})
}

func TestSchemaStruct(t *testing.T) {
	t.Run("string type", func(t *testing.T) {
		schema := Schema{
			Type: SchemaString,
		}

		if schema.Type != SchemaString {
			t.Errorf("expected SchemaString, got %v", schema.Type)
		}
	})

	t.Run("number type", func(t *testing.T) {
		schema := Schema{
			Type: SchemaNumber,
		}

		if schema.Type != SchemaNumber {
			t.Errorf("expected SchemaNumber, got %v", schema.Type)
		}
	})

	t.Run("integer type", func(t *testing.T) {
		schema := Schema{
			Type: SchemaInteger,
		}

		if schema.Type != SchemaInteger {
			t.Errorf("expected SchemaInteger, got %v", schema.Type)
		}
	})

	t.Run("boolean type", func(t *testing.T) {
		schema := Schema{
			Type: SchemaBool,
		}

		if schema.Type != SchemaBool {
			t.Errorf("expected SchemaBool, got %v", schema.Type)
		}
	})

	t.Run("array type", func(t *testing.T) {
		schema := Schema{
			Type:  SchemaArray,
			Items: &Schema{Type: SchemaString},
		}

		if schema.Type != SchemaArray {
			t.Errorf("expected SchemaArray, got %v", schema.Type)
		}
		if schema.Items == nil {
			t.Error("expected Items to be set for array type")
		}
	})

	t.Run("object type with properties", func(t *testing.T) {
		schema := Schema{
			Type: SchemaObject,
			Properties: map[string]*Schema{
				"title":   {Type: SchemaString, MinLength: 1, MaxLength: 100},
				"content": {Type: SchemaString, MinLength: 10},
				"count":   {Type: SchemaInteger, Minimum: float64Ptr(0)},
			},
			Required: []string{"title", "content"},
		}

		if schema.Type != SchemaObject {
			t.Errorf("expected SchemaObject, got %v", schema.Type)
		}
		if len(schema.Properties) != 3 {
			t.Errorf("expected 3 properties, got %d", len(schema.Properties))
		}
		if len(schema.Required) != 2 {
			t.Errorf("expected 2 required fields, got %d", len(schema.Required))
		}
	})

	t.Run("with enum", func(t *testing.T) {
		schema := Schema{
			Type: SchemaString,
			Enum: []string{"draft", "published", "archived"},
		}

		if len(schema.Enum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(schema.Enum))
		}
	})

	t.Run("with pattern", func(t *testing.T) {
		schema := Schema{
			Type:    SchemaString,
			Pattern: `^[a-z]+$`,
		}

		if schema.Pattern != `^[a-z]+$` {
			t.Errorf("Pattern not set correctly")
		}
	})

	t.Run("with length constraints", func(t *testing.T) {
		schema := Schema{
			Type:      SchemaString,
			MinLength: 1,
			MaxLength: 255,
		}

		if schema.MinLength != 1 {
			t.Errorf("expected MinLength 1, got %d", schema.MinLength)
		}
		if schema.MaxLength != 255 {
			t.Errorf("expected MaxLength 255, got %d", schema.MaxLength)
		}
	})

	t.Run("with numeric constraints", func(t *testing.T) {
		min := 0.0
		max := 100.0
		schema := Schema{
			Type:    SchemaNumber,
			Minimum: &min,
			Maximum: &max,
		}

		if schema.Minimum == nil || *schema.Minimum != 0.0 {
			t.Error("Minimum not set correctly")
		}
		if schema.Maximum == nil || *schema.Maximum != 100.0 {
			t.Error("Maximum not set correctly")
		}
	})

	t.Run("with additional properties", func(t *testing.T) {
		schema := Schema{
			Type:                 SchemaObject,
			AdditionalProperties: true,
		}

		if !schema.AdditionalProperties {
			t.Error("expected AdditionalProperties to be true")
		}
	})
}

func TestTemplateRegistryStruct(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		registry := TemplateRegistry{
			Templates: make(map[string]Template),
		}

		if registry.Templates == nil {
			t.Error("Templates map should be initialized")
		}
		if len(registry.Templates) != 0 {
			t.Errorf("expected 0 templates, got %d", len(registry.Templates))
		}
	})

	t.Run("adding templates", func(t *testing.T) {
		registry := TemplateRegistry{
			Templates: make(map[string]Template),
		}

		template1 := Template{Name: "template1"}
		template2 := Template{Name: "template2"}

		registry.Templates["template1"] = template1
		registry.Templates["template2"] = template2

		if len(registry.Templates) != 2 {
			t.Errorf("expected 2 templates, got %d", len(registry.Templates))
		}
	})

	t.Run("template lookup", func(t *testing.T) {
		registry := TemplateRegistry{
			Templates: map[string]Template{
				"article": {Name: "article"},
				"product": {Name: "product"},
			},
		}

		tmpl, ok := registry.Templates["article"]
		if !ok {
			t.Error("template not found")
		}
		if tmpl.Name != "article" {
			t.Errorf("expected name 'article', got %q", tmpl.Name)
		}
	})
}

func float64Ptr(v float64) *float64 {
	return &v
}
