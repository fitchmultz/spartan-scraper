package apperrors

import (
	"errors"
)

// Kind is a coarse-grained error classification used for consistent handling.
type Kind string

const (
	KindValidation           Kind = "validation"
	KindNotFound             Kind = "not_found"
	KindPermission           Kind = "permission"
	KindInternal             Kind = "internal"
	KindMethodNotAllowed     Kind = "method_not_allowed"
	KindUnsupportedMediaType Kind = "unsupported_media_type"
)

// Error wraps an underlying error with a Kind and a safe, user-facing message.
//
// Design note: If Msg is set, Error() returns Msg only (and does NOT append Err.Error()).
// This allows callers to keep rich underlying errors for debugging via Unwrap(),
// while preventing accidental leaks in user-facing surfaces.
type Error struct {
	Kind Kind
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Kind != "" {
		return string(e.Kind)
	}
	return "error"
}

func (e *Error) Unwrap() error { return e.Err }

// New creates a new classified error with a safe message.
func New(kind Kind, msg string) error {
	return &Error{Kind: kind, Msg: msg}
}

// Wrap classifies err as kind, while presenting msg as public-facing Error() string.
// The original err remains accessible via errors.Unwrap/errors.Is/errors.As.
func Wrap(kind Kind, msg string, err error) error {
	return &Error{Kind: kind, Msg: msg, Err: err}
}

// WithKind classifies err as kind without changing its message.
// This is useful for sentinel errors whose text should remain stable.
func WithKind(kind Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Err: err}
}

// KindOf returns the first apperrors.Kind found in err's unwrap chain.
// If none is present, KindInternal is returned as a safe default.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) && e.Kind != "" {
		return e.Kind
	}
	return KindInternal
}

// IsKind reports whether err is classified as kind anywhere in its unwrap chain.
func IsKind(err error, kind Kind) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == kind
}

func Validation(msg string) error           { return New(KindValidation, msg) }
func NotFound(msg string) error             { return New(KindNotFound, msg) }
func Permission(msg string) error           { return New(KindPermission, msg) }
func Internal(msg string) error             { return New(KindInternal, msg) }
func MethodNotAllowed(msg string) error     { return New(KindMethodNotAllowed, msg) }
func UnsupportedMediaType(msg string) error { return New(KindUnsupportedMediaType, msg) }
