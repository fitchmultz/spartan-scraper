# Repository Guidelines

## Project Overview

- **Primary language**: Go (CLI + API + TUI)
- **Frontend**: TypeScript (Vite + React)
- **Local CI gate**: `make ci` — must pass before completing work or committing
- **API contract**: `api/openapi.yaml` → generates TS client via `make generate` (hey-api openapi-ts)

## Development Workflow

### Local CI Gate

`make ci` runs: `install → generate → format → type-check → lint → build → test-ci`

**CRITICAL**: Never end a turn with a failing `make ci`. If `make ci` fails, fix all failures before completing your work.

### Build, Test, and Development Commands

```bash
make install          # Download Go deps + install pnpm deps
make update           # Update all Go/pnpm deps to latest (review before committing)
make generate         # Generate TS API client from openapi.yaml
make format           # Format Go (gofmt) and TS (biome)
make type-check       # Type-check TS (biome/tsc)
make lint             # Lint Go (go vet) and TS (biome)
make build            # Build Go binary + web assets + install to ~/.local/bin
make test             # Run Go tests (including e2e)
make test-ci          # Run Go tests (excluding e2e) + web tests
make ci               # Full CI pipeline: install, generate, format, type-check, lint, build, test-ci
make clean            # Remove build artifacts, dependencies, node_modules, installed binary
make web-dev          # Start web dev server (http://localhost:5173)
```

### Testing Guidelines

- **Go tests**: Use `go test ./...` with `CI=1` for consistent output
- **E2E tests**: Located in `internal/e2e` — excluded from `make test-ci`
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
- **Install Playwright**: `go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install --with-deps`

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

- Go `1.25.6`
- Node `24.13.0`
- pnpm `10.28.0`

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
