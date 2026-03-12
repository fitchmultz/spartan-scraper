/**
 * Helpers for surfacing SDK/API errors consistently in the UI.
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
