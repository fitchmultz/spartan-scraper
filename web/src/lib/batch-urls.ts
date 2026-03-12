/**
 * Purpose: Normalize batch URL text input into a stable list the form can count and submit.
 * Responsibilities: Split comma/newline-delimited text, trim whitespace, and drop empty entries.
 * Scope: Client-side batch form parsing only.
 * Usage: Call from batch form UI and related tests when deriving URL counts or requests.
 * Invariants/Assumptions: Duplicate separators and blank lines should never inflate the visible count.
 */

export const MAX_BATCH_SIZE = 100;

export function parseBatchUrls(input: string): string[] {
  if (!input.trim()) {
    return [];
  }

  return input
    .split(/[\n,]/)
    .map((value) => value.trim())
    .filter(Boolean);
}

export function summarizeBatchUrls(
  urls: readonly string[],
  visibleCount = 3,
): { visible: string[]; remaining: number } {
  const normalizedVisibleCount =
    visibleCount > 0 ? Math.floor(visibleCount) : 0;

  return {
    visible: urls.slice(0, normalizedVisibleCount),
    remaining: Math.max(0, urls.length - normalizedVisibleCount),
  };
}
