/**
 * Application Data Hook
 *
 * Custom React hook for managing all application-wide data fetching.
 * Handles loading and refreshing of jobs, profiles, schedules, templates,
 * crawl states, and manager status. Implements polling for real-time updates.
 *
 * @module useAppData
 */

import { useCallback, useEffect, useState } from "react";
import {
  getV1Jobs,
  getHealthz,
  listTemplates,
  listCrawlStates,
  getV1AuthProfiles,
  getV1Schedules,
  type CrawlState,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";

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
  jobs: import("../types").JobEntry[];
  profiles: Profile[];
  schedules: Schedule[];
  templates: string[];
  crawlStates: CrawlState[];
  managerStatus: ManagerStatus | null;
  jobsTotal: number;
  jobsPage: number;
  crawlStatesTotal: number;
  crawlStatesPage: number;
  error: string | null;
  loading: boolean;
}

export interface AppDataActions {
  refreshJobs: (page?: number) => Promise<void>;
  refreshProfiles: () => Promise<void>;
  refreshSchedules: () => Promise<void>;
  refreshTemplates: () => Promise<void>;
  refreshCrawlStates: (page?: number) => Promise<void>;
  setJobsPage: (page: number) => void;
  setCrawlStatesPage: (page: number) => void;
}

const JOBS_PER_PAGE = 100;
const CRAWL_STATES_PER_PAGE = 100;
const POLL_INTERVAL = 4000;

export function useAppData(): AppDataState & AppDataActions {
  const [jobs, setJobs] = useState<import("../types").JobEntry[]>([]);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [templates, setTemplates] = useState<string[]>([]);
  const [crawlStates, setCrawlStates] = useState<CrawlState[]>([]);
  const [managerStatus, setManagerStatus] = useState<ManagerStatus | null>(
    null,
  );
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [jobsPage, setJobsPage] = useState(1);
  const [jobsTotal, setJobsTotal] = useState(0);
  const [crawlStatesPage, setCrawlStatesPage] = useState(1);
  const [crawlStatesTotal, setCrawlStatesTotal] = useState(0);

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
          },
        });
        if (apiError) {
          setError(String(apiError));
          return;
        }
        setJobs(data?.jobs ?? []);
        const total = response.headers.get("X-Total-Count");
        if (total) {
          setJobsTotal(parseInt(total, 10));
        }
        setError(null);
      } catch (err) {
        setError(String(err));
      } finally {
        setLoading(false);
      }
    },
    [jobsPage],
  );

  const refreshManagerStatus = useCallback(async () => {
    try {
      const { data, error: apiError } = await getHealthz({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch manager status:", apiError);
        return;
      }
      const queueDetails = data?.components?.queue?.details;
      if (queueDetails && typeof queueDetails === "object") {
        const queued =
          typeof queueDetails.queued === "number" ? queueDetails.queued : 0;
        const active =
          typeof queueDetails.active === "number" ? queueDetails.active : 0;
        setManagerStatus({ queued, active });
      }
    } catch (err) {
      console.error("Failed to fetch manager status:", err);
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

  useEffect(() => {
    void refreshJobs();
    void refreshManagerStatus();
    void refreshProfiles();
    void refreshSchedules();
    void refreshTemplates();
    void refreshCrawlStates();
    const handle = window.setInterval(() => {
      void refreshJobs();
      void refreshManagerStatus();
    }, POLL_INTERVAL);
    return () => window.clearInterval(handle);
  }, [
    refreshJobs,
    refreshManagerStatus,
    refreshProfiles,
    refreshSchedules,
    refreshTemplates,
    refreshCrawlStates,
  ]);

  return {
    jobs,
    profiles,
    schedules,
    templates,
    crawlStates,
    managerStatus,
    jobsTotal,
    jobsPage,
    crawlStatesTotal,
    crawlStatesPage,
    error,
    loading,
    refreshJobs,
    refreshProfiles,
    refreshSchedules,
    refreshTemplates,
    refreshCrawlStates,
    setJobsPage,
    setCrawlStatesPage,
  };
}
