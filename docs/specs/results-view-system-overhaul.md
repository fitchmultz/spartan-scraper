# Results View System Overhaul

**Status:** Recently Completed  
**Primary surface:** Web UI `/jobs/:id`

## Summary

Refocus result inspection around a clearer default path: inspect the selected item, understand what changed, and export with confidence. The current experience exposes too many modes and actions at once, which increases cognitive load before the user has even looked at the result.

## Problems This Solves

- The primary toolbar combines mode switching, filtering, search, and five export buttons at once.
- Result detail competes with comparison and transform tools too early.
- Job detail repeats recent jobs below the result view, weakening focus.
- Export options lack preview or explanation of when each format is right.
- Returning to the jobs list loses context.

## Product Decisions

- Keep `/jobs/:id` as the canonical result route.
- Make one default result mode clearly primary.
- Reframe comparison, tree, transform, and visualization as secondary tools.
- Keep export support broad, but reduce default visual noise.

## Goals

- Shorten the time from opening a job to understanding its output.
- Promote a stronger default result reader.
- Reduce toolbar overload.
- Add clearer export intent and export previews where feasible.
- Preserve result-to-jobs continuity.

## Proposed Interaction Model

### Default path

1. result summary/header
2. selected item list or navigator
3. primary detail panel
4. optional secondary tools

### Secondary tools

Move non-primary capabilities behind a clearer secondary layer:

- Compare
- Transform
- Tree/structure view
- Research visualization

These can live in a segmented control, drawer, or tool switcher, but they should not compete equally with the main reading surface on first paint.

## Export Model

Replace the always-expanded export button strip with:

- one primary “Export” action
- format chooser inside a menu or side sheet
- short explanation per format
- preview of the export scope if feasible

## Continuity Requirements

- Preserve jobs-route context when navigating back.
- Avoid rendering a full recent-jobs section below results by default.
- Keep quick navigation to adjacent jobs lightweight.

## Acceptance Criteria

- The result route has one clearly dominant default experience.
- Secondary tools are still available, but not equally loud on first paint.
- Export feels guided instead of button-spammed.
- Result detail remains the center of the route.
- Navigating back to jobs restores the prior monitoring context.
