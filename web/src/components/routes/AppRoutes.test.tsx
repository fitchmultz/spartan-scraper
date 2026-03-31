/**
 * Purpose: Lock the stable `AppRoutes` export surface used by the application shell and route tests.
 * Responsibilities: Verify the shared route entrypoint keeps re-exporting the split job and workspace route containers.
 * Scope: Route export wiring only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: `App.tsx` and related tests import route containers through `AppRoutes`, so broken re-exports should fail fast here.
 */

import { describe, expect, it } from "vitest";

import {
  AutomationRoute,
  JobDetailRoute,
  JobsRoute,
  NewJobRoute,
  SettingsRoute,
  SetupRequiredRoute,
  TemplatesRoute,
} from "./AppRoutes";
import * as JobRoutes from "./JobRoutes";
import * as WorkspaceRoutes from "./WorkspaceRoutes";

describe("AppRoutes", () => {
  it("re-exports the split route containers through the stable entrypoint", () => {
    expect(JobsRoute).toBe(JobRoutes.JobsRoute);
    expect(JobDetailRoute).toBe(JobRoutes.JobDetailRoute);
    expect(NewJobRoute).toBe(JobRoutes.NewJobRoute);
    expect(TemplatesRoute).toBe(JobRoutes.TemplatesRoute);
    expect(AutomationRoute).toBe(WorkspaceRoutes.AutomationRoute);
    expect(SettingsRoute).toBe(WorkspaceRoutes.SettingsRoute);
    expect(SetupRequiredRoute).toBe(WorkspaceRoutes.SetupRequiredRoute);
  });
});
