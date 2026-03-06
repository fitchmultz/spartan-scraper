# Adoption Map

This map links real project artifacts to outcomes that matter in deployment, enablement, and operational adoption.

Use it as supporting material after you understand the product itself. It is not the primary entrypoint for the repo.

## Reliability and correctness

- Deterministic PR gate: `make ci-pr` (`Makefile`, `.github/workflows/ci-pr.yml`)
- Full local gate: `make ci` (`Makefile`)
- High-signal tests: Go unit/integration + web Vitest (`internal/**`, `web/src/**/*.test.ts*`)

## Security and public-readiness

- Public hygiene scanner: `make audit-public` (`scripts/public_audit.mjs`)
- Secret/signature checks + history artifact checks: `scripts/public_audit.mjs`
- WebSocket origin hardening: `internal/api/server.go`, `internal/api/server_websocket_origin_test.go`
- Safe defaults docs: `README.md`, `.env.example`, `docs/usage.md`, `SECURITY.md`

## Developer productivity and onboarding

- One-command pipelines: `make ci`, `make ci-pr`, `make ci-slow`, `make ci-network` (`make ci-slow` provisions Playwright for clean-machine heavy verification)
- Lockfile-strict installs for deterministic setup: `Makefile`
- Validation runbook: `docs/validation_checklist.md`
- Release runbook: `RELEASING.md`, `docs/validation_checklist.md`

## Operational discipline

- Split CI profiles by cost/risk: `docs/ci.md`, workflows in `.github/workflows/`
- Deterministic heavy checks separated from PR-required checks, with optional live smoke kept manual: `ci-slow.yml`, `Makefile`
- Runtime safety controls (timeouts/auth): `internal/config/config.go`, `internal/cli/server/server.go`, `internal/api/middleware.go`
