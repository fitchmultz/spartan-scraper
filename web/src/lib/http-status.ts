/**
 * Purpose: Provide reusable http status helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
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
