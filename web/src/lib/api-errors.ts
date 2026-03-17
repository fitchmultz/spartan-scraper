/**
 * Purpose: Normalize API and runtime error payloads into human-readable UI messages.
 * Responsibilities: Prefer explicit API error strings, unwrap common error object shapes, and fall back to caller-provided copy instead of leaking `[object Object]` into the interface.
 * Scope: Shared web-client error-message formatting only.
 * Usage: Call `getApiErrorMessage(error, fallback)` before surfacing an unknown error in UI state.
 * Invariants/Assumptions: Known string fields win over generic fallbacks, callers provide a user-safe fallback, and this helper never throws while formatting unknown values.
 */

export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === "string" && error.trim()) {
    return error;
  }

  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }

  if (typeof error === "object" && error !== null) {
    const candidate = error as {
      error?: unknown;
      message?: unknown;
      detail?: unknown;
    };

    if (typeof candidate.error === "string" && candidate.error.trim()) {
      return candidate.error;
    }

    if (typeof candidate.message === "string" && candidate.message.trim()) {
      return candidate.message;
    }

    if (typeof candidate.detail === "string" && candidate.detail.trim()) {
      return candidate.detail;
    }
  }

  return fallback;
}
