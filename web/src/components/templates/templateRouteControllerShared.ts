/**
 * Purpose: Share template-route controller types and codec helpers across the split Templates workspace.
 * Responsibilities: Define built-in template names, draft/session shapes, selector draft factories, and normalized draft snapshots used by the route controllers and persistence hooks.
 * Scope: Template route controller state and draft serialization only; network calls and UI presentation stay in sibling files.
 * Usage: Imported by the template route controller hook stack, mutation actions, and editor components.
 * Invariants/Assumptions: Draft sessions snapshot normalized template data, built-in templates are read-only until duplicated, and selector drafts always include a stable local id.
 */

import type {
  JsonldRule,
  NormalizeSpec,
  RegexRule,
  SelectorRule,
  Template,
} from "../../api";
import { deepEqual } from "../../lib/diff-utils";
import {
  formatOptionalJSON,
  parseOptionalJSON,
} from "../settings/settingsAuthoringForm";

export const BUILT_IN_TEMPLATE_NAMES = [
  "article",
  "default",
  "product",
] as const;
type BuiltInTemplateName = (typeof BUILT_IN_TEMPLATE_NAMES)[number];

const EMPTY_SELECTOR: SelectorRule = {
  name: "",
  selector: "",
  attr: "text",
  trim: true,
  all: false,
  required: false,
};

export interface SelectorDraft {
  id: string;
  rule: SelectorRule;
}

export interface TemplateDraftState {
  name: string;
  selectors: SelectorDraft[];
  jsonldText: string;
  regexText: string;
  normalizeText: string;
}

function createDraftId() {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `selector-${Math.random().toString(36).slice(2, 10)}`;
}

function normalizeSelectorRule(rule: SelectorRule): SelectorRule {
  return {
    ...rule,
    name: rule.name?.trim(),
    selector: rule.selector?.trim(),
    attr: rule.attr?.trim() || "text",
    join: rule.join?.trim() || undefined,
  };
}

export function createSelectorDraft(rule?: SelectorRule): SelectorDraft {
  return {
    id: createDraftId(),
    rule: {
      ...EMPTY_SELECTOR,
      ...rule,
    },
  };
}

export function buildDraftFromTemplate(
  template?: Template,
): TemplateDraftState {
  return {
    name: template?.name ?? "",
    selectors: template?.selectors?.length
      ? template.selectors.map((selector) => createSelectorDraft(selector))
      : [createSelectorDraft()],
    jsonldText: formatOptionalJSON(template?.jsonld),
    regexText: formatOptionalJSON(template?.regex),
    normalizeText: formatOptionalJSON(template?.normalize),
  };
}

export function buildTemplateSnapshot(draft: TemplateDraftState): Template {
  let jsonld: JsonldRule[] | undefined;
  try {
    const parsed = parseOptionalJSON<unknown>(
      "JSON-LD rules",
      draft.jsonldText,
    );
    jsonld =
      parsed && Array.isArray(parsed) ? (parsed as JsonldRule[]) : undefined;
  } catch {
    jsonld = undefined;
  }

  let regex: RegexRule[] | undefined;
  try {
    const parsed = parseOptionalJSON<unknown>("Regex rules", draft.regexText);
    regex =
      parsed && Array.isArray(parsed) ? (parsed as RegexRule[]) : undefined;
  } catch {
    regex = undefined;
  }

  let normalize: NormalizeSpec | undefined;
  try {
    const parsed = parseOptionalJSON<unknown>(
      "Normalization settings",
      draft.normalizeText,
    );
    normalize =
      parsed && !Array.isArray(parsed) && typeof parsed === "object"
        ? (parsed as NormalizeSpec)
        : undefined;
  } catch {
    normalize = undefined;
  }

  return {
    name: draft.name.trim(),
    selectors: draft.selectors.map(({ rule }) => normalizeSelectorRule(rule)),
    ...(jsonld ? { jsonld } : {}),
    ...(regex ? { regex } : {}),
    ...(normalize ? { normalize } : {}),
  };
}

export function isBuiltInTemplateName(
  name: string | null | undefined,
): name is BuiltInTemplateName {
  return (
    !!name && BUILT_IN_TEMPLATE_NAMES.includes(name as BuiltInTemplateName)
  );
}

export type DraftSource = "selected" | "create" | "duplicate";

export interface TemplateWorkspaceDraftSession {
  source: DraftSource;
  originalName: string | null;
  selectedName: string | null;
  initialTemplate: Template;
  draft: TemplateDraftState;
  visible: boolean;
}

export function createTemplateWorkspaceDraftSession(
  template: Template | undefined,
  source: DraftSource,
  options?: {
    originalName?: string | null;
    selectedName?: string | null;
    visible?: boolean;
  },
): TemplateWorkspaceDraftSession {
  const draft = buildDraftFromTemplate(template);

  return {
    source,
    originalName:
      options?.originalName !== undefined
        ? options.originalName
        : (template?.name ?? null),
    selectedName:
      options?.selectedName !== undefined
        ? options.selectedName
        : (template?.name ?? null),
    initialTemplate: buildTemplateSnapshot(draft),
    draft,
    visible: options?.visible ?? true,
  };
}

export function isTemplateWorkspaceDraftDirty(
  session: TemplateWorkspaceDraftSession,
): boolean {
  return !deepEqual(
    buildTemplateSnapshot(session.draft),
    session.initialTemplate,
  );
}
