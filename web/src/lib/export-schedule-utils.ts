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
  ExportSchedule,
  ExportScheduleRequest,
  ExportFilters,
  ExportConfig,
  CloudExportConfig,
} from "../api";
import type { ExportScheduleFormData } from "../types/export-schedule";

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
  pathTemplate: "{job_id}.{format}",
  cloudProvider: "s3",
  cloudBucket: "",
  cloudRegion: "",
  cloudPath: "",
  localPath: "",
  webhookUrl: "",
  maxRetries: 3,
  baseDelayMs: 1000,
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
    case "s3":
    case "gcs":
    case "azure": {
      const cloud = config.cloud_config;
      if (cloud?.bucket) {
        return `${config.destination_type.toUpperCase()}: ${cloud.bucket}${cloud.path ? `/${cloud.path}` : ""}`;
      }
      return config.destination_type.toUpperCase();
    }
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

/**
 * Format an ISO date string to a locale-specific string
 * @param dateStr - ISO date string or undefined
 * @returns Formatted date string or "-" if undefined
 */
export function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return "-";
  const date = new Date(dateStr);
  return date.toLocaleString();
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
  const cloud = export_.cloud_config;
  const retry = schedule.retry || {};

  return {
    name: schedule.name || "",
    enabled: schedule.enabled ?? true,
    filterJobKinds: filters.job_kinds || [],
    filterJobStatus: filters.job_status || ["completed"],
    filterTags: filters.tags?.join("\n") || "",
    filterHasResults: filters.has_results ?? true,
    format: export_.format || "json",
    destinationType: export_.destination_type || "local",
    pathTemplate: export_.path_template || "{job_id}.{format}",
    cloudProvider: cloud?.provider || "s3",
    cloudBucket: cloud?.bucket || "",
    cloudRegion: cloud?.region || "",
    cloudPath: cloud?.path || "",
    localPath: export_.local_path || "",
    webhookUrl: export_.webhook_url || "",
    maxRetries: retry.max_retries ?? 3,
    baseDelayMs: retry.base_delay_ms ?? 1000,
  };
}

/**
 * Build CloudExportConfig from form data
 * @param data - Form data
 * @returns CloudExportConfig or undefined if not cloud destination
 */
function buildCloudConfig(
  data: ExportScheduleFormData,
): CloudExportConfig | undefined {
  if (data.destinationType === "local" || data.destinationType === "webhook") {
    return undefined;
  }

  const config: CloudExportConfig = {
    provider: data.cloudProvider,
    bucket: data.cloudBucket,
  };

  if (data.cloudPath) config.path = data.cloudPath;
  if (data.cloudRegion) config.region = data.cloudRegion;

  return config;
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
  } else if (
    ["s3", "gcs", "azure"].includes(data.destinationType) &&
    data.cloudBucket
  ) {
    config.cloud_config = buildCloudConfig(data);
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
    filters.tags = data.filterTags
      .split("\n")
      .map((t) => t.trim())
      .filter((t) => t);
  }

  filters.has_results = data.filterHasResults;

  return filters;
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
