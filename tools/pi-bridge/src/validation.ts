import type {
  BridgeFieldValue,
  ExportShapeConfig,
  ExportShapeResult,
  ExtractResult,
  PipelineJsResult,
  RenderProfileResult,
  ResearchEvidenceHighlight,
  ResearchRefineResult,
  ResearchRefinedContent,
  TemplateResult,
} from "./protocol.js";

interface ExtractResultMetadata {
  model?: string;
  provider?: string;
  route_id?: string;
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

  if (metadata.route_id) {
    result.route_id = metadata.route_id;
  }

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

export function validateRenderProfileResult(
  result: RenderProfileResult,
): RenderProfileResult {
  if (!result.profile || typeof result.profile !== "object" || Array.isArray(result.profile)) {
    throw new Error("render profile result must include a profile object");
  }
  if (!hasAnyMeaningfulValue(result.profile)) {
    throw new Error("render profile result must include at least one configuration field");
  }
  return result;
}

export function validatePipelineJsResult(result: PipelineJsResult): PipelineJsResult {
  if (!result.script || typeof result.script !== "object" || Array.isArray(result.script)) {
    throw new Error("pipeline script result must include a script object");
  }
  if (!hasAnyMeaningfulValue(result.script)) {
    throw new Error("pipeline script result must include at least one configuration field");
  }
  return result;
}

export function validateResearchRefineResult(
  result: ResearchRefineResult,
): ResearchRefineResult {
  if (!result || typeof result !== "object") {
    throw new Error("research refinement result must be an object");
  }
  result.refined = normalizeRefinedContent(result.refined);
  return result;
}

export function validateExportShapeResult(
  result: ExportShapeResult,
): ExportShapeResult {
  if (!result || typeof result !== "object") {
    throw new Error("export shape result must be an object");
  }
  result.shape = normalizeExportShapeConfig(result.shape);
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

function normalizeRefinedContent(raw: unknown): ResearchRefinedContent {
  const refined = expectRecord(raw, "research refinement result refined");
  const summary = expectNonEmptyString(refined.summary, "research refinement summary");
  const conciseSummary = expectNonEmptyString(
    refined.conciseSummary,
    "research refinement conciseSummary",
  );
  const keyFindings = normalizeStringArray(
    refined.keyFindings,
    "research refinement keyFindings",
  );
  if (keyFindings.length === 0) {
    throw new Error("research refinement must include at least one key finding");
  }

  const normalized: ResearchRefinedContent = {
    summary,
    conciseSummary,
    keyFindings,
  };

  const openQuestions = normalizeOptionalStringArray(
    refined.openQuestions,
    "research refinement openQuestions",
  );
  if (openQuestions.length > 0) {
    normalized.openQuestions = openQuestions;
  }

  const recommendedNextSteps = normalizeOptionalStringArray(
    refined.recommendedNextSteps,
    "research refinement recommendedNextSteps",
  );
  if (recommendedNextSteps.length > 0) {
    normalized.recommendedNextSteps = recommendedNextSteps;
  }

  const evidenceHighlights = normalizeEvidenceHighlights(refined.evidenceHighlights);
  if (evidenceHighlights.length > 0) {
    normalized.evidenceHighlights = evidenceHighlights;
  }

  if (refined.confidence !== undefined && refined.confidence !== null) {
    normalized.confidence = normalizeUnitInterval(
      refined.confidence,
      "research refinement confidence",
    );
  }

  return normalized;
}

function normalizeEvidenceHighlights(raw: unknown): ResearchEvidenceHighlight[] {
  if (raw == null) {
    return [];
  }
  if (!Array.isArray(raw)) {
    throw new Error("research refinement evidenceHighlights must be an array");
  }

  return raw.map((entry, index) => {
    const value = expectRecord(
      entry,
      `research refinement evidenceHighlight[${index}]`,
    );
    const highlight: ResearchEvidenceHighlight = {
      url: expectNonEmptyString(
        value.url,
        `research refinement evidenceHighlight[${index}].url`,
      ),
      finding: expectNonEmptyString(
        value.finding,
        `research refinement evidenceHighlight[${index}].finding`,
      ),
    };

    if (typeof value.title === "string" && value.title.trim()) {
      highlight.title = value.title.trim();
    }
    if (typeof value.relevance === "string" && value.relevance.trim()) {
      highlight.relevance = value.relevance.trim();
    }
    if (typeof value.citationUrl === "string" && value.citationUrl.trim()) {
      highlight.citationUrl = value.citationUrl.trim();
    }
    return highlight;
  });
}

function normalizeOptionalStringArray(raw: unknown, label: string): string[] {
  if (raw == null) {
    return [];
  }
  return normalizeStringArray(raw, label);
}

function normalizeExportShapeConfig(raw: unknown): ExportShapeConfig {
  const config = expectRecord(raw, "export shape config");
  const normalized: ExportShapeConfig = {};

  const topLevelFields = normalizeOptionalStringArray(
    config.topLevelFields,
    "export shape topLevelFields",
  );
  if (topLevelFields.length > 0) {
    normalized.topLevelFields = dedupeStrings(topLevelFields);
  }

  const normalizedFields = normalizeOptionalStringArray(
    config.normalizedFields,
    "export shape normalizedFields",
  );
  if (normalizedFields.length > 0) {
    normalized.normalizedFields = dedupeStrings(normalizedFields);
  }

  const evidenceFields = normalizeOptionalStringArray(
    config.evidenceFields,
    "export shape evidenceFields",
  );
  if (evidenceFields.length > 0) {
    normalized.evidenceFields = dedupeStrings(evidenceFields);
  }

  const summaryFields = normalizeOptionalStringArray(
    config.summaryFields,
    "export shape summaryFields",
  );
  if (summaryFields.length > 0) {
    normalized.summaryFields = dedupeStrings(summaryFields);
  }

  if (config.fieldLabels != null) {
    const fieldLabels = expectRecord(config.fieldLabels, "export shape fieldLabels");
    const normalizedLabels: Record<string, string> = {};
    for (const [key, value] of Object.entries(fieldLabels)) {
      const trimmedKey = key.trim();
      if (!trimmedKey) {
        throw new Error("export shape fieldLabels keys must be non-empty");
      }
      normalizedLabels[trimmedKey] = expectNonEmptyString(
        value,
        `export shape fieldLabels.${trimmedKey}`,
      );
    }
    if (Object.keys(normalizedLabels).length > 0) {
      normalized.fieldLabels = normalizedLabels;
    }
  }

  if (config.formatting != null) {
    const formatting = expectRecord(config.formatting, "export shape formatting");
    const normalizedFormatting: ExportShapeConfig["formatting"] = {};
    if (typeof formatting.emptyValue === "string" && formatting.emptyValue.trim()) {
      normalizedFormatting.emptyValue = formatting.emptyValue.trim();
    }
    if (
      typeof formatting.multiValueJoin === "string" &&
      formatting.multiValueJoin.trim()
    ) {
      normalizedFormatting.multiValueJoin = formatting.multiValueJoin.trim();
    }
    if (typeof formatting.markdownTitle === "string" && formatting.markdownTitle.trim()) {
      normalizedFormatting.markdownTitle = formatting.markdownTitle.trim();
    }
    if (Object.keys(normalizedFormatting).length > 0) {
      normalized.formatting = normalizedFormatting;
    }
  }

  return normalized;
}

function normalizeStringArray(raw: unknown, label: string): string[] {
  if (!Array.isArray(raw)) {
    throw new Error(`${label} must be an array`);
  }
  return raw
    .map((entry, index) =>
      expectNonEmptyString(entry, `${label}[${index}]`),
    )
    .filter(Boolean);
}

function dedupeStrings(values: string[]): string[] {
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))];
}

function expectNonEmptyString(raw: unknown, label: string): string {
  if (typeof raw !== "string" || !raw.trim()) {
    throw new Error(`${label} must be a non-empty string`);
  }
  return raw.trim();
}

function normalizeUnitInterval(raw: unknown, label: string): number {
  const value = typeof raw === "string" && raw.trim() ? Number(raw) : raw;
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw new Error(`${label} must be numeric`);
  }
  if (value < 0 || value > 1) {
    throw new Error(`${label} must be between 0 and 1`);
  }
  return value;
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

function hasAnyMeaningfulValue(value: Record<string, unknown>): boolean {
  return Object.values(value).some((entry) => {
    if (entry == null) {
      return false;
    }
    if (typeof entry === "string") {
      return entry.trim().length > 0;
    }
    if (typeof entry === "number" || typeof entry === "boolean") {
      return true;
    }
    if (Array.isArray(entry)) {
      return entry.length > 0;
    }
    if (typeof entry === "object") {
      return Object.keys(entry as Record<string, unknown>).length > 0;
    }
    return false;
  });
}

function expectRecord(raw: unknown, label: string): Record<string, unknown> {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    throw new Error(`${label} must be an object`);
  }
  return raw as Record<string, unknown>;
}
