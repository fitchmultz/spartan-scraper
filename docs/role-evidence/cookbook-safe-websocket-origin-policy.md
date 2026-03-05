# Cookbook: Safe Localhost WebSocket Origin Policy

## Problem

Localhost APIs with browser-accessible WebSockets are vulnerable to cross-site origin abuse if origin is not validated.

## Pattern

For `/v1/ws` upgrades:

- Allow requests with **no `Origin`** header (non-browser clients).
- If `Origin` is present, parse URL and allow only loopback hostnames:
  - `localhost`
  - `127.0.0.1` (and `127.*`)
  - `::1`
- Reject all other browser origins with `403`.

## Why this default

- Strongly reduces local cross-site risk.
- Preserves CLI/tooling use-cases.
- Keeps behavior explicit and testable.

## Implementation reference

- Logic: `internal/api/server.go` (`isAllowedWebSocketOrigin`, `handleWebSocket`)
- Tests: `internal/api/server_websocket_origin_test.go`

## Trade-offs

- Remote/browser deployments need explicit auth+proxy strategy.
- This pattern is intentionally conservative for local-first workflows.
