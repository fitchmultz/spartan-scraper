/**
 * Formatting utilities for shared UI display concerns.
 *
 * Purpose:
 * - Centralize common date, duration, and string truncation behavior.
 *
 * Responsibilities:
 * - Format ISO timestamps for display with safe fallbacks.
 * - Convert second and millisecond durations into compact labels.
 * - Convert second durations into approximate UI-friendly labels.
 * - Render unknown values into readable, UI-safe display strings.
 * - Truncate long identifiers and URLs consistently.
 *
 * Scope:
 * - Stateless presentation helpers for the web client.
 *
 * Usage:
 * - Import from UI components and other lib modules instead of re-implementing
 *   local formatting helpers.
 *
 * Invariants/Assumptions:
 * - Invalid dates fall back to the original input string.
 * - Empty values use explicit caller-provided fallback labels.
 */

export function formatDateTime(
  value: string | undefined,
  emptyLabel = "-",
): string {
  if (!value) {
    return emptyLabel;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export function formatSecondsAsDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
}

export function formatSecondsAsApproximateDuration(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  if (seconds < 3600) {
    return `~${Math.ceil(seconds / 60)}min`;
  }
  if (seconds < 86400) {
    return `~${Math.ceil(seconds / 3600)}h`;
  }
  return `~${Math.ceil(seconds / 86400)}d`;
}

export function formatMillisecondsAsDuration(
  milliseconds: number | undefined,
  emptyLabel = "-",
): string {
  if (milliseconds === undefined || milliseconds === null) {
    return emptyLabel;
  }
  if (milliseconds < 1) return "<1ms";
  if (milliseconds < 1000) return `${Math.round(milliseconds)}ms`;
  return `${(milliseconds / 1000).toFixed(2)}s`;
}

export function formatDisplayValue(
  value: unknown,
  options?: {
    emptyLabel?: string;
    nullLabel?: string;
    undefinedLabel?: string;
    trueLabel?: string;
    falseLabel?: string;
    maxLength?: number;
    objectLabel?: string;
    arrayLabel?: (length: number) => string;
  },
): string {
  const emptyLabel = options?.emptyLabel ?? "-";
  const nullLabel = options?.nullLabel ?? emptyLabel;
  const undefinedLabel = options?.undefinedLabel ?? emptyLabel;

  if (value === undefined) {
    return undefinedLabel;
  }
  if (value === null) {
    return nullLabel;
  }
  if (typeof value === "boolean") {
    return value
      ? (options?.trueLabel ?? "true")
      : (options?.falseLabel ?? "false");
  }
  if (typeof value === "number" || typeof value === "bigint") {
    return String(value);
  }
  if (typeof value === "string") {
    if (value === "") {
      return emptyLabel;
    }
    return options?.maxLength && value.length > options.maxLength
      ? truncateEnd(value, options.maxLength, emptyLabel)
      : value;
  }
  if (Array.isArray(value)) {
    return options?.arrayLabel?.(value.length) ?? `[${value.length} items]`;
  }
  if (typeof value === "object") {
    return options?.objectLabel ?? "{...}";
  }

  return String(value);
}

export function truncateMiddle(
  value: string | undefined,
  maxLength = 60,
  emptyLabel = "-",
): string {
  if (!value) {
    return emptyLabel;
  }
  if (value.length <= maxLength) {
    return value;
  }

  const segmentLength = Math.max(1, Math.floor((maxLength - 3) / 2));
  return `${value.slice(0, segmentLength)}...${value.slice(-segmentLength)}`;
}

export function truncateEnd(
  value: string | undefined,
  maxLength = 50,
  emptyLabel = "-",
): string {
  if (!value) {
    return emptyLabel;
  }
  if (value.length <= maxLength) {
    return value;
  }
  return `${value.slice(0, maxLength)}...`;
}
