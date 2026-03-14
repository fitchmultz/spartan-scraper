/**
 * Export Schedule Utilities Module
 *
 * Provides helper functions for export schedule-related operations including
 * formatting, data transformation between API types and form data, and default
 * value generation.
 *
 * This module does NOT handle:
 * - React state management or hooks
 * - API calls or network operations
 * - UI rendering or component logic
 *
 * @module lib/export-schedule-utils
 */

import type {
  ExportConfig,
  ExportSchedule,
  ExportScheduleRequest,
  ExportFilters,
  ExportShapeConfig,
  ResultTransformConfig,
} from "../api";
import {
  parseLineSeparatedMap,
  parseOptionalList,
  splitAndTrim,
} from "./input-parsing";
import type { ExportScheduleFormData } from "../types/export-schedule";

const SHAPE_SUPPORTED_FORMATS = new Set<ExportScheduleFormData["format"]>([
  "md",
  "csv",
  "xlsx",
]);

/**
 * Default form data for creating a new export schedule
 */
export const defaultFormData: ExportScheduleFormData = {
  name: "",
  enabled: true,
  filterJobKinds: [],
  filterJobStatus: ["completed"],
  filterTags: "",
  filterHasResults: true,
  format: "json",
  destinationType: "local",
  pathTemplate: "exports/{kind}/{job_id}.{format}",
  localPath: "exports/{kind}/{job_id}.{format}",
  webhookUrl: "",
  maxRetries: 3,
  baseDelayMs: 1000,
  transformExpression: "",
  transformLanguage: "jmespath",
  shapeTopLevelFields: "",
  shapeNormalizedFields: "",
  shapeEvidenceFields: "",
  shapeSummaryFields: "",
  shapeFieldLabels: "",
  shapeEmptyValue: "",
  shapeMultiValueJoin: "",
  shapeMarkdownTitle: "",
};

/**
 * Format a destination configuration to a human-readable string
 * @param schedule - Export schedule from API
 * @returns Human-readable destination summary
 */
export function formatDestination(schedule: ExportSchedule): string {
  const config = schedule.export;
  if (!config) return "Unknown";

  switch (config.destination_type) {
    case "local":
      return config.local_path || "Local file";
    case "webhook":
      return config.webhook_url
        ? `Webhook: ${config.webhook_url.substring(0, 30)}...`
        : "Webhook";
    default:
      return String(config.destination_type);
  }
}

/**
 * Format filter criteria to a human-readable string
 * @param filters - Export filters from API
 * @returns Human-readable filter summary
 */
export function formatFilters(filters: ExportFilters | undefined): string {
  if (!filters) return "All jobs";

  const parts: string[] = [];

  if (filters.job_kinds?.length) {
    parts.push(`Kinds: ${filters.job_kinds.join(", ")}`);
  }

  if (filters.job_status?.length) {
    parts.push(`Status: ${filters.job_status.join(", ")}`);
  }

  if (filters.tags?.length) {
    parts.push(`Tags: ${filters.tags.join(", ")}`);
  }

  if (filters.has_results) {
    parts.push("Has results");
  }

  return parts.length ? parts.join(" | ") : "All jobs";
}

/**
 * Format file size in bytes to human-readable string
 * @param bytes - Size in bytes
 * @returns Formatted string (e.g., "1.5 KB", "2.3 MB")
 */
export function formatFileSize(bytes: number | undefined): string {
  if (bytes === undefined || bytes === null) return "-";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024)
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

export function supportsExportShapeFormat(
  format: ExportScheduleFormData["format"] | undefined,
): format is Extract<ExportScheduleFormData["format"], "md" | "csv" | "xlsx"> {
  return format ? SHAPE_SUPPORTED_FORMATS.has(format) : false;
}

export function hasTransformFormData(data: ExportScheduleFormData): boolean {
  return Boolean(data.transformExpression.trim());
}

export function transformConfigToFormData(
  transform: ResultTransformConfig | undefined,
): Pick<ExportScheduleFormData, "transformExpression" | "transformLanguage"> {
  return {
    transformExpression: transform?.expression || "",
    transformLanguage:
      (transform?.language as ExportScheduleFormData["transformLanguage"]) ||
      "jmespath",
  };
}

export function formDataToTransformConfig(
  data: ExportScheduleFormData,
): ResultTransformConfig | undefined {
  const expression = data.transformExpression.trim();
  if (!expression) {
    return undefined;
  }
  return {
    expression,
    language: data.transformLanguage,
  };
}

export function formatExportTransformSummary(
  transform: ResultTransformConfig | undefined,
): string {
  const normalized = transformConfigToFormData({
    expression: transform?.expression,
    language: transform?.language,
  });
  if (!normalized.transformExpression.trim()) {
    return "Default";
  }
  const compactExpression = normalized.transformExpression
    .replace(/\s+/g, " ")
    .trim();
  const preview =
    compactExpression.length > 48
      ? `${compactExpression.slice(0, 45)}...`
      : compactExpression;
  return `${normalized.transformLanguage} · ${preview}`;
}

function formatLineSeparatedList(values: string[] | undefined): string {
  return values?.join("\n") || "";
}

export function formatShapeLabels(
  labels: Record<string, string> | undefined,
): string {
  if (!labels || Object.keys(labels).length === 0) {
    return "";
  }

  return Object.entries(labels)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value]) => `${key}=${value}`)
    .join("\n");
}

export function shapeConfigToFormData(
  shape: ExportShapeConfig | undefined,
): Pick<
  ExportScheduleFormData,
  | "shapeTopLevelFields"
  | "shapeNormalizedFields"
  | "shapeEvidenceFields"
  | "shapeSummaryFields"
  | "shapeFieldLabels"
  | "shapeEmptyValue"
  | "shapeMultiValueJoin"
  | "shapeMarkdownTitle"
> {
  return {
    shapeTopLevelFields: formatLineSeparatedList(shape?.topLevelFields),
    shapeNormalizedFields: formatLineSeparatedList(shape?.normalizedFields),
    shapeEvidenceFields: formatLineSeparatedList(shape?.evidenceFields),
    shapeSummaryFields: formatLineSeparatedList(shape?.summaryFields),
    shapeFieldLabels: formatShapeLabels(shape?.fieldLabels),
    shapeEmptyValue: shape?.formatting?.emptyValue || "",
    shapeMultiValueJoin: shape?.formatting?.multiValueJoin || "",
    shapeMarkdownTitle: shape?.formatting?.markdownTitle || "",
  };
}

export function hasShapeFormData(data: ExportScheduleFormData): boolean {
  return Boolean(
    data.shapeTopLevelFields.trim() ||
      data.shapeNormalizedFields.trim() ||
      data.shapeEvidenceFields.trim() ||
      data.shapeSummaryFields.trim() ||
      data.shapeFieldLabels.trim() ||
      data.shapeEmptyValue.trim() ||
      data.shapeMultiValueJoin.trim() ||
      data.shapeMarkdownTitle.trim(),
  );
}

export function parseShapeFieldLabels(
  input: string,
): Record<string, string> | undefined {
  return parseLineSeparatedMap(input, "=");
}

export function formDataToShapeConfig(
  data: ExportScheduleFormData,
): ExportShapeConfig | undefined {
  if (!supportsExportShapeFormat(data.format)) {
    return undefined;
  }

  const shape: ExportShapeConfig = {
    topLevelFields: parseOptionalList(data.shapeTopLevelFields, /\r?\n/),
    normalizedFields: parseOptionalList(data.shapeNormalizedFields, /\r?\n/),
    evidenceFields: parseOptionalList(data.shapeEvidenceFields, /\r?\n/),
    summaryFields: parseOptionalList(data.shapeSummaryFields, /\r?\n/),
    fieldLabels: parseShapeFieldLabels(data.shapeFieldLabels),
  };

  const emptyValue = data.shapeEmptyValue.trim();
  const multiValueJoin = data.shapeMultiValueJoin.trim();
  const markdownTitle = data.shapeMarkdownTitle.trim();

  if (emptyValue || multiValueJoin || markdownTitle) {
    shape.formatting = {
      emptyValue: emptyValue || undefined,
      multiValueJoin: multiValueJoin || undefined,
      markdownTitle: markdownTitle || undefined,
    };
  }

  if (
    shape.topLevelFields?.length ||
    shape.normalizedFields?.length ||
    shape.evidenceFields?.length ||
    shape.summaryFields?.length ||
    (shape.fieldLabels && Object.keys(shape.fieldLabels).length > 0) ||
    shape.formatting?.emptyValue ||
    shape.formatting?.multiValueJoin ||
    shape.formatting?.markdownTitle
  ) {
    return shape;
  }

  return undefined;
}

export function clearTransformFormData(): Pick<
  ExportScheduleFormData,
  "transformExpression" | "transformLanguage"
> {
  return {
    transformExpression: "",
    transformLanguage: "jmespath",
  };
}

export function clearShapeFormData(): Pick<
  ExportScheduleFormData,
  | "shapeTopLevelFields"
  | "shapeNormalizedFields"
  | "shapeEvidenceFields"
  | "shapeSummaryFields"
  | "shapeFieldLabels"
  | "shapeEmptyValue"
  | "shapeMultiValueJoin"
  | "shapeMarkdownTitle"
> {
  return shapeConfigToFormData(undefined);
}

export function formatExportShapeSummary(
  shape: ExportShapeConfig | undefined,
): string {
  if (!shape) {
    return "Default";
  }

  const counts = [
    shape.topLevelFields?.length ?? 0,
    shape.normalizedFields?.length ?? 0,
    shape.evidenceFields?.length ?? 0,
    shape.summaryFields?.length ?? 0,
  ];
  const selectedFieldCount = counts.reduce((sum, value) => sum + value, 0);
  const labelCount = shape.fieldLabels
    ? Object.keys(shape.fieldLabels).length
    : 0;
  const hasFormatting = Boolean(
    shape.formatting?.emptyValue ||
      shape.formatting?.multiValueJoin ||
      shape.formatting?.markdownTitle,
  );

  if (selectedFieldCount === 0 && labelCount === 0 && !hasFormatting) {
    return "Default";
  }

  const parts: string[] = [];
  if (selectedFieldCount > 0) {
    parts.push(
      `${selectedFieldCount} field${selectedFieldCount === 1 ? "" : "s"}`,
    );
  }
  if (labelCount > 0) {
    parts.push(`${labelCount} label${labelCount === 1 ? "" : "s"}`);
  }
  if (hasFormatting) {
    parts.push("formatting");
  }
  return parts.join(" · ");
}

/**
 * Convert an ExportSchedule API object to ExportScheduleFormData for editing
 * @param schedule - ExportSchedule object from API
 * @returns ExportScheduleFormData for form state
 */
export function scheduleToFormData(
  schedule: ExportSchedule,
): ExportScheduleFormData {
  const filters = schedule.filters || {};
  const export_ = schedule.export || {};
  const retry = schedule.retry || {};

  return {
    name: schedule.name || "",
    enabled: schedule.enabled ?? true,
    filterJobKinds: filters.job_kinds || [],
    filterJobStatus: filters.job_status || ["completed"],
    filterTags: filters.tags?.join("\n") || "",
    filterHasResults: filters.has_results ?? true,
    format: (export_.format as ExportScheduleFormData["format"]) || "json",
    destinationType:
      (export_.destination_type as ExportScheduleFormData["destinationType"]) ||
      "local",
    pathTemplate: export_.path_template || defaultFormData.pathTemplate,
    localPath: export_.local_path || defaultFormData.localPath,
    webhookUrl: export_.webhook_url || "",
    maxRetries: retry.max_retries ?? 3,
    baseDelayMs: retry.base_delay_ms ?? 1000,
    ...transformConfigToFormData(export_.transform),
    ...shapeConfigToFormData(export_.shape),
  };
}

/**
 * Build ExportConfig from form data
 * @param data - Form data
 * @returns ExportConfig for API
 */
function buildExportConfig(data: ExportScheduleFormData): ExportConfig {
  const config: ExportConfig = {
    format: data.format,
    destination_type: data.destinationType,
  };

  if (data.pathTemplate) {
    config.path_template = data.pathTemplate;
  }

  // Add destination-specific config
  if (data.destinationType === "local" && data.localPath) {
    config.local_path = data.localPath;
  } else if (data.destinationType === "webhook" && data.webhookUrl) {
    config.webhook_url = data.webhookUrl;
  }

  const transform = formDataToTransformConfig(data);
  if (transform) {
    config.transform = transform;
  }

  const shape = formDataToShapeConfig(data);
  if (shape && !transform) {
    config.shape = shape;
  }

  return config;
}

/**
 * Build ExportFilters from form data
 * @param data - Form data
 * @returns ExportFilters for API
 */
function buildFilters(data: ExportScheduleFormData): ExportFilters {
  const filters: ExportFilters = {};

  if (data.filterJobKinds.length) {
    filters.job_kinds = data.filterJobKinds;
  }

  if (data.filterJobStatus.length) {
    filters.job_status = data.filterJobStatus;
  }

  if (data.filterTags.trim()) {
    filters.tags = parseOptionalList(data.filterTags, "\n");
  }

  filters.has_results = data.filterHasResults;

  return filters;
}

export function normalizeShapeFieldTextarea(input: string): string {
  return splitAndTrim(input, /\r?\n/).join("\n");
}

/**
 * Convert ExportScheduleFormData to ExportScheduleRequest for API submission
 * @param data - Form data from export schedule form
 * @returns ExportScheduleRequest for API create/update calls
 */
export function formDataToScheduleRequest(
  data: ExportScheduleFormData,
): ExportScheduleRequest {
  const request: ExportScheduleRequest = {
    name: data.name,
    enabled: data.enabled,
    filters: buildFilters(data),
    export: buildExportConfig(data),
  };

  // Only include retry config if values differ from defaults
  if (data.maxRetries !== 3 || data.baseDelayMs !== 1000) {
    request.retry = {
      max_retries: data.maxRetries,
      base_delay_ms: data.baseDelayMs,
    };
  }

  return request;
}
