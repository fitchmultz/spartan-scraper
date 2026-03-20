# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Completed

- [Web UI/UX Audit](specs/web-ui-ux-audit.md) — findings from live product review and code inspection that set the direction for the current Web UI cutover.
- [Web Shell Simplification](specs/web-shell-simplification.md) — thinner, route-first global chrome so operators reach real work faster.
- [Guided Job Submission Wizard](specs/guided-job-wizard.md) — guided 4-step job creation on `/jobs/new` with Expert mode and review-before-submit.
- [Job Monitoring Dashboard](specs/job-monitoring-dashboard.md) — scan-first `/jobs` dashboard with lane-based monitoring and return-context preservation.
- [Results View System Overhaul](specs/results-view-system-overhaul.md) — dominant reader on `/jobs/:id` with quieter secondary tools and guided export.
- [Automation Hub Redesign](specs/automation-hub-redesign.md) — `/automation/:section` hub with explicit in-route sub-navigation for batches, chains, watches, exports, and webhooks.
- [Template Editor Rework](specs/template-editor-rework.md) — inline `/templates` workspace with persistent list/detail, preview, and AI-assisted authoring.
- [Integrated AI Assistant Panel](specs/ai-assistant-panel.md) — persistent route-aware AI surface across job creation, templates, and results.
- [Toast Notification System](specs/toast-notification-system.md) — consistent transient feedback across the Web UI.
- [Verified Job Promotion Contract Audit](specs/job-to-automation-promotion-contract-audit.md) — confirmed the real source-job reuse boundary, destination overlap, redaction constraints, and route-level data requirements before the promotion cutover begins.

## Planning Principles

- Treat the Web UI as a first-class operator surface. When parity work and workflow clarity compete, prioritize the product workflow that helps operators complete real tasks faster and with less confusion.
- Prefer feature symmetry across the primary product interfaces that carry the main operator and automation workflows: API, Web UI, CLI, and MCP, but do not preserve a poor Web UI solely for parity.
- Treat the TUI as an intentionally limited local inspection surface, not a feature-parity target, unless this roadmap explicitly says otherwise.
- Add AI enablement where it improves a real scraping, template-authoring, or results-analysis workflow; do not force AI into surfaces where it adds little operational value.
- Preserve route-per-major-feature information architecture at the top level, then use sub-routing or explicit in-route navigation inside complex Web surfaces when that materially improves clarity.
- Treat interface asymmetry as intentional only when this roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.
- Put meaningful operator-facing product work ahead of maintenance, cleanup, and policy reminders.
- Treat focused failure-path dogfooding as acceptance criteria for major operator workflow cutovers, not as a standalone roadmap epic.

## After

1. [Verified Job Promotion Cutover](specs/job-to-automation-promotion.md) — implement the audited product path from a completed, trusted job into templates, watches, and export schedules, including authoritative job-detail loading, destination-specific seeded drafts, and explicit treatment of unsupported carry-forward.
2. [Promotion Flow Deterministic Regression Coverage](specs/promotion-flow-deterministic-regression.md) — once the promotion path is stable, lock it down with system-first deterministic regression coverage and narrow browser proof that also protects the detail-fetch fallback and redaction boundary.

## Later / Deprioritized

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level Web route model (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), but allow sub-routing or in-route navigation inside complex surfaces when that materially improves operator comprehension.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
