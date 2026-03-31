/**
 * Purpose: Centralize shared route contracts for the web application shell.
 * Responsibilities: Define route prop types, shared navigation helpers, and small DOM utilities used across the extracted route container modules.
 * Scope: Route-level types and helpers only; rendering logic stays in the route component files.
 * Usage: Import from route container modules to keep `AppRoutes.tsx` as a thin barrel.
 * Invariants/Assumptions: Top-level routes remain stable, route help stays attached to each major route, and route-local containers compose existing feature surfaces instead of re-implementing them.
 */

import type { RefObject } from "react";

import type { ComponentStatus, CrawlState, HealthResponse } from "../../api";
import type {
  AppDataActions,
  JobStatusFilter,
  ManagerStatus,
  Profile,
  Schedule,
} from "../../hooks/useAppData";
import type { FormController } from "../../hooks/useFormState";
import type { ShortcutConfig } from "../../hooks/useKeyboard";
import type { ResultsActions, ResultsState } from "../../hooks/useResultsState";
import type { RouteHelpAction } from "../../lib/onboarding";
import type { JobEntry } from "../../types";
import type { JobPreset, JobType, PresetConfig } from "../../types/presets";
import type {
  PromotionSeed,
  PromotionDestination,
} from "../../types/promotion";
import type { AutomationSection } from "../automation/automationSections";
import type { JobSubmissionContainerRef } from "../jobs/JobSubmissionContainer";
import type { SettingsSectionId } from "../settings/settingsSections";

export interface SharedRouteHelpProps {
  shortcuts: ShortcutConfig;
  isMac: boolean;
  onOpenCommandPalette: () => void;
  onOpenShortcuts: () => void;
  onRestartTour: () => void;
  onAction: (actionId: RouteHelpAction["id"]) => void;
}

export type AppNavigate = (
  path: string,
  state?: { promotionSeed?: PromotionSeed | null } | null,
) => void;

export interface JobsRouteProps {
  jobs: JobEntry[];
  failedJobs: JobEntry[];
  error: string | null;
  loading: boolean;
  statusFilter: JobStatusFilter;
  currentPage: number;
  totalJobs: number;
  connectionState: "connected" | "disconnected" | "reconnecting" | "polling";
  managerStatus: ManagerStatus | null;
  routeHelp: SharedRouteHelpProps;
  onStatusFilterChange: (status: JobStatusFilter) => void;
  onViewResults: (jobId: string, format: string, page: number) => void;
  onCancel: (jobId: string) => void;
  onDelete: (jobId: string) => void;
  onRefresh: () => void;
  onCreateJob: () => void;
  onPageChange: (page: number) => void;
}

export interface JobDetailRouteProps {
  jobId: string;
  jobs: JobEntry[];
  routeDetailJob: JobEntry | null;
  detailJobLoading: boolean;
  detailJobError: string | null;
  resultsState: ResultsState & ResultsActions;
  connectionState: "connected" | "disconnected" | "reconnecting" | "polling";
  aiStatus?: ComponentStatus | null;
  routeHelp: SharedRouteHelpProps;
  refreshJobDetail: AppDataActions["refreshJobDetail"];
  clearJobDetail: AppDataActions["clearJobDetail"];
  navigate: AppNavigate;
}

export interface NewJobRouteProps {
  activeTab: JobType;
  formState: FormController;
  loading: boolean;
  profiles: Profile[];
  presets: JobPreset[];
  jobsTotal: number;
  jobStatusFilter: JobStatusFilter;
  aiStatus?: ComponentStatus | null;
  routeHelp: SharedRouteHelpProps;
  jobSubmissionRef: RefObject<JobSubmissionContainerRef | null>;
  savePreset: (
    name: string,
    description: string,
    jobType: JobType,
    config: PresetConfig,
  ) => void;
  setActiveTab: (tab: JobType) => void;
  onSubmitScrape: (request: import("../../api").ScrapeRequest) => Promise<void>;
  onSubmitCrawl: (request: import("../../api").CrawlRequest) => Promise<void>;
  onSubmitResearch: (
    request: import("../../api").ResearchRequest,
  ) => Promise<void>;
  onSelectPreset: (preset: JobPreset) => void;
  onOpenAssistant: () => void;
  onOpenTemplateAssistant: () => void;
}

export interface TemplatesRouteProps {
  templateNames: string[];
  promotionSeed?: PromotionSeed | null;
  aiStatus?: ComponentStatus | null;
  routeHelp: SharedRouteHelpProps;
  onClearPromotionSeed?: () => void;
  onOpenSourceJob?: (jobId: string) => void;
  onTemplatesChanged: () => void;
}

export interface AutomationRouteProps {
  section: AutomationSection;
  promotionSeed?: PromotionSeed | null;
  formState: FormController;
  profiles: Profile[];
  loading: boolean;
  aiStatus?: ComponentStatus | null;
  routeHelp: SharedRouteHelpProps;
  onClearPromotionSeed?: () => void;
  navigate: AppNavigate;
  onRefreshJobs: () => Promise<void>;
}

export interface SettingsRouteProps {
  section: SettingsSectionId;
  path: string;
  health: HealthResponse | null;
  profiles: Profile[];
  schedules: Schedule[];
  crawlStates: CrawlState[];
  crawlStatesPage: number;
  crawlStatesTotal: number;
  jobsTotal: number;
  routeHelp: SharedRouteHelpProps;
  onNavigate: AppNavigate;
  onRefreshHealth: () => Promise<HealthResponse | null>;
  onCrawlStatesPageChange: (page: number) => void;
}

export function scrollWindowToTop() {
  if (typeof document === "undefined") {
    return;
  }

  document.documentElement.scrollTop = 0;
  document.body.scrollTop = 0;
}

export type { PromotionDestination };
