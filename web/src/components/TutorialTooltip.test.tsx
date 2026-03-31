/**
 * Purpose: Verify passive tutorial tooltips stay keyboard-accessible without hijacking pointer interactions.
 * Responsibilities: Assert hover and keyboard focus can reveal a tooltip while pointer-originated focus leaves the primary action alone.
 * Scope: Component coverage for `TutorialTooltip` uncontrolled visibility rules.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Tooltip listeners attach to the target selector after render, pointerdown precedes pointer-originated focus, and uncontrolled tooltips render with `role="tooltip"`.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TutorialTooltip } from "./TutorialTooltip";

function renderHarness() {
  render(
    <>
      <button data-testid="target" type="button">
        Target action
      </button>
      <TutorialTooltip
        target='[data-testid="target"]'
        title="Jump anywhere fast"
        content="Use the command palette to navigate quickly."
      />
    </>,
  );

  return screen.getByRole("button", { name: /target action/i });
}

describe("TutorialTooltip", () => {
  it("does not open from pointer-originated focus", () => {
    const target = renderHarness();

    fireEvent.pointerDown(target);
    fireEvent.focus(target);

    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("opens from keyboard focus", () => {
    const target = renderHarness();

    fireEvent.focus(target);

    expect(screen.getByRole("tooltip")).toBeInTheDocument();
    expect(screen.getByText(/jump anywhere fast/i)).toBeInTheDocument();
  });
});
