# Integrated AI Assistant Panel

**Status:** Roadmap / Now  
**Primary surfaces:** Web UI job submission, templates, and results

## Summary

Replace fragmented modal-only AI workflows with a persistent, collapsible, route-aware assistant panel that stays embedded inside the operator workflow.

Today, AI capabilities exist, but they are split across separate modal entry points such as preview, template generation, debugging, and export-shape assistance. That makes AI feel bolted on rather than operationally useful.

This spec defines one assistant surface that adapts to the current route and context.

## Problems This Solves

- AI actions are isolated in modals that interrupt flow.
- Template editing, job creation, and results analysis each have separate AI entry patterns.
- Operators cannot easily keep AI guidance visible while editing.
- AI capability is hard to discover and feels disconnected from the rest of the product.

## Product Decisions

- Keep the existing top-level routes.
- Introduce a shared assistant shell rather than a separate AI route.
- The panel is **contextual**, not a generic always-on chat toy.
- Reuse existing AI endpoints and components where possible.
- If existing AI logic is trapped inside modal components, extract the underlying behavior and embed it in the new panel.

## Goals

- Create one persistent AI surface across the main Web workflows.
- Keep AI help visible while the user edits forms, templates, or results settings.
- Make AI context-aware so it can act on the current page state.
- Reduce modal churn and route interruption.
- Support collapse/expand behavior and mobile fallback.

## Non-Goals

- Designing a fully open-ended chatbot unrelated to the operator workflow.
- Adding brand-new backend AI capabilities before reusing current ones.
- Replacing all route-specific UI with AI.

## Assistant Contexts

The assistant should adapt to the current route.

| Route / Area | Assistant purpose | Example actions |
| --- | --- | --- |
| `/jobs/new` | Help configure extraction and runtime settings | preview extraction, explain fields, suggest template strategy, help refine AI extraction prompt |
| `/templates` | Help create, debug, and improve templates | generate template, suggest selectors, debug template failures, explain normalization |
| `/jobs/:id` | Help interpret results and exports | summarize results, suggest transformation or export shape, explain anomalies |

## Interaction Model

### Desktop

- Right-side collapsible panel
- Default state can be collapsed but visible via a tab or icon rail
- Width should be resizable or fixed to a sensible range such as 360–420px
- Panel state persists per user

### Mobile

- Bottom sheet or full-screen sheet
- Opened via explicit action button
- Must not depend on keyboard shortcuts

## Panel Sections

A route-aware panel should generally include:

1. **Header**
   - context title
   - current route label
   - collapse / close controls

2. **Context summary**
   - current URL/query/template/job/result selection
   - concise snapshot of what the assistant is acting on

3. **Suggested actions**
   - task-oriented buttons, not just a blank text box

4. **Conversation / output area**
   - results, recommendations, generated content, or explanations

5. **Apply actions**
   - apply generated template
   - copy prompt
   - insert shape config
   - update field values
   - open related route content

## State Model

Create a shared assistant context model.

```ts
type AssistantSurface = "job-submission" | "templates" | "results";

type AssistantContext =
  | {
      surface: "job-submission";
      jobType: "scrape" | "crawl" | "research";
      url?: string;
      query?: string;
      templateName?: string;
      formSnapshot: Record<string, unknown>;
    }
  | {
      surface: "templates";
      templateName?: string;
      templateSnapshot?: Record<string, unknown>;
      selectedUrl?: string;
    }
  | {
      surface: "results";
      jobId: string;
      resultFormat: string;
      selectedResultIndex: number;
      resultSummary?: string | null;
    };
```

## Suggested React Architecture

```ts
interface AIAssistantController {
  isOpen: boolean;
  surface: AssistantSurface | null;
  context: AssistantContext | null;
  open: (context: AssistantContext) => void;
  close: () => void;
  toggle: () => void;
  setContext: (context: AssistantContext) => void;
}
```

Recommended pieces:

- `AIAssistantProvider`
- `useAIAssistant()`
- `AIAssistantPanel`
- route adapters:
  - `JobSubmissionAssistantSection`
  - `TemplateAssistantSection`
  - `ResultsAssistantSection`

## Existing Components to Consolidate

Visible context already shows modal-style AI surfaces such as:

- `AIExtractPreview`
- `AITemplateGenerator`
- `AITemplateDebugger`
- `AIExportShapeAssistant`

Implementation guidance:

1. Inventory all AI modal components in the repo.
2. Extract shared logic and request handling from those modal shells.
3. Re-host that logic inside the assistant panel.
4. Keep temporary modal wrappers only if needed during migration, but the roadmap target is a persistent panel.

## Apply/Commit Rules

AI output must never silently mutate the product state.

Allowed patterns:

- explicit “Apply” button
- explicit “Insert into field” button
- explicit “Replace current config” confirmation
- explicit copy-to-clipboard action

Disallowed pattern:

- assistant auto-overwrites fields without an operator confirmation step

## Persistence

Persist panel UI state separately from content state.

Suggested keys:

```ts
const AI_ASSISTANT_OPEN_KEY = "spartan.ai-assistant.open";
const AI_ASSISTANT_WIDTH_KEY = "spartan.ai-assistant.width";
```

Do **not** persist sensitive prompt/result content unless the product explicitly wants that behavior.

## Acceptance Criteria

- Operators can use AI without leaving their current workflow.
- At least job submission, templates, and results each expose context-aware AI actions.
- Existing modal-based AI tools are consolidated or clearly on a migration path toward consolidation.
- AI-generated changes require explicit operator confirmation.
- The panel is usable on desktop and mobile.
