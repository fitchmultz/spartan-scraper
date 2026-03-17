# Integrated AI Assistant Panel

**Status:** Completed
**Primary surfaces:** Web UI job submission, templates, and results

## Summary

Replace fragmented modal-only AI workflows with a persistent, collapsible, route-aware assistant panel that stays embedded inside the operator workflow.

Today, AI capabilities exist, but they are split across separate modal entry points such as preview, template generation, debugging, and export-shape assistance. That makes AI feel bolted on rather than operationally useful.

This spec defined one assistant surface that adapts to the current route and context.

## Problems This Solved

- AI actions were isolated in modals that interrupted flow.
- Template editing, job creation, and results analysis each had separate AI entry patterns.
- Operators could not keep AI guidance visible while editing.
- AI capability was harder to discover and felt disconnected from the rest of the product.

## Delivery Notes

- `AIAssistantProvider` now owns assistant open/collapse state, route context, and width persistence.
- `/jobs/new`, `/templates`, and `/jobs/:id` each mount route-specific assistant adapters inside a shared assistant shell.
- Job submission now embeds AI extraction preview and explicit apply actions beside the wizard and expert forms.
- Templates now keep preview, generation, and debugging inside the persistent assistant rail instead of route-local rail shell logic or modal assumptions.
- Results now route export-shape and research-refinement flows through the persistent assistant panel instead of modal-only entry points.
- App-level quick-start actions now open the persistent assistant on the appropriate route instead of launching modal overlays.
- AI-generated output still requires explicit operator apply or copy actions.

## Outcome

The web app now has one persistent, route-aware AI surface across the main operator workflows without changing the top-level route model.
