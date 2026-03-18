/**
 * Purpose: Verify route-help next actions stay visible and wired for the active route.
 * Responsibilities: Assert route-specific action labels render and invoke the supplied callback.
 * Scope: RouteHelpPanel interaction coverage only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Route help content comes from shared onboarding configuration and next actions remain route-aware.
 */

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { RouteHelpPanel } from "./RouteHelpPanel";

const shortcuts = {
  commandPalette: "mod+k",
  submitForm: "mod+enter",
  search: "/",
  help: "?",
  escape: "escape",
  navigateJobs: "g j",
  navigateResults: "g r",
  navigateForms: "g f",
};

describe("RouteHelpPanel", () => {
  it("renders route-specific next actions and invokes the handler", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();

    render(
      <RouteHelpPanel
        routeKey="templates"
        shortcuts={shortcuts}
        defaultExpanded
        onOpenCommandPalette={vi.fn()}
        onOpenShortcuts={vi.fn()}
        onRestartTour={vi.fn()}
        onAction={onAction}
      />,
    );

    const createJobButton = screen.getByRole("button", {
      name: /create first job/i,
    });
    expect(
      screen.getByRole("button", { name: /open automation/i }),
    ).toBeInTheDocument();

    await user.click(createJobButton);

    expect(onAction).toHaveBeenCalledWith("create-job");
  });
});
