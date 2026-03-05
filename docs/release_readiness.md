# Release Readiness Report

This is a living report for public-release hardening.

## Scope

Goal: ensure the repository is reviewable with minimal friction by engineers and hiring managers.

## Top risks and mitigations

1. **Public artifact leakage (`.ralph`, `out`, `.data`, `node_modules`, `dist`)**
   - Mitigation: hardened `.gitignore` and `make audit-public` checks for tracked artifacts.

2. **Local path / placeholder text leaks in docs**
   - Mitigation: `scripts/public_audit.mjs` scans docs and metadata for absolute paths and placeholder contacts.

3. **History hygiene blind spots on release branches**
   - Mitigation: public audit now scans current branch history for blocked artifact paths and targeted historical content leakage (`docs/landscape.md`).

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
3. If publishing from a previously private history, maintainers should confirm remote branch history was sanitized per release policy.

## Release checklist

- [ ] `make audit-public`
- [ ] `make ci-pr` on a clean working tree
- [ ] reviewer smoke path validated (`docs/reviewer_checklist.md`)
- [ ] docs links valid and up to date
- [ ] no sensitive values in tracked config files
- [ ] release notes updated in `CHANGELOG.md`
