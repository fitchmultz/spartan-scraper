/**
 * Purpose: Verify the Settings first-run overview stays actionable and capability-oriented.
 * Responsibilities: Assert the overview teaches operators what each Settings surface is for and exposes the primary next steps.
 * Scope: SettingsOverviewPanel rendering only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: First-run Settings guidance should stay calm, comprehensive, and biased toward starting real work before extra configuration.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SettingsOverviewPanel } from "./SettingsOverviewPanel";

describe("SettingsOverviewPanel", () => {
  it("summarizes each major Settings capability and exposes first-run actions", () => {
    const onCreateJob = vi.fn();
    const onOpenJobs = vi.fn();

    render(
      <SettingsOverviewPanel
        onCreateJob={onCreateJob}
        onOpenJobs={onOpenJobs}
      />,
    );

    expect(
      screen.getByText(
        /most settings controls can wait until a workflow proves it needs them/i,
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Auth Profiles")).toBeInTheDocument();
    expect(screen.getByText("Schedules")).toBeInTheDocument();
    expect(screen.getByText("Crawl States")).toBeInTheDocument();
    expect(screen.getByText("Render Profiles")).toBeInTheDocument();
    expect(screen.getByText("Pipeline JavaScript")).toBeInTheDocument();
    expect(screen.getByText("Proxy Pool")).toBeInTheDocument();
    expect(screen.getByText("Retention")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /create job/i }));
    fireEvent.click(screen.getByRole("button", { name: /review jobs/i }));

    expect(onCreateJob).toHaveBeenCalledTimes(1);
    expect(onOpenJobs).toHaveBeenCalledTimes(1);
  });
});
