/**
 * Purpose: Verify the job submission shell hook keeps route handoffs and destructive actions centralized.
 * Responsibilities: Cover pending preset/submission effects plus direct submit and delete flows.
 * Scope: `useJobSubmissionActions` behavior only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: The hook owns the submission ref, command-palette handoffs are deferred until the new-job route is active, and the shared toast controller drives destructive confirmations.
 */

import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { deleteV1JobsById, postV1Scrape, type ScrapeRequest } from "../api";
import type { JobSubmissionContainerRef } from "../components/jobs/JobSubmissionContainer";
import type { ToastController } from "../components/toast";
import { submitScrapeJob } from "../lib/job-actions";
import type { JobPreset, PresetConfig } from "../types/presets";
import type { RouteKind } from "./useAppShellRouting";
import { useJobSubmissionActions } from "./useJobSubmissionActions";

vi.mock("../api", () => ({
  deleteV1JobsById: vi.fn(),
  postV1Scrape: vi.fn(),
  postV1Crawl: vi.fn(),
  postV1Research: vi.fn(),
}));

vi.mock("../lib/job-actions", () => ({
  submitScrapeJob: vi.fn(),
  submitCrawlJob: vi.fn(),
  submitResearchJob: vi.fn(),
}));

beforeEach(() => {
  vi.clearAllMocks();
});

function createToastController(confirmResult = true): ToastController {
  return {
    show: vi.fn(() => "toast-1"),
    update: vi.fn(),
    dismiss: vi.fn(),
    confirm: vi.fn().mockResolvedValue(confirmResult),
  };
}

function createSubmissionRef(
  overrides: Partial<JobSubmissionContainerRef> = {},
): JobSubmissionContainerRef {
  return {
    submitScrape: vi.fn().mockResolvedValue(undefined),
    submitCrawl: vi.fn().mockResolvedValue(undefined),
    submitResearch: vi.fn().mockResolvedValue(undefined),
    setScrapeUrl: vi.fn(),
    setCrawlUrl: vi.fn(),
    setResearchQuery: vi.fn(),
    getScrapeUrl: vi.fn(() => ""),
    getCrawlUrl: vi.fn(() => ""),
    getCurrentConfig: vi.fn(() => ({})),
    applyPreset: vi.fn(),
    clearDraft: vi.fn(),
    ...overrides,
  };
}

function renderActions(
  routeKind: RouteKind = "jobs",
  selectedJobId: string | null = null,
) {
  const navigate = vi.fn();
  const refreshJobs = vi.fn().mockResolvedValue(undefined);
  const toast = createToastController();
  const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

  const hook = renderHook(
    ({ currentRouteKind }: { currentRouteKind: RouteKind }) =>
      useJobSubmissionActions({
        navigate,
        routeKind: currentRouteKind,
        refreshJobs,
        selectedJobId,
        toast,
        getApiBaseUrl,
      }),
    { initialProps: { currentRouteKind: routeKind } },
  );

  return {
    ...hook,
    navigate,
    refreshJobs,
    toast,
    getApiBaseUrl,
  };
}

describe("useJobSubmissionActions", () => {
  it("applies a pending preset once the new-job route becomes active", async () => {
    const { result, rerender, navigate } = renderActions("jobs");
    const applyPreset = vi.fn();

    (
      result.current.jobSubmissionRef as {
        current: JobSubmissionContainerRef | null;
      }
    ).current = createSubmissionRef({ applyPreset });

    const preset = {
      id: "preset-1",
      name: "Preset",
      description: "Preset description",
      icon: "⚙️",
      jobType: "crawl",
      config: {} as PresetConfig,
      resources: { timeSeconds: 10, cpu: "low", memory: "low" },
      useCases: ["Testing"],
      isBuiltIn: false,
    } satisfies JobPreset;

    await act(async () => {
      result.current.handleSelectPreset(preset);
    });

    expect(navigate).toHaveBeenCalledWith("/jobs/new");
    expect(applyPreset).not.toHaveBeenCalled();

    rerender({ currentRouteKind: "new-job" });

    await waitFor(() => {
      expect(applyPreset).toHaveBeenCalledWith(preset.config, "crawl");
    });
  });

  it("submits a pending form once the new-job route becomes active", async () => {
    const { result, rerender, navigate } = renderActions("jobs");
    const submitResearch = vi.fn().mockResolvedValue(undefined);

    (
      result.current.jobSubmissionRef as {
        current: JobSubmissionContainerRef | null;
      }
    ).current = createSubmissionRef({ submitResearch });

    await act(async () => {
      await result.current.handleSubmitForm("research");
    });

    expect(navigate).toHaveBeenCalledWith("/jobs/new");
    expect(submitResearch).not.toHaveBeenCalled();

    rerender({ currentRouteKind: "new-job" });

    await waitFor(() => {
      expect(submitResearch).toHaveBeenCalledTimes(1);
    });
  });

  it("submits a scrape job and returns to jobs on success", async () => {
    const { result, refreshJobs, navigate, toast } = renderActions("jobs");
    const clearDraft = vi.fn();
    const submitScrapeJobMock = vi.mocked(submitScrapeJob);

    (
      result.current.jobSubmissionRef as {
        current: JobSubmissionContainerRef | null;
      }
    ).current = createSubmissionRef({ clearDraft });

    const request = { url: "https://example.com" } as ScrapeRequest;
    submitScrapeJobMock.mockImplementation(async (_submit, context) => {
      await context.refreshJobs();
      return { status: "success" };
    });

    await act(async () => {
      await result.current.handleSubmitScrape(request);
    });

    expect(submitScrapeJobMock).toHaveBeenCalledWith(
      postV1Scrape,
      expect.objectContaining({ request, refreshJobs }),
    );
    expect(refreshJobs).toHaveBeenCalledTimes(1);
    expect(clearDraft).toHaveBeenCalledWith("scrape");
    expect(toast.show).toHaveBeenCalledWith(
      expect.objectContaining({
        tone: "loading",
        title: "Submitting scrape job",
      }),
    );
    expect(toast.update).toHaveBeenCalledWith(
      "toast-1",
      expect.objectContaining({
        tone: "success",
        title: "Scrape job queued",
      }),
    );
    expect(navigate).toHaveBeenCalledWith("/jobs");
  });

  it("deletes the selected job after confirmation", async () => {
    const jobId = "job-123";
    const { result, refreshJobs, navigate, toast } = renderActions(
      "job-detail",
      jobId,
    );
    const deleteV1JobsByIdMock = vi.mocked(deleteV1JobsById);
    deleteV1JobsByIdMock.mockResolvedValue({ error: undefined } as never);

    await act(async () => {
      await result.current.deleteJob(jobId);
    });

    expect(toast.confirm).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "Delete this job permanently?",
        confirmLabel: "Delete job",
        cancelLabel: "Keep job",
        tone: "error",
      }),
    );
    expect(toast.show).toHaveBeenCalledWith(
      expect.objectContaining({
        tone: "loading",
        title: `Deleting job ${jobId}`,
      }),
    );
    expect(deleteV1JobsByIdMock).toHaveBeenCalledWith({
      baseUrl: "http://localhost:8080",
      path: { id: jobId },
      query: { force: true },
    });
    expect(refreshJobs).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith("/jobs");
    expect(toast.update).toHaveBeenCalledWith(
      "toast-1",
      expect.objectContaining({
        tone: "success",
        title: "Job deleted",
      }),
    );
  });
});
