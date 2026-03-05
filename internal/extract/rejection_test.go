// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// It handles template-based extraction, field normalization, and schema validation.
// It does NOT handle fetching or rendering HTML content.
package extract

import (
	"errors"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestApplyRejectionPolicy(t *testing.T) {
	tests := []struct {
		name          string
		doc           NormalizedDocument
		validation    ValidationResult
		policy        RejectionPolicy
		wantSkip      bool
		wantEmpty     bool
		wantError     bool
		wantErrIsKind apperrors.Kind
		wantDocFields map[string]FieldValue
	}{
		{
			name: "validation passed - no rejection",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			validation: ValidationResult{Valid: true},
			policy:     RejectPolicyNone,
			wantSkip:   false,
			wantEmpty:  false,
			wantError:  false,
			wantDocFields: map[string]FieldValue{
				"name": {Values: []string{"test"}, Source: FieldSourceSelector},
			},
		},
		{
			name: "validation failed - policy none",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			validation: ValidationResult{
				Valid:  false,
				Errors: []string{"field 'name' is required"},
			},
			policy:    RejectPolicyNone,
			wantSkip:  false,
			wantEmpty: false,
			wantError: false,
			wantDocFields: map[string]FieldValue{
				"name": {Values: []string{"test"}, Source: FieldSourceSelector},
			},
		},
		{
			name: "validation failed - policy skip",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			validation: ValidationResult{
				Valid:  false,
				Errors: []string{"field 'name' is required"},
			},
			policy:    RejectPolicySkip,
			wantSkip:  true,
			wantEmpty: false,
			wantError: false,
		},
		{
			name: "validation failed - policy empty",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			validation: ValidationResult{
				Valid:  false,
				Errors: []string{"field 'name' is required"},
			},
			policy:    RejectPolicyEmpty,
			wantSkip:  false,
			wantEmpty: true,
			wantError: false,
		},
		{
			name: "validation failed - policy error",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			validation: ValidationResult{
				Valid:  false,
				Errors: []string{"field 'name' is required"},
			},
			policy:        RejectPolicyError,
			wantSkip:      false,
			wantEmpty:     false,
			wantError:     true,
			wantErrIsKind: apperrors.KindValidation,
		},
		{
			name: "validation failed - empty policy string defaults to none",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			validation: ValidationResult{
				Valid:  false,
				Errors: []string{"field 'name' is required"},
			},
			policy:    "",
			wantSkip:  false,
			wantEmpty: false,
			wantError: false,
		},
		{
			name: "validation failed - unknown policy defaults to none",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			validation: ValidationResult{
				Valid:  false,
				Errors: []string{"field 'name' is required"},
			},
			policy:    "unknown_policy",
			wantSkip:  false,
			wantEmpty: false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyRejectionPolicy(tt.doc, tt.validation, tt.policy)

			if result.Skip != tt.wantSkip {
				t.Errorf("ApplyRejectionPolicy() Skip = %v, want %v", result.Skip, tt.wantSkip)
			}
			if result.Empty != tt.wantEmpty {
				t.Errorf("ApplyRejectionPolicy() Empty = %v, want %v", result.Empty, tt.wantEmpty)
			}

			if tt.wantError {
				if result.Error == nil {
					t.Errorf("ApplyRejectionPolicy() Error = nil, want error")
				} else if tt.wantErrIsKind != "" {
					if !apperrors.IsKind(result.Error, tt.wantErrIsKind) {
						t.Errorf("ApplyRejectionPolicy() Error kind = %v, want %v", apperrors.KindOf(result.Error), tt.wantErrIsKind)
					}
				}
			} else if result.Error != nil {
				t.Errorf("ApplyRejectionPolicy() Error = %v, want nil", result.Error)
			}

			if tt.wantEmpty {
				// Empty document should have URL and Template preserved
				if result.Document.URL != tt.doc.URL {
					t.Errorf("ApplyRejectionPolicy() Document.URL = %v, want %v", result.Document.URL, tt.doc.URL)
				}
				if result.Document.Template != tt.doc.Template {
					t.Errorf("ApplyRejectionPolicy() Document.Template = %v, want %v", result.Document.Template, tt.doc.Template)
				}
				// Fields should be empty
				if len(result.Document.Fields) != 0 {
					t.Errorf("ApplyRejectionPolicy() Document.Fields length = %d, want 0", len(result.Document.Fields))
				}
				// Validation errors should be preserved (Valid should be false since validation failed)
				if result.Document.Validation.Valid {
					t.Errorf("ApplyRejectionPolicy() Document.Validation.Valid = true, want false (validation errors preserved)")
				}
				if len(result.Document.Validation.Errors) == 0 {
					t.Errorf("ApplyRejectionPolicy() Document.Validation.Errors is empty, want errors preserved")
				}
			}

			if tt.wantDocFields != nil {
				if len(result.Document.Fields) != len(tt.wantDocFields) {
					t.Errorf("ApplyRejectionPolicy() Document.Fields length = %d, want %d", len(result.Document.Fields), len(tt.wantDocFields))
				}
			}
		})
	}
}

func TestGetEffectiveRejectionPolicy(t *testing.T) {
	tests := []struct {
		name       string
		opts       ExtractOptions
		tmpl       Template
		wantPolicy RejectionPolicy
	}{
		{
			name:       "options takes precedence",
			opts:       ExtractOptions{RejectionPolicy: RejectPolicyError},
			tmpl:       Template{RejectionPolicy: RejectPolicySkip},
			wantPolicy: RejectPolicyError,
		},
		{
			name:       "template used when options empty",
			opts:       ExtractOptions{},
			tmpl:       Template{RejectionPolicy: RejectPolicySkip},
			wantPolicy: RejectPolicySkip,
		},
		{
			name:       "default to none when both empty",
			opts:       ExtractOptions{},
			tmpl:       Template{},
			wantPolicy: RejectPolicyNone,
		},
		{
			name:       "options empty string falls through to template",
			opts:       ExtractOptions{RejectionPolicy: ""},
			tmpl:       Template{RejectionPolicy: RejectPolicyEmpty},
			wantPolicy: RejectPolicyEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEffectiveRejectionPolicy(tt.opts, tt.tmpl)
			if got != tt.wantPolicy {
				t.Errorf("GetEffectiveRejectionPolicy() = %v, want %v", got, tt.wantPolicy)
			}
		})
	}
}

func TestFormatValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors []string
		want   string
	}{
		{
			name:   "no errors",
			errors: []string{},
			want:   "no validation errors",
		},
		{
			name:   "nil errors",
			errors: nil,
			want:   "no validation errors",
		},
		{
			name:   "single error",
			errors: []string{"field 'name' is required"},
			want:   "field 'name' is required",
		},
		{
			name:   "multiple errors",
			errors: []string{"field 'name' is required", "field 'price' must be positive"},
			want:   "2 validation errors: field 'name' is required; field 'price' must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatValidationErrors(tt.errors)
			if got != tt.want {
				t.Errorf("FormatValidationErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRejectionPolicyValid(t *testing.T) {
	tests := []struct {
		name   string
		policy RejectionPolicy
		want   bool
	}{
		{
			name:   "none is valid",
			policy: RejectPolicyNone,
			want:   true,
		},
		{
			name:   "skip is valid",
			policy: RejectPolicySkip,
			want:   true,
		},
		{
			name:   "empty is valid",
			policy: RejectPolicyEmpty,
			want:   true,
		},
		{
			name:   "error is valid",
			policy: RejectPolicyError,
			want:   true,
		},
		{
			name:   "empty string is valid",
			policy: "",
			want:   true,
		},
		{
			name:   "unknown is invalid",
			policy: "unknown_policy",
			want:   false,
		},
		{
			name:   "random string is invalid",
			policy: "random",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRejectionPolicyValid(tt.policy)
			if got != tt.want {
				t.Errorf("IsRejectionPolicyValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRejectionResult_Struct(t *testing.T) {
	// Test that RejectionResult can be properly constructed
	result := RejectionResult{
		Skip:       true,
		Empty:      false,
		Error:      errors.New("test error"),
		Document:   NormalizedDocument{URL: "https://example.com"},
		Validation: ValidationResult{Valid: false, Errors: []string{"error"}},
	}

	if !result.Skip {
		t.Error("Expected Skip to be true")
	}
	if result.Error == nil {
		t.Error("Expected Error to be non-nil")
	}
	if result.Document.URL != "https://example.com" {
		t.Errorf("Expected Document.URL to be 'https://example.com', got %s", result.Document.URL)
	}
}
