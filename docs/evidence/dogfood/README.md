# Dogfood Evidence Bundles

This directory contains timestamped UI dogfood evidence packs used for release-readiness validation.

## Bundles

- [`2026-03-05-focused-ui`](./2026-03-05-focused-ui/)
  - Focused UI pass for onboarding, command palette, keyboard shortcuts, and API-outage behavior.
  - Includes:
    - `report.md` (issue findings with repro steps)
    - `remediation-validation.md` (post-fix status)
    - screenshots and short repro videos
    - truncated console/error logs for reviewer context

- [`2026-03-05-postfix-ui`](./2026-03-05-postfix-ui/)
  - Post-fix live re-dogfood validating all five previously reported regressions.
  - Includes:
    - `report.md` (pass/fail verification with linked artifacts)
    - check-specific snapshots/JSON logs
    - outage-isolated console/error captures
