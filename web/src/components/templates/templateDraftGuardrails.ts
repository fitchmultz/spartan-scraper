/**
 * Purpose: Centralize template draft replacement and discard confirmation policy.
 * Responsibilities: Map draft actions to operator-facing confirmation copy so template draft guardrails stay consistent across persistence, mutation, and promotion flows.
 * Scope: Template draft confirmation copy only.
 * Usage: Imported by template draft hooks before calling the shared toast confirmation UI.
 * Invariants/Assumptions: All draft-replacement prompts share the same confirmation framing, and discard prompts distinguish between dirty and clean local drafts.
 */

export type TemplateDraftReplaceIntent =
  | "create"
  | "duplicate"
  | "switch-template"
  | "apply-template"
  | "promotion";

export interface TemplateDraftPromptOverride {
  title?: string;
  reason?: string;
}

export type TemplateDraftReplacementRequest =
  | TemplateDraftReplaceIntent
  | TemplateDraftPromptOverride
  | undefined;

const DEFAULT_REPLACE_TITLE = "Replace the current template draft?";

const REPLACE_REASONS: Record<TemplateDraftReplaceIntent, string> = {
  create:
    "This starts a new local template draft and discards the edits you have not saved yet. Keep the current draft if you still need it.",
  duplicate:
    "This duplicates another saved template into the workspace and discards the edits you have not saved yet. Keep the current draft if you still need it.",
  "switch-template":
    "This opens another saved template and discards the edits you have not saved yet. Keep the current draft if you still need it.",
  "apply-template":
    "This applies another template to the workspace and discards the edits you have not saved yet. Keep the current draft if you still need it.",
  promotion:
    "This verified-job draft will replace the current local template draft. Keep the current draft if you still need those unsaved edits.",
};

export function resolveTemplateDraftReplacementPrompt(
  request?: TemplateDraftReplacementRequest,
) {
  if (!request) {
    return {
      title: DEFAULT_REPLACE_TITLE,
      description:
        "This opens another local template draft and discards the edits you have not saved yet. Keep the current draft if you still need it.",
    };
  }

  if (typeof request === "string") {
    return {
      title: DEFAULT_REPLACE_TITLE,
      description: REPLACE_REASONS[request],
    };
  }

  return {
    title: request.title ?? DEFAULT_REPLACE_TITLE,
    description:
      request.reason ??
      "This opens another local template draft and discards the edits you have not saved yet. Keep the current draft if you still need it.",
  };
}

export function resolveTemplateDraftDiscardPrompt(
  isDirty: boolean,
  options?: TemplateDraftPromptOverride,
) {
  return {
    title: options?.title ?? "Discard the local template draft?",
    description:
      options?.reason ??
      (isDirty
        ? "This removes the in-progress template draft. Your unsaved edits will be lost."
        : "This removes the current local template draft from this tab."),
  };
}
