/**
 * Purpose: Verify fresh-start jobs and results routes stay coherent through first-run and promotion flows.
 * Responsibilities: Cover jobs landing guidance, result-route recovery, promotion handoffs, and first-job submission.
 * Scope: Route-level app-shell behavior for Jobs and Results only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Shared route mocks come from the FreshStart operator-flow harness and promotion destinations remain canonical.
 */

import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import {
  createJob,
  getAppDataState,
  loadAppModule,
  makeDetailJob,
  renderAppAt,
  setupFreshStartOperatorFlowTest,
} from "./freshStartOperatorFlowHarness";

setupFreshStartOperatorFlowTest();

describe("FreshStartOperatorFlow", () => {
  it("shows the first-run nudge on a pristine workspace and retires it after work starts", async () => {
    const rendered = await renderAppAt("/jobs");

    expect(
      screen.getByRole("heading", { name: /start with one working job/i }),
    ).toBeInTheDocument();

    const appDataState = getAppDataState();
    const { App } = await loadAppModule();
    appDataState.jobsTotal = 1;
    rendered.rerender(<App />);

    await waitFor(() => {
      expect(
        screen.queryByRole("heading", { name: /start with one working job/i }),
      ).not.toBeInTheDocument();
    });
  });

  it("loads authoritative job detail for direct result routes outside the paged jobs list", async () => {
    const appDataState = getAppDataState();
    appDataState.detailJob = makeDetailJob({ id: "job-direct" });

    await renderAppAt("/jobs/job-direct");

    await waitFor(() => {
      expect(appDataState.refreshJobDetail).toHaveBeenCalledWith("job-direct");
    });

    expect(screen.getByText("job-direct")).toBeInTheDocument();
  });

  it("keeps saved results ahead of secondary framing on the results route", async () => {
    const appDataState = getAppDataState();
    appDataState.detailJob = makeDetailJob({ id: "job-results-first" });

    await renderAppAt("/jobs/job-results-first");

    const results = await screen.findByTestId("results-container");
    const routeHelp = screen.getByLabelText(
      /what can i do here\? for this route/i,
    );

    expect(
      screen.queryByText(
        /read saved output first, then open comparison, transform, and export tools only when needed/i,
      ),
    ).not.toBeInTheDocument();
    expect(
      results.compareDocumentPosition(routeHelp) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("keeps job detail errors focused on the recovery path", async () => {
    const appDataState = getAppDataState();
    appDataState.detailJobError = "invalid job id format: bad-job";

    await renderAppAt("/jobs/bad-job");

    expect(
      await screen.findByRole("heading", {
        name: /unable to load this saved job/i,
      }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText(/result context/i)).not.toBeInTheDocument();
    expect(
      screen.queryByLabelText(/what can i do here\? for this route/i),
    ).not.toBeInTheDocument();
    expect(
      screen.getAllByRole("button", { name: /back to jobs/i }).length,
    ).toBeGreaterThan(0);
  });

  it("hands off promotion drafts from results into the canonical destination workspaces", async () => {
    const user = userEvent.setup();
    const appDataState = getAppDataState();
    appDataState.detailJob = makeDetailJob({ id: "job-promote" });

    await renderAppAt("/jobs/job-promote");

    await waitFor(() => {
      expect(screen.getByText("job-promote")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /promote watch/i }));

    await waitFor(() => {
      expect(window.location.pathname).toBe("/automation/watches");
    });
    expect(screen.getByText("Watch seed job-promote")).toBeInTheDocument();
  });

  it("hands template promotion drafts off to the templates workspace", async () => {
    const user = userEvent.setup();
    const appDataState = getAppDataState();
    appDataState.detailJob = makeDetailJob({ id: "job-template" });

    await renderAppAt("/jobs/job-template");

    await waitFor(() => {
      expect(screen.getByText("job-template")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /promote template/i }));

    await waitFor(() => {
      expect(window.location.pathname).toBe("/templates");
    });
    expect(screen.getByText("Template seed job-template")).toBeInTheDocument();
  });

  it("keeps the new job workspace ahead of first-run framing", async () => {
    await renderAppAt("/jobs/new");

    expect(
      screen.queryByRole("heading", { name: /start with one working job/i }),
    ).not.toBeInTheDocument();

    const wizard = screen.getByTestId("job-submission-container");
    const firstRunNotice = screen.getByText(/start with a single page scrape/i);
    const routeHelp = screen.getByLabelText(
      /what can i do here\? for this route/i,
    );

    expect(
      wizard.compareDocumentPosition(firstRunNotice) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      firstRunNotice.compareDocumentPosition(routeHelp) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("submits the first job from the New Job route, redirects to Jobs, and shows the new run", async () => {
    const user = userEvent.setup();
    const appDataState = getAppDataState();
    appDataState.refreshJobs = vi.fn(async () => {
      appDataState.jobs = [createJob("job-first-run-0001")];
      appDataState.jobsTotal = 1;
    });

    await renderAppAt("/jobs/new");

    await user.click(
      screen.getByRole("button", { name: /submit first scrape/i }),
    );

    await waitFor(() => {
      expect(window.location.pathname).toBe("/jobs");
    });

    expect(appDataState.refreshJobs).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("jobs-dashboard")).toHaveTextContent(
      "job-first-run-0001",
    );
  });
});
