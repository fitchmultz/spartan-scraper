# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Treat the Web UI as a first-class operator surface. When parity work and workflow clarity compete, prioritize the product workflow that helps operators complete real tasks faster and with less confusion.
- Prefer feature symmetry across the primary product interfaces that carry the main operator and automation workflows: API, Web UI, CLI, and MCP, but do not preserve a poor Web UI solely for parity.
- Treat the TUI as an intentionally limited local inspection surface, not a feature-parity target, unless this roadmap explicitly says otherwise.
- Add AI enablement where it improves a real scraping, template-authoring, or results-analysis workflow; do not force AI into surfaces where it adds little operational value.
- Preserve route-per-major-feature information architecture at the top level, then use sub-routing or explicit in-route navigation inside complex Web surfaces when that materially improves clarity.
- Treat interface asymmetry as intentional only when this roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.
- Put meaningful operator-facing product work ahead of maintenance, cleanup, and policy reminders.

## Recently Completed

- Automation hub navigation now gives `/automation` explicit section switching and stable deep links for batches, chains, watches, exports, and webhook deliveries, so operators no longer have to scroll a stacked mega-page just to change automation modes.
- Run history observability now spans API, Web UI, CLI, and MCP with recent execution inspection, explicit batch queue progression, recent failed-run views, and structured failure context so operators can understand outcomes without digging through host-local files or internal state.
- Watch management now spans API, Web UI, CLI, and MCP with `watch_list`, `watch_get`, `watch_create`, `watch_update`, `watch_delete`, and `watch_check`, so operators and agents can manage stored watches and run manual checks without falling back to REST-only flows.
- Webhook delivery inspection now spans API, Web UI, CLI, and MCP with sanitized URL/error output, so operators can debug retries and failures from `spartan webhook deliveries ...` and `webhook_delivery_get` without reading host-local files.
- Batch management now spans API, Web UI, CLI, and MCP with authoritative list/detail/cancel flows plus MCP batch submission tools, so operators and agents can create, enumerate, inspect, and stop persisted batches without falling back to REST-only workflows or browser-local tracking.

## Now

- [Web Shell Simplification](./specs/web-shell-simplification.md) — Remove the duplicated masthead/route-intro pattern, reduce repeated metrics and CTA chrome above the fold, and let each route spend its first screen on real work instead of repeated framing.
- [Job Monitoring Dashboard](./specs/job-monitoring-dashboard.md) — Replace the dense recent-run list with a scan-friendly monitoring surface that emphasizes visual progress, failure severity, queue position, and smoother movement between the jobs index and job detail.
- [Guided Job Submission Wizard](./specs/guided-job-wizard.md) — Replace the long single-screen submission flow with a step-based wizard for scrape, crawl, and research jobs while preserving an Expert mode for operators who want dense editing.

## After

- [Results View System Overhaul](./specs/results-view-system-overhaul.md) — Reduce view-mode overload in results exploration, promote clearer default views, add saved views or bookmarks for repeat investigation patterns, and make export and transform workflows easier to understand before download.
- [Template Editor Rework](./specs/template-editor-rework.md) — Move template creation and editing out of blocking modal overlays into an inline workspace with clearer hierarchy, side-by-side builder and preview, and stronger continuity with the broader Templates route.
- [Integrated AI Assistant Panel](./specs/ai-assistant-panel.md) — Replace modal-only AI preview, generation, and debugging flows with a persistent, collapsible, route-aware assistant panel embedded into job submission, templates, and results workflows after the primary jobs, results, and templates layouts settle.
- [Toast Notification System](./specs/toast-notification-system.md) — Introduce a global notification layer for success, error, loading, and progress feedback so transient operations stop relying on `alert()`, `confirm()`, `console.error`, and inconsistent inline messaging.
- Onboarding and Discoverability Expansion — Replace the heavy first-run modal with lighter progressive onboarding, expand guidance beyond the job form, and make command palette, shortcuts, and route-specific help discoverable without prior knowledge.
- Mobile Experience Pass — Make job monitoring, form actions, results inspection, and automation interactions touch-friendly, improve status readability at small sizes, and provide a mobile-accessible alternative to keyboard-first navigation.
- Zero-Friction First Run and Empty-State Resilience — Ensure fresh local startup never requires hidden env toggles like `PROXY_POOL_FILE=`, replace dead-end empty states with guided recovery steps, and make setup/runtime problems visible inside the product instead of only in terminal logs.

## Later / Deprioritized

- Resume export outcome inspection across API, Web UI, CLI, and MCP after the core Web operator workflows above stop imposing major usability cost.
- Resume watch outcome and check-history inspection across the primary operator surfaces after the Automation Hub redesign clarifies where that information should live in the Web UI.
- Keep docs and examples aligned with major UI workflow changes and future parity cuts so the canonical operator workflow stays current as surfaces evolve.
- Polish cross-surface contract consistency discovered during workflow redesign, including pagination, filters, envelope naming, and generated client/doc sync where needed.

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level Web route model (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), but allow sub-routing or in-route navigation inside complex surfaces when that materially improves operator comprehension.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
