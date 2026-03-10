/**
 * Template A/B test helper tests.
 *
 * Purpose:
 * - Verify shared A/B test formatting helpers stay consistent.
 *
 * Responsibilities:
 * - Cover badge-class mappings for test statuses.
 * - Cover formatting helpers for winner, allocation, and confidence output.
 *
 * Scope:
 * - Unit tests for web/src/lib/template-ab-tests.ts only.
 *
 * Usage:
 * - Run through Vitest as part of the web test suite.
 *
 * Invariants/Assumptions:
 * - Missing values should fall back to stable defaults.
 */

import { describe, expect, it } from "vitest";

import {
  formatTemplateABTestAllocation,
  formatTemplateABTestConfidence,
  getTemplateABTestStatusBadgeClass,
  getTemplateABTestWinnerName,
} from "./template-ab-tests";

describe("getTemplateABTestStatusBadgeClass", () => {
  it("maps known statuses to badge classes", () => {
    expect(getTemplateABTestStatusBadgeClass("pending")).toBe("badge--neutral");
    expect(getTemplateABTestStatusBadgeClass("running")).toBe("badge--success");
    expect(getTemplateABTestStatusBadgeClass("paused")).toBe("badge--warning");
    expect(getTemplateABTestStatusBadgeClass("completed")).toBe("badge--info");
  });

  it("falls back to no extra badge class", () => {
    expect(getTemplateABTestStatusBadgeClass(undefined)).toBe("");
  });
});

describe("template A/B test formatters", () => {
  it("formats allocation and confidence with defaults", () => {
    expect(formatTemplateABTestAllocation(undefined)).toBe("50% / 50%");
    expect(formatTemplateABTestConfidence(undefined)).toBe("95%");
  });

  it("formats winner names from the selected side", () => {
    expect(
      getTemplateABTestWinnerName({
        winner: "baseline",
        baseline_template: "baseline-template",
        variant_template: "variant-template",
      }),
    ).toBe("baseline-template");
    expect(
      getTemplateABTestWinnerName({
        winner: "variant",
        baseline_template: "baseline-template",
        variant_template: "variant-template",
      }),
    ).toBe("variant-template");
    expect(
      getTemplateABTestWinnerName({
        winner: undefined,
        baseline_template: "baseline-template",
        variant_template: "variant-template",
      }),
    ).toBeUndefined();
  });
});
