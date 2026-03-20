/**
 * Purpose: Manage authoritative application-wide operator data for the web UI.
 * Responsibilities: Fetch recent runs, failed runs, manager health, metrics, profiles, schedules, templates, and crawl states; keep job data fresh via WebSocket and polling fallbacks; expose pagination and run-filter controls.
 * Scope: Client-side data orchestration only; presentation stays in React components and transport contracts come from the generated API client.
 * Usage: Call `useAppData()` from the application shell and pass the returned state/actions into route-level components.
 * Invariants/Assumptions: Jobs are loaded from the API as recent-run envelopes, failedJobs is a lightweight failure-focused subset, polling only activates when the WebSocket connection is unavailable, and setup-mode health short-circuits normal data loading.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  getV1Jobs,
  getV1JobsById,
  getV1JobsFailures,
  getHealthz,
  getMetrics,
  listTemplates,
  listCrawlStates,
  getV1AuthProfiles,
  getV1Schedules,
  type CrawlState,
  type HealthResponse,
  type MetricsResponse,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import { useWebSocket, type WSMessage } from "./useWebSocket";

type JobEntry = import("../types").JobEntry;
export type JobStatusFilter =
  | ""
  | "queued"
  | "running"
  | "succeeded"
  | "failed"
  | "canceled";

export interface ManagerStatus {
  queued: number;
  active: number;
}

export interface Profile {
  name: string;
  parents: string[];
}

export interface Schedule {
  id: string;
  kind: string;
  intervalSeconds: number;
  nextRun: string;
}

export interface AppDataState {
  jobs: JobEntry[];
  failedJobs: JobEntry[];
  jobStatusFilter: JobStatusFilter;
  profiles: Profile[];
  schedules: Schedule[];
  templates: string[];
  crawlStates: CrawlState[];
  managerStatus: ManagerStatus | null;
  metrics: MetricsResponse | null;
  jobsTotal: number;
  jobsPage: number;
  crawlStatesTotal: number;
  crawlStatesPage: number;
  error: string | null;
  loading: boolean;
  connectionState: "connected" | "disconnected" | "reconnecting" | "polling";
  health: HealthResponse | null;
  setupRequired: boolean;
  detailJob: JobEntry | null;
  detailJobLoading: boolean;
  detailJobError: string | null;
}

export interface AppDataActions {
  refreshJobs: (page?: number) => Promise<void>;
  refreshJobFailures: () => Promise<void>;
  refreshProfiles: () => Promise<void>;
  refreshSchedules: () => Promise<void>;
  refreshTemplates: () => Promise<void>;
  refreshCrawlStates: (page?: number) => Promise<void>;
  refreshHealth: () => Promise<HealthResponse | null>;
  refreshJobDetail: (jobId: string) => Promise<JobEntry | null>;
  clearJobDetail: () => void;
  setJobsPage: (page: number) => void;
  setCrawlStatesPage: (page: number) => void;
  setJobStatusFilter: (status: JobStatusFilter) => void;
}

const JOBS_PER_PAGE = 100;
const FAILED_JOBS_PER_PAGE = 10;
const CRAWL_STATES_PER_PAGE = 100;
const POLL_INTERVAL = 4000;

function getWebSocketUrl(): string {
  const baseUrl = getApiBaseUrl();
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";

  if (!baseUrl) {
    return `${protocol}//${window.location.host}/v1/ws`;
  }

  if (baseUrl.startsWith("http://")) {
    return `${baseUrl.replace("http://", "ws://")}/v1/ws`;
  }
  if (baseUrl.startsWith("https://")) {
    return `${baseUrl.replace("https://", "wss://")}/v1/ws`;
  }

  return `${protocol}//${baseUrl}/v1/ws`;
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
        const {
          data,
          response,
          error: apiError,
        } = await getV1Jobs({
          baseUrl: getApiBaseUrl(),
          query: {
            limit: JOBS_PER_PAGE,
            offset: (page - 1) * JOBS_PER_PAGE,
            ...(jobStatusFilter ? { status: jobStatusFilter } : {}),
          },
        });
        if (apiError) {
          setError(getApiErrorMessage(apiError, "Failed to fetch jobs."));
          return;
        }
        setJobs(data?.jobs ?? []);
        if (typeof data?.total === "number") {
          setJobsTotal(data.total);
        } else {
          const total = response.headers.get("X-Total-Count");
          if (total) {
            setJobsTotal(parseInt(total, 10));
          }
        }
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
      const { data, error: apiError } = await getV1JobsById({
        baseUrl: getApiBaseUrl(),
        path: { id: jobId },
      });
      if (apiError) {
        const message = getApiErrorMessage(apiError, "Failed to load job.");
        setDetailJobError(message);
        setDetailJob(null);
        return null;
      }

      const nextJob = data?.job ?? null;
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
      const { data, error: apiError } = await getV1JobsFailures({
        baseUrl: getApiBaseUrl(),
        query: {
          limit: FAILED_JOBS_PER_PAGE,
          offset: 0,
        },
      });
      if (apiError) {
        console.error("Failed to fetch job failures:", apiError);
        return;
      }
      setFailedJobs(data?.jobs ?? []);
    } catch (err) {
      console.error("Failed to fetch job failures:", err);
    }
  }, []);

  const refreshMetrics = useCallback(async () => {
    try {
      const { data, error: apiError } = await getMetrics({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch metrics:", apiError);
        return;
      }
      setMetrics(data ?? null);
    } catch (err) {
      console.error("Failed to fetch metrics:", err);
    }
  }, []);

  const refreshProfiles = useCallback(async () => {
    try {
      const { data, error: apiError } = await getV1AuthProfiles({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch profiles:", apiError);
        return;
      }
      const profileList = (data?.profiles ?? [])
        .filter((p) => p.name !== undefined)
        .map((p) => ({
          name: p.name as string,
          parents: p.parents || [],
        }));
      setProfiles(profileList);
    } catch (err) {
      console.error("Failed to fetch profiles:", err);
    }
  }, []);

  const refreshSchedules = useCallback(async () => {
    try {
      const { data, error: apiError } = await getV1Schedules({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch schedules:", apiError);
        return;
      }
      setSchedules(data?.schedules || []);
    } catch (err) {
      console.error("Failed to fetch schedules:", err);
    }
  }, []);

  const refreshTemplates = useCallback(async () => {
    try {
      const { data, error: apiError } = await listTemplates({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch templates:", apiError);
        return;
      }
      setTemplates(data?.templates || []);
    } catch (err) {
      console.error("Failed to fetch templates:", err);
    }
  }, []);

  const refreshCrawlStates = useCallback(
    async (page = crawlStatesPage) => {
      try {
        const {
          data,
          response,
          error: apiError,
        } = await listCrawlStates({
          baseUrl: getApiBaseUrl(),
          query: {
            limit: CRAWL_STATES_PER_PAGE,
            offset: (page - 1) * CRAWL_STATES_PER_PAGE,
          },
        });
        if (apiError) {
          console.error("Failed to fetch crawl states:", apiError);
          return;
        }
        setCrawlStates(data?.crawlStates || []);
        const total = response.headers.get("X-Total-Count");
        if (total) {
          setCrawlStatesTotal(parseInt(total, 10));
        }
      } catch (err) {
        console.error("Failed to fetch crawl states:", err);
      }
    },
    [crawlStatesPage],
  );

  const refreshHealth =
    useCallback(async (): Promise<HealthResponse | null> => {
      try {
        const { data, error: apiError } = await getHealthz({
          baseUrl: getApiBaseUrl(),
        });

        if (apiError) {
          setError(
            getApiErrorMessage(apiError, "Failed to fetch system status."),
          );
          setHealthLoaded(true);
          return null;
        }

        const nextHealth = data ?? null;
        setHealth(nextHealth);

        const queueDetails = nextHealth?.components?.queue?.details;
        if (queueDetails && typeof queueDetails === "object") {
          const queued =
            typeof queueDetails.queued === "number" ? queueDetails.queued : 0;
          const active =
            typeof queueDetails.active === "number" ? queueDetails.active : 0;
          setManagerStatus({ queued, active });
        } else {
          setManagerStatus(null);
        }

        setError(null);
        setHealthLoaded(true);
        return nextHealth;
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
    (msg: WSMessage) => {
      switch (msg.type) {
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
          const payload = msg.payload as {
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
          const payload = msg.payload as MetricsResponse;
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
