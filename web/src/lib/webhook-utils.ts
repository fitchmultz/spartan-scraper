/**
 * Purpose: Provide reusable webhook utils helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
 */

export function getDeliveryStatusColor(status?: string): string {
  switch (status?.toLowerCase()) {
    case "delivered":
      return "var(--success, #22c55e)";
    case "failed":
      return "var(--error, #ef4444)";
    case "pending":
      return "var(--warning, #f59e0b)";
    default:
      return "var(--muted, #6b7280)";
  }
}

export function getDeliveryStatusBackgroundColor(status?: string): string {
  switch (status?.toLowerCase()) {
    case "delivered":
      return "rgba(34, 197, 94, 0.1)";
    case "failed":
      return "rgba(239, 68, 68, 0.1)";
    case "pending":
      return "rgba(245, 158, 11, 0.1)";
    default:
      return "rgba(107, 114, 128, 0.1)";
  }
}

export function formatJson(data: unknown): string {
  try {
    return JSON.stringify(data, null, 2);
  } catch {
    return String(data);
  }
}
