// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// It handles template-based extraction, field normalization, and schema validation.
// It does NOT handle fetching or rendering HTML content.
package extract

import (
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// RejectionResult contains the outcome of applying a rejection policy.
type RejectionResult struct {
	Skip       bool
	Empty      bool
	Error      error
	Document   NormalizedDocument
	Validation ValidationResult
}

// ApplyRejectionPolicy applies the specified rejection policy based on validation results.
// It returns a RejectionResult indicating how the document should be handled.
func ApplyRejectionPolicy(
	doc NormalizedDocument,
	validation ValidationResult,
	policy RejectionPolicy,
) RejectionResult {
	// If validation passed, no rejection needed
	if validation.Valid {
		return RejectionResult{
			Skip:       false,
			Empty:      false,
			Error:      nil,
			Document:   doc,
			Validation: validation,
		}
	}

	// Apply policy based on validation failure
	switch policy {
	case RejectPolicyNone, "":
		// Default: store validation result but return document unchanged
		doc.Validation = validation
		return RejectionResult{
			Skip:       false,
			Empty:      false,
			Error:      nil,
			Document:   doc,
			Validation: validation,
		}

	case RejectPolicySkip:
		// Skip the document entirely
		return RejectionResult{
			Skip:       true,
			Empty:      false,
			Error:      nil,
			Document:   NormalizedDocument{},
			Validation: validation,
		}

	case RejectPolicyEmpty:
		// Return empty document with validation errors preserved
		emptyDoc := NormalizedDocument{
			URL:        doc.URL,
			Template:   doc.Template,
			Validation: validation,
		}
		return RejectionResult{
			Skip:       false,
			Empty:      true,
			Error:      nil,
			Document:   emptyDoc,
			Validation: validation,
		}

	case RejectPolicyError:
		// Return validation error
		err := apperrors.Validation("document validation failed: " + strings.Join(validation.Errors, "; "))
		doc.Validation = validation
		return RejectionResult{
			Skip:       false,
			Empty:      false,
			Error:      err,
			Document:   doc,
			Validation: validation,
		}

	default:
		// Unknown policy: treat as "none" but log a warning
		doc.Validation = validation
		return RejectionResult{
			Skip:       false,
			Empty:      false,
			Error:      nil,
			Document:   doc,
			Validation: validation,
		}
	}
}

// GetEffectiveRejectionPolicy returns the rejection policy to use.
// It checks options first, then template, defaulting to "none".
func GetEffectiveRejectionPolicy(opts ExtractOptions, tmpl Template) RejectionPolicy {
	// Check options first (highest priority)
	if opts.RejectionPolicy != "" {
		return opts.RejectionPolicy
	}

	// Check template default
	if tmpl.RejectionPolicy != "" {
		return tmpl.RejectionPolicy
	}

	// Default to "none"
	return RejectPolicyNone
}

// FormatValidationErrors formats validation errors into a single string.
func FormatValidationErrors(errors []string) string {
	if len(errors) == 0 {
		return "no validation errors"
	}
	if len(errors) == 1 {
		return errors[0]
	}
	return fmt.Sprintf("%d validation errors: %s", len(errors), strings.Join(errors, "; "))
}

// IsRejectionPolicyValid returns true if the policy is a known valid policy.
func IsRejectionPolicyValid(policy RejectionPolicy) bool {
	switch policy {
	case RejectPolicyNone, RejectPolicySkip, RejectPolicyEmpty, RejectPolicyError, "":
		return true
	default:
		return false
	}
}
