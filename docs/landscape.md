# Spartan Scraper Landscape

This document explains where Spartan Scraper fits in the scraping/automation ecosystem and why the project is designed the way it is.

For planned work and sequencing, use [docs/roadmap.md](roadmap.md). This landscape document explains direction; the roadmap is the source of truth for what is actually planned.

> Privacy note: this document intentionally excludes local machine inventories, personal directory paths, and unrelated private project references.

## Problem Space

Scraping systems usually break down when they need to support more than one mode of operation. Typical pain points include:

- One-off scripts that cannot scale into repeatable jobs.
- Crawlers that handle HTML but struggle with JS-heavy targets.
- API, CLI, and UI layers that drift because they are built independently.
- Ad-hoc authentication handling that is hard to audit and hard to reuse.
- Weak error boundaries that leak sensitive implementation details.

Spartan Scraper is built to solve those problems in one cohesive system.

## Project Positioning

Spartan Scraper is a self-hosted scraping platform with:

- A Go-first backend for concurrency and operational stability.
- Multiple operator interfaces: CLI, TUI, API, and Web UI.
- Shared domain models and an OpenAPI contract for consistency.
- Pluggable fetch/extract/pipeline behavior for custom workflows.

The project targets engineering teams that need repeatable automation pipelines, not single-use scripts.

## Core Design Decisions

### 1) Go-First Runtime

The backend uses Go for predictable concurrency, strong standard-library primitives, and easy local deployment.

### 2) Multi-Interface Delivery from One Core

CLI, API, TUI, and Web UI all route through shared internal packages so behavior stays aligned across interfaces.

Interface symmetry is intentional, not absolute. Spartan aims for shared capability where it improves operator outcomes, but it does not force every interface to host every workflow. In particular, AI preview and template-authoring flows belong in the Web UI, API, CLI, and MCP rather than the TUI, which remains focused on local inspection and job operations.

### 3) Fetch Strategy Abstraction

The fetch layer supports:

- HTTP (default)
- Chromedp (always available)
- Playwright (optional, for JS-heavy pages)

This avoids locking the project to a single rendering strategy.

### 4) OpenAPI as Contract Source of Truth

`api/openapi.yaml` defines API behavior and generates the web client types. This reduces drift between backend handlers and frontend usage.

### 5) Structured Error Handling + Redaction

`internal/apperrors` classifies errors and sanitizes unsafe path/URL/token details before returning them to users or logs.

## Architecture Shape

At a high level:

- `internal/fetch`: content acquisition
- `internal/extract`: structured extraction
- `internal/crawl`: graph traversal over websites
- `internal/jobs` + `internal/store`: orchestration and persistence
- `internal/pipeline`: composable processors and transformers
- `internal/research`: multi-source evidence workflows
- `internal/api`: HTTP surface
- `web/`: operator dashboard

This separation keeps responsibilities explicit and testable.

## Trade-offs and Non-Goals

### Trade-offs

- The codebase favors explicitness over framework magic.
- Optional browser tooling (Playwright) adds setup complexity in exchange for better JS-site coverage.
- Local-first operation means operators must manage their own runtime and storage.

### Non-Goals

- It is not a hosted SaaS service.
- It does not attempt to bypass legal/contractual restrictions on target sites.
- It does not optimize for ultra-minimal script size over maintainability.

## What This Repository Demonstrates

For engineering review, this repository is intended to show:

- Clean module boundaries in a polyglot system (Go + TypeScript).
- Contract-driven API/client development.
- Local CI discipline through a single `make ci` gate.
- Consistent error classification and redaction.
- Practical interfaces for both developer and operator workflows.

## Future Evolution Areas

Likely next maturity steps include:

- Release process hardening and changelog discipline.
- More explicit compatibility/versioning policy.
- Additional observability metrics and dashboards.
- Continued UX refinement in the web interface.
- Agentic research workflows powered by `pi`, while keeping deterministic `internal/research` as the baseline. The current implementation is additive and bounded: pi can choose follow-up URLs from discovered evidence and synthesize the final brief, but fetch/crawl/extract still run through Spartan primitives.
