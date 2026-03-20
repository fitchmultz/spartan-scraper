# Specs Index

These specs back the roadmap and capture the UI/UX direction for the current cutover and the next operator-facing work.

## Audit

- [Web UI/UX Audit](web-ui-ux-audit.md) — findings from code inspection plus live app walkthroughs and screenshot-based review.

## Next Up

- [Verified Job Promotion Flow](job-to-automation-promotion.md) — let operators turn a completed, trusted job into a reusable template, watch, or export schedule without re-entering known-good configuration.
- [Promotion Flow Deterministic Regression Coverage](promotion-flow-deterministic-regression.md) — define system-first regression coverage and targeted browser proof for the promotion cutover.

## Recently Completed

- Web UI optional-capability follow-through — first-run docs, onboarding, help text, and Settings copy now consistently treat AI, proxy pooling, and retention as optional, off-by-default capabilities instead of prerequisites.
- [Web Shell Simplification](web-shell-simplification.md) — thinner global chrome so routes spend their first screen on work instead of repeated framing.
- [Guided Job Submission Wizard](guided-job-wizard.md) — `/jobs/new` now uses a guided 4-step wizard with Expert mode, per-job draft persistence, and review-before-submit.
- [Job Monitoring Dashboard](job-monitoring-dashboard.md) — `/jobs` now uses a scan-first monitoring dashboard with lanes, stronger progress treatment, and jobs-route return-context preservation.
- [Results View System Overhaul](results-view-system-overhaul.md) — `/jobs/:id` now defaults to one dominant reader, with secondary tools and guided export moved behind quieter drawers.
- [Automation Hub Redesign](automation-hub-redesign.md) — `/automation` now uses explicit in-route section navigation with stable deep links for batches, chains, watches, exports, and webhooks.
- [Template Editor Rework](template-editor-rework.md) — `/templates` now runs as an inline authoring workspace with a persistent library rail, center editor, right-side preview/AI tools, and inline visual-builder mode.
- [Integrated AI Assistant Panel](ai-assistant-panel.md) — `/jobs/new`, `/templates`, and `/jobs/:id` now share a persistent, route-aware AI rail instead of modal-first entry points.
- [Toast Notification System](toast-notification-system.md) — the Web UI now has one reusable transient feedback layer for success, error, loading, and progress states.
