# Release Readiness Report

Date: 2026-03-05

This report captures the current public-release hardening status for Spartan Scraper.

## Current State

- **Stack:** Go 1.25.6 backend/CLI + React/TypeScript web UI (Vite, Vitest, Biome)
- **Primary entrypoints:** `./bin/spartan server`, `make web-dev`, `make ci-pr`
- **Deterministic PR-equivalent gate:** `make ci-pr`
- **Heavy confidence gate:** `make ci-slow` (nightly/manual deterministic heavy lane with Playwright provisioning)
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

## UI Public-Readiness Remediation (2026-03-05)

P0 issue fixed in this pass:

- **Batch UI crash after submit** (`Cannot read properties of undefined ...`) is fixed.
- Root cause: generated API client returned serialized JSON strings for batch endpoints; hook code assumed object payloads.
- Remediation: `web/src/hooks/useBatches.ts` now parses serialized payloads, validates required batch fields, normalizes kind/status/stats/jobs, and rejects malformed payloads safely.
- Regression tests added: `web/src/hooks/useBatches.test.ts` now covers serialized payload parsing and missing-ID rejection.

Live browser verification receipts (agent-browser):

- Verified on 2026-03-05 with `agent-browser` against the live app at `http://localhost:5173`.
- Covered batch scrape, batch crawl, and batch research submit flows.
- Result: each flow completed without a page runtime error and the UI remained interactive after the batch row rendered.

## CI Matrix

PR required (deterministic, resource-capped):

- `make ci-pr`
  - clean-tree guard
  - audit-public
  - lockfile-strict install
  - generate + format + type-check + lint + build
  - `test-ci` with capped vitest workers

Nightly/manual heavy:

- `make ci-slow` (deterministic stress + e2e via local fixture, provisioning Playwright on clean machines)

Optional live-network smoke:

- `make ci-network`

Manual release-tier:

- `make secret-scan` (full-history secret scan)

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

1. **`ci-network` depends on third-party uptime**
   - Keep it optional/manual; do not treat live-network failures as deterministic repo regressions without reproduction.

2. **Deep git-history secret scan is still a release-tier/manual concern**
   - Keep `make audit-public` as fast deterministic gate.
   - Run `make secret-scan` in pre-tag/manual release workflow.

## Validation Snapshot (latest pass)

- `make ci` ✅ (2026-03-05)
- browser-driven batch regression verification ✅
  - batch scrape/crawl/research submit flows verified in the live UI with `agent-browser`

Last-known additional checks (run outside this focused UI fix pass):

- `make audit-public` ✅
- `make secret-scan` ✅
- `make ci-pr` ✅
- `make ci-slow` ✅
- GitHub-hosted `ci-slow` now provisions Playwright instead of assuming a machine-local browser install ✅
- `make ci-network` optional/manual

## Reviewer Validation Path

Use `docs/reviewer_checklist.md` for copy/paste validation of setup, CI-equivalent checks, API/WS safety, and public-hygiene checks.
