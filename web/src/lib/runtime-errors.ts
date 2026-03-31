/**
 * Purpose: Provide one shared frontend runtime-error reporting path for operator-facing web flows.
 * Responsibilities: Normalize unknown runtime errors into user-safe messages, emit one structured console error entry, and let callers reuse the resolved message for UI state or toasts.
 * Scope: Browser-side logging and message derivation only; transport-specific retries and UI presentation stay in calling hooks/components.
 * Usage: Call `reportRuntimeError(scope, error, { fallback })` instead of ad hoc `console.error(...)` handling.
 * Invariants/Assumptions: The provided scope is already user-safe, this helper never throws while formatting/logging unknown errors, and callers own whether the returned message is surfaced to operators.
 */

import { getApiErrorMessage } from "./api-errors";

export interface RuntimeErrorReportOptions {
  fallback?: string;
  logger?: Pick<Console, "error">;
}

export function reportRuntimeError(
  scope: string,
  error: unknown,
  options: RuntimeErrorReportOptions = {},
): string {
  const { fallback = scope, logger = console } = options;
  const message = getApiErrorMessage(error, fallback);
  logger.error(`${scope}: ${message}`, error);
  return message;
}
