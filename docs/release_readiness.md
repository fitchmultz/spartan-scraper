# Release Readiness Report

Date: 2026-03-05

This report captures current public-release hardening status for Spartan Scraper.

## Current State

- **Stack:** Go 1.25.6 backend/CLI + React/TypeScript web UI (Vite, Vitest, Biome)
- **Primary entrypoints:** `./bin/spartan server`, `make web-dev`, `make ci-pr`
- **Deterministic gate:** `make ci-pr`
- **Heavy confidence gate:** `make ci-slow` (nightly/manual only)
- **Public hygiene gate:** `make audit-public`

## Top 10 Risks and Mitigations

1. **Tracked local artifact leakage (`.ralph`, `.data`, `out`, `node_modules`, `dist`)**
   - Mitigation: hardened ignore rules + automated path checks in `scripts/public_audit.mjs`.

2. **Secret exposure in tracked files**
   - Mitigation: `make audit-public` now scans source/config/docs for high-confidence secret patterns (OpenAI, GitHub, AWS, Slack, private key blocks).

3. **Absolute-path and placeholder leaks in public docs**
   - Mitigation: automated content scans in `scripts/public_audit.mjs` across docs/metadata.

4. **History hygiene blind spots before publish**
   - Mitigation: branch-history checks in `make audit-public` + documented sanitized baseline reset on March 5, 2026.

5. **PR gate drift between local and CI environments**
   - Mitigation: `make ci-pr` is the required PR-equivalent pipeline locally and in `.github/workflows/ci-pr.yml`.

6. **Resource-heavy checks saturating PR runs**
   - Mitigation: heavy checks isolated to `.github/workflows/ci-slow.yml` (nightly/manual) and Vitest workers capped with `CI_VITEST_MAX_WORKERS=2`.

7. **Onboarding completion failure in UI**
   - Mitigation: onboarding step-count source of truth + Joyride callback hardening (`web/src/lib/onboarding.ts`, `web/src/components/OnboardingFlow.tsx`) with regression tests.

8. **Keyboard help shortcut inconsistency (`?`)**
   - Mitigation: shortcut matcher normalization fix in `web/src/hooks/useKeyboard.ts` + targeted tests.

9. **Command palette focus not reliably capturing typing**
   - Mitigation: explicit focus handoff to command input on open in `web/src/components/CommandPalette.tsx`.

10. **WebSocket callback instability under API outage**
    - Mitigation: callback-ref stabilization + disconnect-path cleanup in `web/src/hooks/useWebSocket.ts`, covered by regression tests.

## Remaining Known Issues and Next Steps

1. **`ci-slow` variance in network/e2e environments**
   - Next step: keep nightly/manual-only policy; track failure patterns in CI artifacts before tightening as PR-required.

2. **Dogfood evidence bundle size growth over time**
   - Next step: keep evidence logs truncated and prefer screenshot-first bundles; store long raw traces externally when needed.

3. **Post-fix live UI verification artifacts**
   - Next step: append a short post-remediation browser pass to `docs/evidence/dogfood/2026-03-05-focused-ui/`.

## Before/After DX Notes

### Before
- Reviewer-critical UI paths had reproducible failures (onboarding completion, `?` help shortcut, command-palette input focus, WS instability under outage).
- Public audit focused on path/content hygiene but not explicit secret signatures.

### After
- UI regressions above are fixed with dedicated tests:
  - `web/src/components/OnboardingFlow.test.tsx`
  - `web/src/hooks/useOnboarding.test.ts`
  - `web/src/hooks/useKeyboard.test.ts`
  - `web/src/hooks/useWebSocket.test.ts`
- Public-readiness gate now includes high-confidence secret detection via `make audit-public`.
- Local deterministic pipeline remains green with `make ci` and `make ci-pr`.

## Validation Snapshot

- `make audit-public` ✅
- `make ci` ✅
- Web tests: 18 files / 231 tests passing ✅

## Reviewer Validation Path

Use `docs/reviewer_checklist.md` for copy/paste verification of setup, run, CI-equivalent checks, and public-readiness checks.
