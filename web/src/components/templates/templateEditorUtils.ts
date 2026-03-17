/**
 * Purpose: Centralize draft conversion and validation helpers for the Templates workspace.
 * Responsibilities: Create editable selector drafts, build save payloads, format advanced JSON blocks, and describe saved templates.
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

export interface JsonParseResult<T> {
  data?: T;
  error?: string;
}

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
    jsonldText: formatJSON(template?.jsonld),
    regexText: formatJSON(template?.regex),
    normalizeText: formatJSON(template?.normalize),
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

export function parseJSONInput<T>(
  label: string,
  value: string,
): JsonParseResult<T> {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }

  try {
    return { data: JSON.parse(trimmed) as T };
  } catch (error) {
    return {
      error: `${label} must be valid JSON: ${error instanceof Error ? error.message : "Invalid JSON"}`,
    };
  }
}

export function formatJSON(value: unknown) {
  return value ? JSON.stringify(value, null, 2) : "";
}

export function buildTemplateSnapshot(draft: TemplateDraftState): Template {
  const jsonldResult = parseJSONInput<JsonldRule[]>(
    "JSON-LD rules",
    draft.jsonldText,
  );
  const regexResult = parseJSONInput<RegexRule[]>(
    "Regex rules",
    draft.regexText,
  );
  const normalizeResult = parseJSONInput<NormalizeSpec>(
    "Normalization settings",
    draft.normalizeText,
  );

  return {
    name: draft.name.trim(),
    selectors: draft.selectors.map(({ rule }) => ({
      ...rule,
      name: rule.name?.trim(),
      selector: rule.selector?.trim(),
      attr: rule.attr?.trim() || "text",
      join: rule.join?.trim() || undefined,
    })),
    ...(Array.isArray(jsonldResult.data) ? { jsonld: jsonldResult.data } : {}),
    ...(Array.isArray(regexResult.data) ? { regex: regexResult.data } : {}),
    ...(normalizeResult.data && !Array.isArray(normalizeResult.data)
      ? { normalize: normalizeResult.data }
      : {}),
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

  const jsonldResult = parseJSONInput<JsonldRule[]>(
    "JSON-LD rules",
    draft.jsonldText,
  );
  if (jsonldResult.error) {
    return { error: jsonldResult.error };
  }
  if (jsonldResult.data && !Array.isArray(jsonldResult.data)) {
    return { error: "JSON-LD rules must be a JSON array." };
  }

  const regexResult = parseJSONInput<RegexRule[]>(
    "Regex rules",
    draft.regexText,
  );
  if (regexResult.error) {
    return { error: regexResult.error };
  }
  if (regexResult.data && !Array.isArray(regexResult.data)) {
    return { error: "Regex rules must be a JSON array." };
  }

  const normalizeResult = parseJSONInput<NormalizeSpec>(
    "Normalization settings",
    draft.normalizeText,
  );
  if (normalizeResult.error) {
    return { error: normalizeResult.error };
  }
  if (
    normalizeResult.data &&
    (Array.isArray(normalizeResult.data) ||
      typeof normalizeResult.data !== "object")
  ) {
    return { error: "Normalization settings must be a JSON object." };
  }

  return {
    payload: {
      name: trimmedName,
      selectors,
      ...(jsonldResult.data ? { jsonld: jsonldResult.data } : {}),
      ...(regexResult.data ? { regex: regexResult.data } : {}),
      ...(normalizeResult.data ? { normalize: normalizeResult.data } : {}),
    },
  };
}

export function ruleKey(rule: SelectorRule) {
  return `${rule.name ?? "selector"}-${rule.selector ?? ""}-${rule.attr ?? "text"}`;
}
