/**
 * Purpose: Verify the Settings first-run overview only appears for truly pristine workspaces.
 * Responsibilities: Cover the empty-state happy path plus the operator-work and optional-subsystem gates that should suppress the overview.
 * Scope: Pure `shouldShowSettingsOverviewPanel` decision logic only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Any persisted jobs or non-quiet optional subsystem state means the operator has moved beyond first-run Settings guidance.
 */

import { describe, expect, it } from "vitest";
import { shouldShowSettingsOverviewPanel } from "./settings-overview";

describe("shouldShowSettingsOverviewPanel", () => {
  const pristineInput = {
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
  } as const;

  it("shows the overview for a pristine Settings workspace", () => {
    expect(shouldShowSettingsOverviewPanel(pristineInput)).toBe(true);
  });

  it("hides the overview once any job has been created", () => {
    expect(
      shouldShowSettingsOverviewPanel({
        ...pristineInput,
        jobsTotal: 1,
      }),
    ).toBe(false);
  });

  it("hides the overview when an optional subsystem needs attention", () => {
    expect(
      shouldShowSettingsOverviewPanel({
        ...pristineInput,
        proxyStatus: "degraded",
      }),
    ).toBe(false);
  });
});
