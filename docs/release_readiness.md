# Release Readiness Report

Date: 2026-03-05

This report captures the current public-release hardening status for Spartan Scraper.

## Current State

- **Stack:** Go 1.25.6 backend/CLI + React/TypeScript web UI (Vite, Vitest, Biome)
- **Primary entrypoints:** `./bin/spartan server`, `make web-dev`, `make ci-pr`
- **Deterministic PR-equivalent gate:** `make ci-pr`
- **Heavy confidence gate:** `make ci-slow` (nightly/manual only)
- **Public hygiene gate:** `make audit-public`
- **Deep history secret gate (manual release-tier):** `make secret-scan`

## P0 Hardening Completed in This Pass

1. **Audit gate now catches tracked binary artifacts (`bin/`)**
   - Updated `scripts/public_audit.mjs` tracked-path and branch-history path rules.
   - Added regression tests in `scripts/public_audit.test.mjs`.

2. **WebSocket origin safety for browser clients**
   - Added loopback-origin enforcement for `/v1/ws` in `internal/api/server.go`.
   - Policy: browser origins must be loopback (`localhost`, `127.0.0.1`, `::1`); non-browser clients without `Origin` remain supported.
   - Added route/security tests in `internal/api/server_websocket_origin_test.go`.

3. **Deterministic installs by default**
   - `Makefile` now uses `pnpm install --frozen-lockfile` for `install` and `extension-install`.
   - `test-ci` now runs web tests with `NODE_OPTIONS=` to avoid inherited shell-level noise in CI output.

4. **Documentation alignment**
   - Updated `README.md`, `.env.example`, and `docs/usage.md` with explicit API auth auto-enforcement and WebSocket origin behavior.
   - Updated `docs/reviewer_checklist.md` with explicit WebSocket origin validation steps.

5. **First-run warning cleanup**
   - `internal/config` now avoids retention warnings on untouched defaults (warns only when retention limits are explicitly overridden while retention is disabled).
   - `.env.example` no longer uses inline comment values that can be parsed as invalid AI provider strings.

## Top Risks and Mitigations

1. **Tracked local/build/cache artifacts leaking into commits**
   - Mitigation: hardened `.gitignore` + `make audit-public` path checks (including `bin/`).

2. **Secret exposure in tracked source/config/docs**
   - Mitigation: secret signature scanning in `scripts/public_audit.mjs`.

3. **History secret hygiene blind spots**
   - Mitigation: `make secret-scan` (pinned Gitleaks full-history scan with reviewed `.gitleaksignore` baseline) plus branch-history artifact checks in `make audit-public`.

4. **Cross-site localhost WebSocket abuse**
   - Mitigation: `/v1/ws` now rejects non-loopback browser origins with `403`.

5. **PR gate nondeterminism from mutable dependency installs**
   - Mitigation: lockfile-strict install (`pnpm --frozen-lockfile`) in Make targets used by CI.

## Remaining Known Issues / Next Steps

1. **`ci-slow` network/e2e variance**
   - Keep nightly/manual-only policy; use artifacts to track flaky signatures before promoting checks.

2. **Deep git-history secret scan is still a release-tier/manual concern**
   - Keep `make audit-public` as fast deterministic gate.
   - Run `make secret-scan` in pre-tag/manual release workflow.

## Validation Snapshot

- `make audit-public` ✅
- `make secret-scan` ✅
- `make ci-pr` ✅
- `make ci` ✅
- `make ci-slow` ✅

## Reviewer Validation Path

Use `docs/reviewer_checklist.md` for copy/paste validation of setup, CI-equivalent checks, API/WS safety, and public-hygiene checks.
