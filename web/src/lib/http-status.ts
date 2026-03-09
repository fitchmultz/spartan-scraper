/**
 * Shared HTTP status presentation helpers.
 *
 * Purpose:
 * - Centralize CSS class mapping for HTTP status code badges.
 *
 * Responsibilities:
 * - Provide one simple success/running/failed mapping for compact badges.
 * - Provide one detailed mapping for inspector-style status displays.
 *
 * Scope:
 * - Web UI display helpers only.
 *
 * Usage:
 * - Import from components that render HTTP status badges.
 *
 * Invariants/Assumptions:
 * - Non-positive statuses can optionally render as an empty class.
 * - Unknown or missing statuses map to explicit fallback classes.
 */

export function getSimpleHttpStatusClass(
  status: number,
  options?: { emptyWhenZero?: boolean },
): string {
  if (options?.emptyWhenZero && status === 0) {
    return "";
  }
  if (status >= 200 && status < 300) return "success";
  if (status >= 400) return "failed";
  return "running";
}

export function getDetailedHttpStatusClass(status?: number): string {
  if (status === undefined || status === null) return "unknown";
  if (status >= 200 && status < 300) return "success";
  if (status >= 300 && status < 400) return "redirect";
  if (status >= 400 && status < 500) return "client-error";
  if (status >= 500) return "server-error";
  return "unknown";
}
