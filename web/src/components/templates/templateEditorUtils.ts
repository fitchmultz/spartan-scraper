/**
 * Purpose: Centralize draft conversion and validation helpers for the Templates workspace.
 * Responsibilities: Create editable selector drafts, build save payloads, reuse shared optional JSON codecs, and describe saved templates.
 * Scope: Template-editor state helpers only; network requests and route orchestration stay in workspace components.
 * Usage: Imported by template workspace components to keep draft handling consistent across inline editing, preview, AI assistance, and builder handoffs.
 * Invariants/Assumptions: Template names are required for persistence, meaningful selector rows need both a field name and CSS selector, and invalid advanced JSON should only block save flows.
 */

import type {
  CreateTemplateRequest,
  JsonldRule,
  NormalizeSpec,
  RegexRule,
  SelectorRule,
  Template,
  TemplateDetail,
} from "../../api";
import {
  formatOptionalJSON,
  parseOptionalJSON,
} from "../settings/settingsAuthoringForm";

export const BUILT_IN_TEMPLATE_NAMES = [
  "article",
  "default",
  "product",
] as const;

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

function hasRuleContent(rule: SelectorRule) {
  return [rule.name, rule.selector, rule.join].some(
    (value) => (value?.trim().length ?? 0) > 0,
  );
}

export function createDraftId() {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `selector-${Math.random().toString(36).slice(2, 10)}`;
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

export function getDuplicateName(name: string) {
  return `${name}-copy`;
}

export function describeTemplate(detail: TemplateDetail | null) {
  const selectors = detail?.template?.selectors?.length ?? 0;
  const jsonld = detail?.template?.jsonld?.length ?? 0;
  const regex = detail?.template?.regex?.length ?? 0;
  return `${selectors} selector${selectors === 1 ? "" : "s"} · ${jsonld} JSON-LD · ${regex} regex`;
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
    selectors: draft.selectors.map(({ rule }) => ({
      ...rule,
      name: rule.name?.trim(),
      selector: rule.selector?.trim(),
      attr: rule.attr?.trim() || "text",
      join: rule.join?.trim() || undefined,
    })),
    ...(jsonld ? { jsonld } : {}),
    ...(regex ? { regex } : {}),
    ...(normalize ? { normalize } : {}),
  };
}

export function buildTemplatePayload(draft: TemplateDraftState): {
  payload?: CreateTemplateRequest;
  error?: string;
} {
  const trimmedName = draft.name.trim();
  if (!trimmedName) {
    return { error: "Template name is required." };
  }

  const selectors = draft.selectors
    .map(({ rule }) => ({
      ...rule,
      name: rule.name?.trim() ?? "",
      selector: rule.selector?.trim() ?? "",
      attr: rule.attr?.trim() || "text",
      join: rule.join?.trim() || undefined,
    }))
    .filter((rule) => hasRuleContent(rule));

  if (selectors.length === 0) {
    return { error: "Add at least one selector rule before saving." };
  }

  if (selectors.some((rule) => rule.name.length === 0)) {
    return { error: "Each selector rule needs a field name." };
  }

  if (selectors.some((rule) => rule.selector.length === 0)) {
    return { error: "Each selector rule needs a CSS selector." };
  }

  let jsonld: JsonldRule[] | undefined;
  try {
    const parsed = parseOptionalJSON<unknown>(
      "JSON-LD rules",
      draft.jsonldText,
    );
    if (parsed && !Array.isArray(parsed)) {
      return { error: "JSON-LD rules must be a JSON array." };
    }
    jsonld =
      parsed && Array.isArray(parsed) ? (parsed as JsonldRule[]) : undefined;
  } catch (error) {
    return {
      error:
        error instanceof Error
          ? error.message
          : "JSON-LD rules must be valid JSON.",
    };
  }

  let regex: RegexRule[] | undefined;
  try {
    const parsed = parseOptionalJSON<unknown>("Regex rules", draft.regexText);
    if (parsed && !Array.isArray(parsed)) {
      return { error: "Regex rules must be a JSON array." };
    }
    regex =
      parsed && Array.isArray(parsed) ? (parsed as RegexRule[]) : undefined;
  } catch (error) {
    return {
      error:
        error instanceof Error
          ? error.message
          : "Regex rules must be valid JSON.",
    };
  }

  let normalize: NormalizeSpec | undefined;
  try {
    const parsed = parseOptionalJSON<unknown>(
      "Normalization settings",
      draft.normalizeText,
    );
    if (parsed && (Array.isArray(parsed) || typeof parsed !== "object")) {
      return { error: "Normalization settings must be a JSON object." };
    }
    normalize =
      parsed && !Array.isArray(parsed) && typeof parsed === "object"
        ? (parsed as NormalizeSpec)
        : undefined;
  } catch (error) {
    return {
      error:
        error instanceof Error
          ? error.message
          : "Normalization settings must be valid JSON.",
    };
  }

  return {
    payload: {
      name: trimmedName,
      selectors,
      ...(jsonld ? { jsonld } : {}),
      ...(regex ? { regex } : {}),
      ...(normalize ? { normalize } : {}),
    },
  };
}

export function ruleKey(rule: SelectorRule) {
  return `${rule.name ?? "selector"}-${rule.selector ?? ""}-${rule.attr ?? "text"}`;
}
