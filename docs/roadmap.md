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

- Mobile Experience Pass — Makes the shell, route headers, jobs dashboard, new-job wizard, results reader, automation hub, templates workspace, settings panels, and command/help surfaces genuinely usable at `≤720px` with touch-friendly controls, preserved visible onboarding affordances, and overflow-safe layouts.
- Integrated AI assistant panel now gives `/jobs/new`, `/templates`, and `/jobs/:id` one persistent, collapsible, route-aware AI rail with explicit apply actions, replacing modal-only preview, generation, debugging, shape, and refinement entry points in the core operator workflows.
- Template editor rework now makes `/templates` a real inline workspace with a persistent library rail, center editor, right-side preview/AI tools, and inline visual-builder mode instead of blocking modal-first authoring.
- Results view system overhaul now keeps `/jobs/:id` centered on one dominant reader, moves comparison/tree/transform/visualization into an explicit secondary layer, replaces the export button strip with guided export, and preserves jobs-route continuity without reintroducing extra below-the-fold job clutter.
- Web shell simplification now uses a compact global top bar, route-owned headers, and reduced repeated metrics so `/jobs` lands directly on the scan-first monitoring surface, `/jobs/:id` stays result-focused, and top-level routes spend their first screen on real work instead of stacked framing.
- Automation hub navigation now gives `/automation` explicit section switching and stable deep links for batches, chains, watches, exports, and webhook deliveries, so operators no longer have to scroll a stacked mega-page just to change automation modes.
- Run history observability now spans API, Web UI, CLI, and MCP with recent execution inspection, explicit batch queue progression, recent failed-run views, and structured failure context so operators can understand outcomes without digging through host-local files or internal state.
- Watch management now spans API, Web UI, CLI, and MCP with `watch_list`, `watch_get`, `watch_create`, `watch_update`, `watch_delete`, and `watch_check`, so operators and agents can manage stored watches and run manual checks without falling back to REST-only flows.
- Webhook delivery inspection now spans API, Web UI, CLI, and MCP with sanitized URL/error output, so operators can debug retries and failures from `spartan webhook deliveries ...` and `webhook_delivery_get` without reading host-local files.
- Batch management now spans API, Web UI, CLI, and MCP with authoritative list/detail/cancel flows plus MCP batch submission tools, so operators and agents can create, enumerate, inspect, and stop persisted batches without falling back to REST-only workflows or browser-local tracking.

## After

- Zero-Friction First Run and Empty-State Resilience — Keep fresh local startup free of optional-subsystem footguns, replace dead-end empty states with guided recovery steps, and make setup/runtime problems visible inside the product instead of only in terminal logs.

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
