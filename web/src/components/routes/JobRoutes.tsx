/**
 * Purpose: Provide route-local containers for the job creation, monitoring, and template workflows.
 * Responsibilities: Render the jobs dashboard, job detail drill-down, new-job authoring surface, and template workspace while composing shared route help and promotion flows.
 * Scope: Job and template route presentation only; top-level route parsing and global shell state stay in `App.tsx`.
 * Usage: Re-export through `AppRoutes.tsx` and render from the application shell after the active route has been parsed.
 * Invariants/Assumptions: Jobs and template routes keep their existing onboarding hooks, promotion flows only run for supported destinations, and route containers compose existing feature surfaces instead of re-implementing them.
 */

import { useCallback, useEffect, useMemo } from "react";

import {
  buildExportSchedulePromotionSeed,
  buildTemplatePromotionSeed,
  buildWatchPromotionSeed,
} from "../../lib/promotion";
import type {
  PromotionSeed,
  TemplatePromotionSeed,
} from "../../types/promotion";
import { ActionEmptyState } from "../ActionEmptyState";
import { getAutomationPath } from "../automation/automationSections";
import { JobSubmissionContainer } from "../jobs/JobSubmissionContainer";
import { JobMonitoringDashboard } from "../jobs/JobMonitoringDashboard";
import { ResultsContainer } from "../results/ResultsContainer";
import { RouteHelpPanel } from "../RouteHelpPanel";
import {
  RouteHeader,
  RouteSignals,
  type RouteSignal,
} from "../shell/ShellPrimitives";
import { TemplateManager } from "../templates/TemplateManager";
import type {
  JobDetailRouteProps,
  JobsRouteProps,
  NewJobRouteProps,
  PromotionDestination,
  TemplatesRouteProps,
} from "./routeTypes";

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
