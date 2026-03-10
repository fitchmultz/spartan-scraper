/**
 * trafficInspectorUtils
 *
 * Purpose:
 * - Hold pure helpers for traffic filtering, summarization, and replay request assembly.
 *
 * Responsibilities:
 * - Normalize traffic entry display values.
 * - Apply filter/search logic for intercepted entries.
 * - Build replay payloads without empty filter fields.
 *
 * Scope:
 * - Pure utility logic only; no React state or network calls.
 *
 * Usage:
 * - Used by TrafficInspector and focused utility tests.
 *
 * Invariants/Assumptions:
 * - Unknown resource types are treated as "other".
 * - Empty replay filter inputs are omitted from the request payload.
 * - Entry keys remain stable for the same captured traffic record.
 */

import type { InterceptedEntry, TrafficReplayRequest } from "../../api";

export type TrafficFilterType =
  | "all"
  | "xhr"
  | "fetch"
  | "document"
  | "script"
  | "stylesheet"
  | "image"
  | "other";

export function formatTrafficBytes(bytes?: number | null): string {
  if (bytes === undefined || bytes === null) return "-";
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  return `${(bytes / 1024 ** index).toFixed(1)} ${units[index]}`;
}

export function getResourceTypeLabel(type?: string): string {
  return type || "other";
}

export function getTrafficEntryKey(entry: InterceptedEntry): string {
  return (
    entry.request?.requestId ||
    [
      entry.request?.url || "unknown-url",
      entry.request?.method || "GET",
      entry.request?.timestamp || entry.response?.timestamp || "no-timestamp",
      entry.response?.status?.toString() || "no-status",
    ].join("::")
  );
}

export function filterTrafficEntries(
  entries: InterceptedEntry[],
  filterType: TrafficFilterType,
  searchQuery: string,
): InterceptedEntry[] {
  const normalizedQuery = searchQuery.trim().toLowerCase();

  return entries.filter((entry) => {
    const resourceType = entry.request?.resourceType || "other";
    const matchesType =
      filterType === "all"
        ? true
        : filterType === "other"
          ? ![
              "xhr",
              "fetch",
              "document",
              "script",
              "stylesheet",
              "image",
            ].includes(resourceType)
          : resourceType === filterType;

    if (!matchesType) {
      return false;
    }

    if (!normalizedQuery) {
      return true;
    }

    const url = entry.request?.url?.toLowerCase() || "";
    const method = entry.request?.method?.toLowerCase() || "";
    return url.includes(normalizedQuery) || method.includes(normalizedQuery);
  });
}

export function summarizeTrafficEntries(entries: InterceptedEntry[]) {
  const total = entries.length;
  const withResponse = entries.filter((entry) => entry.response).length;
  const totalSize = entries.reduce(
    (sum, entry) => sum + (entry.response?.bodySize || 0),
    0,
  );
  const avgDuration =
    total > 0
      ? entries.reduce((sum, entry) => sum + (entry.duration || 0), 0) / total
      : 0;

  return { total, withResponse, totalSize, avgDuration };
}

export function parseReplayFilterValues(value: string): string[] | undefined {
  const items = value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);

  return items.length > 0 ? items : undefined;
}

export function buildReplayRequest(
  jobId: string,
  targetBaseUrl: string,
  compareResponses: boolean,
  replayFilterURL: string,
  replayFilterMethod: string,
): TrafficReplayRequest {
  return {
    jobId,
    targetBaseUrl,
    compareResponses,
    filter: {
      urlPatterns: parseReplayFilterValues(replayFilterURL),
      methods: parseReplayFilterValues(replayFilterMethod),
    },
  };
}
