// Package pipeline provides a plugin system for extending scrape and crawl workflows.
// It handles plugin hooks at pre/post stages of fetch, extract, and output operations,
// plugin registration, and JavaScript plugin execution.
// It does NOT handle workflow execution or plugin implementations.
package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/stretchr/testify/assert"
)

func TestValidatorTransformer_Name(t *testing.T) {
	vt := NewValidatorTransformer()
	assert.Equal(t, "validator", vt.Name())
}

func TestValidatorTransformer_Priority(t *testing.T) {
	vt := NewValidatorTransformer()
	assert.Equal(t, 100, vt.Priority())
}

func TestValidatorTransformer_Enabled(t *testing.T) {
	vt := NewValidatorTransformer()
	target := Target{URL: "https://example.com"}
	opts := Options{}

	// Validator is always enabled by default
	assert.True(t, vt.Enabled(target, opts))
}

func TestValidatorTransformer_Transform_NormalizedDocument(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		schema    *extract.Schema
		policy    extract.RejectionPolicy
		wantErr   bool
		wantErrIs error
		wantSkip  bool
		wantValid bool
	}{
		{
			name: "valid normalized document",
			input: extract.NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
			},
			schema: &extract.Schema{
				Format:   extract.SchemaFormatCustom,
				Type:     extract.SchemaObject,
				Required: []string{"url"},
			},
			policy:    extract.RejectPolicyNone,
			wantErr:   false,
			wantSkip:  false,
			wantValid: true,
		},
		{
			name: "invalid document with error policy",
			input: extract.NormalizedDocument{
				URL:   "",
				Title: "Test",
			},
			schema: &extract.Schema{
				Format:   extract.SchemaFormatCustom,
				Type:     extract.SchemaObject,
				Required: []string{"url"},
			},
			policy:    extract.RejectPolicyError,
			wantErr:   true,
			wantSkip:  false,
			wantValid: false,
		},
		{
			name: "invalid document with skip policy",
			input: extract.NormalizedDocument{
				URL:   "",
				Title: "Test",
			},
			schema: &extract.Schema{
				Format:   extract.SchemaFormatCustom,
				Type:     extract.SchemaObject,
				Required: []string{"url"},
			},
			policy:    extract.RejectPolicySkip,
			wantErr:   true,
			wantErrIs: extract.ErrDocumentSkipped,
			wantSkip:  true,
			wantValid: false,
		},
		{
			name: "invalid document with empty policy",
			input: extract.NormalizedDocument{
				URL:   "",
				Title: "Test",
			},
			schema: &extract.Schema{
				Format:   extract.SchemaFormatCustom,
				Type:     extract.SchemaObject,
				Required: []string{"url"},
			},
			policy:    extract.RejectPolicyEmpty,
			wantErr:   false,
			wantSkip:  false,
			wantValid: false,
		},
		{
			name:    "non-document input passes through",
			input:   "not a document",
			schema:  nil,
			policy:  extract.RejectPolicyNone,
			wantErr: false,
		},
		{
			name: "pointer to normalized document",
			input: &extract.NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
			},
			schema: &extract.Schema{
				Format:   extract.SchemaFormatCustom,
				Type:     extract.SchemaObject,
				Required: []string{"url"},
			},
			policy:    extract.RejectPolicyNone,
			wantErr:   false,
			wantSkip:  false,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewValidatorTransformer(
				WithSchema(tt.schema),
				WithRejectionPolicy(tt.policy),
			)

			ctx := HookContext{
				Context:   context.Background(),
				RequestID: "test-123",
				Stage:     StagePreOutput,
				Target:    Target{URL: "https://example.com"},
				Now:       time.Now(),
				Options:   Options{},
			}

			in := OutputInput{
				Target:     Target{URL: "https://example.com"},
				Kind:       "test",
				Raw:        []byte("test"),
				Structured: tt.input,
			}

			out, err := vt.Transform(ctx, in)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrIs != nil {
					assert.True(t, errors.Is(err, tt.wantErrIs))
				}
				if tt.wantSkip {
					assert.Nil(t, out.Structured)
				}
			} else {
				assert.NoError(t, err)
				if tt.input == "not a document" {
					assert.Equal(t, tt.input, out.Structured)
				}
			}
		})
	}
}

func TestValidatorTransformer_Transform_Extracted(t *testing.T) {
	vt := NewValidatorTransformer(
		WithSchema(&extract.Schema{
			Format:   extract.SchemaFormatCustom,
			Type:     extract.SchemaObject,
			Required: []string{"url"},
		}),
		WithRejectionPolicy(extract.RejectPolicyNone),
	)

	ctx := HookContext{
		Context:   context.Background(),
		RequestID: "test-123",
		Stage:     StagePreOutput,
		Target:    Target{URL: "https://example.com"},
		Now:       time.Now(),
		Options:   Options{},
	}

	extracted := extract.Extracted{
		URL:      "https://example.com",
		Title:    "Test",
		Template: "test",
	}

	in := OutputInput{
		Target:     Target{URL: "https://example.com"},
		Kind:       "test",
		Raw:        []byte("test"),
		Structured: extracted,
	}

	out, err := vt.Transform(ctx, in)
	assert.NoError(t, err)
	assert.NotNil(t, out.Structured)

	// Should be converted to NormalizedDocument
	_, ok := out.Structured.(extract.NormalizedDocument)
	assert.True(t, ok)
}

func TestValidatorTransformer_WithSchema(t *testing.T) {
	schema := &extract.Schema{
		Format: extract.SchemaFormatJSONSchema,
		JSONSchema: map[string]any{
			"type": "object",
		},
	}

	vt := NewValidatorTransformer(WithSchema(schema))
	assert.Equal(t, schema, vt.schema)
}

func TestValidatorTransformer_WithRejectionPolicy(t *testing.T) {
	vt := NewValidatorTransformer(WithRejectionPolicy(extract.RejectPolicyError))
	assert.Equal(t, extract.RejectPolicyError, vt.rejectionPolicy)
}

func TestExtractDocument(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantDoc extract.NormalizedDocument
		wantOK  bool
	}{
		{
			name: "NormalizedDocument",
			input: extract.NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
			},
			wantDoc: extract.NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
			},
			wantOK: true,
		},
		{
			name: "*NormalizedDocument",
			input: &extract.NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
			},
			wantDoc: extract.NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
			},
			wantOK: true,
		},
		{
			name: "Extracted",
			input: extract.Extracted{
				URL:   "https://example.com",
				Title: "Test",
			},
			wantDoc: extract.NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
			},
			wantOK: true,
		},
		{
			name: "*Extracted",
			input: &extract.Extracted{
				URL:   "https://example.com",
				Title: "Test",
			},
			wantDoc: extract.NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
			},
			wantOK: true,
		},
		{
			name:    "string - not extractable",
			input:   "not a document",
			wantDoc: extract.NormalizedDocument{},
			wantOK:  false,
		},
		{
			name:    "nil - not extractable",
			input:   nil,
			wantDoc: extract.NormalizedDocument{},
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDoc, gotOK := ExtractDocument(tt.input)
			assert.Equal(t, tt.wantOK, gotOK)
			if tt.wantOK {
				assert.Equal(t, tt.wantDoc.URL, gotDoc.URL)
				assert.Equal(t, tt.wantDoc.Title, gotDoc.Title)
			}
		})
	}
}

func TestValidatorTransformer_getSchema(t *testing.T) {
	tests := []struct {
		name       string
		vtSchema   *extract.Schema
		doc        extract.NormalizedDocument
		wantSchema *extract.Schema
	}{
		{
			name: "configured schema takes precedence",
			vtSchema: &extract.Schema{
				Format: extract.SchemaFormatCustom,
				Type:   extract.SchemaObject,
			},
			doc: extract.NormalizedDocument{
				Template: "test",
			},
			wantSchema: &extract.Schema{
				Format: extract.SchemaFormatCustom,
				Type:   extract.SchemaObject,
			},
		},
		{
			name:     "no configured schema returns nil",
			vtSchema: nil,
			doc: extract.NormalizedDocument{
				Template: "test",
			},
			wantSchema: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewValidatorTransformer(WithSchema(tt.vtSchema))
			gotSchema := vt.getSchema(tt.doc)
			assert.Equal(t, tt.wantSchema, gotSchema)
		})
	}
}

func TestValidatorTransformer_getRejectionPolicy(t *testing.T) {
	tests := []struct {
		name       string
		vtPolicy   extract.RejectionPolicy
		doc        extract.NormalizedDocument
		wantPolicy extract.RejectionPolicy
	}{
		{
			name:       "configured policy takes precedence",
			vtPolicy:   extract.RejectPolicyError,
			doc:        extract.NormalizedDocument{},
			wantPolicy: extract.RejectPolicyError,
		},
		{
			name:       "default to none when not configured",
			vtPolicy:   "",
			doc:        extract.NormalizedDocument{},
			wantPolicy: extract.RejectPolicyNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewValidatorTransformer(WithRejectionPolicy(tt.vtPolicy))
			gotPolicy := vt.getRejectionPolicy(tt.doc)
			assert.Equal(t, tt.wantPolicy, gotPolicy)
		})
	}
}

func TestRegistry_RegisterValidator(t *testing.T) {
	registry := NewRegistry()

	schema := &extract.Schema{
		Format: extract.SchemaFormatCustom,
		Type:   extract.SchemaObject,
	}

	policy := extract.RejectPolicyError

	registry.RegisterValidator(schema, policy)

	// Verify the validator was registered by checking transformers
	// The validator should be in the registry's transformers list
	assert.Equal(t, 1, len(registry.transformers))

	// Verify it's a ValidatorTransformer
	entry := registry.transformers[0]
	vt, ok := entry.transformer.(*ValidatorTransformer)
	assert.True(t, ok)
	assert.Equal(t, schema, vt.schema)
	assert.Equal(t, policy, vt.rejectionPolicy)
}

func TestNewValidatorTransformer_Defaults(t *testing.T) {
	vt := NewValidatorTransformer()

	assert.Nil(t, vt.schema)
	assert.Equal(t, extract.RejectPolicyNone, vt.rejectionPolicy)
	assert.Equal(t, "validator", vt.Name())
	assert.Equal(t, 100, vt.Priority())
}
