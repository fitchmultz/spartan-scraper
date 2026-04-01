/**
 * Purpose: Verify fresh-start Settings journeys stay canonical and resilient.
 * Responsibilities: Cover first-visit Settings guidance, section URL binding, and local failure containment.
 * Scope: Route-level Settings behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Shared route mocks come from the FreshStart operator-flow harness and Settings deep links remain canonical.
 */

import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import {
  getAppDataState,
  loadAppModule,
  renderAppAt,
  setSettingsPanelMessages,
  setupFreshStartOperatorFlowTest,
} from "./freshStartOperatorFlowHarness";

setupFreshStartOperatorFlowTest();

describe("FreshStartOperatorFlowSettings", () => {
  it("shows first-visit Settings guidance, then retires the overview after the first job", async () => {
    const user = userEvent.setup();

    const firstRender = await renderAppAt("/settings");

    await waitFor(() => {
      expect(window.location.pathname).toBe("/settings/authoring");
    });

    expect(
      screen.queryByRole("heading", { name: /start with one working job/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /show details/i }),
    ).toBeInTheDocument();
    const settingsOverview = await screen.findByText(
      /most settings controls can wait until a workflow proves it needs them/i,
    );
    expect(settingsOverview).toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: /settings sections/i }),
    ).toBeInTheDocument();
    const authoringHeading = screen.getByRole("heading", {
      name: /authoring tools/i,
    });
    expect(authoringHeading).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: /saved state and history/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: /operational controls/i }),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("render-profile-editor")).toBeInTheDocument();
    expect(screen.getByTestId("pipeline-js-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("proxy-pool-status")).not.toBeInTheDocument();
    expect(screen.queryByTestId("retention-status")).not.toBeInTheDocument();

    const routeHelp = screen.getByLabelText(
      /what can i do here\? for this route/i,
    );
    expect(
      authoringHeading.compareDocumentPosition(settingsOverview) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      settingsOverview.compareDocumentPosition(routeHelp) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();

    await user.click(
      within(routeHelp).getByRole("button", { name: /^create job$/i }),
    );

    await waitFor(() => {
      expect(window.location.pathname).toBe("/jobs/new");
    });

    firstRender.unmount();

    const secondRender = await renderAppAt("/settings/authoring");

    expect(
      screen.getByLabelText(/what can i do here\? for this route/i),
    ).toBeInTheDocument();

    const appDataState = getAppDataState();
    const { App } = await loadAppModule();
    appDataState.jobsTotal = 1;
    secondRender.rerender(<App />);

    await waitFor(() => {
      expect(
        screen.queryByText(
          /most settings controls can wait until a workflow proves it needs them/i,
        ),
      ).not.toBeInTheDocument();
    });
  });

  it("binds Settings section selection to the canonical URL and browser history", async () => {
    const user = userEvent.setup();

    await renderAppAt("/settings/authoring");

    const authoringButton = screen.getByRole("button", {
      name: /authoring tools/i,
    });
    const operationsButton = screen.getByRole("button", {
      name: /operations/i,
    });

    expect(authoringButton).toHaveAttribute("aria-current", "page");
    expect(operationsButton).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("render-profile-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("proxy-pool-status")).not.toBeInTheDocument();

    await user.click(operationsButton);

    await waitFor(() => {
      expect(window.location.pathname).toBe("/settings/operations");
    });

    expect(operationsButton).toHaveAttribute("aria-current", "page");
    expect(authoringButton).not.toHaveAttribute("aria-current");
    expect(
      screen.getByRole("heading", { name: /operational controls/i }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("proxy-pool-status")).toBeInTheDocument();
    expect(
      screen.queryByTestId("render-profile-editor"),
    ).not.toBeInTheDocument();

    window.history.back();

    await waitFor(() => {
      expect(window.location.pathname).toBe("/settings/authoring");
    });

    expect(authoringButton).toHaveAttribute("aria-current", "page");
    expect(operationsButton).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("render-profile-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("proxy-pool-status")).not.toBeInTheDocument();
  });

  it("keeps shared Settings chrome visible when Operations panels fail locally", async () => {
    setSettingsPanelMessages({
      proxyMessage: "Proxy pool metadata could not be loaded.",
      retentionMessage: "Retention metadata could not be loaded.",
    });

    await renderAppAt("/settings/operations");

    expect(
      screen.getByRole("heading", { name: /operational controls/i }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("proxy-pool-status")).toHaveTextContent(
      "Proxy pool metadata could not be loaded.",
    );
    expect(screen.getByTestId("retention-status")).toHaveTextContent(
      "Retention metadata could not be loaded.",
    );
    expect(
      screen.getByRole("heading", { name: /^settings$/i }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/settings sections/i)).toBeInTheDocument();
    expect(
      screen.getByLabelText(/what can i do here\? for this route/i),
    ).toBeInTheDocument();
  });
});
