/**
 * Purpose: Define the stable type contract for web application data orchestration.
 * Responsibilities: Centralize the public state and action interfaces exposed by `useAppData` along with supporting shared types used by route containers.
 * Scope: Type definitions only; network loading, WebSocket handling, and stateful hooks stay in the app-data hook modules.
 * Usage: Import from `useAppData` via re-exports or directly from app-data internals when splitting hook implementation files.
 * Invariants/Assumptions: Job status filters remain aligned with API status values, manager status stays a simple queued/active snapshot, and app-data actions remain async where network work is involved.
 */

import type { CrawlState, HealthResponse, MetricsResponse } from "../../api";

type JobEntry = import("../../types").JobEntry;

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
