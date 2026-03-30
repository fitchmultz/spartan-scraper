/**
 * Purpose: Centralize job submission and lifecycle actions for the web shell.
 * Responsibilities: Own the active job tab, queue pending preset/submission handoffs, and expose toast-driven submit/cancel/delete handlers.
 * Scope: New-job workflow orchestration plus job mutation UX only.
 * Usage: Call from `App.tsx` once per shell render and pass the returned state/actions into the route containers and command palette.
 * Invariants/Assumptions: The shell owns the job submission ref, route changes are canonicalized through `useAppShellRouting`, and destructive actions confirm through the shared toast controller.
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type RefObject,
} from "react";

import {
  deleteV1JobsById,
  postV1Crawl,
  postV1Research,
  postV1Scrape,
  type CrawlRequest,
  type ResearchRequest,
  type ScrapeRequest,
} from "../api";
import type { JobSubmissionContainerRef } from "../components/jobs/JobSubmissionContainer";
import type { ToastController } from "../components/toast";
import { getApiErrorMessage } from "../lib/api-errors";
import {
  submitCrawlJob,
  submitResearchJob,
  submitScrapeJob,
} from "../lib/job-actions";
import type { RouteKind } from "./useAppShellRouting";
import type { JobPreset, JobType } from "../types/presets";

export interface UseJobSubmissionActionsOptions {
  navigate: (path: string) => void;
  routeKind: RouteKind;
  refreshJobs: () => Promise<void>;
  selectedJobId: string | null;
  toast: ToastController;
  getApiBaseUrl: () => string;
}

export interface UseJobSubmissionActionsReturn {
  activeTab: JobType;
  setActiveTab: (tab: JobType) => void;
  jobSubmissionRef: RefObject<JobSubmissionContainerRef | null>;
  handleSelectPreset: (preset: JobPreset) => void;
  handleSubmitForm: (formType: JobType) => void;
  handleSubmitScrape: (request: ScrapeRequest) => Promise<void>;
  handleSubmitCrawl: (request: CrawlRequest) => Promise<void>;
  handleSubmitResearch: (request: ResearchRequest) => Promise<void>;
  cancelJob: (jobId: string) => Promise<void>;
  deleteJob: (jobId: string) => Promise<void>;
}

function formatShortJobId(id: string): string {
  if (id.length <= 14) {
    return id;
  }

  return `${id.slice(0, 8)}…${id.slice(-4)}`;
}

export function useJobSubmissionActions(
  options: UseJobSubmissionActionsOptions,
): UseJobSubmissionActionsReturn {
  const {
    navigate,
    routeKind,
    refreshJobs,
    selectedJobId,
    toast,
    getApiBaseUrl,
  } = options;
  const [activeTab, setActiveTab] = useState<JobType>("scrape");
  const [pendingPreset, setPendingPreset] = useState<JobPreset | null>(null);
  const [pendingSubmission, setPendingSubmission] = useState<JobType | null>(
    null,
  );
  const jobSubmissionRef = useRef<JobSubmissionContainerRef | null>(null);

  const handleSubmitScrape = useCallback(
    async (request: ScrapeRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting scrape job",
        description:
          "Queueing your scrape request and refreshing the Jobs view.",
      });

      const result = await submitScrapeJob(postV1Scrape, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Scrape job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("scrape");
      toast.update(toastId, {
        tone: "success",
        title: "Scrape job queued",
        description: "The new run is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [getApiBaseUrl, navigate, refreshJobs, toast],
  );

  const handleSubmitCrawl = useCallback(
    async (request: CrawlRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting crawl job",
        description:
          "Queueing your crawl request and refreshing the Jobs view.",
      });

      const result = await submitCrawlJob(postV1Crawl, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Crawl job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("crawl");
      toast.update(toastId, {
        tone: "success",
        title: "Crawl job queued",
        description: "The crawl is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [getApiBaseUrl, navigate, refreshJobs, toast],
  );

  const handleSubmitResearch = useCallback(
    async (request: ResearchRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting research job",
        description:
          "Queueing your research request and refreshing the Jobs view.",
      });

      const result = await submitResearchJob(postV1Research, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Research job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("research");
      toast.update(toastId, {
        tone: "success",
        title: "Research job queued",
        description: "The research run is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [getApiBaseUrl, navigate, refreshJobs, toast],
  );

  const cancelJob = useCallback(
    async (jobId: string) => {
      const toastId = toast.show({
        tone: "loading",
        title: `Canceling job ${formatShortJobId(jobId)}`,
        description: "Requesting a graceful stop for the active run.",
      });

      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
        });
        if (apiError) {
          toast.update(toastId, {
            tone: "error",
            title: "Failed to cancel job",
            description: getApiErrorMessage(
              apiError,
              "Unable to stop the selected job.",
            ),
          });
          return;
        }
        await refreshJobs();
        toast.update(toastId, {
          tone: "success",
          title: "Job canceled",
          description: `Job ${formatShortJobId(jobId)} is no longer running.`,
        });
      } catch (error) {
        toast.update(toastId, {
          tone: "error",
          title: "Failed to cancel job",
          description: getApiErrorMessage(
            error,
            "Unable to stop the selected job.",
          ),
        });
      }
    },
    [getApiBaseUrl, refreshJobs, toast],
  );

  const deleteJob = useCallback(
    async (jobId: string) => {
      const confirmed = await toast.confirm({
        title: "Delete this job permanently?",
        description:
          "This removes the saved run and its local artifacts. This action cannot be undone.",
        confirmLabel: "Delete job",
        cancelLabel: "Keep job",
        tone: "error",
      });
      if (!confirmed) {
        return;
      }

      const toastId = toast.show({
        tone: "loading",
        title: `Deleting job ${formatShortJobId(jobId)}`,
        description: "Removing the saved run from local storage.",
      });

      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
          query: { force: true },
        });
        if (apiError) {
          toast.update(toastId, {
            tone: "error",
            title: "Failed to delete job",
            description: getApiErrorMessage(
              apiError,
              "Unable to delete the selected job.",
            ),
          });
          return;
        }
        await refreshJobs();
        if (selectedJobId === jobId) {
          navigate("/jobs");
        }
        toast.update(toastId, {
          tone: "success",
          title: "Job deleted",
          description: `Job ${formatShortJobId(jobId)} has been removed.`,
        });
      } catch (error) {
        toast.update(toastId, {
          tone: "error",
          title: "Failed to delete job",
          description: getApiErrorMessage(
            error,
            "Unable to delete the selected job.",
          ),
        });
      }
    },
    [getApiBaseUrl, navigate, refreshJobs, selectedJobId, toast],
  );

  const handleSelectPreset = useCallback(
    (preset: JobPreset) => {
      navigate("/jobs/new");
      setActiveTab(preset.jobType);
      setPendingPreset(preset);
    },
    [navigate],
  );

  useEffect(() => {
    if (!pendingPreset || routeKind !== "new-job") {
      return;
    }
    if (pendingPreset.jobType !== activeTab) {
      return;
    }

    jobSubmissionRef.current?.applyPreset(
      pendingPreset.config,
      pendingPreset.jobType,
    );
    setPendingPreset(null);
  }, [activeTab, pendingPreset, routeKind]);

  const handleSubmitForm = useCallback(
    async (formType: JobType) => {
      navigate("/jobs/new");
      setActiveTab(formType);
      setPendingSubmission(formType);
    },
    [navigate],
  );

  useEffect(() => {
    if (!pendingSubmission || routeKind !== "new-job") {
      return;
    }
    if (pendingSubmission !== activeTab) {
      return;
    }

    const submit = async () => {
      if (pendingSubmission === "scrape") {
        await jobSubmissionRef.current?.submitScrape();
      } else if (pendingSubmission === "crawl") {
        await jobSubmissionRef.current?.submitCrawl();
      } else {
        await jobSubmissionRef.current?.submitResearch();
      }
      setPendingSubmission(null);
    };

    void submit();
  }, [activeTab, pendingSubmission, routeKind]);

  return {
    activeTab,
    setActiveTab,
    jobSubmissionRef,
    handleSelectPreset,
    handleSubmitForm,
    handleSubmitScrape,
    handleSubmitCrawl,
    handleSubmitResearch,
    cancelJob,
    deleteJob,
  };
}
