/**
 * Purpose: Verify fresh-start navigation helpers and command-palette routing remain deterministic.
 * Responsibilities: Cover command-palette route navigation, reopened search state, and path normalization helpers.
 * Scope: App-shell navigation behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Shared route mocks come from the FreshStart operator-flow harness and canonical route parsing stays centralized.
 */

import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import {
  createJob,
  getAppDataState,
  getKeyboardState,
  loadApiConfigHelpers,
  loadAppModule,
  loadRoutingHelpers,
  renderAppAt,
  setupFreshStartOperatorFlowTest,
} from "./freshStartOperatorFlowHarness";

setupFreshStartOperatorFlowTest();

describe("FreshStartOperatorFlowNavigation", () => {
  it("navigates to every major route from the command palette", async () => {
    const user = userEvent.setup();

    const appDataState = getAppDataState();
    const keyboardState = getKeyboardState();
    appDataState.jobs = [createJob("job-12345678")];
    appDataState.jobsTotal = 1;
    keyboardState.isCommandPaletteOpen = true;

    await renderAppAt("/jobs");

    const palette = screen.getByRole("dialog", { name: /command palette/i });

    const routeCases = [
      ["Open Jobs", "/jobs", "Jobs"],
      ["Create Job", "/jobs/new", "Create Job"],
      ["Open Templates", "/templates", "Templates"],
      ["Open Settings", "/settings/authoring", "Settings"],
      ["Open Automation / Batches", "/automation/batches", "Automation"],
      ["Open Automation / Chains", "/automation/chains", "Automation"],
      ["Open Automation / Watches", "/automation/watches", "Automation"],
      ["Open Automation / Exports", "/automation/exports", "Automation"],
      ["Open Automation / Webhooks", "/automation/webhooks", "Automation"],
    ] as const;

    for (const [label, path, heading] of routeCases) {
      await user.click(within(palette).getByText(label));

      await waitFor(() => {
        expect(window.location.pathname).toBe(path);
      });

      expect(
        screen.getByRole("heading", { name: heading }),
      ).toBeInTheDocument();
    }

    expect(screen.getByTestId("automation-active-section")).toHaveTextContent(
      "webhooks",
    );
  });

  it("reopens the command palette with a fresh search input", async () => {
    const user = userEvent.setup();

    const keyboardState = getKeyboardState();
    keyboardState.isCommandPaletteOpen = true;
    const rendered = await renderAppAt("/jobs");

    await user.type(screen.getByLabelText("Search commands"), "template");
    expect(screen.getByLabelText("Search commands")).toHaveValue("template");

    const { App } = await loadAppModule();

    keyboardState.isCommandPaletteOpen = false;
    rendered.rerender(<App />);

    await waitFor(() => {
      expect(
        screen.queryByRole("dialog", { name: /command palette/i }),
      ).not.toBeInTheDocument();
    });

    keyboardState.isCommandPaletteOpen = true;
    rendered.rerender(<App />);

    await waitFor(() => {
      expect(screen.getByLabelText("Search commands")).toHaveValue("");
    });
  });

  it("keeps browser API base URL resolution deterministic in test mode", async () => {
    const { getApiBaseUrl } = await loadApiConfigHelpers();
    expect(getApiBaseUrl()).toBe("");
  });

  it("normalizes paths, resolves Settings sections, and falls back unknown routes to Jobs", async () => {
    const { normalizePath, parseRoute } = await loadRoutingHelpers();

    expect(normalizePath("")).toBe("/jobs");
    expect(normalizePath("/")).toBe("/jobs");
    expect(normalizePath("/settings///")).toBe("/settings");
    expect(normalizePath("/jobs/new/")).toBe("/jobs/new");

    expect(parseRoute("/settings/operations")).toMatchObject({
      kind: "settings",
      path: "/settings/operations",
      settingsSection: "operations",
    });
    expect(parseRoute("/settings/unknown")).toMatchObject({
      kind: "settings",
      path: "/settings/unknown",
      settingsSection: "authoring",
    });
    expect(parseRoute("/mystery-path")).toMatchObject({
      kind: "jobs",
      path: "/jobs",
    });
  });
});
