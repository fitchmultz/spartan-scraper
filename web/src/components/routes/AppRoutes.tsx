/**
 * Purpose: Provide route-local containers for the major Web operator workflows.
 * Responsibilities: Own route framing, route help wiring, route-specific derived state, and route-local action composition for jobs, results, templates, automation, and settings.
 * Scope: Web route containers only; top-level path parsing, global navigation chrome, and shared data hooks stay in `App.tsx`.
 * Usage: Render from `App.tsx` after the active route has been parsed.
 * Invariants/Assumptions: Top-level routes remain stable, route help stays attached to each major route, and route-local containers compose existing feature surfaces instead of re-implementing them.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
  type RefObject,
} from "react";

import type { ComponentStatus, CrawlState, HealthResponse } from "../../api";
import type {
  AppDataActions,
  JobStatusFilter,
  ManagerStatus,
  Profile,
  Schedule,
} from "../../hooks/useAppData";
import type { FormController } from "../../hooks/useFormState";
import type { ResultsActions, ResultsState } from "../../hooks/useResultsState";
import type { ShortcutConfig } from "../../hooks/useKeyboard";
import type { RouteHelpAction } from "../../lib/onboarding";
import {
  buildExportSchedulePromotionSeed,
  buildTemplatePromotionSeed,
  buildWatchPromotionSeed,
} from "../../lib/promotion";
import { shouldShowSettingsOverviewPanel } from "../../lib/settings-overview";
import type { JobEntry } from "../../types";
import type { JobPreset, JobType, PresetConfig } from "../../types/presets";
import type {
  ExportSchedulePromotionSeed,
  PromotionDestination,
  PromotionSeed,
  TemplatePromotionSeed,
  WatchPromotionSeed,
} from "../../types/promotion";
import { ActionEmptyState } from "../ActionEmptyState";
import { InfoSections } from "../InfoSections";
import { ProxyPoolStatusPanel } from "../ProxyPoolStatusPanel";
import { RetentionStatusPanel } from "../RetentionStatusPanel";
import { RouteHelpPanel } from "../RouteHelpPanel";
import { SettingsOverviewPanel } from "../SettingsOverviewPanel";
import { ExportScheduleContainer } from "../export-schedules/ExportScheduleContainer";
import {
  JobSubmissionContainer,
  type JobSubmissionContainerRef,
} from "../jobs/JobSubmissionContainer";
import { JobMonitoringDashboard } from "../jobs/JobMonitoringDashboard";
import { ResultsContainer } from "../results/ResultsContainer";
import { RenderProfileEditor } from "../render-profiles";
import {
  RouteHeader,
  RouteSignals,
  type RouteSignal,
} from "../shell/ShellPrimitives";
import {
  getSettingsPath,
  SETTINGS_SECTION_META,
  type SettingsSectionId,
} from "../settings/settingsSections";
import { SettingsSubnav } from "../settings/SettingsSubnav";
import { TemplateManager } from "../templates/TemplateManager";
import { AutomationLayout } from "../automation/AutomationLayout";
import {
  getAutomationPath,
  type AutomationSection,
} from "../automation/automationSections";
import { AutomationSubnav } from "../automation/AutomationSubnav";
import { BatchContainer } from "../batches/BatchContainer";
import { ChainContainer } from "../chains/ChainContainer";
import { PipelineJSEditor } from "../pipeline-js/PipelineJSEditor";
import { WatchContainer } from "../watches/WatchContainer";
import { WebhookDeliveryContainer } from "../webhooks/WebhookDeliveryContainer";

interface SharedRouteHelpProps {
  shortcuts: ShortcutConfig;
  isMac: boolean;
  onOpenCommandPalette: () => void;
  onOpenShortcuts: () => void;
  onRestartTour: () => void;
  onAction: (actionId: RouteHelpAction["id"]) => void;
}

type AppNavigate = (
  path: string,
  state?: { promotionSeed?: PromotionSeed | null } | null,
) => void;

interface JobsRouteProps {
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

interface JobDetailRouteProps {
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

interface NewJobRouteProps {
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

interface TemplatesRouteProps {
  templateNames: string[];
  promotionSeed?: PromotionSeed | null;
  aiStatus?: ComponentStatus | null;
  routeHelp: SharedRouteHelpProps;
  onClearPromotionSeed?: () => void;
  onOpenSourceJob?: (jobId: string) => void;
  onTemplatesChanged: () => void;
}

interface AutomationRouteProps {
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

interface SettingsRouteProps {
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

function scrollWindowToTop() {
  if (typeof document === "undefined") {
    return;
  }

  document.documentElement.scrollTop = 0;
  document.body.scrollTop = 0;
}

export function JobsRoute({
  jobs,
  failedJobs,
  error,
  loading,
  statusFilter,
  currentPage,
  totalJobs,
  connectionState,
  managerStatus,
  routeHelp,
  onStatusFilterChange,
  onViewResults,
  onCancel,
  onDelete,
  onRefresh,
  onCreateJob,
  onPageChange,
}: JobsRouteProps) {
  return (
    <div className="route-stack">
      <RouteHeader title="Jobs" />

      <section id="jobs" data-tour="jobs-dashboard">
        <JobMonitoringDashboard
          jobs={jobs}
          failedJobs={failedJobs}
          error={error}
          loading={loading}
          statusFilter={statusFilter}
          onStatusFilterChange={onStatusFilterChange}
          onViewResults={onViewResults}
          onCancel={onCancel}
          onDelete={onDelete}
          onRefresh={onRefresh}
          onCreateJob={onCreateJob}
          currentPage={currentPage}
          totalJobs={totalJobs}
          jobsPerPage={100}
          onPageChange={onPageChange}
          connectionState={connectionState}
          managerStatus={managerStatus}
        />
      </section>

      <RouteHelpPanel routeKey="jobs" {...routeHelp} />
    </div>
  );
}

export function JobDetailRoute({
  jobId,
  jobs,
  routeDetailJob,
  detailJobLoading,
  detailJobError,
  resultsState,
  connectionState,
  aiStatus = null,
  routeHelp,
  refreshJobDetail,
  clearJobDetail,
  navigate,
}: JobDetailRouteProps) {
  const { loadResults, resultFormat, totalResults } = resultsState;

  const detailJob = useMemo(
    () =>
      routeDetailJob && routeDetailJob.id === jobId
        ? routeDetailJob
        : (jobs.find((job) => job.id === jobId) ?? null),
    [jobId, jobs, routeDetailJob],
  );

  useEffect(() => {
    void loadResults(jobId);
    void refreshJobDetail(jobId);

    return () => {
      clearJobDetail();
    };
  }, [clearJobDetail, jobId, loadResults, refreshJobDetail]);

  const jobDetailSignals = useMemo<RouteSignal[]>(
    () => [
      {
        label: "Status",
        value: detailJob?.status ?? "unknown",
      },
      {
        label: "Results",
        value: totalResults,
      },
      {
        label: "Format",
        value: resultFormat.toUpperCase(),
      },
      {
        label: "Connection",
        value: connectionState,
      },
    ],
    [connectionState, detailJob?.status, resultFormat, totalResults],
  );

  const handlePromoteJob = useCallback(
    (
      destination: PromotionDestination,
      options?: {
        preferredExportFormat?: "json" | "jsonl" | "md" | "csv" | "xlsx";
      },
    ) => {
      if (!detailJob || detailJob.status !== "succeeded") {
        return;
      }

      let promotionSeed: PromotionSeed;
      let path = "/templates";

      if (destination === "template") {
        promotionSeed = buildTemplatePromotionSeed(detailJob);
        path = "/templates";
      } else if (destination === "watch") {
        const watchSeed = buildWatchPromotionSeed(detailJob);
        if (!watchSeed.eligible) {
          return;
        }
        promotionSeed = watchSeed;
        path = getAutomationPath("watches");
      } else {
        promotionSeed = buildExportSchedulePromotionSeed(
          detailJob,
          options?.preferredExportFormat,
        );
        path = getAutomationPath("exports");
      }

      navigate(path, { promotionSeed });
    },
    [detailJob, navigate],
  );

  return (
    <div className="route-stack">
      <RouteHeader
        title={`Job ${jobId.length <= 14 ? jobId : `${jobId.slice(0, 8)}…${jobId.slice(-4)}`}`}
        actions={
          <button
            type="button"
            className="secondary"
            onClick={() => navigate("/jobs")}
          >
            Back to Jobs
          </button>
        }
      />

      {detailJobLoading && !detailJob ? (
        <section className="panel">
          <div className="loading-placeholder">Loading job details…</div>
        </section>
      ) : null}

      {detailJobError && !detailJob ? (
        <ActionEmptyState
          eyebrow="Job detail"
          title="Unable to load this saved job"
          description={detailJobError}
          actions={[
            {
              label: "Back to jobs",
              onClick: () => navigate("/jobs"),
            },
          ]}
        />
      ) : null}

      {!detailJobError ? (
        <>
          <ResultsContainer
            resultsState={resultsState}
            jobs={jobs}
            currentJob={detailJob}
            aiStatus={aiStatus}
            onPromote={handlePromoteJob}
          />

          <RouteSignals ariaLabel="Result context" items={jobDetailSignals} />

          <RouteHelpPanel routeKey="job-detail" {...routeHelp} />
        </>
      ) : null}
    </div>
  );
}

export function NewJobRoute({
  activeTab,
  formState,
  loading,
  profiles,
  presets,
  jobsTotal,
  jobStatusFilter,
  aiStatus = null,
  routeHelp,
  jobSubmissionRef,
  savePreset,
  setActiveTab,
  onSubmitScrape,
  onSubmitCrawl,
  onSubmitResearch,
  onSelectPreset,
  onOpenAssistant,
  onOpenTemplateAssistant,
}: NewJobRouteProps) {
  return (
    <div className="route-stack">
      <RouteHeader title="Create Job" />

      <div className="route-stack" data-tour="job-wizard">
        <JobSubmissionContainer
          ref={jobSubmissionRef}
          activeTab={activeTab}
          setActiveTab={setActiveTab}
          formState={formState}
          aiStatus={aiStatus}
          onSubmitScrape={onSubmitScrape}
          onSubmitCrawl={onSubmitCrawl}
          onSubmitResearch={onSubmitResearch}
          loading={loading}
          profiles={profiles}
          presets={presets}
          savePreset={savePreset}
          onSelectPreset={onSelectPreset}
          onOpenAssistant={onOpenAssistant}
          onOpenTemplateAssistant={onOpenTemplateAssistant}
        />
      </div>

      {jobsTotal === 0 && jobStatusFilter === "" ? (
        <ActionEmptyState
          eyebrow="First run"
          title="Start with a single page scrape"
          description="Paste a URL into the form below, keep the defaults, and submit one successful run before moving on to templates or automation."
        />
      ) : null}

      <RouteHelpPanel routeKey="new-job" {...routeHelp} />
    </div>
  );
}

export function TemplatesRoute({
  templateNames,
  promotionSeed = null,
  aiStatus = null,
  routeHelp,
  onClearPromotionSeed,
  onOpenSourceJob,
  onTemplatesChanged,
}: TemplatesRouteProps) {
  const templatePromotionSeed =
    promotionSeed?.kind === "template"
      ? (promotionSeed as TemplatePromotionSeed)
      : null;

  return (
    <div className="route-stack">
      <RouteHeader title="Templates" />

      <div data-tour="templates-workspace">
        <TemplateManager
          templateNames={templateNames}
          aiStatus={aiStatus}
          promotionSeed={templatePromotionSeed}
          onClearPromotionSeed={onClearPromotionSeed}
          onOpenSourceJob={onOpenSourceJob}
          onTemplatesChanged={onTemplatesChanged}
        />
      </div>

      <RouteHelpPanel routeKey="templates" {...routeHelp} />
    </div>
  );
}

export function AutomationRoute({
  section,
  promotionSeed = null,
  formState,
  profiles,
  loading,
  aiStatus = null,
  routeHelp,
  onClearPromotionSeed,
  navigate,
  onRefreshJobs,
}: AutomationRouteProps) {
  const watchPromotionSeed =
    section === "watches" && promotionSeed?.kind === "watch"
      ? (promotionSeed as WatchPromotionSeed)
      : null;
  const exportPromotionSeed =
    section === "exports" && promotionSeed?.kind === "export-schedule"
      ? (promotionSeed as ExportSchedulePromotionSeed)
      : null;

  const renderSection = useCallback(
    (activeSection: AutomationSection): ReactNode => {
      switch (activeSection) {
        case "batches":
          return (
            <BatchContainer
              formState={formState}
              profiles={profiles}
              loading={loading}
            />
          );
        case "chains":
          return <ChainContainer onChainSubmit={onRefreshJobs} />;
        case "watches":
          return (
            <WatchContainer
              promotionSeed={watchPromotionSeed}
              onClearPromotionSeed={onClearPromotionSeed}
              onOpenSourceJob={(jobId) => navigate(`/jobs/${jobId}`)}
            />
          );
        case "exports":
          return (
            <ExportScheduleContainer
              aiStatus={aiStatus}
              promotionSeed={exportPromotionSeed}
              onClearPromotionSeed={onClearPromotionSeed}
              onOpenSourceJob={(jobId) => navigate(`/jobs/${jobId}`)}
            />
          );
        case "webhooks":
          return <WebhookDeliveryContainer />;
      }
    },
    [
      aiStatus,
      exportPromotionSeed,
      formState,
      loading,
      navigate,
      onClearPromotionSeed,
      onRefreshJobs,
      profiles,
      watchPromotionSeed,
    ],
  );

  return (
    <div className="route-stack">
      <RouteHeader
        title="Automation"
        subnav={
          <div data-tour="automation-subnav">
            <AutomationSubnav
              activeSection={section}
              onSectionChange={(nextSection) =>
                navigate(getAutomationPath(nextSection))
              }
            />
          </div>
        }
      />

      <section data-tour="automation-hub">
        <AutomationLayout
          activeSection={section}
          renderSection={renderSection}
        />
      </section>

      <RouteHelpPanel routeKey="automation" {...routeHelp} />
    </div>
  );
}

export function SettingsRoute({
  section,
  path,
  health,
  profiles,
  schedules,
  crawlStates,
  crawlStatesPage,
  crawlStatesTotal,
  jobsTotal,
  routeHelp,
  onNavigate,
  onRefreshHealth,
  onCrawlStatesPageChange,
}: SettingsRouteProps) {
  const [renderProfileCount, setRenderProfileCount] = useState<number | null>(
    null,
  );
  const [pipelineScriptCount, setPipelineScriptCount] = useState<number | null>(
    null,
  );

  const showSettingsOverview = useMemo(
    () =>
      shouldShowSettingsOverviewPanel({
        isSettingsRoute: true,
        setupRequired: false,
        jobsTotal,
        profilesCount: profiles.length,
        schedulesCount: schedules.length,
        crawlStatesTotal,
        renderProfileCount,
        pipelineScriptCount,
        proxyStatus: health?.components?.proxy_pool?.status,
        retentionStatus: health?.components?.retention?.status,
      }),
    [
      crawlStatesTotal,
      health?.components?.proxy_pool?.status,
      health?.components?.retention?.status,
      jobsTotal,
      pipelineScriptCount,
      profiles.length,
      renderProfileCount,
      schedules.length,
    ],
  );

  useEffect(() => {
    if (!path) {
      return;
    }

    scrollWindowToTop();
  }, [path]);

  const scrollToSettingsSection = useCallback(
    (nextSection: SettingsSectionId) => {
      if (nextSection === section) {
        scrollWindowToTop();
        return;
      }

      onNavigate(getSettingsPath(nextSection));
    },
    [onNavigate, section],
  );

  const renderSection = useCallback(
    (activeSection: SettingsSectionId): ReactNode => {
      switch (activeSection) {
        case "authoring":
          return (
            <div className="settings-route__section-stack">
              <section className="panel">
                <RenderProfileEditor
                  aiStatus={health?.components?.ai ?? null}
                  onInventoryChange={setRenderProfileCount}
                />
              </section>

              <section className="panel">
                <PipelineJSEditor
                  aiStatus={health?.components?.ai ?? null}
                  onInventoryChange={setPipelineScriptCount}
                />
              </section>
            </div>
          );
        case "inventory":
          return (
            <InfoSections
              profiles={profiles}
              schedules={schedules}
              crawlStates={crawlStates}
              crawlStatesPage={crawlStatesPage}
              crawlStatesTotal={crawlStatesTotal}
              crawlStatesPerPage={100}
              onCrawlStatesPageChange={onCrawlStatesPageChange}
              onCreateJob={() => onNavigate("/jobs/new")}
              onOpenAutomation={() => onNavigate("/automation/batches")}
              onOpenJobs={() => onNavigate("/jobs")}
            />
          );
        case "operations":
          return (
            <div className="settings-route__section-stack">
              <ProxyPoolStatusPanel
                health={health}
                onNavigate={onNavigate}
                onRefreshHealth={onRefreshHealth}
              />
              <RetentionStatusPanel
                health={health}
                onNavigate={onNavigate}
                onRefreshHealth={onRefreshHealth}
                onCreateJob={() => onNavigate("/jobs/new")}
                onOpenAutomation={() => onNavigate("/automation/batches")}
              />
            </div>
          );
      }
    },
    [
      crawlStates,
      crawlStatesPage,
      crawlStatesTotal,
      health,
      onCrawlStatesPageChange,
      onNavigate,
      onRefreshHealth,
      profiles,
      schedules,
    ],
  );

  return (
    <div className="route-stack">
      <RouteHeader
        title="Settings"
        subnav={
          <SettingsSubnav
            activeSection={section}
            onSectionChange={scrollToSettingsSection}
          />
        }
      />

      <div data-tour="settings-workspace" className="settings-route">
        <section
          id={SETTINGS_SECTION_META[section].elementId}
          className="settings-route__section"
          aria-labelledby={`settings-route-${section}-title`}
        >
          <div className="settings-route__section-header">
            <div className="settings-route__section-eyebrow">
              {SETTINGS_SECTION_META[section].label}
            </div>
            <h2 id={`settings-route-${section}-title`}>
              {SETTINGS_SECTION_META[section].title}
            </h2>
            <p>{SETTINGS_SECTION_META[section].description}</p>
          </div>

          {renderSection(section)}
        </section>

        {showSettingsOverview ? (
          <SettingsOverviewPanel
            onCreateJob={() => onNavigate("/jobs/new")}
            onOpenJobs={() => onNavigate("/jobs")}
          />
        ) : null}
      </div>

      <RouteHelpPanel routeKey="settings" {...routeHelp} />
    </div>
  );
}

export function SetupRequiredRoute({
  health,
}: {
  health: HealthResponse | null;
}) {
  return (
    <div className="route-stack">
      <RouteHeader
        title="Setup required"
        description="Spartan is running in guided recovery mode so the issue is visible in-product instead of only in terminal output."
      />

      <ActionEmptyState
        eyebrow="Guided recovery"
        title={health?.setup?.title ?? "Setup required"}
        description={
          health?.setup?.message ??
          "Resolve the setup issue, then restart the server."
        }
      />
    </div>
  );
}
