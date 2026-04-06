# CI Strategy (Local-First)

This repository uses **local CI as the source of truth** through Makefile targets.

`.tool-versions` is the authoritative Go/Node/pnpm contract for both local runs and GitHub workflows.

## Why this structure

- Keep PR checks deterministic and resource-aware.
- Keep expensive live-network checks opt-in while preserving a deterministic heavy lane.
- Make local and CI-equivalent commands identical.

## CI profiles

### GitHub workflow mapping

- **PR required**: `.github/workflows/ci-pr.yml` → `make ci-pr`
- **Nightly/manual heavy**: `.github/workflows/ci-slow.yml` → `make ci-slow` (deterministic local fixture with Playwright provisioning)

### PR-equivalent gate (`make ci-pr`)

Use from a **clean git state**.

```bash
make verify-toolchain
make ci-pr
```

Runs:

```text
verify-clean-tree
→ verify-toolchain
→ audit-public
→ install
→ generate
→ format
→ type-check
→ lint
→ build
→ test-ci
→ verify-clean-tree
```

Use this to ensure generation/format steps do not create uncommitted drift.
`install` uses lockfile-strict dependency resolution (`pnpm install --frozen-lockfile`) for deterministic results across both JavaScript packages (`web` and `tools/pi-bridge`).

### Full local gate (`make ci`)

```bash
make verify-toolchain
make ci
```

Runs the full deterministic pipeline but does **not** enforce clean-tree checks before/after.
Useful during active development before commit.

Go transitive overrides are **not** proactively audited in CI. If a root `go.mod` `replace` is ever needed, it should be a rare, temporary response to a high-severity security or correctness emergency rather than routine dependency-freshness maintenance.

### Heavy checks (`make ci-slow`)

```bash
make ci-slow
```

Runs deterministic heavy checks against the shared local fixture. Before the heavy lane runs, `make ci-slow` installs Playwright browsers via `make install-playwright` so clean machines and GitHub runners do not depend on a pre-warmed browser cache.

- `./scripts/stress_test.sh`
- `go test -v ./internal/e2e/...`

### Optional live-network smoke (`make ci-network`)

```bash
make ci-network
```

Runs the stress profile against live Internet targets. Keep this manual/optional because third-party uptime is outside the repo's control.

### Manual full sweep (`make ci-manual`)

```bash
make ci-manual
```

Runs both `make ci-slow` and `make ci-network`.

### Manual release-tier secret scan (`make secret-scan`)

```bash
make secret-scan
```

Runs a pinned Gitleaks history scan (`--log-opts="--all"`) with the reviewed false-positive baseline from `.gitleaksignore`.
Keep this out of PR-required checks to preserve deterministic runtime budgets; run it before releases and in manual sweeps.

## Runtime and resource guidance

These timings are practical targets on GitHub-hosted `ubuntu-latest` runners and modern local laptops:

- `make ci-pr`: ~8-15 minutes (deterministic, resource-capped)
- `make ci`: ~8-15 minutes (same checks without clean-tree assertions)
- `make ci-slow`: ~10-20 minutes (deterministic heavy lane, including Playwright provisioning on clean machines)
- `make ci-network`: variable (live-network dependency)
- `make secret-scan`: ~1-5 minutes (repo history size dependent)

Resource controls:

- Vitest worker count is capped with `CI_VITEST_MAX_WORKERS` (default `2` in `Makefile`).
- PR workflow uses `concurrency` cancellation to avoid redundant parallel runs.
- Heavy checks are isolated to nightly/manual workflow to avoid saturating PR runners.

Local expectations stay similar:

- `make ci-pr`: medium (installs + build + tests)
- `make ci`: medium (same pipeline, no clean-tree assertions)
- `make ci-slow`: medium-high (deterministic heavy lane; highest on first run when Playwright is provisioned)
- `make ci-network`: variable/high (live-network smoke)
- `make secret-scan`: medium (full-history scan)

Recommended usage:

- **PR / merge readiness**: `make ci-pr`
- **Inner dev loop**: targeted commands (`make test-ci`, `make lint`, `make type-check`) and periodic `make ci`
- **Nightly confidence sweep**: `make ci-slow`
- **Optional live smoke**: `make ci-network`
- **Manual pre-release sweep**: `make ci-manual` + `make secret-scan`

## Convenience wrapper

Use `run_ci.sh` when you want profile-based invocation:

```bash
./run_ci.sh --profile pr
./run_ci.sh --profile full
./run_ci.sh --profile slow
./run_ci.sh --profile network
./run_ci.sh --profile manual
```

Run `make secret-scan` directly when you need the release-tier history scan; `run_ci.sh` does not provide a dedicated secret-scan profile.
