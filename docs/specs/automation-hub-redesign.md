# Automation Hub Redesign

**Status:** Implemented  
**Primary surface:** Web UI `/automation`

## Summary

Turn `/automation` from a long stacked mega-page into a true orchestration hub with explicit in-route navigation, stable deep links, and focused workspace sections for batches, chains, watches, export schedules, and webhook deliveries.

## Problems This Solves

- Multiple major workflows are stacked into one long page.
- Batch creation dominates the route while other automation surfaces are effectively below-the-fold afterthoughts.
- Jump links are not strong enough information architecture.
- Operators have to scroll extensively to switch between creation, monitoring, and maintenance tasks.
- Deep-linking and “come back to where I was” behavior are weak.

## Product Decisions

- Keep `/automation` as a top-level route.
- Add in-route sub-navigation with deep links.
- Each automation area gets a focused landing state and quick actions.
- Support list/detail behavior within each automation section.
- No attempt to preserve the stacked page layout once the cutover lands.

## Goals

- Make automation learnable.
- Reduce scrolling and context loss.
- Separate creation flows from monitoring/detail flows.
- Give every automation surface a clear ownership zone.
- Preserve parity-backed capabilities while improving how they are presented.

## Non-Goals

- Removing batches, chains, watches, export schedules, or webhook deliveries.
- Converting automation into multiple top-level routes.
- Changing backend contracts before the IA cutover proves it is necessary.

## Information Architecture

Recommended primary sections:

- `/automation/batches`
- `/automation/chains`
- `/automation/watches`
- `/automation/exports`
- `/automation/webhooks`

If route-level subpaths are too disruptive in the first cut, use query-state or local sub-navigation with the exact same information model so the final cutover can become route-backed cleanly.

## Section Requirements

### Batches

- Overview of recent batches
- Clear create batch action
- Batch type selector
- Recent submissions and status summary
- Batch detail drill-down without losing section context

### Chains

- List of chains
- Create/edit chain action
- Last run status and next action cues

### Watches

- List of watches
- Recent outcomes and latest change status
- Add watch action and detail/edit flow

### Exports

- Export schedule list
- Quick enable/disable status
- Creation/edit flow
- Outcome summary once observability work resumes

### Webhooks

- Delivery inspection
- Filters that feel native to this sub-surface
- Retry/failure context where supported

## Layout Model

### Desktop

- Route header
- Horizontal or vertical sub-navigation
- Main content region for the active section
- Optional right rail for section-specific quick actions or recent signals

### Mobile

- Section switcher as segmented control or compact menu
- No giant stacked page
- Section actions placed near the top of the active view

## State and Deep-Linking

Persist and restore:

- active automation section
- current filters within a section
- selected detail item where useful

Suggested key shape:

```ts
type AutomationSection = "batches" | "chains" | "watches" | "exports" | "webhooks";
```

Prefer URL state over local-only state so links remain shareable and revisitable.

## Suggested Component Plan

- `AutomationLayout`
- `AutomationSubnav`
- `AutomationSectionHeader`
- route/section adapters for batches, chains, watches, exports, and webhooks

Existing containers can remain the data/workflow owners initially, but they should render inside the new layout instead of being stacked unconditionally.

## Acceptance Criteria

- `/automation` no longer renders as a long stacked mega-page.
- Each automation capability has a clearly named section with direct navigation.
- Users can revisit a specific automation subsection via a stable URL or equivalent stateful deep link.
- Creation and monitoring are easier to separate mentally.
- The route becomes faster to scan and easier to learn.
