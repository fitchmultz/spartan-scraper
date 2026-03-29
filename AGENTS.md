# Repository Guidelines

## Project Overview

- **Primary language**: Go (CLI + API + TUI)
- **Frontend**: TypeScript (Vite + React)
- **Local CI gate**: `make ci` — must pass before completing work or committing
- **API contract**: `api/openapi.yaml` → generates TS client via `make generate` (hey-api openapi-ts)

## Development Workflow

### Local CI Gate

`make ci` runs: `audit-public → install → generate → format → type-check → lint → build → test-ci`
`make ci-pr` runs the same pipeline with `verify-clean-tree` before and after (clean git state required).
GitHub Actions split mirrors this: PR-required checks in `.github/workflows/ci-pr.yml`; heavy nightly/manual checks in `.github/workflows/ci-slow.yml`, and `make ci-slow` now provisions Playwright so clean machines/runners exercise the same auth/browser path.

**CRITICAL**: Never end a turn with a failing `make ci`. If `make ci` fails, fix all failures before completing your work.

### Build, Test, and Development Commands

```bash
make audit-public     # Scan tracked files + branch history for public-readiness leaks/placeholders
make secret-scan      # Deep git-history secret scan (manual/nightly release-tier)
make install          # Download Go deps + install pnpm deps (lockfile-strict)
make update           # Update all Go/pnpm deps to latest (review before committing)
make generate         # Generate TS API client from openapi.yaml
make format           # Format Go (gofmt) and TS (biome)
make type-check       # Type-check web TS
make lint             # Lint Go (go vet) and TS (biome)
make build            # Build Go binary + web assets (no install side effects)
make install-bin      # Install built binary to ~/.local/bin (or $XDG_BIN_HOME)
make test             # Run Go tests (including e2e)
make test-ci          # Run Go tests (excluding e2e but including PR-safe internal/system coverage) + web tests (Vitest capped by CI_VITEST_MAX_WORKERS, localstorage path pinned for warning-free Node runs)
make ci-pr            # PR-equivalent deterministic gate (requires clean git state)
make ci               # Full CI pipeline: audit-public, install, generate, format, type-check, lint, build, test-ci
make ci-slow          # Deterministic heavy stress + e2e checks (local fixture; provisions Playwright)
make ci-network       # Optional live-Internet smoke validation
make ci-manual        # Manual full heavy profile (ci-slow + ci-network)
CI_VITEST_MAX_WORKERS=2 make ci-pr  # Optional local worker cap override
make clean            # Remove build artifacts, dependencies, node_modules, installed binary
make web-dev          # Start web dev server (http://localhost:5173)
```

### Automation Cleanup Hygiene

- If an agent starts any repo-owned runtime resource for this project—a local server, browser/desktop automation (Playwright, agent-browser, Peekaboo, XCTest, Chrome/Chromium sessions, or similar), temp browser profile, temp workspace, simulator/device run, worker, or other background helper—it must stop it and remove the related ephemeral artifacts before ending its turn, and cleanup must still happen on success, failure, and interruption.
- Cleanup is not documentation-only: enforce teardown in the relevant repo scripts/tests/tooling where practical with `trap`, `defer`, `afterEach`, `t.Cleanup`, or equivalent failure-safe hooks.
- Before ending a turn, verify no agent-started resources remain for this repo: dev servers, browser automation processes, temporary browser profiles, or other repo-started automation helpers.
- Only terminate resources started by the agent for this repo, or clearly orphaned repo automation. If ownership is ambiguous, do not kill it silently; report it.
- Never leave repo-started dev servers, background workers, or automation browsers running after verification or visual inspection completes.

### Roadmap Authoring

- Roadmap items must be meaningful implementation slices, not micro-tasks. Bundle tightly related work in the same surface—UI behavior, state handling, error handling, cleanup, and the direct regression coverage needed to ship it—into one item unless a hard dependency or contract boundary forces a split.
- When new follow-up work is discovered in the same area, rewrite adjacent roadmap items so the roadmap reflects the real cutover shape and lowest-churn sequence instead of appending another tiny task.

### Recent Learned Patterns

- Public-facing docs should stay value-first: lead README with the core URL-to-result workflow, keep a fast 5-minute demo near the top, and point evidence docs at the smallest set of high-signal verification artifacts instead of archival inventories.
- Keep validation material secondary to the real product path. Top-level docs should send readers through README → demo → validation checklist before any optional supporting documentation.
- Local web development should keep browser requests same-origin through the Vite proxy. Use `DEV_API_PROXY_TARGET` when the backend is not on `http://127.0.0.1:8741`; reserve `VITE_API_BASE_URL` for intentional browser-visible cross-origin deployments.
- `PROXY_POOL_FILE=` is the explicit disable path for the optional proxy pool. Preserve empty overrides in config loading so fresh CLI smoke runs stay warning-free when users intentionally turn the feature off.
- `useWebSocket` should defer its initial connect by one tick and ignore stale/manual socket errors. React `StrictMode` double-mount in Vite dev otherwise produces a false `WebSocket ... closed before the connection is established` warning even when the transport is healthy.
- Export schedule history is wired through `ExportScheduleManager`; keep regression coverage at the manager level so browser-harness ref flakiness does not masquerade as a product bug.
- The web shell must declare a real favicon asset. Otherwise fresh browser sessions emit a load-time `/favicon.ico` 404 even when the app itself is healthy.
- API handlers should use `decodeJSONBody`, `writeJSONStatus`/`writeCreatedJSON`, and the shared current-user helpers instead of hand-rolled JSON decoding or `WriteHeader` + `writeJSON` sequences. That keeps `Content-Type`, body limits, unknown-field rejection, and auth checks consistent.
- When upgrading Biome, update `web/biome.json`'s `$schema` URL in the same change. Newer Biome releases hard-fail lint if the config schema version lags the CLI.
- Keep `go.mod`'s `toolchain` line aligned with the pinned Go patch version in `.tool-versions`; current Go docs treat it as the suggested reproducible main-module toolchain.
- Do not proactively maintain transitive Go `go.mod` overrides. Use a root-level `replace` only for rare, temporary high-severity security or correctness emergencies, document the reason inline, and remove it as soon as upstream modules absorb the fix.
- Chains and watch-triggered jobs must persist operator-facing `request` payloads, not typed job specs, so automation surfaces reuse the same request-to-job conversion path as live jobs and schedules.
- Job and batch control-plane responses should use the shared envelopes: `{ job }`, `{ jobs, total, limit, offset }`, and `{ batch, jobs, total, limit, offset }` across REST, Web, CLI, and MCP.
- Watch and crawl-state APIs must never expose host-local screenshot or diff paths; return explicit artifact download URLs from `/v1/watch/{id}/artifacts/{kind}` and keep crawl-state responses sanitized to the documented fields.
- Webhook validation is split intentionally: use `webhook.ValidateConfigURL` for create-time syntax checks, and keep SSRF/private-target enforcement in dispatch-time delivery planning (`ValidateURL` / pinned dialing). `WEBHOOK_ALLOW_INTERNAL=true` is a trusted-environment escape hatch only.
- `internal/submission` is the canonical operator-facing request-to-spec and validation layer for single and batch scrape/crawl/research flows. API handlers, CLI batch preflight, and direct batch execution should delegate there instead of rebuilding `jobs.JobSpec` defaults or validation locally.
- In React 19 code, prefer `useEffectEvent` for effect-owned listeners/timers that need latest render values without re-subscribing the effect.
- Settings authoring flows should treat Close as non-destructive and Discard as explicit removal. Persist AI sessions, AI handoff drafts, and native create/edit drafts in `sessionStorage`, then reserve `beforeunload` warnings for truly unsaved local edits that would otherwise be lost on tab close.

### Testing Guidelines

- **Go tests**: Use `go test ./...` with `CI=1` for consistent output
- **E2E tests**: Located in `internal/e2e` — excluded from `make test-ci`; the deterministic PR-safe system subset now lives in `internal/system`
- **Web tests**: Run with `cd web && pnpm run test`
- **Flaky E2E tests**: May be retried up to 3 times before considering them a real failure

## Project Structure

### Source Code Organization

```
cmd/spartan/          # Main CLI entry point
internal/             # Go packages (internal only)
  api/                # REST API server and route handlers
  auth/               # Auth profile management and vault
  cli/                # CLI subcommand implementations
  config/             # Global configuration and logging
  crawl/              # Concurrent website crawling logic
  e2e/                # End-to-end integration tests
  extract/            # HTML content extraction and normalization
  exporter/           # Result exporters (markdown, CSV, JSON)
  fetch/              # Content fetching (HTTP, Chromedp, Playwright)
  jobs/               # Job manager and worker pool
  mcp/                # MCP stdio server for agent orchestration
  model/              # Shared domain models and constants
  pipeline/           # Pipeline hooks, processors, and transformers
  research/           # Multi-source research workflows
  scheduler/          # Recurring job scheduler
  scrape/             # Single-page scraping logic
  store/              # Persistent storage for jobs and crawl states
  ui/tui/             # Terminal User Interface
web/                  # Frontend (Vite + React)
  src/                # TypeScript source
  src/api/            # Generated API client (from openapi.yaml)
api/                  # OpenAPI contract (api/openapi.yaml)
scripts/              # Utility scripts
docs/                 # Documentation (usage, architecture, landscape)
```

### Data Storage

- **Job store**: Local on-disk under `DATA_DIR` (default `.data`)
- **Auth vault**: `.data/auth_vault.json` (profiles + presets + inheritance)
- **Render profiles**: `.data/render_profiles.json`
- **Extraction templates**: `DATA_DIR/extract_templates.json`
- **Pipeline JS config**: `DATA_DIR/pipeline_js.json`

## Coding Style & Naming Conventions

### Go

- **Formatting**: `gofmt -w ./cmd ./internal` (enforced via `make format`)
- **Linting**: `go vet ./...` (enforced via `make lint`)
- **Naming**: Follow Go conventions (camelCase for variables, PascalCase for exported types)
- **Package structure**: Each internal package has a single, clear responsibility

### TypeScript

- **Formatting**: `biome format . --write` (enforced via `make format`)
- **Linting**: `biome lint .` (enforced via `make lint`)
- **Type checking**: `tsc --noEmit` (enforced via `make type-check`)
- **Framework**: Vite + React with strict TypeScript

## Architecture & Patterns

### Configuration (immutability + concurrency)

- `internal/config.Load()` is called once at process startup and returns a `config.Config` value (not a pointer).
- `config.Config` is treated as **immutable after loading** and is safe for concurrent read access when used this way.
- `AuthOverrides.Headers` and `AuthOverrides.Cookies` are maps (reference types). Treat them as read-only; deep-copy before modifying.

### Extraction Pipeline

- Centralized in `internal/extract`
- Templates stored in `DATA_DIR/extract_templates.json`
- Pipeline hooks and plugin contracts in `internal/pipeline` (pre/post fetch/extract/output + transformers)
- Headless per-target JS configured in `DATA_DIR/pipeline_js.json`

### Content Fetching

- **HTTP**: Default fetcher
- **Chromedp**: Headless Chromium (built-in, always available)
- **Playwright**: Optional for JS-heavy pages — enable with `USE_PLAYWRIGHT=1` or `--playwright` flag
- **Install Playwright**: `go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install --with-deps`

### Robots.txt

- **Ignored by default** — robots.txt compliance is opt-in via `--respect-robots` flag or `RESPECT_ROBOTS_TXT=true` env var
- When enabled, the crawler will:
  - Fetch and parse robots.txt for each host
  - Respect Allow/Disallow rules for the configured User-Agent
  - Apply Crawl-Delay if specified
  - Cache robots.txt per host with 1-hour TTL
  - Fail-open (allow crawl) if robots.txt fetch fails

### Error Handling

- **Use `internal/apperrors` package**: All new error handling must use the `apperrors` package for classification and consistent handling
- **Error kinds**: Use `apperrors.Validation()`, `apperrors.NotFound()`, `apperrors.Permission()`, `apperrors.Internal()` for appropriate error types
- **Wrapping**: Use `apperrors.Wrap(kind, "safe message", err)` to add context without exposing secrets in user-facing messages
- **Sentinel errors**: Use `apperrors.WithKind(kind, sentinelErr)` for stable error text that can be compared with `errors.Is()`
- **Never log secrets**: Always use `apperrors.SafeMessage(err)` when logging or returning errors to clients
- **HTTP handlers**: Use `writeError(w, err)` from `internal/api/util.go` for consistent status code mapping (validation→400, not_found→404, permission→403, internal→500)
- **Check error kinds**: Use `apperrors.IsKind(err, apperrors.KindValidation)` to check error types, or `apperrors.KindOf(err)` to get the kind
- **Error checking**: Use `errors.Is(err, sentinelErr)` and `errors.As(err, &typedErr)` for error inspection

See `internal/apperrors/README.md` for detailed usage patterns and examples.

## Toolchain

Pinned in `.tool-versions`:

- Go `1.26.1`
- Node `25.8` (any `25.8.x` patch)
- pnpm `10.32.1`

## Commit & Pull Request Guidelines

- **Local CI**: Run `make ci` before committing — it must pass
- **Commit messages**: Use clear, descriptive messages (no enforced format currently)
- **PR requirements**: Ensure `make ci` passes, describe changes clearly

## Generated Code Exceptions

Files generated from API contracts (e.g., `web/src/api/*.gen.ts` from `api/openapi.yaml`) are exempt from the 1000 LOC threshold requirement because:

1. They are machine-generated from a single source of truth
2. The generator controls output structure (limited splitting options)
3. They are regenerated on every `make generate` — manual organization would be overwritten
4. The types are conceptually cohesive (all API contract types)
5. They are not hand-maintained — complexity concerns for human editing don't apply

The relevant files are:
- `web/src/api/types.gen.ts` - Domain types and endpoint operation types
- `web/src/api/core/*.gen.ts` - Core client infrastructure types
- `web/src/api/client/*.gen.ts` - Client-specific types

## Documentation

- `README.md`: Project overview and quickstart
- `AGENTS.md`: This file — repository guidelines for AI agents
- `docs/usage.md`: CLI/API/Web/MCP entry points and flags
- `docs/architecture.md`: System structure and data flow
- `docs/landscape.md`: Project context and design decisions
ns
 and data flow
- `docs/landscape.md`: Project context and design decisions
ns
decisions
ns
decisions
ns
