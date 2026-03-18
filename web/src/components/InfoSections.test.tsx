/**
 * Purpose: Verify the Settings inventory stays capability-aware even when no reusable data exists yet.
 * Responsibilities: Assert guided empty-state sections remain visible for auth profiles, schedules, and crawl states.
 * Scope: InfoSections rendering only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Empty settings should still teach operators what each capability is for and what to do next.
 */

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { InfoSections } from "./InfoSections";

describe("InfoSections", () => {
  it("shows guided empty-state sections instead of hiding settings capabilities", () => {
    render(
      <InfoSections
        profiles={[]}
        schedules={[]}
        crawlStates={[]}
        crawlStatesPage={1}
        crawlStatesTotal={0}
        crawlStatesPerPage={100}
        onCrawlStatesPageChange={vi.fn()}
        onCreateJob={vi.fn()}
        onOpenAutomation={vi.fn()}
      />,
    );

    expect(screen.getByText("Auth Profiles")).toBeInTheDocument();
    expect(
      screen.getByText("No reusable auth profiles yet"),
    ).toBeInTheDocument();
    expect(screen.getByText("Schedules")).toBeInTheDocument();
    expect(screen.getByText("No recurring schedules yet")).toBeInTheDocument();
    expect(screen.getByText("Crawl States")).toBeInTheDocument();
    expect(
      screen.getByText("No crawl state has been recorded yet"),
    ).toBeInTheDocument();
  });
});
