/**
 * Purpose: Provide reusable status display helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
 */

import type { ExportInspection, Watch, WatchCheckInspection } from "../api";

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

export function getWatchCheckStatusTone(
  status?: WatchCheckInspection["status"],
): StatusTone {
  switch (status) {
    case "baseline":
      return "info";
    case "changed":
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
