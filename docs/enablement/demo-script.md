# Demo Script (5–10 minutes)

## Goal

Prove public-release readiness with deterministic checks and core-flow smoke tests.

## Steps

1. **Clone + gate**
   ```bash
   git clone <repo-url>
   cd spartan-scraper
   make ci-pr
   ```
   Expected: exits `0`; clean-tree checks pass; no warnings in test output.

2. **CLI smoke**
   ```bash
   ./bin/spartan --help
   ./bin/spartan version
   ```

3. **Server health + auth behavior**
   ```bash
   ./bin/spartan server
   # in another terminal
   curl -sS http://127.0.0.1:8741/healthz
   ```
   Optional: restart with `BIND_ADDR=0.0.0.0` and verify non-health routes require `X-API-Key`.

4. **WebSocket origin protection**
   ```bash
   curl -i -H "Origin: https://example.com" http://127.0.0.1:8741/v1/ws
   ```
   Expected: `403 Forbidden` + `forbidden websocket origin`.

5. **Web UI smoke**
   ```bash
   make web-dev
   ```
   Open `http://localhost:5173` and verify app loads and API-backed sections render.

## Troubleshooting

- If `make ci-pr` fails clean-tree checks, run `git status` and inspect generated drift.
- If web dev cannot reach API, verify server is running on `127.0.0.1:8741`.
