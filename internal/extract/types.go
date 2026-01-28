// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// It handles template-based extraction, field normalization, and schema validation.
// It does NOT handle fetching or rendering HTML content.
package extract

import (
	"encoding/json"
	"time"
)

type ExtractOptions struct {
	Template string    `json:"template,omitempty"`
	Inline   *Template `json:"inline,omitempty"`
	Validate bool      `json:"validate,omitempty"`
}

type Template struct {
	Name      string         `json:"name"`
	Selectors []SelectorRule `json:"selectors,omitempty"`
	JSONLD    []JSONLDRule   `json:"jsonld,omitempty"`
	Regex     []RegexRule    `json:"regex,omitempty"`
	Schema    *Schema        `json:"schema,omitempty"`
	Normalize NormalizeSpec  `json:"normalize,omitempty"`
}

type SelectorRule struct {
	Name     string `json:"name"`
	Selector string `json:"selector"`
	Attr     string `json:"attr,omitempty"` // "text" for text, or attribute name
	All      bool   `json:"all,omitempty"`
	Join     string `json:"join,omitempty"`
	Trim     bool   `json:"trim,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type JSONLDRule struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"` // match @type
	Path     string `json:"path,omitempty"` // dot path in JSON-LD object
	All      bool   `json:"all,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type RegexSource string

const (
	RegexSourceText RegexSource = "text"
	RegexSourceHTML RegexSource = "html"
	RegexSourceURL  RegexSource = "url"
)

type RegexRule struct {
	Name     string      `json:"name"`
	Pattern  string      `json:"pattern"`
	Group    int         `json:"group,omitempty"`
	All      bool        `json:"all,omitempty"`
	Source   RegexSource `json:"source,omitempty"`
	Required bool        `json:"required,omitempty"`
}

type NormalizeSpec struct {
	TitleField       string            `json:"titleField,omitempty"`
	DescriptionField string            `json:"descriptionField,omitempty"`
	TextField        string            `json:"textField,omitempty"`
	MetaFields       map[string]string `json:"metaFields,omitempty"` // normalizedKey -> fieldName
}

type FieldSource string

const (
	FieldSourceSelector FieldSource = "selector"
	FieldSourceJSONLD   FieldSource = "jsonld"
	FieldSourceRegex    FieldSource = "regex"
	FieldSourceDerived  FieldSource = "derived"
)

type FieldValue struct {
	Values    []string    `json:"values,omitempty"`
	Source    FieldSource `json:"source"`
	RawObject string      `json:"rawObject,omitempty"` // JSON-encoded object for SchemaObject validation
}

type Extracted struct {
	URL         string                `json:"url"`
	Title       string                `json:"title"`
	Text        string                `json:"text"`
	Links       []string              `json:"links"`
	Metadata    map[string]string     `json:"metadata,omitempty"`
	Fields      map[string]FieldValue `json:"fields,omitempty"`
	JSONLD      []map[string]any      `json:"jsonld,omitempty"`
	Raw         map[string][]string   `json:"raw,omitempty"` // optional: per-rule raw values
	Template    string                `json:"template"`
	ExtractedAt time.Time             `json:"extractedAt"`
}

type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

type NormalizedDocument struct {
	URL         string                `json:"url"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Text        string                `json:"text"`
	Links       []string              `json:"links"`
	Metadata    map[string]string     `json:"metadata,omitempty"`
	Fields      map[string]FieldValue `json:"fields,omitempty"`
	JSONLD      []map[string]any      `json:"jsonld,omitempty"`
	Template    string                `json:"template"`
	ExtractedAt time.Time             `json:"extractedAt"`
	Validation  ValidationResult      `json:"validation"`
}

type ExecuteInput struct {
	URL      string
	HTML     string
	Options  ExtractOptions
	DataDir  string
	Registry *TemplateRegistry
}

type ExecuteOutput struct {
	Extracted  Extracted
	Normalized NormalizedDocument
}

type SchemaType string

const (
	SchemaString  SchemaType = "string"
	SchemaNumber  SchemaType = "number"
	SchemaInteger SchemaType = "integer"
	SchemaBool    SchemaType = "boolean"
	SchemaArray   SchemaType = "array"
	SchemaObject  SchemaType = "object"
)

type Schema struct {
	Type                 SchemaType         `json:"type"`
	Required             []string           `json:"required,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Enum                 []string           `json:"enum,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	MinLength            int                `json:"minLength,omitempty"`
	MaxLength            int                `json:"maxLength,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	AdditionalProperties bool               `json:"additionalProperties,omitempty"`
}

// NewObjectFieldValue creates a FieldValue containing a nested object.
// The nested map will be JSON-serialized for storage and can be validated
// against a SchemaObject schema.
func NewObjectFieldValue(fields map[string]FieldValue, source FieldSource) (FieldValue, error) {
	data, err := json.Marshal(fields)
	if err != nil {
		return FieldValue{}, err
	}
	return FieldValue{
		RawObject: string(data),
		Source:    source,
	}, nil
}

type TemplateRegistry struct {
	Templates map[string]Template
}

type TemplateFile struct {
	Templates []Template `json:"templates"`
}
