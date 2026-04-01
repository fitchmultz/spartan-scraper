/**
 * Purpose: Verify browser execution controls behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BrowserExecutionControls } from "./BrowserExecutionControls";

describe("BrowserExecutionControls", () => {
  it("explains why Playwright is unavailable until headless is enabled", () => {
    render(
      <BrowserExecutionControls
        headless={false}
        setHeadless={vi.fn()}
        usePlaywright={false}
        setUsePlaywright={vi.fn()}
        timeoutSeconds={30}
        setTimeoutSeconds={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("Playwright")).toBeDisabled();
    expect(
      screen.getByText(
        /enable headless to unlock playwright, device emulation, and browser-only diagnostics/i,
      ),
    ).toBeInTheDocument();
  });

  it("removes the dependency hint once headless is enabled", () => {
    const setUsePlaywright = vi.fn();

    render(
      <BrowserExecutionControls
        headless
        setHeadless={vi.fn()}
        usePlaywright={false}
        setUsePlaywright={setUsePlaywright}
        timeoutSeconds={30}
        setTimeoutSeconds={vi.fn()}
      />,
    );

    const playwright = screen.getByLabelText("Playwright");
    expect(playwright).toBeEnabled();
    expect(
      screen.queryByText(/enable headless to unlock playwright/i),
    ).not.toBeInTheDocument();

    fireEvent.click(playwright);
    expect(setUsePlaywright).toHaveBeenCalledWith(true);
  });
});
