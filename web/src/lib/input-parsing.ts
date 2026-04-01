/**
 * Purpose: Provide reusable input parsing helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
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

export function parseOptionalNumber(
  label: string,
  value: string,
): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }

  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed)) {
    throw new Error(`${label} must be a valid number`);
  }

  return parsed;
}

export function parseOptionalWholeNumber(
  label: string,
  value: string,
): number | undefined {
  const parsed = parseOptionalNumber(label, value);
  if (parsed === undefined) {
    return undefined;
  }
  if (!Number.isInteger(parsed)) {
    throw new Error(`${label} must be a whole number`);
  }

  return parsed;
}

export function parseOptionalNonNegativeInteger(
  label: string,
  value: string,
): number | undefined {
  const parsed = parseOptionalWholeNumber(label, value);
  if (parsed === undefined) {
    return undefined;
  }
  if (parsed < 0) {
    throw new Error(`${label} must be non-negative`);
  }

  return parsed;
}

export function parseOptionalNumberInRange(
  label: string,
  value: string,
  min: number,
  max: number,
): number | undefined {
  const parsed = parseOptionalNumber(label, value);
  if (parsed === undefined) {
    return undefined;
  }
  if (parsed < min || parsed > max) {
    throw new Error(`${label} must be between ${min} and ${max}`);
  }

  return parsed;
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
