# Spartan Scraper

[![CI PR](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-pr.yml/badge.svg)](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-pr.yml)
[![CI Slow](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-slow.yml/badge.svg)](https://github.com/fitchmultz/spartan-scraper/actions/workflows/ci-slow.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fitchmultz/spartan-scraper)](https://goreportcard.com/report/github.com/fitchmultz/spartan-scraper)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![Go Version](https://img.shields.io/badge/Go-1.26.1-00ADD8)
![TypeScript](https://img.shields.io/badge/TypeScript-5.9.3-3178C6)

Spartan Scraper is a local-first scraping workbench for turning a URL into a clean result, a bounded crawl, or a research job without standing up cloud infrastructure.

It is built for people who want one dependable workflow from fetch to stored artifacts: open the UI or CLI, submit work, inspect results locally, promote a verified job into reusable automation, and reopen saved watch/export history when daily operations or failure recovery matter.

A healthy first run works without any AI, proxy-pool, or retention setup. Those are optional subsystems you can enable later when a workflow actually calls for them.

If you want the fastest path in, start with the 5-minute demo below. If you are integrating it into a real workflow, the API, MCP server, schedules, and local artifact model all build on that same core path.

Planning and future work live in [docs/roadmap.md](docs/roadmap.md). That document is the canonical source of truth for what is in flight, next, and explicitly out of scope for the current cutover.

## Why It Exists

- Start from a URL and get something useful quickly: extracted content, crawl output, or a research bundle.
- Keep everything local by default: jobs, artifacts, auth profiles, schedules, and render rules stay on disk.
- Use the same job model everywhere: Web UI, CLI, API, TUI, and MCP all operate on the same persisted workflows.
- Stay practical for real sites: HTTP-first by default, Chromedp/Playwright when pages are JS-heavy or need login flows.

## 5-Minute Demo

This walkthrough uses the default out-of-the-box path. Leave AI, proxy pooling, and retention off; they are optional and not needed for the first successful scrape.

```bash
git clone https://github.com/fitchmultz/spartan-scraper.git
cd spartan-scraper

make verify-toolchain
make install
make generate
make build

# terminal 1
./bin/spartan server

# terminal 2
make web-dev
```

If startup detects a legacy or unsupported `.data` directory, Spartan now serves guided setup mode instead of failing only in the terminal. Run `./bin/spartan reset-data` once to archive the current data directory under `output/cutover/` and recreate `.data` for a fresh Balanced 1.0 boot.

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

## Validated Operator Workflow

The current operator path is:

1. submit a scrape, crawl, or research job
2. inspect the saved result on the job detail route
3. promote that verified job into a template, watch, or export schedule draft
4. save the automation from its destination surface
5. inspect persisted watch checks and export outcomes from `/automation/watches` and `/automation/exports`
6. follow guided recovery actions when a watch check or export run fails

The README keeps the fast first-success path near the top. For the guided continuation into promotion, saved automation, history inspection, and failure recovery, use:

- [docs/demo.md](docs/demo.md)
- [docs/operator.md](docs/operator.md)

## What It Covers

- Single pages, full websites, and deep research workflows.
- Works for static HTML and JS‑heavy sites (headless Chromium or Playwright).
- Unified interfaces: CLI, TUI, and Web UI.
- Clean API contract (OpenAPI) with generated TS client.
- Local, self‑hosted, no SaaS dependencies.
- Webhook integrations that now distinguish JSON job events from multipart export deliveries, so downstream receivers get actual exported bytes on `export.completed` instead of placeholder path metadata.

## Project Status

Spartan Scraper is in **1.0.0-rc1 release prep**. Current release-readiness evidence for this tree lives in [docs/evidence/release-readiness/2026-04-05/README.md](docs/evidence/release-readiness/2026-04-05/README.md).

## Quickstart

```bash
# Quick install (CLI-focused; requires Go 1.26+)
go install github.com/fitchmultz/spartan-scraper/cmd/spartan@latest

# Full local setup (recommended for contributors and operators)
git clone https://github.com/fitchmultz/spartan-scraper.git
cd spartan-scraper
make verify-toolchain
make install
make generate
make build
./bin/spartan --help
./bin/spartan server
make web-dev
```

If the default `.data` directory came from a pre-cutover build, reset it with:

```bash
./bin/spartan reset-data
```

Open `http://localhost:5173`, submit a scrape for `https://example.com`, and confirm the saved result contains `Example Domain`.

For a full local validation pass, run:

```bash
make ci-pr
```

Optional: install the binary into `~/.local/bin` (or `$XDG_BIN_HOME`) with:

```bash
make install-bin
```

## Developer And Agent Workflows

- Agents get an MCP surface, a deterministic local API, and a persistent job store they can inspect and reuse.
- Developers get one local system for UI, CLI, and API validation instead of separate throwaway scripts.
- Saved results can be re-authored locally too: bounded AI helpers can now refine research output, shape recurring exports, and generate validated JMESPath/JSONata transforms from representative persisted artifacts without launching new jobs.
- Teams get reproducible CI, generated API types, and stored artifacts that make behavior easier to verify.

## Community

- [LICENSE](LICENSE) - Apache License 2.0
- [NOTICE](NOTICE) - Apache 2.0 notice file
- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) - Code of conduct
- [SECURITY.md](SECURITY.md) - Security policy
- [CHANGELOG.md](CHANGELOG.md) - Release history
- [RELEASING.md](RELEASING.md) - Release workflow

## Documentation

- [docs/README.md](docs/README.md): docs index and navigation.
- [docs/usage.md](docs/usage.md): CLI/API/Web/MCP entry points and flags.
- [docs/architecture.md](docs/architecture.md): system structure and flow.
- [docs/demo.md](docs/demo.md): a fast clone-to-value walkthrough that continues into promotion and saved automation.
- [docs/operator.md](docs/operator.md): canonical operator workflow for promotion, saved automation, history inspection, and failure recovery.
- [docs/validation_checklist.md](docs/validation_checklist.md): copy/paste validation steps for setup, runtime checks, and public-readiness smoke tests.
- [RELEASING.md](RELEASING.md): release workflow and pre-tag checklist.
- [docs/ci.md](docs/ci.md): CI tiers, runtime expectations, and resource profile guidance.
- [docs/performance.md](docs/performance.md): tuning and scaling guidance.
- [docs/landscape.md](docs/landscape.md): ecosystem positioning and design trade-offs.

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

# If you later load a proxy pool, prefer residential us-east proxies from it for one request
./bin/spartan scrape \
  --url https://example.com \
  --proxy-region us-east \
  --proxy-tag residential \
  --out ./out/example.json

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
./bin/spartan export --job-id <id> --schedule-id <export-schedule-id>
./bin/spartan export --inspect-id <export-id>
./bin/spartan export --history-job-id <job-id>
./bin/spartan export-schedule history --id <export-schedule-id>

# Watch inspection
./bin/spartan watch check <watch-id>
./bin/spartan watch history <watch-id>
./bin/spartan watch history <watch-id> --check-id <check-id>

# Schedules
./bin/spartan schedule add --kind scrape --interval 3600 --url https://example.com
./bin/spartan schedule list

# Run API server + background worker (API binds to localhost by default)
./bin/spartan server

# Archive and recreate a legacy/default .data directory for Balanced 1.0
./bin/spartan reset-data

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

Validated operator continuation after a successful job:

- `/jobs/:id` is the promotion starting point for templates, watches, and export schedules
- `/templates` is where promoted templates are saved and previewed
- `/automation/watches` is where promoted watches are saved, checked, and inspected through persisted history
- `/automation/exports` is where promoted export schedules are saved and where export outcome history stays inspectable across success and failure states

The dev server proxies API requests to `http://localhost:8741` by default.
WebSocket upgrades to `/v1/ws` accept browser origins from loopback hosts only (`localhost`, `127.0.0.1`, `::1`).
Non-browser clients without an `Origin` header remain supported.
If you run the backend on a different local port, set `DEV_API_PROXY_TARGET=http://127.0.0.1:<port>` in `web/.env` so the dev proxy stays same-origin.
Use `VITE_API_BASE_URL` only for deployed cross-origin builds where the browser should call a remote API directly.

Repo-local AI defaults live in `.env` and `config/pi-routes.json`, but core scrape/crawl/research workflows work without AI. Leave `PI_ENABLED=false` for the default first run; when you later set `PI_ENABLED=true`, Spartan asks pi for routes in this order: `kimi-coding/k2p5`, `zai/glm-5`, `openai-codex/gpt-5.4`. Auth, account selection, and billing stay in pi; if you want a different route order or different provider/model IDs, override `PI_CONFIG_PATH` or edit that routes file locally.

Proxy pooling and retention are optional too: leave `PROXY_POOL_FILE` unset and `RETENTION_ENABLED=false` until you actually need pooled routing or automated cleanup.

## Interfaces

- Web UI for job submission, monitoring, automation, and settings
- CLI for scripting and local automation
- REST + WebSocket APIs for integrations
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
- `internal/exporter`: exports results to json/jsonl/csv/markdown/xlsx.
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

- Go `1.26.1`
- Node `25.9` (any `25.9.x` patch)
- pnpm `10.33.0`

Use a `.tool-versions`-compatible version manager (for example `mise install`) to provision those pinned versions, then run `make verify-toolchain` before build/test work.

## Local CI

GitHub workflow split:

- **PR required:** `.github/workflows/ci-pr.yml` (`make ci-pr`)
- **Nightly/manual heavy checks:** `.github/workflows/ci-slow.yml` (`make ci-slow`, deterministic local-fixture heavy lane that provisions Playwright browsers)

```bash
make verify-toolchain  # Print and enforce the exact Go/Node/pnpm contract from .tool-versions
make audit-public      # Scan tracked files + branch history for public-readiness leaks/secrets/placeholders
make secret-scan       # Deep git-history secret scan (manual/nightly release-tier check)
make ci-pr             # PR-equivalent deterministic gate (requires clean git state)
make ci                # Full local gate (Go + web + pi-bridge install/build/tests)
make ci-slow           # Deterministic heavy stress/e2e checks (local fixture; provisions Playwright)
make ci-network        # Optional live-Internet smoke validation
CI_VITEST_MAX_WORKERS=2 make ci-pr  # Optional local worker cap override
make ci-manual         # Manual full heavy sweep (ci-slow + ci-network)
```
