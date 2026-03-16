/**
 * Purpose: Derive scan-friendly jobs dashboard models and persist jobs-route view state.
 * Responsibilities: Transform raw job envelopes into dashboard summary/lane/card models, format operator-facing timing and recency labels, and save/restore jobs filter/page/scroll state.
 * Scope: Web UI presentation helpers for the `/jobs` monitoring dashboard and `/jobs` → `/jobs/:id` return flow.
 * Usage: Import from jobs dashboard components and the application shell when rendering or preserving jobs-route context.
 * Invariants/Assumptions: Job entries follow the generated API contract, failedJobs is a lightweight recent-failure subset, and localStorage persistence failures must never break navigation.
 */

import type { ManagerStatus } from "../hooks/useAppData";
import type { JobEntry } from "../types";

export type ConnectionState =
  | "connected"
  | "disconnected"
  | "reconnecting"
  | "polling";

export type JobStatusFilter =
  | ""
  | "queued"
  | "running"
  | "succeeded"
  | "failed"
  | "canceled";

export interface JobsViewState {
  statusFilter: JobStatusFilter;
  currentPage: number;
  scrollY: number;
}

export interface JobMonitorCardModel {
  id: string;
  shortId: string;
  rawId: string;
  kind: string;
  status: NonNullable<JobEntry["status"]>;
  dependencyStatus?: JobEntry["dependencyStatus"];
  chainId?: string;
  updatedAtLabel: string;
  progress?: {
    label: string;
    percent?: number;
    valueText: string;
    indeterminate?: boolean;
  };
  timeline: Array<{
    label: string;
    value: string;
  }>;
  failure?: {
    tone: "danger" | "warning";
    category: string;
    summary: string;
  };
  activityText: string;
  dependsOnLabel?: string;
  canViewResults: boolean;
  canCancel: boolean;
  canDelete: boolean;
}

export interface JobMonitoringDashboardModel {
  summary: {
    totalJobs: number;
    queued: number;
    running: number;
    recentFailures: number;
    connectionState: ConnectionState;
  };
  lanes: {
    attention: JobMonitorCardModel[];
    progress: JobMonitorCardModel[];
    completed: JobMonitorCardModel[];
  };
}

const TERMINAL_STATUSES = new Set<JobEntry["status"]>([
  "succeeded",
  "failed",
  "canceled",
]);
const VALID_FILTERS = new Set<JobStatusFilter>([
  "",
  "queued",
  "running",
  "succeeded",
  "failed",
  "canceled",
]);

export const JOBS_VIEW_STATE_KEY = "spartan.jobs.view-state";

function formatShortId(id: string): string {
  if (id.length <= 14) {
    return id;
  }

  return `${id.slice(0, 8)}…${id.slice(-4)}`;
}

function formatRelativeTime(timestamp: string, now = Date.now()): string {
  const value = Date.parse(timestamp);
  if (Number.isNaN(value)) {
    return "just now";
  }

  const diffMs = now - value;
  const absMs = Math.abs(diffMs);
  const minuteMs = 60_000;
  const hourMs = 60 * minuteMs;
  const dayMs = 24 * hourMs;
  const suffix = diffMs >= 0 ? "ago" : "from now";

  if (absMs < minuteMs) {
    return "just now";
  }

  if (absMs < hourMs) {
    const minutes = Math.round(absMs / minuteMs);
    return `${minutes}m ${suffix}`;
  }

  if (absMs < dayMs) {
    const hours = Math.round(absMs / hourMs);
    return `${hours}h ${suffix}`;
  }

  const days = Math.round(absMs / dayMs);
  return `${days}d ${suffix}`;
}

function formatDuration(ms?: number): string {
  if (ms == null || Number.isNaN(ms)) {
    return "—";
  }

  if (ms < 1_000) {
    return `${Math.max(0, Math.round(ms))}ms`;
  }

  const seconds = ms / 1_000;
  if (seconds < 10) {
    return `${seconds.toFixed(1)}s`;
  }
  if (seconds < 60) {
    return `${Math.round(seconds)}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.round(seconds % 60);
  return `${minutes}m ${remainingSeconds}s`;
}

function getDependsOnLabel(dependsOn?: string[]): string | undefined {
  if (!dependsOn || dependsOn.length === 0) {
    return undefined;
  }

  const [first, second, ...rest] = dependsOn;
  const shown = [first, second].filter(Boolean).map((id) => formatShortId(id));

  if (rest.length === 0) {
    return `Depends on ${shown.join(" and ")}`;
  }

  return `Depends on ${shown.join(", ")} +${rest.length} more`;
}

function clampPercent(value?: number): number | undefined {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return undefined;
  }

  return Math.max(0, Math.min(100, Math.round(value)));
}

function getActivityText(job: JobEntry): string {
  if (job.dependencyStatus === "pending") {
    return "Waiting for dependencies";
  }

  if (job.dependencyStatus === "failed") {
    return "Blocked by failed dependency";
  }

  switch (job.status) {
    case "queued":
      return "Queued for execution";
    case "running":
      return `Running ${job.kind}`;
    case "succeeded":
      return "Completed successfully";
    case "failed":
      return "Needs triage";
    case "canceled":
      return "Canceled";
  }
}

function getFailure(job: JobEntry): JobMonitorCardModel["failure"] | undefined {
  if (job.run?.failure) {
    return {
      tone:
        job.run.failure.retryable && !job.run.failure.terminal
          ? "warning"
          : "danger",
      category: job.run.failure.category,
      summary: job.run.failure.summary,
    };
  }

  if (job.error) {
    return {
      tone: "danger",
      category: "Error",
      summary: job.error,
    };
  }

  if (job.dependencyStatus === "failed") {
    return {
      tone: "danger",
      category: "Dependency",
      summary: "An upstream dependency failed and blocked this job.",
    };
  }

  return undefined;
}

function getProgress(
  job: JobEntry,
): JobMonitorCardModel["progress"] | undefined {
  const queue = job.run?.queue;
  const computedPercent =
    typeof queue?.percent === "number"
      ? queue.percent
      : typeof queue?.completed === "number" &&
          typeof queue?.total === "number" &&
          queue.total > 0
        ? (queue.completed / queue.total) * 100
        : undefined;
  const percent = clampPercent(computedPercent);

  if (queue) {
    const label =
      typeof queue.index === "number" && typeof queue.total === "number"
        ? `Batch ${queue.index} of ${queue.total}`
        : "Batch progress";
    const detailParts = [
      typeof queue.completed === "number"
        ? `${queue.completed} complete`
        : null,
      typeof queue.running === "number" && queue.running > 0
        ? `${queue.running} running`
        : null,
      typeof queue.queued === "number" ? `${queue.queued} queued` : null,
      typeof percent === "number" ? `${percent}%` : null,
    ].filter(Boolean);

    return {
      label,
      percent,
      valueText: detailParts.join(" · ") || "Batch progress available",
      indeterminate: job.status === "running" && typeof percent !== "number",
    };
  }

  if (job.status === "running") {
    return {
      label: "Running",
      valueText: "In progress",
      indeterminate: true,
    };
  }

  return undefined;
}

function isDependencyBlocked(job: JobEntry): boolean {
  return (
    job.dependencyStatus === "pending" || job.dependencyStatus === "failed"
  );
}

export function getConnectionStateLabel(state: ConnectionState): string {
  switch (state) {
    case "connected":
      return "Live";
    case "reconnecting":
      return "Reconnecting";
    case "disconnected":
      return "Disconnected";
    case "polling":
      return "Polling";
  }
}

export function toJobMonitorCardModel(
  job: JobEntry,
  now = Date.now(),
): JobMonitorCardModel {
  return {
    id: job.id,
    shortId: formatShortId(job.id),
    rawId: job.id,
    kind: job.kind,
    status: job.status,
    dependencyStatus: job.dependencyStatus,
    chainId: job.chainId,
    updatedAtLabel: `Updated ${formatRelativeTime(job.updatedAt, now)}`,
    progress: getProgress(job),
    timeline: [
      { label: "Wait", value: formatDuration(job.run?.waitMs) },
      { label: "Run", value: formatDuration(job.run?.runMs) },
      { label: "Total", value: formatDuration(job.run?.totalMs) },
    ],
    failure: getFailure(job),
    activityText: getActivityText(job),
    dependsOnLabel: getDependsOnLabel(job.dependsOn),
    canViewResults: job.status === "succeeded",
    canCancel: job.status === "queued" || job.status === "running",
    canDelete: TERMINAL_STATUSES.has(job.status),
  };
}

export function buildJobMonitoringDashboardModel({
  jobs,
  failedJobs,
  totalJobs,
  connectionState,
  managerStatus,
  now = Date.now(),
}: {
  jobs: JobEntry[];
  failedJobs: JobEntry[];
  totalJobs: number;
  connectionState: ConnectionState;
  managerStatus?: ManagerStatus | null;
  now?: number;
}): JobMonitoringDashboardModel {
  const attentionMap = new Map<string, JobEntry>();

  for (const job of failedJobs) {
    attentionMap.set(job.id, job);
  }

  for (const job of jobs) {
    if (job.status === "failed" || isDependencyBlocked(job)) {
      attentionMap.set(job.id, job);
    }
  }

  return {
    summary: {
      totalJobs: Math.max(totalJobs, jobs.length),
      queued:
        typeof managerStatus?.queued === "number"
          ? managerStatus.queued
          : jobs.filter((job) => job.status === "queued").length,
      running:
        typeof managerStatus?.active === "number"
          ? managerStatus.active
          : jobs.filter((job) => job.status === "running").length,
      recentFailures: failedJobs.length,
      connectionState,
    },
    lanes: {
      attention: [...attentionMap.values()].map((job) =>
        toJobMonitorCardModel(job, now),
      ),
      progress: jobs
        .filter(
          (job) =>
            (job.status === "queued" || job.status === "running") &&
            !attentionMap.has(job.id),
        )
        .map((job) => toJobMonitorCardModel(job, now)),
      completed: jobs
        .filter(
          (job) => job.status === "succeeded" || job.status === "canceled",
        )
        .map((job) => toJobMonitorCardModel(job, now)),
    },
  };
}

export function saveJobsViewState(state: JobsViewState): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(JOBS_VIEW_STATE_KEY, JSON.stringify(state));
  } catch {
    // Ignore storage failures; navigation should still proceed.
  }
}

export function loadJobsViewState(): JobsViewState | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const raw = window.localStorage.getItem(JOBS_VIEW_STATE_KEY);
    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw) as Partial<JobsViewState>;
    if (
      !VALID_FILTERS.has((parsed.statusFilter ?? "") as JobStatusFilter) ||
      typeof parsed.currentPage !== "number" ||
      typeof parsed.scrollY !== "number"
    ) {
      window.localStorage.removeItem(JOBS_VIEW_STATE_KEY);
      return null;
    }

    return {
      statusFilter: parsed.statusFilter as JobStatusFilter,
      currentPage: Math.max(1, Math.floor(parsed.currentPage)),
      scrollY: Math.max(0, parsed.scrollY),
    };
  } catch {
    window.localStorage.removeItem(JOBS_VIEW_STATE_KEY);
    return null;
  }
}

export function clearJobsViewState(): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.removeItem(JOBS_VIEW_STATE_KEY);
  } catch {
    // Ignore storage failures.
  }
}
