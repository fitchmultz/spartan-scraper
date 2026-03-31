/**
 * Purpose: Encapsulate the network-facing helpers used by `useAppData`.
 * Responsibilities: Execute the generated API client calls for app shell data, normalize totals and derived manager status, and expose shared polling/WebSocket constants.
 * Scope: Client-side transport and normalization only; React state management stays in `useAppData`.
 * Usage: Import these helpers from `useAppData` to keep the hook focused on orchestration instead of raw request plumbing.
 * Invariants/Assumptions: Generated client envelopes remain the source of truth, header fallbacks are only used when the API omits totals in-body, and WebSocket URLs derive from the configured API base URL.
 */

import {
  getHealthz,
  getMetrics,
  getV1AuthProfiles,
  getV1Jobs,
  getV1JobsById,
  getV1JobsFailures,
  getV1Schedules,
  listCrawlStates,
  listTemplates,
  type HealthResponse,
  type MetricsResponse,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import type {
  JobStatusFilter,
  ManagerStatus,
  Profile,
  Schedule,
} from "./types";

type JobEntry = import("../../types").JobEntry;
type CrawlState = import("../../api").CrawlState;

export const JOBS_PER_PAGE = 100;
export const FAILED_JOBS_PER_PAGE = 10;
export const CRAWL_STATES_PER_PAGE = 100;
export const POLL_INTERVAL = 4000;

function parseOptionalTotal(
  response: Response | undefined,
  fallback?: number,
): number {
  if (typeof fallback === "number") {
    return fallback;
  }

  const totalHeader = response?.headers.get("X-Total-Count");
  return totalHeader ? Number.parseInt(totalHeader, 10) : 0;
}

function toManagerStatus(health: HealthResponse | null): ManagerStatus | null {
  const queueDetails = health?.components?.queue?.details;
  if (!queueDetails || typeof queueDetails !== "object") {
    return null;
  }

  const queued =
    typeof queueDetails.queued === "number" ? queueDetails.queued : 0;
  const active =
    typeof queueDetails.active === "number" ? queueDetails.active : 0;
  return { queued, active };
}

export function getWebSocketUrl(): string {
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

export async function loadJobs(args: {
  page: number;
  jobStatusFilter: JobStatusFilter;
}): Promise<{ jobs: JobEntry[]; total: number }> {
  const { data, response, error } = await getV1Jobs({
    baseUrl: getApiBaseUrl(),
    query: {
      limit: JOBS_PER_PAGE,
      offset: (args.page - 1) * JOBS_PER_PAGE,
      ...(args.jobStatusFilter ? { status: args.jobStatusFilter } : {}),
    },
  });

  if (error) {
    throw new Error(getApiErrorMessage(error, "Failed to fetch jobs."));
  }

  return {
    jobs: data?.jobs ?? [],
    total: parseOptionalTotal(response, data?.total),
  };
}

export async function loadJobDetail(jobId: string): Promise<JobEntry | null> {
  const { data, error } = await getV1JobsById({
    baseUrl: getApiBaseUrl(),
    path: { id: jobId },
  });

  if (error) {
    throw new Error(getApiErrorMessage(error, "Failed to load job."));
  }

  return data?.job ?? null;
}

export async function loadJobFailures(): Promise<JobEntry[]> {
  const { data, error } = await getV1JobsFailures({
    baseUrl: getApiBaseUrl(),
    query: {
      limit: FAILED_JOBS_PER_PAGE,
      offset: 0,
    },
  });

  if (error) {
    throw new Error(getApiErrorMessage(error, "Failed to fetch job failures."));
  }

  return data?.jobs ?? [];
}

export async function loadMetrics(): Promise<MetricsResponse | null> {
  const { data, error } = await getMetrics({
    baseUrl: getApiBaseUrl(),
  });

  if (error) {
    throw new Error(getApiErrorMessage(error, "Failed to fetch metrics."));
  }

  return data ?? null;
}

export async function loadProfiles(): Promise<Profile[]> {
  const { data, error } = await getV1AuthProfiles({
    baseUrl: getApiBaseUrl(),
  });

  if (error) {
    throw new Error(getApiErrorMessage(error, "Failed to fetch profiles."));
  }

  return (data?.profiles ?? [])
    .filter((profile) => profile.name !== undefined)
    .map((profile) => ({
      name: profile.name as string,
      parents: profile.parents || [],
    }));
}

export async function loadSchedules(): Promise<Schedule[]> {
  const { data, error } = await getV1Schedules({
    baseUrl: getApiBaseUrl(),
  });

  if (error) {
    throw new Error(getApiErrorMessage(error, "Failed to fetch schedules."));
  }

  return data?.schedules || [];
}

export async function loadTemplates(): Promise<string[]> {
  const { data, error } = await listTemplates({
    baseUrl: getApiBaseUrl(),
  });

  if (error) {
    throw new Error(getApiErrorMessage(error, "Failed to fetch templates."));
  }

  return data?.templates || [];
}

export async function loadCrawlStates(page: number): Promise<{
  crawlStates: CrawlState[];
  total: number;
}> {
  const { data, response, error } = await listCrawlStates({
    baseUrl: getApiBaseUrl(),
    query: {
      limit: CRAWL_STATES_PER_PAGE,
      offset: (page - 1) * CRAWL_STATES_PER_PAGE,
    },
  });

  if (error) {
    throw new Error(getApiErrorMessage(error, "Failed to fetch crawl states."));
  }

  return {
    crawlStates: data?.crawlStates || [],
    total: parseOptionalTotal(response),
  };
}

export async function loadHealth(): Promise<{
  health: HealthResponse | null;
  managerStatus: ManagerStatus | null;
}> {
  const { data, error } = await getHealthz({
    baseUrl: getApiBaseUrl(),
  });

  if (error) {
    throw new Error(
      getApiErrorMessage(error, "Failed to fetch system status."),
    );
  }

  const health = data ?? null;
  return {
    health,
    managerStatus: toManagerStatus(health),
  };
}
