/**
 * Purpose: Verify the automation hub sub-navigation renders the full section set and reports selection changes.
 * Responsibilities: Confirm active-state semantics and click handling for automation section switching.
 * Scope: AutomationSubnav UI behavior only.
 * Usage: Run with Vitest and React Testing Library as part of the web test suite.
 * Invariants/Assumptions: Each navigation item is rendered as a button and the active section exposes `aria-current="page"`.
 */

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AutomationSubnav } from "./AutomationSubnav";

describe("AutomationSubnav", () => {
  it("renders all automation sections and marks the active section", () => {
    render(
      <AutomationSubnav activeSection="watches" onSectionChange={vi.fn()} />,
    );

    expect(screen.getByRole("button", { name: "Watches" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByRole("button", { name: "Batches" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Chains" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Exports" })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Webhooks" }),
    ).toBeInTheDocument();
  });

  it("calls onSectionChange with the selected section", async () => {
    const user = userEvent.setup();
    const onSectionChange = vi.fn();

    render(
      <AutomationSubnav
        activeSection="batches"
        onSectionChange={onSectionChange}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Chains" }));

    expect(onSectionChange).toHaveBeenCalledWith("chains");
  });
});
