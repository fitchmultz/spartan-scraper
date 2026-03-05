# Releasing Spartan Scraper

This document describes the local release process for Spartan Scraper.

## Release Policy

- Versioning follows Semantic Versioning (`MAJOR.MINOR.PATCH`).
- `main` is the only supported branch until post-`1.0.0` support policy is defined.
- `CHANGELOG.md` must be updated before tagging a release.

## Prerequisites

- Toolchain installed per `.tool-versions`.
- Clean working tree (`git status` shows no unstaged/uncommitted changes).
- Local CI is green (`make ci`).

## Release Steps

1. **Prepare release notes**
   - Update `CHANGELOG.md` under `[Unreleased]`.
   - Create a dated/versioned section (e.g. `[0.2.0] - 2026-04-01`).

2. **Run full local verification**

   ```bash
   make ci
   make ci-slow
   ```

3. **Create release commit**

   ```bash
   git add CHANGELOG.md
   git commit -m "release: vX.Y.Z"
   ```

4. **Tag release**

   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   ```

5. **Push commit and tag**

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
- [ ] `make ci` passes
- [ ] `CHANGELOG.md` updated
- [ ] tag created and pushed
- [ ] release notes drafted from changelog entries
