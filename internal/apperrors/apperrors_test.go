package apperrors

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
)

func TestKindOfAndIsKind(t *testing.T) {
	base := Validation("bad input")
	wrapped := fmt.Errorf("outer: %w", base)

	if got := KindOf(wrapped); got != KindValidation {
		t.Fatalf("KindOf() = %s, want %s", got, KindValidation)
	}
	if !IsKind(wrapped, KindValidation) {
		t.Fatalf("IsKind(validation) = false, want true")
	}
	if IsKind(wrapped, KindNotFound) {
		t.Fatalf("IsKind(not_found) = true, want false")
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
