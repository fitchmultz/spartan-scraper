# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

## [1.0.0-rc1] - 2026-03-11

### Added

- Public-readiness audit script (`scripts/public_audit.mjs`) and test coverage.
- `make audit-public` target integrated into `make ci`.
- Documentation index (`docs/README.md`) for faster navigation.
- Repository hygiene metadata (`.editorconfig`, `.gitattributes`).
- Validation and release docs: `docs/validation_checklist.md`, `RELEASING.md`, and `docs/ci.md`.
- `make ci-pr` and `make ci-manual` targets plus profile support in `run_ci.sh`.
- GitHub Actions workflows for fast PR checks (`.github/workflows/ci-pr.yml`) and heavy nightly/manual checks (`.github/workflows/ci-slow.yml`).
- Per-job artifact manifests with spec hash, file inventory, checksums, and export records.
- Deterministic PR-safe system tests in `internal/system` covering REST jobs, schedules, WebSocket lifecycle events, exports, manifests, and MCP.
- Operator-facing 1.0 docs for architecture, API, MCP, security, support matrix, and validation.

### Changed

- Narrowed the supported 1.0 product surface to the single-node local-first core: scrape, crawl, research, schedules, batches, chains, watches, webhooks, REST, WebSocket, CLI, TUI, Web UI, MCP, retention, backup/restore, and core-five exporters.
- Removed GraphQL, plugins, distributed mode, workspaces, extension, replay, template A/B metrics, and exotic exporters from the supported product boundary.
- Converted persisted jobs to typed `specVersion` + `spec_json` contracts and converted recurring schedules to typed `specVersion` + `spec` contracts with hard rejection of legacy schedule `params`.
- Reworked the web shell around the retained route set and regenerated the OpenAPI TypeScript client around the reduced 1.0 API.
- Tightened CI so `make ci-pr` exercises the retained 1.0 core through deterministic local system flows while `make ci-slow` keeps the heavy browser and stress lane.
- Rewrote `docs/landscape.md` to remove local-path/private inventory leakage.
- Updated contributing and policy docs for public issue/security workflows.
- Expanded `.gitignore` to block env files, local caches, logs, and build artifacts.
- `make build` now builds artifacts only; `make install-bin` handles binary install side effects explicitly.
- `docs/architecture.md` now starts with a concise quick overview section.
- History reset to a sanitized public baseline and force-updated `main` to remove legacy private artifacts from branch history.
- Removed tracked local runtime artifacts (`.ralph/*`, `out/smoke_test/*`) from the repository tree.

## [0.1.0] - 2026-03-04

### Added

- Initial public baseline version marker.
