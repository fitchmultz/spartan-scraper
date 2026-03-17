/**
 * Spartan Scraper Web UI - Route-aware application shell.
 *
 * Purpose:
 * - Provide the route-based application shell for the local-first operator workbench.
 *
 * Responsibilities:
 * - Route between jobs, templates, automation, and settings views.
 * - Wire shared data hooks, job submission, and result loading.
 * - Keep route-specific surfaces above the fold instead of repeating a shared
 *   marketing-style stack on every route.
 *
 * Scope:
 * - Application shell and cross-page coordination only.
 *
 * Usage:
 * - Rendered once from `main.tsx` as the root React application.
 *
 * Invariants/Assumptions:
 * - Supported routes are `/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`,
 *   `/automation`, `/automation/:section`, and `/settings`.
 * - Results detail routes load job results directly from the canonical REST API.
 * - Deleted feature areas stay out of navigation and render tree entirely.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  deleteV1JobsById,
  postV1Crawl,
  postV1Research,
  postV1Scrape,
  type CrawlRequest,
  type ResearchRequest,
  type ScrapeRequest,
} from "./api";
import { InfoSections } from "./components/InfoSections";
import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { WelcomeModal } from "./components/WelcomeModal";
import { OnboardingFlow } from "./components/OnboardingFlow";
import { AutomationLayout } from "./components/automation/AutomationLayout";
import { AutomationSubnav } from "./components/automation/AutomationSubnav";
import {
  DEFAULT_AUTOMATION_SECTION,
  getAutomationPath,
  getAutomationSectionFromHash,
  getAutomationSectionFromPath,
  type AutomationSection,
} from "./components/automation/automationSections";
import { WatchContainer } from "./components/watches/WatchContainer";
import { ExportScheduleContainer } from "./components/export-schedules/ExportScheduleContainer";
import { WebhookDeliveryContainer } from "./components/webhooks/WebhookDeliveryContainer";
import { RetentionStatusPanel } from "./components/RetentionStatusPanel";
import { ProxyPoolStatusPanel } from "./components/ProxyPoolStatusPanel";
import { ChainContainer } from "./components/chains/ChainContainer";
import { BatchContainer } from "./components/batches/BatchContainer";
import { AIAssistantProvider, useAIAssistant } from "./components/ai-assistant";
import { TemplateManager } from "./components/templates/TemplateManager";
import { PresetContainer } from "./components/presets/PresetContainer";
import {
  JobSubmissionContainer,
  type JobSubmissionContainerRef,
} from "./components/jobs/JobSubmissionContainer";
import { JobMonitoringDashboard } from "./components/jobs/JobMonitoringDashboard";
import { ResultsContainer } from "./components/results/ResultsContainer";
import { RenderProfileEditor } from "./components/render-profiles";
import { PipelineJSEditor } from "./components/pipeline-js/PipelineJSEditor";
import {
  AppTopBar,
  RouteHeader,
  RouteSignals,
  type RouteSignal,
} from "./components/shell/ShellPrimitives";
import { ThemeToggle } from "./components/ThemeToggle";
import { useKeyboard } from "./hooks/useKeyboard";
import { useAppData } from "./hooks/useAppData";
import { useFormState } from "./hooks/useFormState";
import { useResultsState } from "./hooks/useResultsState";
import { useTheme } from "./hooks/useTheme";
import { usePresets } from "./hooks/usePresets";
import { useOnboarding } from "./hooks/useOnboarding";
import {
  submitCrawlJob,
  submitResearchJob,
  submitScrapeJob,
} from "./lib/job-actions";
import { getApiBaseUrl } from "./lib/api-config";
import { saveJobsViewState } from "./lib/job-monitoring";
import type { JobPreset, JobType } from "./types/presets";

type RouteKind =
  | "jobs"
  | "new-job"
  | "job-detail"
  | "templates"
  | "automation"
  | "settings";

interface AppRoute {
  kind: RouteKind;
  path: string;
  jobId?: string;
  automationSection?: AutomationSection;
}

interface NavItem {
  kind: Exclude<RouteKind, "job-detail">;
  label: string;
  path: string;
  description: string;
}

interface RouteMeta {
  title: string;
  description?: string;
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
      "Manage extraction templates and AI-assisted template generation.",
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
    path: "/settings",
    description:
      "Profiles, schedules, crawl state inventory, retention, and pipeline scripts.",
  },
] as const satisfies readonly NavItem[];

function normalizePath(pathname: string): string {
  if (!pathname || pathname === "/") {
    return "/jobs";
  }

  const trimmed = pathname.replace(/\/+$/, "");
  return trimmed === "" ? "/jobs" : trimmed;
}

function parseRoute(pathname: string): AppRoute {
  const path = normalizePath(pathname);
  if (path === "/jobs") {
    return { kind: "jobs", path };
  }
  if (path === "/jobs/new") {
    return { kind: "new-job", path };
  }
  if (path.startsWith("/jobs/")) {
    const jobId = decodeURIComponent(path.slice("/jobs/".length));
    return { kind: "job-detail", path, jobId };
  }
  if (path === "/templates") {
    return { kind: "templates", path };
  }
  if (path === "/automation" || path.startsWith("/automation/")) {
    return {
      kind: "automation",
      path,
      automationSection:
        getAutomationSectionFromPath(path) ?? DEFAULT_AUTOMATION_SECTION,
    };
  }
  if (path === "/settings") {
    return { kind: "settings", path };
  }
  return { kind: "jobs", path: "/jobs" };
}

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
  const appData = useAppData();
  const formState = useFormState();
  const resultsState = useResultsState();
  const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme();
  const { presets, savePreset } = usePresets();
  const [pathname, setPathname] = useState(() =>
    normalizePath(window.location.pathname),
  );
  const [activeTab, setActiveTab] = useState<JobType>("scrape");
  const [pendingPreset, setPendingPreset] = useState<JobPreset | null>(null);
  const [pendingSubmission, setPendingSubmission] = useState<JobType | null>(
    null,
  );
  const jobSubmissionRef = useRef<JobSubmissionContainerRef>(null);

  const {
    shouldShowWelcome,
    isTourActive,
    currentStep,
    startOnboarding,
    skipOnboarding,
    resetOnboarding,
    goToStep,
    finishOnboarding,
  } = useOnboarding();

  const {
    isCommandPaletteOpen,
    isHelpOpen,
    closeCommandPalette,
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
    refreshJobs,
    refreshTemplates,
    setJobsPage,
    setCrawlStatesPage,
    setJobStatusFilter,
  } = appData;

  const { selectedJobId, loadResults } = resultsState;

  const route = useMemo(() => parseRoute(pathname), [pathname]);

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

  const navigate = useCallback(
    (path: string) => {
      const nextPath = normalizePath(path);
      if (nextPath === pathname) {
        return;
      }
      window.history.pushState({}, "", nextPath);
      setPathname(nextPath);
    },
    [pathname],
  );

  useEffect(() => {
    const handlePopState = () => {
      setPathname(normalizePath(window.location.pathname));
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  useEffect(() => {
    if (route.kind !== "automation") {
      return;
    }

    const nextSection =
      pathname === "/automation"
        ? (getAutomationSectionFromHash(window.location.hash) ??
          route.automationSection ??
          DEFAULT_AUTOMATION_SECTION)
        : (route.automationSection ?? DEFAULT_AUTOMATION_SECTION);
    const canonicalPath = getAutomationPath(nextSection);

    if (pathname === canonicalPath) {
      return;
    }

    window.history.replaceState({}, "", canonicalPath);
    setPathname(canonicalPath);
  }, [pathname, route.automationSection, route.kind]);

  useEffect(() => {
    if (route.kind === "job-detail" && route.jobId) {
      void loadResults(route.jobId);
    }
  }, [route, loadResults]);

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
      await submitScrapeJob(postV1Scrape, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });
      jobSubmissionRef.current?.clearDraft("scrape");
      navigate("/jobs");
    },
    [refreshJobs, navigate],
  );

  const handleSubmitCrawl = useCallback(
    async (request: CrawlRequest) => {
      await submitCrawlJob(postV1Crawl, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });
      jobSubmissionRef.current?.clearDraft("crawl");
      navigate("/jobs");
    },
    [refreshJobs, navigate],
  );

  const handleSubmitResearch = useCallback(
    async (request: ResearchRequest) => {
      await submitResearchJob(postV1Research, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });
      jobSubmissionRef.current?.clearDraft("research");
      navigate("/jobs");
    },
    [refreshJobs, navigate],
  );

  const cancelJob = useCallback(
    async (jobId: string) => {
      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
        });
        if (apiError) {
          console.error(String(apiError));
          return;
        }
        await refreshJobs();
      } catch (err) {
        console.error(String(err));
      }
    },
    [refreshJobs],
  );

  const deleteJob = useCallback(
    async (jobId: string) => {
      if (!confirm("Are you sure you want to permanently delete this job?")) {
        return;
      }
      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
          query: { force: true },
        });
        if (apiError) {
          console.error(String(apiError));
          return;
        }
        await refreshJobs();
        if (selectedJobId === jobId) {
          navigate("/jobs");
        }
      } catch (err) {
        console.error(String(err));
      }
    },
    [navigate, refreshJobs, selectedJobId],
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

  const activeAutomationSection =
    route.kind === "automation"
      ? (route.automationSection ?? DEFAULT_AUTOMATION_SECTION)
      : DEFAULT_AUTOMATION_SECTION;

  const renderAutomationSection = useCallback(
    (section: AutomationSection): ReactNode => {
      switch (section) {
        case "batches":
          return (
            <BatchContainer
              formState={formState}
              profiles={profiles}
              loading={loading}
            />
          );
        case "chains":
          return <ChainContainer onChainSubmit={refreshJobs} />;
        case "watches":
          return <WatchContainer />;
        case "exports":
          return <ExportScheduleContainer />;
        case "webhooks":
          return <WebhookDeliveryContainer />;
      }
    },
    [formState, loading, profiles, refreshJobs],
  );

  const activeRouteForNav = route.kind === "job-detail" ? "jobs" : route.kind;

  const routeMeta = useMemo<RouteMeta>(() => {
    switch (route.kind) {
      case "jobs":
        return {
          title: "Jobs",
          description: "Monitor live work and open results fast.",
        };
      case "job-detail":
        return {
          title: route.jobId
            ? `Job ${formatShortJobId(route.jobId)}`
            : "Results",
          description:
            "Read saved output first, then open comparison, transform, and export tools only when needed.",
        };
      case "new-job":
        return {
          title: "Create Job",
          description:
            "Step through scrape, crawl, or research setup without losing access to expert controls.",
        };
      case "templates":
        return {
          title: "Templates",
          description:
            "Manage extraction templates and AI-assisted generation in one workspace.",
        };
      case "automation":
        return {
          title: "Automation",
          description:
            "Switch sections from the sub-navigation and stay in the workspace.",
        };
      case "settings":
        return {
          title: "Settings",
          description:
            "Configure runtime profiles, schedules, crawl state, and pipeline tools.",
        };
    }
  }, [route.jobId, route.kind]);

  const detailJob =
    route.kind === "job-detail"
      ? (jobs.find((job) => job.id === route.jobId) ?? null)
      : null;

  const jobDetailSignals = useMemo<RouteSignal[]>(() => {
    if (route.kind !== "job-detail") {
      return [];
    }

    return [
      {
        label: "Status",
        value: detailJob?.status ?? "unknown",
      },
      {
        label: "Results",
        value: resultsState.totalResults,
      },
      {
        label: "Format",
        value: resultsState.resultFormat.toUpperCase(),
      },
      {
        label: "Connection",
        value: connectionState,
      },
    ];
  }, [
    connectionState,
    detailJob?.status,
    resultsState.resultFormat,
    resultsState.totalResults,
    route.kind,
  ]);

  return (
    <div className={`app app--${route.kind}`}>
      <AppTopBar
        activeRoute={activeRouteForNav}
        navItems={NAV_ITEMS}
        onNavigate={navigate}
        globalAction={
          route.kind !== "new-job" ? (
            <button type="button" onClick={() => navigate("/jobs/new")}>
              Create Job
            </button>
          ) : null
        }
        utilities={
          <ThemeToggle
            theme={theme}
            resolvedTheme={resolvedTheme}
            onThemeChange={setTheme}
            onToggle={toggleTheme}
          />
        }
      />

      <CommandPalette
        isOpen={isCommandPaletteOpen}
        onClose={closeCommandPalette}
        jobs={jobs}
        onNavigate={handleNavigate}
        onSubmitForm={handleSubmitForm}
        onCancelJob={cancelJob}
        activeJobId={activeJob?.id}
        isMac={isMac}
        presets={presets}
        onSelectPreset={handleSelectPreset}
        onRestartTour={resetOnboarding}
      />

      <KeyboardShortcutsHelp
        isOpen={isHelpOpen}
        onClose={closeHelp}
        shortcuts={shortcuts}
        isMac={isMac}
      />

      <WelcomeModal
        isOpen={shouldShowWelcome}
        onStartTour={startOnboarding}
        onSkip={skipOnboarding}
      />

      <OnboardingFlow
        isRunning={isTourActive}
        currentStep={currentStep}
        onComplete={finishOnboarding}
        onSkip={skipOnboarding}
        onStepChange={goToStep}
      />

      <ErrorBanner message={error} />

      {route.kind === "jobs" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
            actions={
              selectedJobId ? (
                <button
                  type="button"
                  className="secondary"
                  onClick={() => {
                    persistJobsViewState();
                    navigate(`/jobs/${selectedJobId}`);
                  }}
                >
                  Open Last Results
                </button>
              ) : null
            }
          />

          <section id="jobs">
            <JobMonitoringDashboard
              jobs={jobs}
              failedJobs={failedJobs}
              error={error}
              loading={loading}
              statusFilter={jobStatusFilter}
              onStatusFilterChange={setJobStatusFilter}
              onViewResults={handleViewResults}
              onCancel={cancelJob}
              onDelete={deleteJob}
              onRefresh={refreshJobs}
              currentPage={jobsPage}
              totalJobs={jobsTotal}
              jobsPerPage={100}
              onPageChange={setJobsPage}
              connectionState={connectionState}
              managerStatus={managerStatus}
            />
          </section>
        </div>
      )}

      {route.kind === "job-detail" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
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

          <RouteSignals ariaLabel="Result context" items={jobDetailSignals} />

          <ResultsContainer resultsState={resultsState} jobs={jobs} />
        </div>
      )}

      {route.kind === "new-job" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
          />

          <div className="route-grid route-grid--new-job">
            <div className="route-primary route-stack">
              <JobSubmissionContainer
                ref={jobSubmissionRef}
                activeTab={activeTab}
                setActiveTab={setActiveTab}
                formState={formState}
                onSubmitScrape={handleSubmitScrape}
                onSubmitCrawl={handleSubmitCrawl}
                onSubmitResearch={handleSubmitResearch}
                loading={loading}
                profiles={profiles}
              />
            </div>
            <aside className="route-sidebar">
              <PresetContainer
                presets={presets}
                activeTab={activeTab}
                setActiveTab={setActiveTab}
                savePreset={savePreset}
                getCurrentConfig={getCurrentConfig}
                getCurrentUrl={getCurrentUrl}
                onSelectPreset={handleSelectPreset}
                onOpenAssistant={openJobAssistant}
                onOpenTemplateAssistant={openTemplateAssistant}
              />
            </aside>
          </div>
        </div>
      )}

      {route.kind === "templates" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
          />

          <TemplateManager
            templateNames={templates}
            onTemplatesChanged={() => {
              void refreshTemplates();
            }}
          />
        </div>
      )}

      {route.kind === "automation" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
            subnav={
              <AutomationSubnav
                activeSection={activeAutomationSection}
                onSectionChange={(section) =>
                  navigate(getAutomationPath(section))
                }
              />
            }
          />
          <AutomationLayout
            activeSection={activeAutomationSection}
            renderSection={renderAutomationSection}
          />
        </div>
      )}

      {route.kind === "settings" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
            actions={
              <button
                type="button"
                className="secondary"
                onClick={() => navigate("/templates")}
              >
                Open Templates
              </button>
            }
          />

          <InfoSections
            profiles={profiles}
            schedules={schedules}
            crawlStates={crawlStates}
            crawlStatesPage={crawlStatesPage}
            crawlStatesTotal={crawlStatesTotal}
            crawlStatesPerPage={100}
            onCrawlStatesPageChange={setCrawlStatesPage}
          />

          <section className="panel">
            <RenderProfileEditor
              onError={(message) => console.error(message)}
            />
          </section>

          <section className="panel">
            <PipelineJSEditor onError={(message) => console.error(message)} />
          </section>

          <ProxyPoolStatusPanel />
          <RetentionStatusPanel />
        </div>
      )}

      <div className="footer">Spartan Scraper 1.0 local-first workbench.</div>
    </div>
  );
}
