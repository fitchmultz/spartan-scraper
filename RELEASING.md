# Releasing Spartan Scraper

This document describes the local release process for Spartan Scraper.

## Release Policy

- Versioning follows Semantic Versioning (`MAJOR.MINOR.PATCH`).
- `main` is the only supported branch until post-`1.0.0` support policy is defined.
- `CHANGELOG.md` must be updated before tagging a release.

## Prerequisites

- Toolchain installed per `.tool-versions`.
- Clean working tree (`git status` shows no unstaged/uncommitted changes).
- Local CI is green (`make ci-pr`, `make ci`, and `make ci-slow`).
- `docs/validation_checklist.md` has been walked on a fresh data directory.

## Release Steps

1. **Prepare release notes**
   - Update `CHANGELOG.md` under `[Unreleased]`.
   - Create a dated/versioned section (e.g. `[0.2.0] - 2026-04-01`).

2. **Run full local verification**

   ```bash
   make ci-pr
   make ci
   make ci-slow          # provisions Playwright for clean-machine heavy validation
   ```

   Run `make ci-network` separately only if you want an optional live-Internet smoke pass before a public tag.

3. **Run deep history secret scan (manual release-tier check)**

   ```bash
   make secret-scan
   ```

4. **Finalize release metadata**

   - Set `VERSION`-bearing files to the release target (for example `1.0.0-rc1` or `1.0.0`).
   - Update `CHANGELOG.md`.
   - Update `README.md` project status if the release changes support posture.
   - Review `SECURITY.md` supported-version language.

5. **Create release commit**

   ```bash
   git add CHANGELOG.md README.md SECURITY.md RELEASING.md Makefile api/openapi.yaml internal/buildinfo/version.go web/package.json
   git commit -m "release: vX.Y.Z"
   ```

6. **Tag release**

   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   ```

7. **Push commit and tag**

   ```bash
   git push origin main
   git push origin vX.Y.Z
   ```

## Build Metadata

The Go binary build metadata is injected by `Makefile` via ldflags:

- `internal/buildinfo.Version`
- `internal/buildinfo.Commit`
- `internal/buildinfo.Date`

Set `VERSION` explicitly when needed:

```bash
make build VERSION=vX.Y.Z
```

## Verification Checklist

- [ ] `make audit-public` passes
- [ ] `make ci-pr` passes
- [ ] `make ci` passes
- [ ] `make ci-slow` passes
- [ ] `make secret-scan` run and reviewed
- [ ] `docs/validation_checklist.md` smoke pass completed
- [ ] `CHANGELOG.md` updated
- [ ] tag created and pushed
- [ ] release notes drafted from changelog entries
