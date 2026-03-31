/**
 * Purpose: Verify job card rendering and operator actions inside the jobs monitoring dashboard.
 * Responsibilities: Assert failure/dependency presentation and callback wiring for card actions.
 * Scope: Unit coverage for `JobRunCard`.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: The card receives precomputed action flags and failure context from the dashboard model.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { JobMonitorCardModel } from "../../../lib/job-monitoring";
import { JobRunCard } from "../JobRunCard";

const succeededModel: JobMonitorCardModel = {
  id: "job-success",
  shortId: "job-succ…0001",
  rawId: "job-success-0000-1111",
  kind: "scrape",
  status: "succeeded",
  updatedAtLabel: "Updated 2m ago",
  timeline: [
    { label: "Wait", value: "500ms" },
    { label: "Run", value: "1.2s" },
    { label: "Total", value: "1.7s" },
  ],
  activityText: "Completed successfully",
  canViewResults: true,
  canCancel: false,
  canDelete: true,
};

describe("JobRunCard", () => {
  it("renders failure context and dependency badges for attention cards", () => {
    render(
      <JobRunCard
        lane="attention"
        model={{
          ...succeededModel,
          id: "job-failed",
          shortId: "job-fail…0001",
          rawId: "job-failed-0000-0001",
          status: "failed",
          dependencyStatus: "failed",
          canViewResults: false,
          failure: {
            tone: "danger",
            category: "Dependency",
            summary: "An upstream dependency failed and blocked this run.",
          },
        }}
        onViewResults={vi.fn()}
        onCancel={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByText(/deps: failed/i)).toBeInTheDocument();
    expect(screen.getAllByText(/dependency/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/upstream dependency failed/i)).toBeInTheDocument();
  });

  it("fires action callbacks", () => {
    const onViewResults = vi.fn();
    const onDelete = vi.fn();

    render(
      <JobRunCard
        lane="completed"
        model={succeededModel}
        onViewResults={onViewResults}
        onCancel={vi.fn()}
        onDelete={onDelete}
      />,
    );

    const viewResultsLink = screen.getByRole("link", {
      name: /view results/i,
    });

    expect(viewResultsLink).toHaveAttribute("href", "/jobs/job-success");

    fireEvent.click(viewResultsLink);
    fireEvent.click(screen.getByRole("button", { name: /delete/i }));

    expect(onViewResults).toHaveBeenCalledWith("job-success");
    expect(onDelete).toHaveBeenCalledWith("job-success");
  });
});
