/**
 * Input parsing utilities for shared form and text processing.
 *
 * Purpose:
 * - Eliminate repeated ad-hoc splitting and trimming logic across the web app.
 *
 * Responsibilities:
 * - Split delimited text into trimmed token arrays.
 * - Convert optional text blobs into list and map structures.
 * - Preserve empty-input semantics consistently for API request builders.
 *
 * Scope:
 * - Stateless parsing helpers for form libraries and components.
 *
 * Usage:
 * - Import from request builders, form adapters, and UI components that parse
 *   textarea or comma-delimited user input.
 *
 * Invariants/Assumptions:
 * - Empty or whitespace-only input returns `undefined` for optional helpers.
 * - Map parsing ignores malformed or blank lines instead of throwing.
 */

function toOptionalList(items: string[]): string[] | undefined {
  return items.length > 0 ? items : undefined;
}

export function splitAndTrim(
  input: string,
  separator: string | RegExp,
): string[] {
  return input
    .split(separator)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function parseOptionalList(
  input: string,
  separator: string | RegExp,
): string[] | undefined {
  if (!input.trim()) {
    return undefined;
  }
  return toOptionalList(splitAndTrim(input, separator));
}

export function parseLineSeparatedMap(
  input: string,
  separator: ":" | "=",
): Record<string, string> | undefined {
  if (!input.trim()) {
    return undefined;
  }

  const entries = splitAndTrim(input, "\n");
  const result: Record<string, string> = {};

  for (const entry of entries) {
    const separatorIndex = entry.indexOf(separator);
    if (separatorIndex <= 0) {
      continue;
    }

    const key = entry.slice(0, separatorIndex).trim();
    const value = entry.slice(separatorIndex + 1).trim();
    if (!key || !value) {
      continue;
    }

    result[key] = value;
  }

  return Object.keys(result).length > 0 ? result : undefined;
}
