import type { BridgeFieldValue, ExtractResult, TemplateResult } from "./protocol.js";

interface ExtractResultMetadata {
  model?: string;
  provider?: string;
  tokens_used?: number;
}

export function normalizeExtractResult(
  raw: unknown,
  metadata: ExtractResultMetadata = {},
): ExtractResult {
  const value = expectRecord(raw, "extract result");
  const result: ExtractResult = {
    fields: normalizeFields(value.fields),
    confidence: normalizeConfidence(value.confidence),
    provider: metadata.provider,
    model: metadata.model,
    tokens_used: metadata.tokens_used,
  };

  if (typeof value.explanation === "string" && value.explanation.trim()) {
    result.explanation = value.explanation;
  }

  return validateExtractResult(result);
}

export function validateExtractResult(result: ExtractResult): ExtractResult {
  if (!result.fields || typeof result.fields !== "object" || Array.isArray(result.fields)) {
    throw new Error("extract result must include a fields object");
  }
  if (
    typeof result.confidence !== "number" ||
    Number.isNaN(result.confidence) ||
    !Number.isFinite(result.confidence)
  ) {
    throw new Error("extract result must include numeric confidence");
  }
  for (const [fieldName, fieldValue] of Object.entries(result.fields)) {
    if (!fieldName.trim()) {
      throw new Error("extract result contains an empty field name");
    }
    validateFieldValue(fieldName, fieldValue);
  }
  return result;
}

export function validateTemplateResult(result: TemplateResult): TemplateResult {
  if (!result.template || typeof result.template !== "object") {
    throw new Error("template result must include a template object");
  }
  if (!result.template.name?.trim()) {
    throw new Error("template result must include a template.name");
  }
  if (!Array.isArray(result.template.selectors) || result.template.selectors.length === 0) {
    throw new Error("template result must include at least one selector");
  }
  return result;
}

function validateFieldValue(fieldName: string, value: BridgeFieldValue) {
  if (!Array.isArray(value.values)) {
    throw new Error(`extract result field ${fieldName} must include values array`);
  }
  if (value.values.some((entry) => typeof entry !== "string")) {
    throw new Error(`extract result field ${fieldName} values must be strings`);
  }
  if (value.source !== "llm") {
    throw new Error(`extract result field ${fieldName} must use source llm`);
  }
  if (value.rawObject !== undefined && typeof value.rawObject !== "string") {
    throw new Error(`extract result field ${fieldName} rawObject must be a string`);
  }
}

function normalizeFields(raw: unknown): Record<string, BridgeFieldValue> {
  const fields = expectRecord(raw, "extract result fields");
  const normalized: Record<string, BridgeFieldValue> = {};
  for (const [fieldName, fieldValue] of Object.entries(fields)) {
    const trimmedName = fieldName.trim();
    if (!trimmedName) {
      throw new Error("extract result contains an empty field name");
    }
    normalized[trimmedName] = normalizeFieldValue(fieldValue);
  }
  return normalized;
}

function normalizeFieldValue(raw: unknown): BridgeFieldValue {
  if (raw == null) {
    return { values: [], source: "llm" };
  }
  if (typeof raw === "string" || typeof raw === "number" || typeof raw === "boolean") {
    return { values: [String(raw)], source: "llm" };
  }
  if (Array.isArray(raw)) {
    return { values: normalizePrimitiveArray(raw), source: "llm" };
  }

  const value = expectRecord(raw, "extract field");
  if (!looksLikeWrappedField(value)) {
    return {
      values: [],
      source: "llm",
      rawObject: JSON.stringify(value),
    };
  }

  const normalized: BridgeFieldValue = {
    values: normalizeMaybeValues(value.values),
    source: "llm",
  };
  const rawObject = normalizeRawObject(value.rawObject);
  if (rawObject !== undefined) {
    normalized.rawObject = rawObject;
  }
  return normalized;
}

function normalizeConfidence(raw: unknown): number {
  if (typeof raw === "number" && Number.isFinite(raw)) {
    return clampConfidence(raw);
  }
  if (typeof raw === "string" && raw.trim()) {
    const parsed = Number(raw);
    if (Number.isFinite(parsed)) {
      return clampConfidence(parsed);
    }
  }
  throw new Error("extract result must include numeric confidence");
}

function clampConfidence(value: number): number {
  return Math.min(1, Math.max(0, value));
}

function normalizeMaybeValues(raw: unknown): string[] {
  if (raw == null) {
    return [];
  }
  if (typeof raw === "string" || typeof raw === "number" || typeof raw === "boolean") {
    return [String(raw)];
  }
  if (Array.isArray(raw)) {
    return normalizePrimitiveArray(raw);
  }
  throw new Error("extract field values must be a primitive or primitive array");
}

function normalizePrimitiveArray(raw: unknown[]): string[] {
  return raw.map((entry) => {
    if (
      typeof entry === "string" ||
      typeof entry === "number" ||
      typeof entry === "boolean"
    ) {
      return String(entry);
    }
    throw new Error("extract field array entries must be primitive values");
  });
}

function normalizeRawObject(raw: unknown): string | undefined {
  if (raw == null) {
    return undefined;
  }
  if (typeof raw === "string") {
    return raw;
  }
  return JSON.stringify(raw);
}

function looksLikeWrappedField(value: Record<string, unknown>): boolean {
  return "values" in value || "source" in value || "rawObject" in value;
}

function expectRecord(raw: unknown, label: string): Record<string, unknown> {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    throw new Error(`${label} must be an object`);
  }
  return raw as Record<string, unknown>;
}
