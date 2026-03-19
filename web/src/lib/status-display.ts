/**
 * Shared status display helpers.
 *
 * Purpose:
 * - Centralize tone and color mapping for compact status-pill UI elements.
 *
 * Responsibilities:
 * - Provide typed status-to-tone helpers for watch and export states.
 * - Expose one shared tone palette for pill-like inline status labels.
 *
 * Scope:
 * - Web UI presentation helpers only.
 *
 * Usage:
 * - Import from shared status-pill components and feature views that need
 *   consistent visual status treatment.
 *
 * Invariants/Assumptions:
 * - Unknown statuses fall back to a neutral tone.
 * - Success, warning, danger, neutral, and info tones share one palette.
 */

import type { ExportInspection, Watch } from "../api";

export type StatusTone = "success" | "warning" | "danger" | "neutral" | "info";

export function getStatusToneColors(tone: StatusTone): {
  backgroundColor: string;
  color: string;
} {
  switch (tone) {
    case "success":
      return {
        backgroundColor: "rgba(34, 197, 94, 0.15)",
        color: "#22c55e",
      };
    case "warning":
      return {
        backgroundColor: "rgba(234, 179, 8, 0.15)",
        color: "#eab308",
      };
    case "danger":
      return {
        backgroundColor: "rgba(239, 68, 68, 0.15)",
        color: "#ef4444",
      };
    case "info":
      return {
        backgroundColor: "rgba(59, 130, 246, 0.15)",
        color: "#3b82f6",
      };
    default:
      return {
        backgroundColor: "rgba(156, 163, 175, 0.15)",
        color: "var(--muted)",
      };
  }
}

export function getWatchStatusTone(status?: Watch["status"]): StatusTone {
  switch (status) {
    case "active":
      return "success";
    case "error":
      return "danger";
    default:
      return "neutral";
  }
}

export function getExportHistoryStatusTone(
  status?: ExportInspection["status"] | "success",
): StatusTone {
  switch (status) {
    case "succeeded":
    case "success":
      return "success";
    case "pending":
      return "warning";
    case "failed":
      return "danger";
    default:
      return "neutral";
  }
}

export function getEnabledStatusTone(enabled: boolean | undefined): StatusTone {
  return enabled ? "success" : "neutral";
}
