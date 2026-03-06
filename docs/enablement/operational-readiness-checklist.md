# Operational Readiness Checklist

## Config and secrets

- [ ] `.env` is local-only and never committed
- [ ] `.env.example` matches current config surface
- [ ] `make audit-public` passes before release

## Build and verification

- [ ] `make ci-pr` passes from a clean tree
- [ ] `make ci` passes locally
- [ ] `make ci-slow` run for deterministic release confidence on a clean machine (Playwright is provisioned by the target)
- [ ] `make ci-network` run if live-Internet smoke validation is desired before release

## Runtime safety

- [ ] API bind policy validated (`BIND_ADDR` + API auth behavior)
- [ ] WebSocket origin policy validated (`/v1/ws` rejects non-loopback browser origins)
- [ ] Request timeout and server timeout defaults reviewed

## Release operations

- [ ] `CHANGELOG.md` updated
- [ ] `RELEASING.md` checklist completed
- [ ] Deep history secret scan run (manual release-tier)
- [ ] Validation checklist executed (`docs/validation_checklist.md`)

## Rollout/rollback notes

- Rollout: tag + push using `RELEASING.md`
- Rollback: revert release commit/tag and republish patch with corrected artifacts
