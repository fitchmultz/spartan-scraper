/**
 * Watch Utilities Module
 *
 * Provides helper functions for watch-related operations including formatting,
 * data transformation between API types and form data, and default value generation.
 *
 * This module does NOT handle:
 * - React state management or hooks
 * - API calls or network operations
 * - UI rendering or component logic
 *
 * @module lib/watch-utils
 */

import type { Watch, WatchInput } from "../api";
import { formatDateTime, formatSecondsAsDuration } from "./formatting";
import { parseOptionalList } from "./input-parsing";
import type { WatchFormData } from "../types/watch";

/**
 * Default form data for creating a new watch
 */
export const defaultFormData: WatchFormData = {
  url: "",
  selector: "",
  intervalSeconds: 3600,
  enabled: true,
  diffFormat: "unified",
  notifyOnChange: false,
  webhookUrl: "",
  webhookSecret: "",
  headless: false,
  usePlaywright: false,
  extractMode: "",
  minChangeSize: "",
  ignorePatterns: "",
  screenshotEnabled: false,
  screenshotFullPage: true,
  screenshotFormat: "png",
  visualDiffThreshold: "0.1",
};

/**
 * Format a duration in seconds to a human-readable string
 * @param seconds - Duration in seconds
 * @returns Formatted string (e.g., "60s", "5m", "2h", "1d")
 */
export function formatDuration(seconds: number): string {
  return formatSecondsAsDuration(seconds);
}

/**
 * Format an ISO date string to a locale-specific string
 * @param dateStr - ISO date string or undefined
 * @returns Formatted date string or "Never" if undefined
 */
export function formatDate(dateStr: string | undefined): string {
  return formatDateTime(dateStr, "Never");
}

/**
 * Convert a Watch API object to WatchFormData for editing
 * @param watch - Watch object from API
 * @returns WatchFormData for form state
 */
export function watchToFormData(watch: Watch): WatchFormData {
  return {
    url: watch.url,
    selector: watch.selector || "",
    intervalSeconds: watch.intervalSeconds,
    enabled: watch.enabled ?? true,
    diffFormat: (watch.diffFormat || "unified") as WatchFormData["diffFormat"],
    notifyOnChange: watch.notifyOnChange ?? false,
    webhookUrl: watch.webhookConfig?.url || "",
    webhookSecret: watch.webhookConfig?.secret || "",
    headless: watch.headless ?? false,
    usePlaywright: watch.usePlaywright ?? false,
    extractMode: (watch.extractMode as WatchFormData["extractMode"]) || "",
    minChangeSize: watch.minChangeSize?.toString() || "",
    ignorePatterns: watch.ignorePatterns?.join("\n") || "",
    screenshotEnabled: watch.screenshotEnabled ?? false,
    screenshotFullPage: watch.screenshotConfig?.fullPage ?? true,
    screenshotFormat:
      (watch.screenshotConfig?.format as "png" | "jpeg") || "png",
    visualDiffThreshold: watch.visualDiffThreshold?.toString() || "0.1",
  };
}

/**
 * Convert WatchFormData to WatchInput for API submission
 * Only includes fields that have values set
 * @param data - Form data from watch form
 * @returns WatchInput for API create/update calls
 */
export function formDataToWatchInput(data: WatchFormData): WatchInput {
  const input: WatchInput = {
    url: data.url,
    intervalSeconds: data.intervalSeconds,
    enabled: data.enabled,
    diffFormat: data.diffFormat,
    notifyOnChange: data.notifyOnChange,
    headless: data.headless,
    usePlaywright: data.usePlaywright,
    screenshotEnabled: data.screenshotEnabled,
  };

  if (data.selector) input.selector = data.selector;
  if (data.extractMode) input.extractMode = data.extractMode;
  if (data.minChangeSize)
    input.minChangeSize = parseInt(data.minChangeSize, 10);
  input.ignorePatterns = parseOptionalList(data.ignorePatterns, "\n");
  if (data.webhookUrl && data.notifyOnChange) {
    input.webhookConfig = {
      url: data.webhookUrl,
      secret: data.webhookSecret || undefined,
    };
  }
  if (data.screenshotEnabled) {
    input.screenshotConfig = {
      enabled: true,
      fullPage: data.screenshotFullPage,
      format: data.screenshotFormat,
    };
    if (data.visualDiffThreshold) {
      input.visualDiffThreshold = parseFloat(data.visualDiffThreshold);
    }
  }

  return input;
}
