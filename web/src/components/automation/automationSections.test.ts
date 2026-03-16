/**
 * Purpose: Verify automation hub path and legacy-hash helpers stay stable.
 * Responsibilities: Cover canonical section parsing, URL building, and legacy anchor migration behavior.
 * Scope: Shared automation section helper behavior only.
 * Usage: Run with Vitest as part of the web unit test suite.
 * Invariants/Assumptions: `/automation/:section` remains the canonical deep-link shape and known legacy hashes keep mapping to supported sections.
 */

import { describe, expect, it } from "vitest";
import {
  DEFAULT_AUTOMATION_SECTION,
  getAutomationPath,
  getAutomationSectionFromHash,
  getAutomationSectionFromPath,
} from "./automationSections";

describe("automationSections", () => {
  it("resolves canonical automation paths", () => {
    expect(getAutomationSectionFromPath("/automation")).toBe(
      DEFAULT_AUTOMATION_SECTION,
    );
    expect(getAutomationSectionFromPath("/automation/chains")).toBe("chains");
    expect(getAutomationSectionFromPath("/automation/webhooks")).toBe(
      "webhooks",
    );
    expect(getAutomationSectionFromPath("/jobs")).toBeNull();
  });

  it("maps legacy hash links into the new section model", () => {
    expect(getAutomationSectionFromHash("#chains")).toBe("chains");
    expect(getAutomationSectionFromHash("#export-schedules")).toBe("exports");
    expect(getAutomationSectionFromHash("#unknown")).toBeNull();
  });

  it("builds stable automation section URLs", () => {
    expect(getAutomationPath("watches")).toBe("/automation/watches");
  });
});
