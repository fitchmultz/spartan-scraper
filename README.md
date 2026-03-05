# Spartan Scraper

[![CI PR](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-pr.yml/badge.svg)](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-pr.yml)
[![CI Slow](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-slow.yml/badge.svg)](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-slow.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fitchmultz/spartan-scraper)](https://goreportcard.com/report/github.com/fitchmultz/spartan-scraper)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
![Go Version](https://img.shields.io/badge/Go-1.25.6-00ADD8)
![TypeScript](https://img.shields.io/badge/TypeScript-5.9.3-3178C6)

A high‑performance, Go‑first web scraping + automation standard for all future projects.

## Goals

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

# Full local setup (recommended for contributors/reviewers)
make install
make generate
make build
./bin/spartan --help
# Optional: install binary into ~/.local/bin (or $XDG_BIN_HOME)
make install-bin
```

## Reviewer Quickstart

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
- [docs/reviewer_checklist.md](docs/reviewer_checklist.md): copy/paste reviewer validation steps.
- [docs/release_readiness.md](docs/release_readiness.md): release hardening report and risk log.
- [docs/ci.md](docs/ci.md): CI tiers, runtime expectations, and resource profile guidance.
- [docs/performance.md](docs/performance.md): tuning and scaling guidance.
- [docs/landscape.md](docs/landscape.md): ecosystem positioning and design trade-offs.
- [docs/evidence/dogfood/README.md](docs/evidence/dogfood/README.md): timestamped UI dogfood evidence bundles.
- [docs/role-evidence/evidence-map.md](docs/role-evidence/evidence-map.md): production-readiness evidence pack (demo/workshop/cookbook/ops).

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
  --login-pass '***' \
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
./bin/spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
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
If you change the backend `PORT` in the root `.env`, you must also update `VITE_API_BASE_URL` in `web/.env` to match.

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
- Playwright can be enabled with `USE_PLAYWRIGHT=1` or `--playwright` (CLI/API). Install browsers with:

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install --with-deps
```

## Toolchain

Pinned in `.tool-versions`:

- Go `1.25.6`
- Node `24.13.0`
- pnpm `10.28.0`

## Local CI

GitHub workflow split:

- **PR required:** `.github/workflows/ci-pr.yml` (`make ci-pr`)
- **Nightly/manual heavy checks:** `.github/workflows/ci-slow.yml` (`make ci-slow`)

```bash
make audit-public  # Scan tracked files + branch history for public-readiness leaks/secrets/placeholders
make secret-scan   # Deep git-history secret scan (manual/nightly release-tier check)
make ci-pr         # PR-equivalent deterministic gate (requires clean git state)
make ci            # Full local gate (includes install + build + tests)
make ci-slow       # Heavy stress/e2e checks (network required)
CI_VITEST_MAX_WORKERS=2 make ci-pr  # Optional local worker cap override
make ci-manual     # Alias for manual heavy profile
```
