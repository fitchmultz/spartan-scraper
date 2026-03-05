// Package apperrors_test contains tests for the apperrors package.
//
// This test file verifies:
// - Error kind classification (KindOf, IsKind)
// - Error wrapping and unwrapping behavior
// - Safe message redaction of sensitive content
// - Sentinel error preservation with WithKind
//
// Test coverage includes:
// - All error kinds (validation, not_found, permission, internal, etc.)
// - Wrapped error chains
// - Generic errors (default to KindInternal)
package apperrors

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
)

func TestKindOfAndIsKind(t *testing.T) {
	tests := []struct {
		name string
		err  error
		kind Kind
	}{
		{"Validation", Validation("bad input"), KindValidation},
		{"NotFound", NotFound("not found"), KindNotFound},
		{"Permission", Permission("denied"), KindPermission},
		{"Internal", Internal("failed"), KindInternal},
		{"MethodNotAllowed", MethodNotAllowed("bad method"), KindMethodNotAllowed},
		{"UnsupportedMediaType", UnsupportedMediaType("bad media"), KindUnsupportedMediaType},
		{"Generic", errors.New("generic"), KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("outer: %w", tt.err)
			if got := KindOf(wrapped); got != tt.kind {
				t.Fatalf("KindOf() = %s, want %s", got, tt.kind)
			}
			// IsKind only returns true if an apperrors.Error with that kind is in the chain.
			// Generic errors don't have it, even if KindOf returns KindInternal.
			if tt.name != "Generic" {
				if !IsKind(wrapped, tt.kind) {
					t.Fatalf("IsKind(%s) = false, want true", tt.kind)
				}
			} else {
				if IsKind(wrapped, tt.kind) {
					t.Fatalf("IsKind(%s) = true, want false for generic error", tt.kind)
				}
			}
		})
	}
}

func TestWithKindPreservesErrorsIs(t *testing.T) {
	err := WithKind(KindValidation, ErrInvalidURLScheme)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, ErrInvalidURLScheme) {
		t.Fatalf("expected errors.Is(err, ErrInvalidURLScheme) == true")
	}
	if KindOf(err) != KindValidation {
		t.Fatalf("expected KindValidation, got %s", KindOf(err))
	}
}

func TestWrapUsesMsgButPreservesUnwrap(t *testing.T) {
	inner := errors.New("low-level detail")
	err := Wrap(KindInternal, "safe message", inner)

	if err.Error() != "safe message" {
		t.Fatalf("Error() = %q, want %q", err.Error(), "safe message")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("expected errors.Is to match inner error")
	}
}

func TestSafeMessage_Redacts(t *testing.T) {
	in := errors.New(`Authorization: Bearer abc123 token=xyz password=hunter2 {"apiKey":"shh"}`)
	out := SafeMessage(in)

	if out == "" {
		t.Fatal("expected non-empty SafeMessage")
	}
	if out == in.Error() {
		t.Fatalf("expected redaction to change output; got %q", out)
	}
	if !regexpContains(out, `Bearer \[REDACTED\]`) {
		t.Fatalf("expected Bearer redaction, got %q", out)
	}
	if !regexpContains(out, `token=\[REDACTED\]`) {
		t.Fatalf("expected token= redaction, got %q", out)
	}
	if !regexpContains(out, `password=\[REDACTED\]`) {
		t.Fatalf("expected password= redaction, got %q", out)
	}
	if !regexpContains(out, `"apiKey":"\[REDACTED\]"`) {
		t.Fatalf("expected JSON apiKey redaction, got %q", out)
	}
}

func regexpContains(s, pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.FindStringIndex(s) != nil
}
