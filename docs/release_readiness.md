# Release Readiness Report

This report captures the current public-release hardening state.

## Scope

Goal: ensure the repository is reviewable with minimal friction by engineers and hiring managers.

## Top risks and mitigations

1. **Public artifact leakage (`.ralph`, `out`, `.data`, `node_modules`, `dist`)**
   - Mitigation: removed tracked local runtime artifacts, hardened `.gitignore`, and enforced `make audit-public` checks.

2. **Local path / placeholder text leaks in docs**
   - Mitigation: `scripts/public_audit.mjs` scans docs and metadata for absolute paths and placeholder contacts.

3. **History hygiene blind spots on release branches**
   - Mitigation: public audit scans branch history, and `main` history was reset on **March 5, 2026** to a sanitized public baseline before publish.

4. **Inconsistent issue/PR quality from contributors**
   - Mitigation: added issue templates and PR checklist under `.github/`.

5. **Security reporting ambiguity**
   - Mitigation: standardized reporting channels in `SECURITY.md`.

6. **Code of conduct enforcement ambiguity**
   - Mitigation: explicit enforcement contact in `CODE_OF_CONDUCT.md`.

7. **CI profile ambiguity (fast vs heavy checks)**
   - Mitigation: explicit CI profiles in Makefile (`ci-pr`, `ci`, `ci-slow`, `ci-manual`) and profile runner support in `run_ci.sh`.

8. **Generate/format drift sneaking through PR validation**
   - Mitigation: `ci-pr` enforces clean-tree assertions before and after pipeline.

9. **Reviewer onboarding friction (web UI dependency ordering)**
   - Mitigation: README and reviewer checklist now explicitly require backend server before launching `web-dev`.

10. **Documentation discoverability gaps**
    - Mitigation: added dedicated reviewer checklist and CI strategy docs, linked from README/docs index.

## Current state snapshot

- Core gate: `make ci-pr` (clean-state deterministic gate)
- Full local gate: `make ci`
- Heavy checks: `make ci-slow`
- Public hygiene gate: `make audit-public`

## Remaining known issues

1. `ci-slow` is intentionally network/e2e heavy and can be flaky outside controlled environments.
2. Browser-driven Playwright workflows remain opt-in because of install/runtime cost.

## Release checklist

- [x] `make audit-public`
- [x] `make ci-pr` on a clean working tree
- [ ] reviewer smoke path validated (`docs/reviewer_checklist.md`)
- [x] docs links valid and up to date
- [x] no sensitive values in tracked config files
- [x] release notes updated in `CHANGELOG.md`
