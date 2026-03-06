# Spartan Scraper

[![CI PR](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-pr.yml/badge.svg)](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-pr.yml)
[![CI Slow](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-slow.yml/badge.svg)](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-slow.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fitchmultz/spartan-scraper)](https://goreportcard.com/report/github.com/fitchmultz/spartan-scraper)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
![Go Version](https://img.shields.io/badge/Go-1.25.6-00ADD8)
![TypeScript](https://img.shields.io/badge/TypeScript-5.9.3-3178C6)

Spartan Scraper is a local-first scraping workbench for turning a URL into a clean result, a bounded crawl, or a research job without standing up cloud infrastructure.

It is built for people who want one dependable workflow from fetch to stored artifacts: open the UI or CLI, submit work, inspect results locally, and only reach for headless browsers when a target actually needs them.

If you want the fastest path in, start with the 5-minute demo below. If you are integrating it into a real workflow, the API, MCP server, schedules, and local artifact model all build on that same core path.

## Why It Exists

- Start from a URL and get something useful quickly: extracted content, crawl output, or a research bundle.
- Keep everything local by default: jobs, artifacts, auth profiles, schedules, and render rules stay on disk.
- Use the same job model everywhere: Web UI, CLI, API, TUI, and MCP all operate on the same persisted workflows.
- Stay practical for real sites: HTTP-first by default, Chromedp/Playwright when pages are JS-heavy or need login flows.

## 5-Minute Demo

```bash
git clone <repo-url>
cd spartan-scraper

make install
make generate
make build

# terminal 1
./bin/spartan server

# terminal 2
make web-dev
```

Open `http://localhost:5173`, submit a scrape job for `https://example.com`, and expect:

- the dashboard to show a new job move into `succeeded`
- the results panel to include `Example Domain`
- the metrics widget to show a live WebSocket connection

If you want a CLI-only proof first:

```bash
./bin/spartan scrape --url https://example.com --out ./out/example.json
cat ./out/example.json
```

Expected output includes the page title text `Example Domain`.

For a more guided version with expected checkpoints, see [docs/demo.md](docs/demo.md).

## What It Covers

- Single pages, full websites, and deep research workflows.
- Works for static HTML and JS‑heavy sites (headless Chromium or Playwright).
- Unified interfaces: CLI, TUI, and Web UI.
- Clean API contract (OpenAPI) with generated TS client.
- Local, self‑hosted, no SaaS dependencies.

## Project Status

Spartan Scraper is currently **pre-1.0** and actively evolving. Public APIs and internal package layouts may change between minor releases.

## Quickstart

```bash
# Quick install (CLI-focused; requires Go 1.25+)
go install github.com/fitchmultz/spartan-scraper/cmd/spartan@latest

# Full local setup (recommended for contributors and operators)
make install
make generate
make build
./bin/spartan --help
# Optional: install binary into ~/.local/bin (or $XDG_BIN_HOME)
make install-bin
```

After the server is running, the fastest way to see value is:

1. Open the Web UI at `http://localhost:5173`
2. Submit a scrape for `https://example.com`
3. Confirm the saved result contains `Example Domain`

## Validation Quickstart

```bash
git clone <repo-url>
cd spartan-scraper

make ci-pr         # Clean-state PR-equivalent gate
./bin/spartan --help
./bin/spartan server
```

In another terminal:

```bash
make web-dev
```

Open `http://localhost:5173`.

## Developer And Agent Workflows

- Agents get an MCP surface, a deterministic local API, and a persistent job store they can inspect and reuse.
- Developers get one local system for UI, CLI, and API validation instead of separate throwaway scripts.
- Teams get reproducible CI, generated API types, and stored artifacts that make behavior easier to verify.

## Community

- [LICENSE](LICENSE) - MIT License
- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) - Code of conduct
- [SECURITY.md](SECURITY.md) - Security policy
- [CHANGELOG.md](CHANGELOG.md) - Release history
- [RELEASING.md](RELEASING.md) - Release workflow

## Documentation

- [docs/README.md](docs/README.md): docs index and navigation.
- [docs/usage.md](docs/usage.md): CLI/API/Web/MCP entry points and flags.
- [docs/architecture.md](docs/architecture.md): system structure and flow.
- [docs/demo.md](docs/demo.md): a fast clone-to-value walkthrough with expected output.
- [docs/validation_checklist.md](docs/validation_checklist.md): copy/paste validation steps for setup, runtime checks, and public-readiness smoke tests.
- [docs/release_readiness.md](docs/release_readiness.md): release hardening report and risk log.
- [docs/ci.md](docs/ci.md): CI tiers, runtime expectations, and resource profile guidance.
- [docs/performance.md](docs/performance.md): tuning and scaling guidance.
- [docs/landscape.md](docs/landscape.md): ecosystem positioning and design trade-offs.
- [docs/evidence/dogfood/README.md](docs/evidence/dogfood/README.md): curated UI verification evidence.
- [docs/enablement/adoption-map.md](docs/enablement/adoption-map.md): optional demo/workshop/cookbook/ops material derived from the project.

### CLI examples

```bash
# Single page scrape (HTTP)
./bin/spartan scrape \
  --url https://example.com \
  --out ./out/example.json

# Headless scrape with login (form-based)
./bin/spartan scrape \
  --url https://example.com/dashboard \
  --headless \
  --playwright \
  --login-url https://example.com/login \
  --login-user-selector '#email' \
  --login-pass-selector '#password' \
  --login-submit-selector 'button[type=submit]' \
  --login-user you@example.com \
  --login-pass 'demo-password' \
  --out ./out/dashboard.json

# Crawl a site (depth-limited)
./bin/spartan crawl \
  --url https://example.com \
  --max-depth 2 \
  --max-pages 200 \
  --out ./out/site.jsonl

# Deep research
./bin/spartan research \
  --query "pricing model" \
  --urls https://example.com,https://example.com/docs \
  --out ./out/research.jsonl

# MCP server (stdio)
./bin/spartan mcp

# Auth profiles
./bin/spartan auth set --name acme --auth-basic user:pass --header "X-API-Key: token-from-provider"
./bin/spartan scrape --url https://example.com --auth-profile acme

# Extraction Templates
./bin/spartan scrape --url https://example.com/product/123 --extract-template product
./bin/spartan scrape --url https://example.com --extract-config my-template.json

# Export results
./bin/spartan export --job-id <id> --format md --out ./out/report.md

# Schedules
./bin/spartan schedule add --kind scrape --interval 3600 --url https://example.com
./bin/spartan schedule list

# Run API server + background worker (API binds to localhost by default)
./bin/spartan server

# Expose API on all interfaces (use with caution)
# API key auth is auto-enforced when BIND_ADDR is non-localhost.
BIND_ADDR=0.0.0.0 ./bin/spartan server

# Launch TUI
./bin/spartan tui
```

### Web UI

```bash
./bin/spartan server
# in a second terminal:
make web-dev
```

Open `http://localhost:5173` for the UI.

The dev server proxies API requests to `http://localhost:8741` by default.
WebSocket upgrades to `/v1/ws` accept browser origins from loopback hosts only (`localhost`, `127.0.0.1`, `::1`).
Non-browser clients without an `Origin` header remain supported.
If you run the backend on a different local port, set `DEV_API_PROXY_TARGET=http://127.0.0.1:<port>` in `web/.env` so the dev proxy stays same-origin.
Use `VITE_API_BASE_URL` only for deployed cross-origin builds where the browser should call a remote API directly.

## Interfaces

- Web UI for job submission, monitoring, and admin workflows
- CLI for scripting and local automation
- REST + GraphQL APIs for integrations
- MCP server for agent orchestration
- TUI for terminal-first inspection

## Architecture at a glance

- `cmd/spartan`: main entrypoint for CLI/TUI/API server.
- `internal/fetch`: HTTP + headless fetchers (Chromedp + Playwright).
- `internal/extract`: HTML parsing + text/metadata extraction.
- `internal/crawl`: BFS crawler with depth/limit and domain scoping (robots ignored by default; opt-in support available).
- `internal/jobs`: persistent job store + queue runner.
- `internal/api`: HTTP API + OpenAPI contract.
- `web`: Vite + React UI; generated API client under `web/src/api`.
- `internal/research`: multi-source workflow (scrape/crawl → evidence → simhash dedup → clustering → citations + confidence → summary).
- `internal/mcp`: MCP stdio server for agent orchestration.
- `internal/exporter`: exports results to markdown/csv/json.
- `internal/scheduler`: recurring job runner with interval schedules.

## Notes

- Robots.txt is ignored by default; enable compliance with `--respect-robots` or `RESPECT_ROBOTS_TXT=true`.
- Auth support: headers, cookies, basic auth, tokens, query params, and form login (headless).
- Auth vault is stored in `.data/auth_vault.json` (profiles + presets + inheritance).
- Render profiles (adaptive rules) are stored in `.data/render_profiles.json`.
- Rate limiting + retries are configurable via `.env`.
- Playwright can be enabled with `USE_PLAYWRIGHT=1` or `--playwright` (CLI/API). Install browsers with `make install-playwright`. `make ci-slow` provisions Playwright automatically for clean-machine heavy runs.

## Toolchain

Pinned in `.tool-versions`:

- Go `1.25.6`
- Node `24.13.0`
- pnpm `10.28.0`

## Local CI

GitHub workflow split:

- **PR required:** `.github/workflows/ci-pr.yml` (`make ci-pr`)
- **Nightly/manual heavy checks:** `.github/workflows/ci-slow.yml` (`make ci-slow`, deterministic local-fixture heavy lane that provisions Playwright browsers)

```bash
make audit-public  # Scan tracked files + branch history for public-readiness leaks/secrets/placeholders
make secret-scan   # Deep git-history secret scan (manual/nightly release-tier check)
make ci-pr         # PR-equivalent deterministic gate (requires clean git state)
make ci            # Full local gate (includes install + build + tests)
make ci-slow       # Deterministic heavy stress/e2e checks (local fixture; provisions Playwright)
make ci-network     # Optional live-Internet smoke validation
CI_VITEST_MAX_WORKERS=2 make ci-pr  # Optional local worker cap override
make ci-manual      # Manual full heavy sweep (ci-slow + ci-network)
```
