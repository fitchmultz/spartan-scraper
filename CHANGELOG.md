# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Public-readiness audit script (`scripts/public_audit.mjs`) and test coverage.
- `make audit-public` target integrated into `make ci`.
- Documentation index (`docs/README.md`) for faster navigation.
- Repository hygiene metadata (`.editorconfig`, `.gitattributes`).
- Reviewer-focused docs: `docs/reviewer_checklist.md`, `docs/release_readiness.md`, and `docs/ci.md`.
- `make ci-pr` and `make ci-manual` targets plus profile support in `run_ci.sh`.
- GitHub Actions workflows for fast PR checks (`.github/workflows/ci-pr.yml`) and heavy nightly/manual checks (`.github/workflows/ci-slow.yml`).

### Changed

- Rewrote `docs/landscape.md` to remove local-path/private inventory leakage.
- Updated contributing and policy docs for public issue/security workflows.
- Expanded `.gitignore` to block env files, local caches, logs, and build artifacts.
- `make build` now builds artifacts only; `make install-bin` handles binary install side effects explicitly.
- `docs/architecture.md` now starts with a concise reviewer-oriented overview section.
- History reset to a sanitized public baseline and force-updated `main` to remove legacy private artifacts from branch history.
- Removed tracked local runtime artifacts (`.ralph/*`, `out/smoke_test/*`) from the repository tree.

## [0.1.0] - 2026-03-04

### Added

- Initial public baseline version marker.
