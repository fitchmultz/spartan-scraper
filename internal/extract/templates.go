package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var (
	builtInTemplates = map[string]Template{
		"default": {
			Name: "default",
			Selectors: []SelectorRule{
				{Name: "title", Selector: "title", Attr: "text", Trim: true},
				{Name: "description", Selector: "meta[name=description]", Attr: "content", Trim: true},
				{Name: "og:description", Selector: "meta[property='og:description']", Attr: "content", Trim: true},
				{Name: "h1", Selector: "h1", Attr: "text", Trim: true, All: true},
			},
			Normalize: NormalizeSpec{
				TitleField:       "title",
				DescriptionField: "description",
			},
		},
		"article": {
			Name: "article",
			Selectors: []SelectorRule{
				{Name: "title", Selector: "title", Attr: "text", Trim: true},
				{Name: "h1", Selector: "h1", Attr: "text", Trim: true},
				{Name: "author", Selector: "meta[name=author]", Attr: "content", Trim: true},
				{Name: "date", Selector: "meta[property='article:published_time']", Attr: "content", Trim: true},
				{Name: "content", Selector: "article", Attr: "text", Trim: true},
			},
			JSONLD: []JSONLDRule{
				{Name: "headline", Type: "Article", Path: "headline"},
				{Name: "author", Type: "Article", Path: "author.name"},
				{Name: "datePublished", Type: "Article", Path: "datePublished"},
			},
			Normalize: NormalizeSpec{
				TitleField: "title",
				TextField:  "content",
				MetaFields: map[string]string{
					"author":        "author",
					"datePublished": "date",
				},
			},
		},
		"product": {
			Name: "product",
			Selectors: []SelectorRule{
				{Name: "title", Selector: "title", Attr: "text", Trim: true},
				{Name: "name", Selector: "h1", Attr: "text", Trim: true},
				{Name: "price", Selector: "[itemprop=price]", Attr: "content", Trim: true},
				{Name: "currency", Selector: "[itemprop=priceCurrency]", Attr: "content", Trim: true},
			},
			JSONLD: []JSONLDRule{
				{Name: "name", Type: "Product", Path: "name"},
				{Name: "price", Type: "Product", Path: "offers.price"},
				{Name: "currency", Type: "Product", Path: "offers.priceCurrency"},
			},
			Normalize: NormalizeSpec{
				TitleField: "name",
				MetaFields: map[string]string{
					"price":    "price",
					"currency": "currency",
				},
			},
		},
	}
)

func LoadTemplateRegistry(dataDir string) (*TemplateRegistry, error) {
	registry := &TemplateRegistry{
		Templates: make(map[string]Template),
	}

	// Load built-ins first
	for k, v := range builtInTemplates {
		registry.Templates[k] = v
	}

	// Load from file if exists
	path := filepath.Join(dataDir, "extract_templates.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return registry, nil
		}
		return nil, err
	}

	var tf TemplateFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, err
	}

	for _, t := range tf.Templates {
		registry.Templates[t.Name] = t
	}

	return registry, nil
}

func ResolveTemplate(opts ExtractOptions, registry *TemplateRegistry) (Template, error) {
	if opts.Inline != nil {
		return *opts.Inline, nil
	}

	name := opts.Template
	if name == "" {
		name = "default"
	}

	// Check registry (which includes built-ins + file loaded)
	if registry != nil {
		if t, ok := registry.Templates[name]; ok {
			return t, nil
		}
	}

	// Fallback to built-in directly if registry is nil or missing
	if t, ok := builtInTemplates[name]; ok {
		return t, nil
	}

	// Fallback to default if named one not found
	return builtInTemplates["default"], nil
}
