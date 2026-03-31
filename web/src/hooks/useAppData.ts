/**
 * Purpose: Manage authoritative application-wide operator data for the web UI.
 * Responsibilities: Fetch recent runs, failed runs, manager health, metrics, profiles, schedules, templates, and crawl states; keep job data fresh via WebSocket and polling fallbacks; expose pagination and run-filter controls.
 * Scope: Client-side data orchestration only; presentation stays in React components and transport contracts come from the generated API client.
 * Usage: Call `useAppData()` from the application shell and pass the returned state/actions into route-level components.
 * Invariants/Assumptions: Jobs are loaded from the API as recent-run envelopes, failedJobs is a lightweight failure-focused subset, polling only activates when the WebSocket connection is unavailable, and setup-mode health short-circuits normal data loading.
 */

import { useCallback, useEffect, useMemo, useState } from "react";

import type { HealthResponse, MetricsResponse } from "../api";
import {
  getWebSocketUrl,
  loadCrawlStates,
  loadHealth,
  loadJobDetail,
  loadJobFailures,
  loadJobs,
  loadMetrics,
  loadProfiles,
  loadSchedules,
  loadTemplates,
  POLL_INTERVAL,
} from "./app-data/api";
import type {
  AppDataActions,
  AppDataState,
  JobStatusFilter,
  ManagerStatus,
  Profile,
  Schedule,
} from "./app-data/types";
import { getApiErrorMessage } from "../lib/api-errors";
import { reportRuntimeError } from "../lib/runtime-errors";
import { useWebSocket, type WSMessage } from "./useWebSocket";

export type {
  AppDataActions,
  AppDataState,
  JobStatusFilter,
  ManagerStatus,
  Profile,
  Schedule,
} from "./app-data/types";

type JobEntry = import("../types").JobEntry;
type CrawlState = import("../api").CrawlState;

function reportAppDataBackgroundError(scope: string, error: unknown) {
  reportRuntimeError(`Failed to fetch ${scope}`, error, {
    fallback: `Failed to fetch ${scope}.`,
  });
}

export function useAppData(): AppDataState & AppDataActions {
  const [jobs, setJobs] = useState<JobEntry[]>([]);
  const [failedJobs, setFailedJobs] = useState<JobEntry[]>([]);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [templates, setTemplates] = useState<string[]>([]);
  const [crawlStates, setCrawlStates] = useState<CrawlState[]>([]);
  const [managerStatus, setManagerStatus] = useState<ManagerStatus | null>(
    null,
  );
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [detailJob, setDetailJob] = useState<JobEntry | null>(null);
  const [detailJobLoading, setDetailJobLoading] = useState(false);
  const [detailJobError, setDetailJobError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [jobsPage, setJobsPage] = useState(1);
  const [jobsTotal, setJobsTotal] = useState(0);
  const [jobStatusFilter, setJobStatusFilterState] =
    useState<JobStatusFilter>("");
  const [crawlStatesPage, setCrawlStatesPage] = useState(1);
  const [crawlStatesTotal, setCrawlStatesTotal] = useState(0);
  const [usePolling, setUsePolling] = useState(false);
  const [healthLoaded, setHealthLoaded] = useState(false);

  const refreshJobs = useCallback(
    async (page = jobsPage) => {
      setLoading(true);
      try {
        const nextJobs = await loadJobs({ page, jobStatusFilter });
        setJobs(nextJobs.jobs);
        setJobsTotal(nextJobs.total);
        setError(null);
      } catch (err) {
        setError(getApiErrorMessage(err, "Failed to fetch jobs."));
      } finally {
        setLoading(false);
      }
    },
    [jobStatusFilter, jobsPage],
  );

  const refreshJobDetail = useCallback(async (jobId: string) => {
    if (!jobId) {
      setDetailJob(null);
      setDetailJobError(null);
      return null;
    }

    setDetailJobLoading(true);
    try {
      const nextJob = await loadJobDetail(jobId);
      setDetailJob(nextJob);
      setDetailJobError(null);
      return nextJob;
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to load job.");
      setDetailJobError(message);
      setDetailJob(null);
      return null;
    } finally {
      setDetailJobLoading(false);
    }
  }, []);

  const clearJobDetail = useCallback(() => {
    setDetailJob(null);
    setDetailJobError(null);
    setDetailJobLoading(false);
  }, []);

  const refreshJobFailures = useCallback(async () => {
    try {
      setFailedJobs(await loadJobFailures());
    } catch (err) {
      reportAppDataBackgroundError("job failures", err);
    }
  }, []);

  const refreshMetrics = useCallback(async () => {
    try {
      setMetrics(await loadMetrics());
    } catch (err) {
      reportAppDataBackgroundError("metrics", err);
    }
  }, []);

  const refreshProfiles = useCallback(async () => {
    try {
      setProfiles(await loadProfiles());
    } catch (err) {
      reportAppDataBackgroundError("profiles", err);
    }
  }, []);

  const refreshSchedules = useCallback(async () => {
    try {
      setSchedules(await loadSchedules());
    } catch (err) {
      reportAppDataBackgroundError("schedules", err);
    }
  }, []);

  const refreshTemplates = useCallback(async () => {
    try {
      setTemplates(await loadTemplates());
    } catch (err) {
      reportAppDataBackgroundError("templates", err);
    }
  }, []);

  const refreshCrawlStates = useCallback(
    async (page = crawlStatesPage) => {
      try {
        const nextState = await loadCrawlStates(page);
        setCrawlStates(nextState.crawlStates);
        setCrawlStatesTotal(nextState.total);
      } catch (err) {
        reportAppDataBackgroundError("crawl states", err);
      }
    },
    [crawlStatesPage],
  );

  const refreshHealth =
    useCallback(async (): Promise<HealthResponse | null> => {
      try {
        const nextHealth = await loadHealth();
        setHealth(nextHealth.health);
        setManagerStatus(nextHealth.managerStatus);
        setError(null);
        setHealthLoaded(true);
        return nextHealth.health;
      } catch (err) {
        setError(getApiErrorMessage(err, "Failed to fetch system status."));
        setHealthLoaded(true);
        return null;
      }
    }, []);

  const setJobStatusFilter = useCallback((status: JobStatusFilter) => {
    setJobsPage(1);
    setJobStatusFilterState(status);
  }, []);

  const handleWebSocketMessage = useCallback(
    (message: WSMessage) => {
      switch (message.type) {
        case "job_created":
        case "job_started":
        case "job_status_changed":
        case "job_completed":
          void Promise.all([
            refreshJobs(),
            refreshJobFailures(),
            refreshHealth(),
            detailJob?.id
              ? refreshJobDetail(detailJob.id)
              : Promise.resolve(null),
          ]);
          break;
        case "manager_status": {
          const payload = message.payload as {
            queuedJobs: number;
            activeJobs: number;
          };
          setManagerStatus({
            queued: payload.queuedJobs,
            active: payload.activeJobs,
          });
          break;
        }
        case "metrics": {
          const payload = message.payload as MetricsResponse;
          setMetrics(payload);
          break;
        }
        default:
          break;
      }
    },
    [
      detailJob?.id,
      refreshHealth,
      refreshJobDetail,
      refreshJobFailures,
      refreshJobs,
    ],
  );

  const setupRequired = health?.setup?.required ?? false;
  const wsUrl = useMemo(() => getWebSocketUrl(), []);
  const { state: wsState } = useWebSocket({
    url: wsUrl,
    enabled: healthLoaded && !setupRequired,
    onMessage: handleWebSocketMessage,
    onConnect: () => {
      setUsePolling(false);
    },
    onDisconnect: () => {
      if (!setupRequired) {
        setUsePolling(true);
      }
    },
  });

  const connectionState = useMemo(() => {
    if (setupRequired) return "disconnected";
    if (wsState === "connected") return "connected";
    if (wsState === "reconnecting") return "reconnecting";
    if (usePolling) return "polling";
    return "disconnected";
  }, [setupRequired, usePolling, wsState]);

  useEffect(() => {
    void (async () => {
      setLoading(true);
      const currentHealth = await refreshHealth();
      if (currentHealth?.setup?.required) {
        setJobs([]);
        setFailedJobs([]);
        setProfiles([]);
        setSchedules([]);
        setTemplates([]);
        setCrawlStates([]);
        setMetrics(null);
        clearJobDetail();
        setLoading(false);
        return;
      }

      await Promise.all([
        refreshJobs(),
        refreshJobFailures(),
        refreshMetrics(),
        refreshProfiles(),
        refreshSchedules(),
        refreshTemplates(),
        refreshCrawlStates(),
      ]);
      setLoading(false);
    })();
  }, [
    clearJobDetail,
    refreshCrawlStates,
    refreshHealth,
    refreshJobFailures,
    refreshJobs,
    refreshMetrics,
    refreshProfiles,
    refreshSchedules,
    refreshTemplates,
  ]);

  useEffect(() => {
    if (!usePolling || setupRequired) {
      return;
    }

    const handle = window.setInterval(() => {
      void refreshHealth();
      void refreshJobs();
      void refreshJobFailures();
      void refreshMetrics();
      if (detailJob?.id) {
        void refreshJobDetail(detailJob.id);
      }
    }, POLL_INTERVAL);

    return () => window.clearInterval(handle);
  }, [
    detailJob?.id,
    refreshHealth,
    refreshJobDetail,
    refreshJobFailures,
    refreshJobs,
    refreshMetrics,
    setupRequired,
    usePolling,
  ]);

  return {
    jobs,
    failedJobs,
    jobStatusFilter,
    profiles,
    schedules,
    templates,
    crawlStates,
    managerStatus,
    metrics,
    jobsTotal,
    jobsPage,
    crawlStatesTotal,
    crawlStatesPage,
    error,
    loading,
    connectionState,
    health,
    setupRequired,
    detailJob,
    detailJobLoading,
    detailJobError,
    refreshJobs,
    refreshJobFailures,
    refreshProfiles,
    refreshSchedules,
    refreshTemplates,
    refreshCrawlStates,
    refreshHealth,
    refreshJobDetail,
    clearJobDetail,
    setJobsPage,
    setCrawlStatesPage,
    setJobStatusFilter,
  };
}
