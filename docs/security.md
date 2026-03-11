# Security Model

## Supported Boundary

- Single trusted operator deployment only.
- No multi-user isolation guarantees.
- No distributed trust boundary.

## Authentication

- Loopback usage can run without API keys.
- Non-loopback binds must use API keys.
- Target-site credentials belong in the auth vault, not in job specs checked into source control.

## Redaction

- Job errors and exposed job specs are redacted before API and MCP responses.
- Filesystem paths are removed from public-facing job views.
