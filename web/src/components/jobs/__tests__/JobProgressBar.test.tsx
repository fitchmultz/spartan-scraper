/**
 * Purpose: Verify accessible determinate and indeterminate progress rendering for job cards.
 * Responsibilities: Assert progress semantics and visible value text for batch and running states.
 * Scope: Unit coverage for `JobProgressBar`.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Progress labels and values come from the dashboard view-model layer.
 */

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { JobProgressBar } from "../JobProgressBar";

describe("JobProgressBar", () => {
  it("renders determinate progress", () => {
    render(
      <JobProgressBar
        progress={{
          label: "Batch 2 of 5",
          percent: 40,
          valueText: "2 complete · 40%",
        }}
      />,
    );

    const progressbar = screen.getByRole("progressbar", {
      name: /batch 2 of 5/i,
    });

    expect(progressbar).toHaveAttribute("value", "40");
    expect(screen.getByText(/2 complete · 40%/i)).toBeInTheDocument();
  });

  it("renders indeterminate running progress", () => {
    render(
      <JobProgressBar
        progress={{
          label: "Running",
          valueText: "In progress",
          indeterminate: true,
        }}
      />,
    );

    const progressbar = screen.getByRole("progressbar", {
      name: /running/i,
    });

    expect(progressbar).toHaveAttribute("aria-valuetext", "In progress");
    expect(progressbar).not.toHaveAttribute("value");
  });
});
