# Contributing to Spartan Scraper

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to Spartan Scraper.

## Getting Started

### Prerequisites

- Go 1.26.1
- Node 25.8.1
- pnpm 10.32.1
- A `.tool-versions`-compatible version manager (for example `mise`) or those exact versions already active on `PATH`

### Quick Setup

```bash
make verify-toolchain # Print and enforce the exact Go/Node/pnpm contract from .tool-versions
make install          # Download Go deps + install pnpm deps for web + pi-bridge
make generate         # Generate TS API client from openapi.yaml
make build            # Build Go binary + web + pi-bridge assets
make install-bin      # Optional: install built binary to ~/.local/bin
```

After setup, verify the installation:

```bash
./bin/spartan --help
```

## Development Workflow

### Makefile Targets

The Makefile is the canonical interface for all development tasks:

```bash
make verify-toolchain # Print and enforce the exact Go/Node/pnpm contract from .tool-versions
make audit-public     # Scan tracked files + branch history for public-readiness leaks/placeholders
make secret-scan      # Deep git-history secret scan (manual/release-tier)
make install          # Download Go deps + install pnpm deps for web + pi-bridge
make update           # Update all Go deps + pnpm deps (web + pi-bridge) to latest, majors included
make generate         # Generate TS API client from openapi.yaml
make format           # Format Go (gofmt) and TS (biome)
make type-check       # Type-check web TS
make lint             # Lint Go (go vet) and TS (biome)
make build            # Build Go binary + web + pi-bridge assets (no install side effects)
make install-bin      # Install built binary to ~/.local/bin (or $XDG_BIN_HOME)
make test             # Run Go tests (including e2e)
make test-ci          # Run Go tests (excluding e2e) + web tests (Vitest workers capped by CI_VITEST_MAX_WORKERS)
make ci-pr            # PR-equivalent deterministic gate (clean git state required)
make ci               # Full local CI pipeline (audit-public, install, generate, format, type-check, lint, build, test-ci)
make ci-slow          # Deterministic heavy stress + e2e checks (local fixture; provisions Playwright)
make ci-network       # Optional live-Internet smoke validation
make ci-manual        # Manual full heavy CI profile (ci-slow + ci-network)
CI_VITEST_MAX_WORKERS=2 make ci-pr  # Optional local worker cap override
make clean            # Remove build artifacts, dependencies, node_modules, installed binary
make web-dev          # Start web dev server (http://localhost:5173)
```

### Local CI Gate

Use CI profiles intentionally based on scope (mirrors GitHub workflows):

- `.github/workflows/ci-pr.yml` → required PR checks (`make ci-pr`)
- `.github/workflows/ci-slow.yml` → nightly/manual deterministic heavy checks (`make ci-slow`, with Playwright provisioning)

- **PR-equivalent gate**: `make ci-pr` (requires clean git state)
- **Full local gate**: `make ci` (includes dependency install plus Go/web/extension verification)
- **Heavy/nightly checks**: `make ci-slow` (deterministic local fixture with Playwright provisioning)
- **Optional live smoke**: `make ci-network`
- **Manual full pre-release sweep**: `make ci-manual` + `make secret-scan`

`make ci-pr` and `make ci` run:

```
audit-public → install → generate → format → type-check → lint → build → test-ci
```

`make ci-pr` adds clean-tree assertions before and after the pipeline, so generation/format drift cannot pass silently.

Do not proactively add root `go.mod` overrides for transitive dependency freshness. Reserve `replace` directives for rare, temporary high-severity security or correctness emergencies only.

### Branch and Commit Workflow

- No enforced commit message format currently
- Keep commit messages clear and descriptive
- Ensure `make ci` passes before committing
- Pull requests are welcome; describe changes clearly

## Code Standards

### Go

- **Formatting**: `gofmt -w ./cmd ./internal` (enforced via `make format`)
- **Linting**: `go vet ./...` (enforced via `make lint`)
- **Naming**: Follow Go conventions (camelCase for variables, PascalCase for exported types)
- **Package structure**: Each internal package has a single, clear responsibility
- **Documentation**: All code files must have docstrings explaining:
  - What the module/file is responsible for
  - What it explicitly does NOT handle
  - Any invariants or assumptions callers must respect

### TypeScript

- **Formatting**: `biome format . --write` (enforced via `make format`)
- **Linting**: `biome lint .` (enforced via `make lint`)
- **Type checking**: `tsc --noEmit` (enforced via `make type-check`)
- **Framework**: Vite + React with strict TypeScript

### Error Handling

All new error handling must use the `internal/apperrors` package for classification and consistent handling:

- Use `apperrors.Validation()`, `apperrors.NotFound()`, `apperrors.Permission()`, `apperrors.Internal()` for appropriate error types
- Use `apperrors.Wrap(kind, "safe message", err)` to add context without exposing secrets
- Use `apperrors.SafeMessage(err)` when logging or returning errors to clients
- For HTTP handlers, use `writeError(w, err)` from `internal/api/util.go` for consistent status code mapping

See `internal/apperrors/README.md` for detailed usage patterns and examples.

### Architecture Patterns

- **Configuration immutability**: `internal/config.Load()` is called once at startup and returns an immutable value. Treat `config.Config` as read-only after loading.
- **Pipeline hooks**: See `internal/pipeline` for plugin contracts (pre/post fetch/extract/output + transformers)
- **Extraction templates**: Stored in `DATA_DIR/extract_templates.json`
- **Auth vault**: Stored in `.data/auth_vault.json` (profiles + presets + inheritance)

## Testing

### Running Tests

```bash
make test         # Run Go tests (including e2e)
make test-ci      # Run Go tests (excluding e2e) + web tests
```

For Go tests specifically:
```bash
go test ./...     # Run all Go tests
CI=1 go test ./... # Run with consistent output (no race detector)
```

For web tests:
```bash
cd web && pnpm run test
```

### Testing Guidelines

- Tests must live as close as possible to the code they validate (same module or per-concept tests folder)
- Full expected behavior and failure modes must be covered
- New or changed code must have tests
- E2E tests are located in `internal/e2e` and excluded from `make test-ci`
- Flaky E2E tests may be retried up to 3 times before considering them a real failure

### Modules and File Boundaries

- Files/modules should represent a single cohesive responsibility
- Individual source files should remain under ~400 LOC
- Files exceeding ~700 LOC require explicit justification
- Files exceeding ~1,000 LOC are presumed to be mis-scoped and must be split

## Project Structure

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

## Reporting Bugs

Use the repository issue tracker for non-security bugs and feature requests.

When reporting a bug, please include:
- Reproduction steps
- Expected behavior vs. actual behavior
- Environment information (Go version, OS, etc.)
- Relevant logs or error messages
- Any configuration that may affect the issue

## Security Issues

**Do NOT open public issues for vulnerabilities.**

See [SECURITY.md](SECURITY.md) for instructions on how to report security vulnerabilities responsibly.

## Documentation

- Keep [AGENTS.md](AGENTS.md) updated with lessons learned and repository philosophy
- All new features should be documented in `docs/usage.md`
- Update [README.md](README.md) for user-facing changes
- Architecture changes should be reflected in `docs/architecture.md`

## Content Fetching Options

- **HTTP**: Default fetcher
- **Chromedp**: Headless Chromium (built-in, always available)
- **Playwright**: Optional for JS-heavy pages — enable with `USE_PLAYWRIGHT=1` or `--playwright` flag

Install Playwright browsers:
```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install --with-deps
```

## Questions?

If you have questions about contributing or need help getting started, open a discussion or issue in this repository (for non-security topics).

---

Thank you for contributing to Spartan Scraper!
r!
ns?

If you have questions about contributing or need help getting started, open a discussion or issue in this repository (for non-security topics).

---

Thank you for contributing to Spartan Scraper!
