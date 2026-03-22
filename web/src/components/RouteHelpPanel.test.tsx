/**
 * Purpose: Verify route-help summary actions stay visible while detailed help remains opt-in.
 * Responsibilities: Assert route-specific action labels render collapsed by default and still invoke the supplied callback.
 * Scope: RouteHelpPanel interaction coverage only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Route help content comes from shared onboarding configuration, next actions remain route-aware, and details only expand on demand.
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
  it("renders the route-specific next action while keeping details collapsed by default", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();

    render(
      <RouteHelpPanel
        routeKey="templates"
        shortcuts={shortcuts}
        onOpenCommandPalette={vi.fn()}
        onOpenShortcuts={vi.fn()}
        onRestartTour={vi.fn()}
        onAction={onAction}
      />,
    );

    const createJobButton = screen.getByRole("button", {
      name: /create job/i,
    });
    expect(
      screen.queryByRole("heading", { name: /shortcuts for this route/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /show details/i }),
    ).toBeInTheDocument();

    await user.click(createJobButton);

    expect(onAction).toHaveBeenCalledWith("create-job");
  });
});
