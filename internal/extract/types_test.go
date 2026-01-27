package extract

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFieldValueStruct(t *testing.T) {
	t.Run("single value", func(t *testing.T) {
		fv := FieldValue{
			Values: []string{"test"},
			Source: FieldSourceSelector,
		}

		if len(fv.Values) != 1 {
			t.Errorf("expected 1 value, got %d", len(fv.Values))
		}
		if fv.Values[0] != "test" {
			t.Errorf("expected 'test', got %q", fv.Values[0])
		}
	})

	t.Run("multiple values", func(t *testing.T) {
		fv := FieldValue{
			Values: []string{"val1", "val2", "val3"},
			Source: FieldSourceJSONLD,
		}

		if len(fv.Values) != 3 {
			t.Errorf("expected 3 values, got %d", len(fv.Values))
		}
	})

	t.Run("with RawObject", func(t *testing.T) {
		fv := FieldValue{
			Values:    []string{"test"},
			Source:    FieldSourceRegex,
			RawObject: `{"key":"value"}`,
		}

		if fv.RawObject != `{"key":"value"}` {
			t.Errorf("unexpected RawObject: %q", fv.RawObject)
		}
	})

	t.Run("field source constants", func(t *testing.T) {
		sources := []FieldSource{
			FieldSourceSelector,
			FieldSourceJSONLD,
			FieldSourceRegex,
			FieldSourceDerived,
		}

		for _, source := range sources {
			fv := FieldValue{Source: source}
			if fv.Source != source {
				t.Errorf("source not preserved: expected %v, got %v", source, fv.Source)
			}
		}
	})
}

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

func TestSelectorRuleStruct(t *testing.T) {
	t.Run("text attribute", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "title",
			Selector: "h1",
			Attr:     "text",
		}

		if rule.Attr != "text" {
			t.Errorf("expected Attr 'text', got %q", rule.Attr)
		}
	})

	t.Run("html attribute", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "content",
			Selector: ".content",
			Attr:     "html",
		}

		if rule.Attr != "html" {
			t.Errorf("expected Attr 'html', got %q", rule.Attr)
		}
	})

	t.Run("content attribute", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "desc",
			Selector: "meta[name=description]",
			Attr:     "content",
		}

		if rule.Attr != "content" {
			t.Errorf("expected Attr 'content', got %q", rule.Attr)
		}
	})

	t.Run("href attribute", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "link",
			Selector: "a",
			Attr:     "href",
		}

		if rule.Attr != "href" {
			t.Errorf("expected Attr 'href', got %q", rule.Attr)
		}
	})

	t.Run("with All flag", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "headings",
			Selector: "h2",
			Attr:     "text",
			All:      true,
		}

		if !rule.All {
			t.Error("expected All to be true")
		}
	})

	t.Run("with Trim flag", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "title",
			Selector: "title",
			Attr:     "text",
			Trim:     true,
		}

		if !rule.Trim {
			t.Error("expected Trim to be true")
		}
	})

	t.Run("with Required flag", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "title",
			Selector: "title",
			Attr:     "text",
			Required: true,
		}

		if !rule.Required {
			t.Error("expected Required to be true")
		}
	})

	t.Run("with Join parameter", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "items",
			Selector: ".item",
			Attr:     "text",
			All:      true,
			Join:     ", ",
		}

		if rule.Join != ", " {
			t.Errorf("expected Join ', ', got %q", rule.Join)
		}
	})
}

func TestJSONLDRuleStruct(t *testing.T) {
	t.Run("with type matching", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "headline",
			Type: "Article",
			Path: "headline",
		}

		if rule.Type != "Article" {
			t.Errorf("expected Type 'Article', got %q", rule.Type)
		}
	})

	t.Run("with path traversal", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "author",
			Type: "Article",
			Path: "author.name",
		}

		if rule.Path != "author.name" {
			t.Errorf("expected Path 'author.name', got %q", rule.Path)
		}
	})

	t.Run("nested offers path", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "price",
			Type: "Product",
			Path: "offers.price",
		}

		if rule.Path != "offers.price" {
			t.Errorf("expected Path 'offers.price', got %q", rule.Path)
		}
	})

	t.Run("with All flag", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "author",
			Type: "Article",
			Path: "author.name",
			All:  true,
		}

		if !rule.All {
			t.Error("expected All to be true")
		}
	})

	t.Run("with Required flag", func(t *testing.T) {
		rule := JSONLDRule{
			Name:     "headline",
			Type:     "Article",
			Path:     "headline",
			Required: true,
		}

		if !rule.Required {
			t.Error("expected Required to be true")
		}
	})

	t.Run("no type filter", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "name",
			Path: "name",
		}

		if rule.Type != "" {
			t.Errorf("expected empty Type, got %q", rule.Type)
		}
	})
}

func TestRegexRuleStruct(t *testing.T) {
	t.Run("text source", func(t *testing.T) {
		rule := RegexRule{
			Name:    "email",
			Pattern: `\S+@\S+\.\S+`,
			Source:  RegexSourceText,
		}

		if rule.Source != RegexSourceText {
			t.Errorf("expected Source RegexSourceText, got %v", rule.Source)
		}
	})

	t.Run("html source", func(t *testing.T) {
		rule := RegexRule{
			Name:    "link",
			Pattern: `href="([^"]*)"`,
			Source:  RegexSourceHTML,
		}

		if rule.Source != RegexSourceHTML {
			t.Errorf("expected Source RegexSourceHTML, got %v", rule.Source)
		}
	})

	t.Run("url source", func(t *testing.T) {
		rule := RegexRule{
			Name:    "path",
			Pattern: `/[^/]+$`,
			Source:  RegexSourceURL,
		}

		if rule.Source != RegexSourceURL {
			t.Errorf("expected Source RegexSourceURL, got %v", rule.Source)
		}
	})

	t.Run("with pattern", func(t *testing.T) {
		rule := RegexRule{
			Name:    "phone",
			Pattern: `\d{3}-\d{3}-\d{4}`,
		}

		if rule.Pattern != `\d{3}-\d{3}-\d{4}` {
			t.Errorf("unexpected Pattern: %q", rule.Pattern)
		}
	})

	t.Run("with group extraction", func(t *testing.T) {
		rule := RegexRule{
			Name:    "email",
			Pattern: `(\S+)@\S+\.\S+`,
			Group:   1,
		}

		if rule.Group != 1 {
			t.Errorf("expected Group 1, got %d", rule.Group)
		}
	})

	t.Run("default group", func(t *testing.T) {
		rule := RegexRule{
			Name:    "email",
			Pattern: `\S+@\S+\.\S+`,
		}

		if rule.Group != 0 {
			t.Errorf("expected default Group 0, got %d", rule.Group)
		}
	})

	t.Run("with All flag", func(t *testing.T) {
		rule := RegexRule{
			Name:    "email",
			Pattern: `\S+@\S+\.\S+`,
			All:     true,
		}

		if !rule.All {
			t.Error("expected All to be true")
		}
	})

	t.Run("with Required flag", func(t *testing.T) {
		rule := RegexRule{
			Name:     "email",
			Pattern:  `\S+@\S+\.\S+`,
			Required: true,
		}

		if !rule.Required {
			t.Error("expected Required to be true")
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

func TestExtractedStruct(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		ext := Extracted{
			URL:         "http://example.com",
			Title:       "Test Title",
			Text:        "Test text content",
			Links:       []string{"/link1", "/link2"},
			Template:    "default",
			ExtractedAt: time.Now(),
		}

		if ext.URL != "http://example.com" {
			t.Errorf("URL not set correctly")
		}
		if ext.Title != "Test Title" {
			t.Errorf("Title not set correctly")
		}
		if ext.Text != "Test text content" {
			t.Errorf("Text not set correctly")
		}
		if len(ext.Links) != 2 {
			t.Errorf("expected 2 links, got %d", len(ext.Links))
		}
	})

	t.Run("with optional fields", func(t *testing.T) {
		metadata := map[string]string{
			"author": "John Doe",
			"date":   "2024-01-01",
		}
		fields := map[string]FieldValue{
			"headline": {Values: []string{"Headline"}, Source: FieldSourceSelector},
		}
		jsonld := []map[string]any{
			{"@type": "Article", "headline": "Headline"},
		}

		ext := Extracted{
			URL:         "http://example.com",
			Title:       "Test Title",
			Text:        "Test text",
			Links:       []string{},
			Metadata:    metadata,
			Fields:      fields,
			JSONLD:      jsonld,
			Template:    "article",
			ExtractedAt: time.Now(),
		}

		if ext.Metadata == nil {
			t.Error("Metadata should be set")
		}
		if ext.Metadata["author"] != "John Doe" {
			t.Errorf("Metadata author not set correctly")
		}
		if ext.Fields == nil {
			t.Error("Fields should be set")
		}
		if len(ext.JSONLD) != 1 {
			t.Errorf("expected 1 JSON-LD object, got %d", len(ext.JSONLD))
		}
	})

	t.Run("with Raw field", func(t *testing.T) {
		raw := map[string][]string{
			"title":   {"Page Title"},
			"content": {"Content here"},
		}

		ext := Extracted{
			URL:         "http://example.com",
			Title:       "Test",
			Text:        "Text",
			Links:       []string{},
			Raw:         raw,
			Template:    "default",
			ExtractedAt: time.Now(),
		}

		if ext.Raw == nil {
			t.Error("Raw should be set")
		}
		if len(ext.Raw) != 2 {
			t.Errorf("expected 2 Raw entries, got %d", len(ext.Raw))
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		ext := Extracted{
			URL:         "http://example.com",
			Title:       "Test",
			Text:        "Text",
			Links:       []string{"/link"},
			Template:    "default",
			ExtractedAt: time.Now(),
		}

		data, err := json.Marshal(ext)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled Extracted
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.URL != ext.URL {
			t.Error("URL not preserved")
		}
		if unmarshaled.Title != ext.Title {
			t.Error("Title not preserved")
		}
	})
}

func TestNormalizedDocumentStruct(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		norm := NormalizedDocument{
			URL:         "http://example.com",
			Title:       "Test Title",
			Description: "Test description",
			Text:        "Test text",
			Links:       []string{"/link1"},
			Template:    "default",
			ExtractedAt: time.Now(),
			Validation:  ValidationResult{Valid: true},
		}

		if norm.URL != "http://example.com" {
			t.Errorf("URL not set correctly")
		}
		if norm.Title != "Test Title" {
			t.Errorf("Title not set correctly")
		}
		if norm.Description != "Test description" {
			t.Errorf("Description not set correctly")
		}
		if norm.Text != "Test text" {
			t.Errorf("Text not set correctly")
		}
	})

	t.Run("with optional fields", func(t *testing.T) {
		metadata := map[string]string{
			"author": "Jane Doe",
		}
		fields := map[string]FieldValue{
			"category": {Values: []string{"Tech"}, Source: FieldSourceSelector},
		}
		jsonld := []map[string]any{
			{"@type": "Article", "headline": "Headline"},
		}

		norm := NormalizedDocument{
			URL:         "http://example.com",
			Title:       "Test",
			Description: "Desc",
			Text:        "Text",
			Links:       []string{},
			Metadata:    metadata,
			Fields:      fields,
			JSONLD:      jsonld,
			Template:    "article",
			ExtractedAt: time.Now(),
			Validation:  ValidationResult{Valid: true},
		}

		if norm.Metadata == nil {
			t.Error("Metadata should be set")
		}
		if norm.Metadata["author"] != "Jane Doe" {
			t.Errorf("Metadata author incorrect")
		}
		if len(norm.JSONLD) != 1 {
			t.Errorf("expected 1 JSON-LD object")
		}
	})

	t.Run("with invalid validation", func(t *testing.T) {
		norm := NormalizedDocument{
			URL:         "http://example.com",
			Title:       "Test",
			Description: "Desc",
			Text:        "Text",
			Links:       []string{},
			Template:    "default",
			ExtractedAt: time.Now(),
			Validation: ValidationResult{
				Valid:  false,
				Errors: []string{"title is required"},
			},
		}

		if norm.Validation.Valid {
			t.Error("expected Valid to be false")
		}
		if len(norm.Validation.Errors) != 1 {
			t.Errorf("expected 1 validation error, got %d", len(norm.Validation.Errors))
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		norm := NormalizedDocument{
			URL:         "http://example.com",
			Title:       "Test",
			Description: "Desc",
			Text:        "Text",
			Links:       []string{},
			Template:    "default",
			ExtractedAt: time.Now(),
			Validation:  ValidationResult{Valid: true},
		}

		data, err := json.Marshal(norm)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled NormalizedDocument
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Title != norm.Title {
			t.Error("Title not preserved")
		}
		if unmarshaled.Description != norm.Description {
			t.Error("Description not preserved")
		}
	})
}

func TestValidationResultStruct(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		vr := ValidationResult{
			Valid:  true,
			Errors: []string{},
		}

		if !vr.Valid {
			t.Error("expected Valid to be true")
		}
		if vr.Errors == nil {
			t.Error("expected Errors to be initialized")
		}
		if len(vr.Errors) != 0 {
			t.Errorf("expected 0 errors, got %d", len(vr.Errors))
		}
	})

	t.Run("invalid result", func(t *testing.T) {
		vr := ValidationResult{
			Valid:  false,
			Errors: []string{"title is required", "description too short"},
		}

		if vr.Valid {
			t.Error("expected Valid to be false")
		}
		if len(vr.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(vr.Errors))
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		vr := ValidationResult{
			Valid:  false,
			Errors: []string{"error 1", "error 2"},
		}

		data, err := json.Marshal(vr)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled ValidationResult
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Valid != vr.Valid {
			t.Error("Valid not preserved")
		}
		if len(unmarshaled.Errors) != len(vr.Errors) {
			t.Error("Errors count not preserved")
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
