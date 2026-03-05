# CI Strategy (Local-First)

This repository uses **local CI as the source of truth** through Makefile targets.

## Why this structure

- Keep PR checks deterministic and resource-aware.
- Keep expensive network/e2e checks opt-in or scheduled.
- Make local and CI-equivalent commands identical.

## CI profiles

### GitHub workflow mapping

- **PR required**: `.github/workflows/ci-pr.yml` → `make ci-pr`
- **Nightly/manual heavy**: `.github/workflows/ci-slow.yml` → `make ci-slow`

### PR-equivalent gate (`make ci-pr`)

Use from a **clean git state**.

```bash
make ci-pr
```

Runs:

```text
verify-clean-tree
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

### Full local gate (`make ci`)

```bash
make ci
```

Runs the full deterministic pipeline but does **not** enforce clean-tree checks before/after.
Useful during active development before commit.

### Heavy checks (`make ci-slow`)

```bash
make ci-slow
```

Runs stress + e2e checks that may require network and more CPU/RAM:

- `./scripts/stress_test.sh`
- `go test -v ./internal/e2e/...`

### Manual alias (`make ci-manual`)

```bash
make ci-manual
```

Alias for `make ci-slow` to make intent explicit in scripts.

## Runtime and resource guidance

These timings are practical targets on GitHub-hosted `ubuntu-latest` runners and modern local laptops:

- `make ci-pr`: ~8-15 minutes (deterministic, resource-capped)
- `make ci`: ~8-15 minutes (same checks without clean-tree assertions)
- `make ci-slow`: ~20-60+ minutes (network/e2e/stress variability)

Resource controls:

- Vitest worker count is capped with `CI_VITEST_MAX_WORKERS` (default `2` in `Makefile`).
- PR workflow uses `concurrency` cancellation to avoid redundant parallel runs.
- Heavy checks are isolated to nightly/manual workflow to avoid saturating PR runners.

Local expectations stay similar:

- `make ci-pr`: medium (installs + build + tests)
- `make ci`: medium (same pipeline, no clean-tree assertions)
- `make ci-slow`: high (network/e2e/stress)

Recommended usage:

- **PR / merge readiness**: `make ci-pr`
- **Inner dev loop**: targeted commands (`make test-ci`, `make lint`, `make type-check`) and periodic `make ci`
- **Nightly/manual confidence sweep**: `make ci-slow`

## Convenience wrapper

Use `run_ci.sh` when you want profile-based invocation:

```bash
./run_ci.sh --profile pr
./run_ci.sh --profile full
./run_ci.sh --profile slow
```
