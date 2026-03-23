# Web UI/UX Audit

**Date:** 2026-03-16  
**Scope:** Web UI route shell, jobs, job creation, results, templates, automation, settings, onboarding, and feedback patterns  
**Method:** repo code inspection plus live walkthroughs against a local app instance and screenshot review

## Review Inputs

### Live app walkthrough

- Started the local server and verified the Web UI on `http://127.0.0.1:5173`.
- Submitted a real scrape job for `https://example.com`.
- Visited `/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings/authoring`.
- Captured screenshots under `output/playwright/ui-ux-audit-2026-03-16/screenshots/`.

### Evidence screenshots

- `output/playwright/ui-ux-audit-2026-03-16/screenshots/initial-home.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/jobs-dashboard.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/new-job-scrape.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/new-job-scrolled.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/job-results-overview.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/job-results-content.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/templates-overview.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/automation-overview.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/automation-stack.png`
- `output/playwright/ui-ux-audit-2026-03-16/screenshots/settings-overview.png`

### Code references sampled during the audit

- `web/src/App.tsx`
- `web/src/components/JobList.tsx`
- `web/src/components/jobs/JobFormSections.tsx`
- `web/src/components/ResultsExplorer.tsx`
- `web/src/components/templates/TemplateManager.tsx`
- `web/src/components/OnboardingFlow.tsx`
- `web/src/components/CommandPalette.tsx`

## What This App Needs To Become A Real UI/UX Powerhouse

Meaningful changes, not fluff:

1. **A thinner, route-first shell** so operators see useful work immediately instead of repeated masthead, metrics, and secondary intros on every page.
2. **A guided job creation flow** that feels confident for normal users while preserving an expert mode for power users.
3. **A proper operations center** for jobs and automation, with scan-first monitoring, queue clarity, failures that are easy to triage, and stable deep links.
4. **One coherent AI surface** embedded in workflows instead of separate modal tricks scattered across routes.
5. **A lower-cognitive-load results experience** with fewer competing views, clearer exports, and better continuity between result detail and job history.
6. **Inline authoring workspaces** for templates and configuration tools instead of blocking modal editing.
7. **Consistent product feedback**: toasts, confirmations, loading states, and recovery guidance instead of `alert()`, `confirm()`, console-only errors, and silent failures.
8. **Progressive onboarding and empty-state guidance** that teaches as the user works rather than front-loading a giant welcome moment.
9. **A first-run experience that never requires hidden terminal incantations** just to boot the product locally.

## Findings

### 1. The shell spends too much of every route on repeated framing

**Observed:** `jobs-dashboard.png`, `templates-overview.png`, `automation-overview.png`, and `settings-overview.png` all show a large global masthead followed immediately by a second route intro card with overlapping copy and repeated metrics.

**Why it hurts:** Above-the-fold space is being spent on repeated explanation instead of the actual work surface. This makes the app feel heavier and slower than it is.

**Code evidence:** `web/src/App.tsx` renders both the global `app-shell` and route-specific `PageIntro` panels for most routes.

**Roadmap coverage:** [Web Shell Simplification](web-shell-simplification.md)

### 2. The first-run experience is too blocking

**Observed:** `initial-home.png` shows a large welcome modal that obscures the product before the user can assess the current state.

**Why it hurts:** This is a hard interruption before trust is established. It also competes with the real jobs dashboard rather than complementing it.

**Code evidence:** `WelcomeModal` and `OnboardingFlow` are mounted globally from `web/src/App.tsx`, and `web/src/components/OnboardingFlow.tsx` is narrowly focused on the job form.

**Roadmap coverage:** Onboarding and Discoverability Expansion

### 3. Job monitoring is functional but too dense and ID-heavy

**Observed:** `jobs-list-detail.png` shows rows dominated by UUIDs, flat metadata, and terse actions. `job-results-content.png` shows recent jobs repeated beneath result detail even after the operator has already opened a specific result.

**Why it hurts:** Operators must parse raw text to understand state instead of scanning for failures, active work, and recent completions.

**Code evidence:** `web/src/components/JobList.tsx` renders machine IDs first and relies heavily on inline-styled metadata rows.

**Roadmap coverage:** [Job Monitoring Dashboard](job-monitoring-dashboard.md), [Results View System Overhaul](results-view-system-overhaul.md)

### 4. Job creation still behaves like a long form with hidden drawers

**Observed:** `new-job-scrape.png` shows the route split between primary form and a side rail of AI/preset actions, but `new-job-scrolled.png` shows that much of the real power is hidden inside unlabeled advanced accordions.

**Why it hurts:** The default flow looks simple at the top, but the real configuration model is still long, scroll-heavy, and reveal-on-demand in a way that hides capability instead of clarifying it.

**Code evidence:** `web/src/components/jobs/JobFormSections.tsx` wraps advanced sections in plain `<details>` blocks.

**Roadmap coverage:** [Guided Job Submission Wizard](guided-job-wizard.md), [Integrated AI Assistant Panel](ai-assistant-panel.md)

### 5. The results surface asks the user to juggle too many modes and export buttons at once

**Observed:** `job-results-overview.png` shows multiple view tabs, search, status filter, and five export buttons all in one toolbar before the user even inspects content.

**Why it hurts:** The product is exposing implementation capability instead of guiding the user through the most common interpretation/export path.

**Code evidence:** `web/src/components/ResultsExplorer.tsx` presents multiple modes and an always-on export button strip in the primary toolbar.

**Roadmap coverage:** [Results View System Overhaul](results-view-system-overhaul.md), [Integrated AI Assistant Panel](ai-assistant-panel.md)

### 6. Automation is the biggest current UX overload point

**Observed:** `automation-overview.png` and `automation-stack.png` show batch creation, chain management, watch monitoring, export schedules, and webhook deliveries sharing one long route.

**Why it hurts:** The route mixes creation, monitoring, and maintenance concerns for multiple systems at once. It is powerful, but not learnable.

**Code evidence:** `/automation` is composed as a stacked surface in `web/src/App.tsx`, with multiple large containers mounted in a single page.

**Roadmap coverage:** [Automation Hub Redesign](automation-hub-redesign.md)

### 7. Templates are better than before but still modal-fragmented

**Observed:** `templates-overview.png` shows multiple AI entry points and management actions on one route; code inspection shows the editor itself still opens in modal overlays.

**Why it hurts:** Template authoring is a deep workflow and deserves workspace-level continuity, not modal interruption.

**Code evidence:** `web/src/components/templates/TemplateManager.tsx` uses `.modal-overlay` editors and AI debugger overlays.

**Roadmap coverage:** [Template Editor Rework](template-editor-rework.md), [Integrated AI Assistant Panel](ai-assistant-panel.md)

### 8. Settings is acting as a catch-all instead of a clear control center

**Observed:** `settings-overview.png` shows templates, render profiles, pipeline JS, proxy pool, and retention all presented together.

**Why it hurts:** The route mixes authoring, operations, and maintenance tools without strong grouping or a clear mental model.

**Roadmap coverage:** [Web Shell Simplification](web-shell-simplification.md), Onboarding and Discoverability Expansion

### 9. Feedback patterns are inconsistent and low-trust

**Observed in code:** the web app still uses `alert()`, `confirm()`, and console-only error handling across forms and admin actions.

Examples:

- `web/src/components/ScrapeForm.tsx`
- `web/src/components/CrawlForm.tsx`
- `web/src/components/ResearchForm.tsx`
- `web/src/components/BatchForm.tsx`
- `web/src/components/batches/BatchContainer.tsx`
- `web/src/App.tsx`

**Why it hurts:** This makes the app feel unfinished, interrupts flow, and hides recoverable outcomes.

**Roadmap coverage:** [Toast Notification System](toast-notification-system.md)

### 10. Command palette and onboarding are under-discovered relative to their importance

**Observed:** the primary UI shows route buttons but no durable command palette entry point. The welcome copy mentions the command palette, yet the feature depends on remembered shortcuts.

**Code evidence:** `web/src/components/CommandPalette.tsx` exposes keyboard-first actions; `web/src/components/OnboardingFlow.tsx` spends most of its guidance budget on the job form.

**Roadmap coverage:** Onboarding and Discoverability Expansion

### 11. The UI system is visually inconsistent under the hood

**Observed in code:** `web/src/components/JobList.tsx`, `web/src/components/OnboardingFlow.tsx`, and other components use inline styles and hard-coded colors instead of shared semantic classes and theme tokens.

**Why it hurts:** Inconsistency slows redesign work, weakens theming, and makes the product feel assembled rather than designed.

**Roadmap coverage:** [Web Shell Simplification](web-shell-simplification.md), [Toast Notification System](toast-notification-system.md)

### 12. First-run/local startup resilience and optional-capability framing are now product-grade

**Resolved after audit follow-up:** fresh local startup now boots cleanly with AI, proxy pooling, and retention off by default, and the docs/onboarding/help copy consistently explain those systems as optional capabilities rather than prerequisites or failures.

**Why it mattered:** First-run friction is UX, not just plumbing. Operators should reach a working scrape without hidden environment overrides or setup-first reading, then enable AI, proxy pooling, or retention later only if the workflow benefits.

**Roadmap coverage:** Resolved via Zero-Friction First Run and optional-capability standardization follow-through.

## Priority Call

The most leverage comes from fixing structure before polishing details:

1. Thin the shell.
2. Split automation into a real hub.
3. Rebuild job creation around a guided wizard.
4. Rebuild job monitoring around scan-first operations.
5. Collapse scattered AI affordances into one contextual assistant.

Once those are in place, results, templates, feedback, onboarding, and mobile work can land with less churn.
