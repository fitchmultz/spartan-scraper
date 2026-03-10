/**
 * Shared template A/B test presentation helpers.
 *
 * Purpose:
 * - Centralize small formatting and display decisions for template A/B tests.
 *
 * Responsibilities:
 * - Map A/B test statuses to badge classes.
 * - Format allocation, confidence, and winner labels consistently.
 *
 * Scope:
 * - Web UI helpers for template A/B test views only.
 *
 * Usage:
 * - Import from template A/B test components instead of repeating inline
 *   formatting logic.
 *
 * Invariants/Assumptions:
 * - Missing values fall back to safe display defaults.
 * - Unknown statuses should not receive a styled badge by accident.
 */

import type { TemplateAbTest } from "../api";

export function getTemplateABTestStatusBadgeClass(
  status?: TemplateAbTest["status"],
): string {
  switch (status) {
    case "pending":
      return "badge--neutral";
    case "running":
      return "badge--success";
    case "paused":
      return "badge--warning";
    case "completed":
      return "badge--info";
    default:
      return "";
  }
}

export function formatTemplateABTestAllocation(
  allocation: TemplateAbTest["allocation"] | undefined,
): string {
  return `${allocation?.baseline ?? 50}% / ${allocation?.variant ?? 50}%`;
}

export function formatTemplateABTestConfidence(
  confidenceLevel: number | undefined,
): string {
  return `${((confidenceLevel ?? 0.95) * 100).toFixed(0)}%`;
}

export function getTemplateABTestWinnerName(
  test: Pick<
    TemplateAbTest,
    "winner" | "baseline_template" | "variant_template"
  >,
): string | undefined {
  if (!test.winner) {
    return undefined;
  }

  return test.winner === "baseline"
    ? test.baseline_template
    : test.variant_template;
}
