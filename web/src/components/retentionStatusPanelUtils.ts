/**
 * Purpose: Hold the derived types and helper logic that power the retention status surface.
 * Responsibilities: Parse cleanup filters, derive retention capability guidance, and resolve health/status data into a single panel-facing capability model.
 * Scope: Retention helper logic only; rendered UI stays in `RetentionStatusPanelSections.tsx` and state orchestration stays in `RetentionStatusPanel.tsx`.
 * Usage: Import from the retention panel and its extracted section components.
 * Invariants/Assumptions: Derived guidance must preserve the existing operator copy and remain consistent with retention health/status payloads.
 */

import type {
  HealthResponse,
  RecommendedAction,
  RetentionStatusResponse,
} from "../api";

interface DerivedRetentionCapability {
  status: "disabled" | "warning" | "danger" | "ok";
  title: string;
  message: string;
  actions: RecommendedAction[];
}

export type RetentionCapabilityStatus =
  | DerivedRetentionCapability["status"]
  | "degraded"
  | "error"
  | "setup_required";

export interface ResolvedRetentionCapability {
  status: RetentionCapabilityStatus;
  title: string;
  message: string;
  actions: RecommendedAction[];
}

export interface CleanupActivity {
  origin: "banner" | "controls";
  dryRun: boolean;
  kind: "" | "scrape" | "crawl" | "research";
  olderThanDays: number | null;
  force: boolean;
}

export interface RetentionLoadErrorState {
  eyebrow: string;
  title: string;
  description: string;
}

export function parseOlderThanDays(value: string): number | null {
  const parsed = Number.parseInt(value, 10);
  if (Number.isNaN(parsed) || parsed < 1) {
    return null;
  }
  return parsed;
}

function deriveRetentionCapability(
  status: RetentionStatusResponse | null,
): DerivedRetentionCapability | null {
  if (!status) {
    return null;
  }

  if (!status.enabled) {
    return {
      status: "disabled",
      title: "Automatic retention stays off by default",
      message:
        "Spartan keeps completed jobs and crawl state until you choose an automatic cleanup policy or run a manual cleanup preview. When you are ready to reclaim space, start with a dry run so you can review what would change.",
      actions: [
        {
          label: "Enable retention in the environment",
          kind: "env",
          value: "RETENTION_ENABLED=true",
        },
        {
          label: "Preview cleanup from the CLI",
          kind: "command",
          value: "spartan retention cleanup --dry-run",
        },
      ],
    };
  }

  const storageRatio =
    status.maxStorageGB > 0
      ? status.storageUsedMB / 1024 / status.maxStorageGB
      : 0;
  const jobsRatio = status.maxJobs > 0 ? status.totalJobs / status.maxJobs : 0;

  if (storageRatio >= 0.9 || jobsRatio >= 0.9) {
    return {
      status: "danger",
      title: "Retention limits are close to being hit",
      message:
        "Storage or job-count pressure is high. Preview cleanup now, then run cleanup or raise limits intentionally if this growth is expected.",
      actions: [
        {
          label: "Preview cleanup from the CLI",
          kind: "command",
          value: "spartan retention cleanup --dry-run",
        },
      ],
    };
  }

  if (status.jobsEligible > 0) {
    return {
      status: "warning",
      title: "Cleanup opportunity detected",
      message: `${status.jobsEligible.toLocaleString()} job(s) already meet the current cleanup policy. Preview a cleanup run before pressure becomes urgent.`,
      actions: [
        {
          label: "Preview cleanup from the CLI",
          kind: "command",
          value: "spartan retention cleanup --dry-run",
        },
      ],
    };
  }

  return {
    status: "ok",
    title: "",
    message: "",
    actions: [],
  };
}

export function resolveRetentionCapability(
  health: HealthResponse | null,
  status: RetentionStatusResponse | null,
): ResolvedRetentionCapability {
  const derived = status?.guidance ?? deriveRetentionCapability(status);
  const component = health?.components?.retention;

  if (!component) {
    return {
      status: derived?.status ?? "ok",
      title: derived?.title ?? "",
      message: derived?.message ?? "",
      actions: derived?.actions ?? [],
    };
  }

  if (component.status === "disabled") {
    return {
      status: "disabled",
      title: derived?.title || "Automatic retention stays off by default",
      message:
        component.message ||
        derived?.message ||
        "Spartan keeps completed jobs and crawl state until you choose an automatic cleanup policy or run a manual cleanup preview. When you are ready to reclaim space, start with a dry run so you can review what would change.",
      actions:
        component.actions && component.actions.length > 0
          ? component.actions
          : (derived?.actions ?? []),
    };
  }

  if (
    component.status === "degraded" ||
    component.status === "error" ||
    component.status === "setup_required"
  ) {
    return {
      status: component.status,
      title:
        component.status === "setup_required"
          ? "Retention setup is required"
          : "Retention needs attention",
      message:
        component.message || derived?.message || "Retention needs attention.",
      actions:
        component.actions && component.actions.length > 0
          ? component.actions
          : (derived?.actions ?? []),
    };
  }

  return {
    status: derived?.status ?? "ok",
    title: derived?.title ?? "",
    message: derived?.message ?? "",
    actions:
      component.actions && component.actions.length > 0
        ? component.actions
        : (derived?.actions ?? []),
  };
}
