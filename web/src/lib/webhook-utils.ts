/**
 * Webhook presentation helpers for delivery status and payload rendering.
 *
 * Purpose:
 * - Keep webhook-specific UI formatting logic out of component bodies.
 *
 * Responsibilities:
 * - Map delivery statuses to shared colors and backgrounds.
 * - Safely stringify payloads for debugging surfaces.
 *
 * Scope:
 * - Webhook delivery UI helpers only.
 *
 * Usage:
 * - Import from webhook list/detail components.
 *
 * Invariants/Assumptions:
 * - Unknown statuses use muted styling.
 * - JSON formatting never throws to the caller.
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
