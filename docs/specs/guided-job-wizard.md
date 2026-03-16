# Guided Job Submission Wizard

**Status:** Roadmap / Now  
**Primary surface:** Web UI `/jobs/new`

## Summary

Replace the current long-form job submission experience with a guided, step-based wizard for scrape, crawl, and research jobs. The wizard should reduce scrolling, clarify what belongs together, and still preserve an **Expert mode** for advanced operators who prefer a full-form editing surface.

The current route already has the right top-level placement. The problem is flow design, not route existence.

## Problems This Solves

- Submission forms are too long and require excessive scrolling.
- Advanced options are hidden behind generic `<details>` blocks with weak information scent.
- AI-related help is separated into modals rather than integrated into the flow.
- Operators do not get a clear “review before submit” checkpoint.
- Current loading feedback is minimal and does not make submission feel confident.

## Product Decisions

- Keep `/jobs/new` as the canonical job creation route.
- Make the wizard the default experience.
- Preserve an **Expert mode** toggle for dense editing.
- Reuse the existing `useFormState` and current submit handlers as the canonical data layer.
- Do not duplicate business rules across multiple form implementations if extraction into shared sections is possible.
- If hidden form files make a clean step extraction difficult, wrap existing forms incrementally rather than rewriting everything in one cut.

## Goals

- Break job creation into logical, low-cognitive-load steps.
- Reduce perceived complexity for first-time and occasional operators.
- Keep advanced control available without forcing it on everyone.
- Add clear validation gates and a final review step.
- Preserve presets and existing job-type distinctions.
- Make the flow ready for integration with the AI Assistant Panel.

## Non-Goals

- Changing backend submission contracts.
- Removing power-user controls.
- Turning job creation into multiple top-level routes.
- Rewriting every field definition from scratch if the existing forms can be refactored into sections.

## Wizard Model

Use a shared 4-step structure across job types:

1. **Basics**
2. **Runtime**
3. **Extraction**
4. **Review & Submit**

### Step 1: Basics

Purpose: define what the operator wants to do.

**Shared**
- job type selector: scrape / crawl / research
- preset selection entry point
- short workflow explanation

**Scrape**
- target URL

**Crawl**
- start URL
- crawl scope summary

**Research**
- research query

### Step 2: Runtime

Purpose: configure how the job will run.

Suggested fields:
- fetcher choice / Playwright toggle
- headless toggle
- timeout
- device / browser execution settings if applicable
- auth profile
- headers / cookies / query params
- login settings
- webhook settings if essential enough for guided flow

Group advanced runtime options into clearly labeled sub-panels, not generic unlabeled collapsibles.

### Step 3: Extraction

Purpose: define what should be produced.

Suggested fields:
- extraction template selection
- validation toggle
- AI extraction settings
- processors / transformers where relevant
- research-specific AI or agentic instructions
- preview hooks for template and AI assistance

### Step 4: Review & Submit

Purpose: give operators confidence before execution.

Show:
- job type
- URL or query
- fetcher/runtime summary
- extraction summary
- advanced flags that materially change behavior
- warnings for risky or incomplete configurations
- final submit action

## Expert Mode

Add an explicit toggle near the wizard header:

- **Guided mode**: step-based flow
- **Expert mode**: full form, existing dense editing surface

Rules:
- switching modes must preserve current entered values
- expert mode should not be a separate route
- persist the operator’s last used mode in local storage

Suggested key:

```ts
const JOB_CREATION_MODE_KEY = "spartan.job-creation.mode";
```

## State Model

Keep the current form controller as the source of truth.

```ts
type WizardStepId = "basics" | "runtime" | "extraction" | "review";

interface JobWizardUIState {
  activeStep: WizardStepId;
  completedSteps: WizardStepId[];
  expertMode: boolean;
  validationErrors: Partial<Record<WizardStepId, string[]>>;
}
```

Recommended implementation:

- `JobSubmissionContainer` owns wizard/expert mode orchestration
- step components receive slices of `FormController`
- submit handlers remain:
  - `onSubmitScrape`
  - `onSubmitCrawl`
  - `onSubmitResearch`

## Validation Rules

### Minimum required validation

- **Scrape:** URL required
- **Crawl:** URL required
- **Research:** query required

### Step gating

- Users cannot advance to the next step if required fields for the current step are invalid.
- Validation errors should appear inline and in a compact summary panel at the top of the step.
- The Review step should surface unresolved warnings without forcing every warning to block submission.

## Draft Persistence

Persist unfinished form progress per job type so accidental route changes are less destructive.

```ts
type JobType = "scrape" | "crawl" | "research";
const JOB_DRAFT_KEY_PREFIX = "spartan.job-draft";
```

Recommended behavior:

- autosave on significant field changes
- restore draft on revisit to `/jobs/new`
- allow explicit “Reset draft” action
- clear draft on successful submission unless product decides to preserve as a quick rerun seed

## Layout

Recommended page structure:

1. Wizard header
   - title
   - mode toggle
   - step indicator
   - save state text such as “Draft saved”

2. Primary step panel
   - current step content
   - clear grouping and helper copy

3. Secondary sidebar
   - presets
   - contextual AI actions
   - summary of selected configuration so far

4. Sticky footer actions
   - Back
   - Next
   - Submit on final step

## Step Indicator

Use a clear stepper with labels, not unlabeled dots.

Suggested labels:
- Basics
- Runtime
- Extraction
- Review

Allow backward navigation to completed steps. Avoid free-jumping to invalid future steps.

## Integration with Existing Components

Current context shows:
- `JobSubmissionContainer`
- lazy-loaded `ScrapeForm`, `CrawlForm`, `ResearchForm`

Preferred implementation path:

1. Extract field groups from the existing form components into smaller sections if practical.
2. Recompose those sections into wizard steps.
3. If extraction is too invasive in one pass, wrap existing forms and progressively hide/reveal field groups by step.

Do not fork three entirely separate wizard stacks if a shared shell can host job-type-specific content.

## Loading and Submission Feedback

- Replace generic `Submitting...` with stronger state feedback.
- On submit, show a progress state in the sticky action area.
- Pair successful submission with the Toast Notification System once available.
- After success, navigate to `/jobs` and preserve a success confirmation.

## Onboarding Hooks

Update tour targets so onboarding can teach:

- job-type selection
- wizard steps
- expert mode
- presets
- AI assistance entry points

## Accessibility

- Stepper must announce current step and progress.
- Validation errors must be linked to fields when possible.
- Keyboard users must be able to move through the flow without modal traps.
- Sticky action bars must not obscure focused fields.

## Responsive Behavior

- On small screens, stepper can collapse into “Step X of 4”.
- Sidebar content should move below the primary panel.
- Keep navigation buttons full-width or large enough for touch.

## Acceptance Criteria

- Operators no longer need to scroll through one long page for the default flow.
- Required fields are validated step by step.
- Expert mode is available and preserves data when toggled.
- Drafts survive accidental navigation.
- Review & Submit gives a confident preflight summary.
- The architecture remains compatible with the existing form state and submission handlers.
