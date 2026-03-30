/**
 * Purpose: Provide the route-based application shell for the local-first operator workbench.
 * Responsibilities: Own global navigation chrome, coordinate shared data hooks, and delegate major route workflows to route-local containers.
 * Scope: Application shell and cross-route coordination only.
 * Usage: Rendered once from `main.tsx` as the root React application.
 * Invariants/Assumptions: Supported routes are `/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, `/automation/:section`, `/settings`, and `/settings/:section`, and route-local containers own route framing once selected.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  deleteV1JobsById,
  postV1Crawl,
  postV1Research,
  postV1Scrape,
  type CrawlRequest,
  type ResearchRequest,
  type ScrapeRequest,
} from "./api";
import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { OnboardingFlow } from "./components/OnboardingFlow";
import { OnboardingNudge } from "./components/OnboardingNudge";
import { AIAssistantProvider, useAIAssistant } from "./components/ai-assistant";
import { DEFAULT_AUTOMATION_SECTION } from "./components/automation/automationSections";
import type { JobSubmissionContainerRef } from "./components/jobs/JobSubmissionContainer";
import {
  AutomationRoute,
  JobDetailRoute,
  JobsRoute,
  NewJobRoute,
  SettingsRoute,
  SetupRequiredRoute,
  TemplatesRoute,
} from "./components/routes/AppRoutes";
import { AppTopBar } from "./components/shell/ShellPrimitives";
import {
  DEFAULT_SETTINGS_SECTION,
  getSettingsPath,
} from "./components/settings/settingsSections";
import { ShortcutHint } from "./components/ShortcutHint";
import { SystemStatusPanel } from "./components/SystemStatusPanel";
import { ThemeToggle } from "./components/ThemeToggle";
import { useToast } from "./components/toast";
import { TutorialTooltip } from "./components/TutorialTooltip";
import { useAppData } from "./hooks/useAppData";
import { useFormState } from "./hooks/useFormState";
import { useKeyboard } from "./hooks/useKeyboard";
import { useOnboarding } from "./hooks/useOnboarding";
import { usePresets } from "./hooks/usePresets";
import { useResultsState } from "./hooks/useResultsState";
import { useTheme } from "./hooks/useTheme";
import { type RouteKind, useAppShellRouting } from "./hooks/useAppShellRouting";
import { getApiBaseUrl } from "./lib/api-config";
import { getApiErrorMessage } from "./lib/api-errors";
import {
  submitCrawlJob,
  submitResearchJob,
  submitScrapeJob,
} from "./lib/job-actions";
import { saveJobsViewState } from "./lib/job-monitoring";
import type { OnboardingRouteKey, RouteHelpAction } from "./lib/onboarding";
import type { JobPreset, JobType } from "./types/presets";

interface NavItem {
  kind: Exclude<RouteKind, "job-detail">;
  label: string;
  path: string;
  description: string;
}

const NAV_ITEMS = [
  {
    kind: "jobs",
    label: "Jobs",
    path: "/jobs",
    description: "Recent jobs, live queue state, and result drill-down.",
  },
  {
    kind: "new-job",
    label: "New Job",
    path: "/jobs/new",
    description: "Submit scrape, crawl, or research work with saved presets.",
  },
  {
    kind: "templates",
    label: "Templates",
    path: "/templates",
    description:
      "Manage extraction templates, with optional AI-assisted generation when that capability is enabled.",
  },
  {
    kind: "automation",
    label: "Automation",
    path: "/automation/batches",
    description:
      "Batches, chains, watches, export schedules, and webhook delivery history.",
  },
  {
    kind: "settings",
    label: "Settings",
    path: getSettingsPath(DEFAULT_SETTINGS_SECTION),
    description:
      "Saved auth, reusable runtime tools, and optional maintenance controls.",
  },
] as const satisfies readonly NavItem[];

function formatShortJobId(id: string): string {
  if (id.length <= 14) {
    return id;
  }

  return `${id.slice(0, 8)}…${id.slice(-4)}`;
}

function ErrorBanner({ message }: { message: string | null }) {
  if (!message) {
    return null;
  }

  return (
    <section className="panel">
      <div className="error">{message}</div>
    </section>
  );
}

export function App() {
  return (
    <AIAssistantProvider>
      <AppShell />
    </AIAssistantProvider>
  );
}

function AppShell() {
  const aiAssistant = useAIAssistant();
  const toast = useToast();
  const appData = useAppData();
  const formState = useFormState();
  const resultsState = useResultsState();
  const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme();
  const { presets, savePreset } = usePresets();
  const { route, navigate, routePromotionSeed, clearPromotionSeed } =
    useAppShellRouting();
  const [activeTab, setActiveTab] = useState<JobType>("scrape");
  const [pendingPreset, setPendingPreset] = useState<JobPreset | null>(null);
  const [pendingSubmission, setPendingSubmission] = useState<JobType | null>(
    null,
  );
  const jobSubmissionRef = useRef<JobSubmissionContainerRef>(null);

  const {
    isCommandPaletteOpen,
    isHelpOpen,
    openCommandPalette,
    closeCommandPalette,
    openHelp,
    closeHelp,
    shortcuts,
    isMac,
  } = useKeyboard();

  const {
    jobs,
    failedJobs,
    jobStatusFilter,
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
    connectionState,
    health,
    setupRequired,
    detailJob,
    detailJobLoading,
    detailJobError,
    refreshHealth,
    refreshJobs,
    refreshTemplates,
    refreshJobDetail,
    clearJobDetail,
    setJobsPage,
    setCrawlStatesPage,
    setJobStatusFilter,
  } = appData;

  const {
    shouldShowFirstRunHint,
    isTourActive,
    currentStep,
    startOnboarding,
    skipOnboarding,
    resetOnboarding,
    goToStep,
    finishOnboarding,
    dismissFirstRunHint,
  } = useOnboarding({ hasStartedWork: jobsTotal > 0 });

  const { selectedJobId } = resultsState;
  const routeKey = route.kind as OnboardingRouteKey;
  const showGlobalFirstRunPrompt =
    shouldShowFirstRunHint && route.kind === "jobs";

  const persistJobsViewState = useCallback(() => {
    if (typeof window === "undefined") {
      return;
    }

    saveJobsViewState({
      statusFilter: jobStatusFilter,
      currentPage: jobsPage,
      scrollY: window.scrollY,
    });
  }, [jobStatusFilter, jobsPage]);

  const handleNavigate = useCallback(
    (view: "jobs" | "results" | "forms") => {
      if (view === "forms") {
        navigate("/jobs/new");
        return;
      }

      if (view === "results" && selectedJobId) {
        if (route.kind === "jobs") {
          persistJobsViewState();
        }
        navigate(`/jobs/${selectedJobId}`);
        return;
      }

      navigate("/jobs");
    },
    [navigate, persistJobsViewState, route.kind, selectedJobId],
  );

  const handlePaletteNavigate = useCallback(
    (path: string) => {
      if (route.kind === "jobs" && path.startsWith("/jobs/")) {
        persistJobsViewState();
      }
      navigate(path);
    },
    [navigate, persistJobsViewState, route.kind],
  );

  useEffect(() => {
    const handleKeyboardNavigate = (event: CustomEvent) => {
      const { destination } = event.detail;
      if (destination === "navigateJobs") handleNavigate("jobs");
      if (destination === "navigateResults") handleNavigate("results");
      if (destination === "navigateForms") handleNavigate("forms");
    };

    window.addEventListener(
      "keyboard-navigate",
      handleKeyboardNavigate as EventListener,
    );
    return () => {
      window.removeEventListener(
        "keyboard-navigate",
        handleKeyboardNavigate as EventListener,
      );
    };
  }, [handleNavigate]);

  const handleSubmitScrape = useCallback(
    async (request: ScrapeRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting scrape job",
        description:
          "Queueing your scrape request and refreshing the Jobs view.",
      });

      const result = await submitScrapeJob(postV1Scrape, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Scrape job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("scrape");
      toast.update(toastId, {
        tone: "success",
        title: "Scrape job queued",
        description: "The new run is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [navigate, refreshJobs, toast],
  );

  const handleSubmitCrawl = useCallback(
    async (request: CrawlRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting crawl job",
        description:
          "Queueing your crawl request and refreshing the Jobs view.",
      });

      const result = await submitCrawlJob(postV1Crawl, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Crawl job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("crawl");
      toast.update(toastId, {
        tone: "success",
        title: "Crawl job queued",
        description: "The crawl is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [navigate, refreshJobs, toast],
  );

  const handleSubmitResearch = useCallback(
    async (request: ResearchRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting research job",
        description:
          "Queueing your research request and refreshing the Jobs view.",
      });

      const result = await submitResearchJob(postV1Research, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Research job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("research");
      toast.update(toastId, {
        tone: "success",
        title: "Research job queued",
        description: "The research run is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [navigate, refreshJobs, toast],
  );

  const cancelJob = useCallback(
    async (jobId: string) => {
      const toastId = toast.show({
        tone: "loading",
        title: `Canceling job ${formatShortJobId(jobId)}`,
        description: "Requesting a graceful stop for the active run.",
      });

      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
        });
        if (apiError) {
          toast.update(toastId, {
            tone: "error",
            title: "Failed to cancel job",
            description: getApiErrorMessage(
              apiError,
              "Unable to stop the selected job.",
            ),
          });
          return;
        }
        await refreshJobs();
        toast.update(toastId, {
          tone: "success",
          title: "Job canceled",
          description: `Job ${formatShortJobId(jobId)} is no longer running.`,
        });
      } catch (error) {
        toast.update(toastId, {
          tone: "error",
          title: "Failed to cancel job",
          description: getApiErrorMessage(
            error,
            "Unable to stop the selected job.",
          ),
        });
      }
    },
    [refreshJobs, toast],
  );

  const deleteJob = useCallback(
    async (jobId: string) => {
      const confirmed = await toast.confirm({
        title: "Delete this job permanently?",
        description:
          "This removes the saved run and its local artifacts. This action cannot be undone.",
        confirmLabel: "Delete job",
        cancelLabel: "Keep job",
        tone: "error",
      });
      if (!confirmed) {
        return;
      }

      const toastId = toast.show({
        tone: "loading",
        title: `Deleting job ${formatShortJobId(jobId)}`,
        description: "Removing the saved run from local storage.",
      });

      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
          query: { force: true },
        });
        if (apiError) {
          toast.update(toastId, {
            tone: "error",
            title: "Failed to delete job",
            description: getApiErrorMessage(
              apiError,
              "Unable to delete the selected job.",
            ),
          });
          return;
        }
        await refreshJobs();
        if (selectedJobId === jobId) {
          navigate("/jobs");
        }
        toast.update(toastId, {
          tone: "success",
          title: "Job deleted",
          description: `Job ${formatShortJobId(jobId)} has been removed.`,
        });
      } catch (error) {
        toast.update(toastId, {
          tone: "error",
          title: "Failed to delete job",
          description: getApiErrorMessage(
            error,
            "Unable to delete the selected job.",
          ),
        });
      }
    },
    [navigate, refreshJobs, selectedJobId, toast],
  );

  const handleViewResults = useCallback(
    (jobId: string, _format: string, _page: number) => {
      if (route.kind === "jobs") {
        persistJobsViewState();
      }

      navigate(`/jobs/${jobId}`);
    },
    [navigate, persistJobsViewState, route.kind],
  );

  const activeJob = jobs.find((job) => job.status === "running");

  const handleSelectPreset = useCallback(
    (preset: JobPreset) => {
      navigate("/jobs/new");
      setActiveTab(preset.jobType);
      setPendingPreset(preset);
    },
    [navigate],
  );

  useEffect(() => {
    if (!pendingPreset || route.kind !== "new-job") {
      return;
    }
    if (pendingPreset.jobType !== activeTab) {
      return;
    }

    jobSubmissionRef.current?.applyPreset(
      pendingPreset.config,
      pendingPreset.jobType,
    );
    setPendingPreset(null);
  }, [activeTab, pendingPreset, route.kind]);

  const getCurrentConfig = useCallback(() => {
    return jobSubmissionRef.current?.getCurrentConfig() ?? {};
  }, []);

  const getCurrentUrl = useCallback(() => {
    switch (activeTab) {
      case "scrape":
        return jobSubmissionRef.current?.getScrapeUrl() ?? "";
      case "crawl":
        return jobSubmissionRef.current?.getCrawlUrl() ?? "";
      default:
        return "";
    }
  }, [activeTab]);

  const openJobAssistant = useCallback(() => {
    const currentConfig = getCurrentConfig();

    navigate("/jobs/new");
    aiAssistant.open({
      surface: "job-submission",
      jobType: activeTab,
      url:
        activeTab === "scrape"
          ? jobSubmissionRef.current?.getScrapeUrl()
          : activeTab === "crawl"
            ? jobSubmissionRef.current?.getCrawlUrl()
            : undefined,
      query:
        activeTab === "research"
          ? (currentConfig.query as string | undefined)
          : undefined,
      templateName: formState.extractTemplate || undefined,
      formSnapshot: currentConfig as Record<string, unknown>,
    });
  }, [
    activeTab,
    aiAssistant,
    formState.extractTemplate,
    getCurrentConfig,
    navigate,
  ]);

  const openTemplateAssistant = useCallback(() => {
    navigate("/templates");
    aiAssistant.open({
      surface: "templates",
      templateName: undefined,
      templateSnapshot: undefined,
      selectedUrl: getCurrentUrl() || undefined,
    });
  }, [aiAssistant, getCurrentUrl, navigate]);

  const handleSubmitForm = useCallback(
    async (formType: "scrape" | "crawl" | "research") => {
      navigate("/jobs/new");
      setActiveTab(formType);
      setPendingSubmission(formType);
    },
    [navigate],
  );

  useEffect(() => {
    if (!pendingSubmission || route.kind !== "new-job") {
      return;
    }
    if (pendingSubmission !== activeTab) {
      return;
    }

    const submit = async () => {
      if (pendingSubmission === "scrape") {
        await jobSubmissionRef.current?.submitScrape();
      } else if (pendingSubmission === "crawl") {
        await jobSubmissionRef.current?.submitCrawl();
      } else {
        await jobSubmissionRef.current?.submitResearch();
      }
      setPendingSubmission(null);
    };

    void submit();
  }, [activeTab, pendingSubmission, route.kind]);

  const handleTourRouteChange = useCallback(
    (targetRoute: OnboardingRouteKey) => {
      switch (targetRoute) {
        case "jobs":
          navigate("/jobs");
          return;
        case "new-job":
          navigate("/jobs/new");
          return;
        case "job-detail": {
          const targetJobId =
            selectedJobId ??
            jobs.find((job) => job.status === "succeeded")?.id ??
            jobs[0]?.id;
          if (targetJobId) {
            navigate(`/jobs/${targetJobId}`);
            return;
          }
          navigate("/jobs");
          return;
        }
        case "templates":
          navigate("/templates");
          return;
        case "automation":
          navigate("/automation/batches");
          return;
        case "settings":
          navigate(getSettingsPath(DEFAULT_SETTINGS_SECTION));
          return;
      }
    },
    [jobs, navigate, selectedJobId],
  );

  const handleRouteHelpAction = useCallback(
    (actionId: RouteHelpAction["id"]) => {
      switch (actionId) {
        case "create-job":
          navigate("/jobs/new");
          return;
        case "open-jobs":
          navigate("/jobs");
          return;
      }
    },
    [navigate],
  );

  const routeHelpProps = useMemo(
    () => ({
      shortcuts,
      isMac,
      onOpenCommandPalette: openCommandPalette,
      onOpenShortcuts: openHelp,
      onRestartTour: resetOnboarding,
      onAction: handleRouteHelpAction,
    }),
    [
      handleRouteHelpAction,
      isMac,
      openCommandPalette,
      openHelp,
      resetOnboarding,
      shortcuts,
    ],
  );

  const shellUtilities = (
    <>
      <button
        type="button"
        className="secondary app-toolbar-shortcut"
        data-tour="command-palette"
        onClick={openCommandPalette}
      >
        Command Palette
        <ShortcutHint shortcut={shortcuts.commandPalette} isMac={isMac} />
      </button>
      <button
        type="button"
        className="secondary app-toolbar-shortcut"
        data-tour="keyboard-help"
        onClick={openHelp}
      >
        Shortcuts
        <ShortcutHint shortcut={shortcuts.help} isMac={isMac} />
      </button>
      <ThemeToggle
        theme={theme}
        resolvedTheme={resolvedTheme}
        onThemeChange={setTheme}
        onToggle={toggleTheme}
      />
    </>
  );

  const activeRouteForNav = route.kind === "job-detail" ? "jobs" : route.kind;

  return (
    <div className={`app app--${route.kind}`}>
      <AppTopBar
        activeRoute={activeRouteForNav}
        navItems={NAV_ITEMS}
        onNavigate={navigate}
        globalAction={
          !setupRequired && route.kind !== "new-job" ? (
            <button type="button" onClick={() => navigate("/jobs/new")}>
              Create Job
            </button>
          ) : null
        }
        utilities={shellUtilities}
      />

      {!setupRequired ? (
        <OnboardingNudge
          isVisible={showGlobalFirstRunPrompt}
          isMac={isMac}
          onStartTour={startOnboarding}
          onOpenHelp={openHelp}
          onDismiss={dismissFirstRunHint}
          onCreateJob={() => navigate("/jobs/new")}
          health={health}
          hasTemplates={templates.length > 0}
        />
      ) : null}

      <TutorialTooltip
        target='[data-tour="command-palette"]'
        title="Jump anywhere fast"
        content="Use the command palette to navigate routes, submit work, select presets, and restart onboarding."
        position="bottom"
        showBeacon={showGlobalFirstRunPrompt}
        showDelay={500}
      />

      <TutorialTooltip
        target='[data-tour="keyboard-help"]'
        title="Shortcut help is visible now"
        content="Open this anytime to see global shortcuts and a route-specific section for what matters on the current screen."
        position="bottom"
        showBeacon={showGlobalFirstRunPrompt}
        showDelay={500}
      />

      {isCommandPaletteOpen ? (
        <CommandPalette
          isOpen={isCommandPaletteOpen}
          onClose={closeCommandPalette}
          jobs={jobs}
          onNavigateToPath={handlePaletteNavigate}
          onSubmitForm={handleSubmitForm}
          onCancelJob={cancelJob}
          activeJobId={activeJob?.id}
          isMac={isMac}
          presets={presets}
          onSelectPreset={handleSelectPreset}
          onRestartTour={resetOnboarding}
        />
      ) : null}

      <KeyboardShortcutsHelp
        isOpen={isHelpOpen}
        onClose={closeHelp}
        shortcuts={shortcuts}
        isMac={isMac}
        routeKind={routeKey}
      />

      <OnboardingFlow
        isRunning={isTourActive}
        currentStep={currentStep}
        currentRoute={routeKey}
        onComplete={finishOnboarding}
        onSkip={skipOnboarding}
        onStepChange={goToStep}
        onRouteChange={handleTourRouteChange}
      />

      <ErrorBanner message={error} />

      <SystemStatusPanel
        health={health}
        onNavigate={navigate}
        onRefresh={refreshHealth}
      />

      {setupRequired ? <SetupRequiredRoute health={health} /> : null}

      {!setupRequired && route.kind === "jobs" ? (
        <JobsRoute
          jobs={jobs}
          failedJobs={failedJobs}
          error={error}
          loading={loading}
          statusFilter={jobStatusFilter}
          currentPage={jobsPage}
          totalJobs={jobsTotal}
          connectionState={connectionState}
          managerStatus={managerStatus}
          routeHelp={routeHelpProps}
          onStatusFilterChange={setJobStatusFilter}
          onViewResults={handleViewResults}
          onCancel={cancelJob}
          onDelete={deleteJob}
          onRefresh={refreshJobs}
          onCreateJob={() => navigate("/jobs/new")}
          onPageChange={setJobsPage}
        />
      ) : null}

      {!setupRequired && route.kind === "job-detail" && route.jobId ? (
        <JobDetailRoute
          jobId={route.jobId}
          jobs={jobs}
          routeDetailJob={detailJob}
          detailJobLoading={detailJobLoading}
          detailJobError={detailJobError}
          resultsState={resultsState}
          connectionState={connectionState}
          aiStatus={health?.components?.ai ?? null}
          routeHelp={routeHelpProps}
          refreshJobDetail={refreshJobDetail}
          clearJobDetail={clearJobDetail}
          navigate={navigate}
        />
      ) : null}

      {!setupRequired && route.kind === "new-job" ? (
        <NewJobRoute
          activeTab={activeTab}
          formState={formState}
          loading={loading}
          profiles={profiles}
          presets={presets}
          jobsTotal={jobsTotal}
          jobStatusFilter={jobStatusFilter}
          aiStatus={health?.components?.ai ?? null}
          routeHelp={routeHelpProps}
          jobSubmissionRef={jobSubmissionRef}
          savePreset={savePreset}
          setActiveTab={setActiveTab}
          onSubmitScrape={handleSubmitScrape}
          onSubmitCrawl={handleSubmitCrawl}
          onSubmitResearch={handleSubmitResearch}
          onSelectPreset={handleSelectPreset}
          onOpenAssistant={openJobAssistant}
          onOpenTemplateAssistant={openTemplateAssistant}
        />
      ) : null}

      {!setupRequired && route.kind === "templates" ? (
        <TemplatesRoute
          templateNames={templates}
          promotionSeed={routePromotionSeed}
          aiStatus={health?.components?.ai ?? null}
          routeHelp={routeHelpProps}
          onClearPromotionSeed={clearPromotionSeed}
          onOpenSourceJob={(jobId) => navigate(`/jobs/${jobId}`)}
          onTemplatesChanged={() => {
            void refreshTemplates();
          }}
        />
      ) : null}

      {!setupRequired && route.kind === "automation" ? (
        <AutomationRoute
          section={route.automationSection ?? DEFAULT_AUTOMATION_SECTION}
          promotionSeed={routePromotionSeed}
          formState={formState}
          profiles={profiles}
          loading={loading}
          aiStatus={health?.components?.ai ?? null}
          routeHelp={routeHelpProps}
          onClearPromotionSeed={clearPromotionSeed}
          navigate={navigate}
          onRefreshJobs={refreshJobs}
        />
      ) : null}

      {!setupRequired && route.kind === "settings" ? (
        <SettingsRoute
          section={route.settingsSection ?? DEFAULT_SETTINGS_SECTION}
          path={route.path}
          health={health}
          profiles={profiles}
          schedules={schedules}
          crawlStates={crawlStates}
          crawlStatesPage={crawlStatesPage}
          crawlStatesTotal={crawlStatesTotal}
          jobsTotal={jobsTotal}
          routeHelp={routeHelpProps}
          onNavigate={navigate}
          onRefreshHealth={refreshHealth}
          onCrawlStatesPageChange={setCrawlStatesPage}
        />
      ) : null}
    </div>
  );
}
