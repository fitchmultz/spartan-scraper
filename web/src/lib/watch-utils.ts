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

import type { Watch, WatchArtifact, WatchInput } from "../api";
import { buildApiUrl } from "./api-config";
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
  jobTriggerKind: "",
  jobTriggerRequest: "",
};

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
    jobTriggerKind:
      (watch.jobTrigger?.kind as WatchFormData["jobTriggerKind"]) || "",
    jobTriggerRequest: watch.jobTrigger
      ? JSON.stringify(watch.jobTrigger.request, null, 2)
      : "",
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

  if (data.jobTriggerKind && data.jobTriggerRequest.trim()) {
    input.jobTrigger = {
      kind: data.jobTriggerKind,
      request: JSON.parse(data.jobTriggerRequest) as Record<string, unknown>,
    };
  }

  return input;
}

export function getWatchArtifact(
  result: { artifacts?: WatchArtifact[] } | null | undefined,
  kind: WatchArtifact["kind"],
): WatchArtifact | undefined {
  return result?.artifacts?.find(
    (artifact: WatchArtifact) => artifact.kind === kind,
  );
}

export function getWatchArtifactUrl(
  artifact: Pick<WatchArtifact, "downloadUrl"> | null | undefined,
): string {
  if (!artifact?.downloadUrl) {
    return "";
  }
  return buildApiUrl(artifact.downloadUrl);
}

export function getWatchArtifactLabel(kind: WatchArtifact["kind"]): string {
  switch (kind) {
    case "current-screenshot":
      return "Current Screenshot";
    case "previous-screenshot":
      return "Previous Screenshot";
    case "visual-diff":
      return "Visual Diff";
    default:
      return kind;
  }
}
