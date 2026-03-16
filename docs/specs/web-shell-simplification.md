# Web Shell Simplification

**Status:** Roadmap / Now  
**Primary surfaces:** All top-level Web UI routes

## Summary

Reduce the global chrome so each route spends its first screen on work, not repeated framing.

The current shell combines a large global masthead with a second route intro, repeated metrics, and repeated action buttons. The result is a product that feels heavier than it is and hides the actual workflow below duplicated context.

## Problems This Solves

- Repeated masthead plus route intro burns above-the-fold space.
- Metrics and CTAs are repeated even when they are not the main thing the user came to do.
- Route content starts too low on the page.
- The jobs detail route repeats job history beneath results, weakening focus.
- Settings and templates inherit the same large-shell treatment even when they need dense workspace room.

## Product Decisions

- Keep the current top-level routes.
- Preserve a recognizable global shell, but make it thinner and more consistent.
- Prefer route-owned context over app-wide decorative framing.
- Allow each route to opt into the minimal shell by default and only promote metrics/actions that matter for that route.
- No backwards-compatibility shims are required for the old stacked shell.

## Goals

- Move meaningful route content above the fold.
- Remove redundant page copy and repeated metrics.
- Make navigation, global actions, and route context easier to parse.
- Support future sub-navigation for complex routes such as `/automation` and `/settings`.
- Establish cleaner layout rules for desktop and mobile.

## Non-Goals

- Rebranding the product.
- Replacing the theme system.
- Rewriting every route in one PR if the shell can be extracted cleanly first.

## Target Layout Model

### Global shell

- Compact top bar
  - product identity
  - primary navigation
  - one global action slot
  - theme toggle / utility actions
- Optional compact signal row only on routes where live queue state materially helps
- No second hero-like introduction by default

### Route header

Each route can render a lightweight route header containing:

- route title
- one-sentence value statement only when needed
- route-local actions
- route-local sub-navigation when present

This replaces the current pattern of a large shared masthead followed by a large route intro card.

## Route-Specific Guidance

### `/jobs`

- Keep a concise route title and only the most useful live signals.
- Land directly in the monitoring surface.

### `/jobs/:id`

- Prioritize the result surface.
- Remove the large lower repeated jobs section from the default layout.
- Replace it with a lighter “jump back to jobs” affordance plus related actions.

### `/templates`

- Give more vertical room to the template workspace and list/detail flow.

### `/automation`

- Use the route header for sub-navigation, not long prose.

### `/settings`

- Treat settings as a control center with subsections, not as another dashboard hero.

## Suggested React Architecture

- Extract shell responsibilities into focused primitives:
  - `AppTopBar`
  - `RouteHeader`
  - `RouteSubnav`
  - optional `RouteSignals`
- Stop treating `PageIntro` as the default route wrapper.
- Let routes compose their own header density.

## Implementation Notes

In current code, `web/src/App.tsx` renders both the global `app-shell` and `PageIntro` across routes. The cutover should:

1. minimize the height of `app-shell`
2. remove default duplication between `app-shell` and `PageIntro`
3. replace route intros with a compact `RouteHeader`
4. move route-specific signals into each route layout instead of repeating the same five-pill summary everywhere

## Styling Guidance

- Prefer class-based semantic layout tokens over route-specific inline spacing.
- Reduce vertical padding in the global shell.
- Reserve high-emphasis surfaces for the operator’s current task, not the route frame.
- Ensure the top bar stays usable on smaller widths without wrapping into a giant block.

## Acceptance Criteria

- Every top-level route shows useful workflow content above the fold without duplicated introductory cards.
- The global shell becomes meaningfully shorter.
- Route-local context is clear without repeating the same messaging twice.
- Job detail no longer defaults to a heavy repeated recent-jobs section below results.
- The new shell leaves room for in-route sub-navigation and future mobile treatment.
