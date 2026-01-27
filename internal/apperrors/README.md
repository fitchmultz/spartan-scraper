# apperrors

Package `apperrors` provides structured error handling with classification, context wrapping, and secret redaction.

## Design Principles

- **Classification**: Errors are classified into `Kind` (validation, not_found, permission, internal) for consistent handling
- **Context vs Debug**: Error messages in user-facing APIs use safe messages; underlying errors preserve full debug context via `Unwrap()`
- **Secret Safety**: `RedactString` and `SafeMessage` automatically redact common secret patterns (Authorization headers, API keys, passwords)
- **Error Is/As**: Full support for `errors.Is` and `errors.As` with proper `Unwrap()` chains

## Error Kinds

| Kind           | HTTP Status | Use Case                                    |
|----------------|-------------|---------------------------------------------|
| `validation`   | 400         | Input validation failures (bad input)          |
| `not_found`    | 404         | Resource not found (job, schedule, etc.)      |
| `permission`    | 403         | Access denied / authorization failures          |
| `internal`      | 500         | System errors (infrastructure, unexpected)     |

## Usage Patterns

### 1. Create Classified Errors

```go
// Simple validation error
err := apperrors.Validation("url is required")

// Not found error
err := apperrors.NotFound("job not found")

// Internal system error
err := apperrors.Internal("job queue full")
```

### 2. Wrap Existing Errors

```go
// Wrap with safe public message, preserve original error for debugging
return apperrors.Wrap(apperrors.KindInternal, "failed to connect to database", connErr)

// Classify sentinel error without changing message
return apperrors.WithKind(apperrors.KindValidation, validate.ErrInvalidURLScheme)
```

### 3. Check Error Kinds

```go
// Check if error has a specific kind
if apperrors.IsKind(err, apperrors.KindValidation) {
    // Handle validation errors
}

// Get the kind (defaults to KindInternal)
switch apperrors.KindOf(err) {
case apperrors.KindValidation:
    // ...
case apperrors.KindNotFound:
    // ...
default:
    // ...
}
```

### 4. Sentinel Errors

Sentinel errors are pre-defined errors that can be compared with `errors.Is`:

```go
import "spartan-scraper/internal/apperrors"
import "spartan-scraper/internal/validate"

// Check for specific sentinel
if errors.Is(err, apperrors.ErrInvalidURLScheme) {
    // Handle specifically
}

// Re-export sentinel errors for external packages
var (
    ErrInvalidURLScheme = apperrors.ErrInvalidURLScheme
    ErrInvalidURLHost   = apperrors.ErrInvalidURLHost
)
```

## HTTP Handler Pattern

Use `writeError(w, err)` in HTTP handlers for consistent status code mapping:

```go
import (
    "net/http"
    "spartan-scraper/internal/apperrors"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // ... validate input ...
    if err != nil {
        // writeError automatically maps:
        // - validation → 400
        // - not_found → 404
        // - permission → 403
        // - internal → 500
        writeError(w, err)
        return
    }
    // ... success ...
}
```

## Secret Redaction

`SafeMessage` and `RedactString` automatically redact common secret patterns:

```go
err := fmt.Errorf(`Authorization: Bearer abc123 token=xyz {"apiKey":"shh"}`)
safeMsg := apperrors.SafeMessage(err)
// Output: Authorization: Bearer [REDACTED] token=[REDACTED] {"apiKey":"[REDACTED]"}

// Redact any string
redacted := apperrors.RedactString(`password=hunter2`)
// Output: password=[REDACTED]
```

Redaction patterns cover:
- Authorization headers: `Bearer <token>`, `Basic <creds>`
- Key-value secrets: `password=`, `token=`, `api_key=`, etc.
- JSON secrets: `"password":"..."`, `"apiKey":"..."`, etc.

## Common Conventions

1. **Validation errors**: Use `apperrors.Validation(msg)` for input failures
2. **Not found errors**: Use `apperrors.NotFound(msg)` when a resource doesn't exist
3. **Wrapping**: Use `apperrors.Wrap(kind, "safe message", err)` to add context without exposing secrets
4. **Sentinels**: Use `apperrors.WithKind(kind, sentinelErr)` for stable sentinel error text
5. **Never log secrets**: Always use `apperrors.SafeMessage(err)` when logging or returning to clients

## Examples

### API Handler Example

```go
func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    if id == "" {
        writeError(w, apperrors.Validation("id is required"))
        return
    }

    job, err := s.store.Get(r.Context(), id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            writeError(w, apperrors.NotFound("job not found"))
            return
        }
        writeError(w, apperrors.Wrap(apperrors.KindInternal, "failed to get job", err))
        return
    }

    writeJSON(w, job)
}
```

### Validation Function Example

```go
func ValidateURL(rawURL string) error {
    if rawURL == "" {
        return apperrors.Validation("url is required")
    }
    u, err := url.Parse(rawURL)
    if err != nil {
        return apperrors.WithKind(apperrors.KindValidation, fmt.Errorf("invalid url: %w", err))
    }
    if u.Scheme != "http" && u.Scheme != "https" {
        return apperrors.WithKind(apperrors.KindValidation, apperrors.ErrInvalidURLScheme)
    }
    return nil
}
```

### Job Manager Example

```go
func (m *Manager) Enqueue(job model.Job) error {
    select {
    case m.queue <- job:
        return nil
    default:
        // Use sentinel error for consistent detection
        return apperrors.ErrQueueFull
    }
}
```

## Error Message Guidelines

- **Include operation context**: `"failed to get job"` (not just `"error"`)
- **Avoid exposing secrets**: Never include raw tokens, passwords, or keys in messages
- **Keep messages user-facing**: Safe messages should be actionable, technical details go in wrapped error
- **Use consistent phrasing**: `"X is required"` for validation, `"failed to X"` for operations
