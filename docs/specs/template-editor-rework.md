# Template Editor Rework

**Status:** Roadmap / After  
**Primary surface:** Web UI `/templates`

## Summary

Turn template authoring into an inline workspace with persistent context, side-by-side preview, and integrated AI assistance instead of modal editing and fragmented actions.

## Problems This Solves

- Template editing currently happens in modal overlays.
- AI preview, generation, and debugging are fragmented entry points.
- Template list, details, and editing do not feel like one continuous workspace.
- Deep editing tasks lose context when the modal closes.

## Product Decisions

- Keep `/templates` as the template home.
- Prefer a split-pane or list/detail workspace.
- Remove modal-first editing as the default pattern.
- Integrate AI assistance through the shared assistant panel.

## Goals

- Make template authoring feel like a real workspace.
- Keep the template list visible while editing when practical.
- Support side-by-side preview with editable rules.
- Make debugging and generation feel like parts of the same flow.

## Target Layout

### Desktop

- template list or library rail
- selected template editor
- preview/debug rail or tabbed pane
- inline save/state feedback

### Mobile

- stacked list/detail flow
- explicit preview switcher
- no giant modal overlays

## Workspace Sections

- template metadata
- selector rules
- advanced extraction and normalization
- preview target / sample page
- AI suggestions and debugging

## Migration Guidance

- Keep existing template data structures and save APIs.
- Extract the modal editor body into reusable inline sections.
- Re-host AI debugger and generator actions into the shared assistant panel or inline workspace controls.

## Acceptance Criteria

- Template creation and editing no longer depend on modal-first flows.
- The user can edit and preview with stronger continuity.
- AI-assisted generation/debugging feels integrated, not detached.
- The templates route reads as a workspace, not a list plus popups.
