/**
 * Purpose: Verify the Settings first-run overview only appears for truly pristine workspaces.
 * Responsibilities: Cover route gating, loading suppression, inventory retirement, and optional-subsystem quiet-state handling.
 * Scope: Pure `shouldShowSettingsOverviewPanel` decision logic only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Any persisted jobs, saved reusable inventory, or non-quiet optional subsystem state means the operator has moved beyond first-run Settings guidance.
 */

import { describe, expect, it } from "vitest";
import { shouldShowSettingsOverviewPanel } from "./settings-overview";

type SettingsOverviewInput = Parameters<
  typeof shouldShowSettingsOverviewPanel
>[0];

function buildInput(
  overrides: Partial<SettingsOverviewInput> = {},
): SettingsOverviewInput {
  return {
    isSettingsRoute: true,
    setupRequired: false,
    jobsTotal: 0,
    profilesCount: 0,
    schedulesCount: 0,
    crawlStatesTotal: 0,
    renderProfileCount: 0,
    pipelineScriptCount: 0,
    proxyStatus: "disabled",
    retentionStatus: "disabled",
    ...overrides,
  };
}

describe("shouldShowSettingsOverviewPanel", () => {
  it("returns false when the current route is not Settings", () => {
    expect(
      shouldShowSettingsOverviewPanel(buildInput({ isSettingsRoute: false })),
    ).toBe(false);
  });

  it("returns false when setup recovery mode is active", () => {
    expect(
      shouldShowSettingsOverviewPanel(buildInput({ setupRequired: true })),
    ).toBe(false);
  });

  it("returns false while render profile inventory is still loading", () => {
    expect(
      shouldShowSettingsOverviewPanel(buildInput({ renderProfileCount: null })),
    ).toBe(false);
  });

  it("returns false while pipeline script inventory is still loading", () => {
    expect(
      shouldShowSettingsOverviewPanel(
        buildInput({ pipelineScriptCount: null }),
      ),
    ).toBe(false);
  });

  it("returns true for a pristine Settings workspace", () => {
    expect(shouldShowSettingsOverviewPanel(buildInput())).toBe(true);
  });

  it.each([
    ["first job", { jobsTotal: 1 }],
    ["saved auth profile", { profilesCount: 1 }],
    ["saved schedule", { schedulesCount: 1 }],
    ["crawl state history", { crawlStatesTotal: 1 }],
    ["render profile inventory", { renderProfileCount: 1 }],
    ["pipeline script inventory", { pipelineScriptCount: 1 }],
  ] as const)("returns false after %s exists", (_label, overrides) => {
    expect(shouldShowSettingsOverviewPanel(buildInput(overrides))).toBe(false);
  });

  it("returns false when the proxy subsystem is non-quiet", () => {
    expect(
      shouldShowSettingsOverviewPanel(buildInput({ proxyStatus: "degraded" })),
    ).toBe(false);
  });

  it("returns false when the retention subsystem is non-quiet", () => {
    expect(
      shouldShowSettingsOverviewPanel(
        buildInput({ retentionStatus: "setup_required" }),
      ),
    ).toBe(false);
  });

  it("treats undefined optional subsystem states as quiet", () => {
    expect(
      shouldShowSettingsOverviewPanel(
        buildInput({ proxyStatus: undefined, retentionStatus: undefined }),
      ),
    ).toBe(true);
  });
});
