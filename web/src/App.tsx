/**
 * Spartan Scraper Web UI - Route-aware application shell.
 *
 * Purpose:
 * - Provide the route-based 1.0 shell for the reduced local-first product.
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
  Suspense,
  lazy,
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
import { JobList } from "./components/JobList";
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
import { AIExtractPreview } from "./components/AIExtractPreview";
import { AITemplateGenerator } from "./components/AITemplateGenerator";
import { TemplateManager } from "./components/templates/TemplateManager";
import { PresetContainer } from "./components/presets/PresetContainer";
import {
  JobSubmissionContainer,
  type JobSubmissionContainerRef,
} from "./components/jobs/JobSubmissionContainer";
import { ResultsContainer } from "./components/results/ResultsContainer";
import { RenderProfileEditor } from "./components/render-profiles";
import { PipelineJSEditor } from "./components/pipeline-js/PipelineJSEditor";
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
import type { JobPreset, JobType } from "./types/presets";

const MetricsDashboard = lazy(() =>
  import("./components/MetricsDashboard").then((mod) => ({
    default: mod.MetricsDashboard,
  })),
);

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
  eyebrow?: string;
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

function AppNavigation({
  activeRoute,
  onNavigate,
}: {
  activeRoute: RouteKind;
  onNavigate: (path: string) => void;
}) {
  return (
    <nav className="app-nav" aria-label="Primary">
      <div className="app-nav__items">
        {NAV_ITEMS.map((item) => {
          const isActive = activeRoute === item.kind;
          return (
            <button
              key={item.path}
              type="button"
              className={
                isActive ? "app-nav__button" : "app-nav__button secondary"
              }
              onClick={() => onNavigate(item.path)}
              aria-current={isActive ? "page" : undefined}
              title={item.description}
            >
              {item.label}
            </button>
          );
        })}
      </div>
    </nav>
  );
}

function RouteHeader({
  eyebrow,
  title,
  description,
  actions,
  subnav,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  actions?: ReactNode;
  subnav?: ReactNode;
}) {
  return (
    <section className="route-header" aria-label={`${title} overview`}>
      <div className="route-header__copy">
        {eyebrow ? (
          <div className="route-header__eyebrow">{eyebrow}</div>
        ) : null}
        <div className="route-header__title-row">
          <h2>{title}</h2>
          {actions ? (
            <div className="route-header__actions">{actions}</div>
          ) : null}
        </div>
        {description ? <p>{description}</p> : null}
        {subnav ? <div className="route-header__subnav">{subnav}</div> : null}
      </div>
    </section>
  );
}

function SignalPill({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  return (
    <div className="signal-pill">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
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
  const appData = useAppData();
  const formState = useFormState();
  const resultsState = useResultsState();
  const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme();
  const { presets, savePreset } = usePresets();
  const [pathname, setPathname] = useState(() =>
    normalizePath(window.location.pathname),
  );
  const [activeTab, setActiveTab] = useState<JobType>("scrape");
  const [isAIPreviewOpen, setIsAIPreviewOpen] = useState(false);
  const [aiPreviewInitialURL, setAIPreviewInitialURL] = useState("");
  const [isAIGeneratorOpen, setIsAIGeneratorOpen] = useState(false);
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
    metrics,
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
        navigate(`/jobs/${selectedJobId}`);
        return;
      }
      navigate("/jobs");
    },
    [navigate, selectedJobId],
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
    async (jobId: string, format: string, page: number) => {
      navigate(`/jobs/${jobId}`);
      await loadResults(jobId, format, page);
    },
    [loadResults, navigate],
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

    if (pendingPreset.config.url) {
      if (pendingPreset.jobType === "scrape") {
        jobSubmissionRef.current?.setScrapeUrl(pendingPreset.config.url);
      } else if (pendingPreset.jobType === "crawl") {
        jobSubmissionRef.current?.setCrawlUrl(pendingPreset.config.url);
      }
    }
    if (pendingPreset.config.query) {
      jobSubmissionRef.current?.setResearchQuery(pendingPreset.config.query);
    }
    setPendingPreset(null);
  }, [activeTab, pendingPreset, route.kind]);

  const getCurrentConfig = useCallback(() => {
    const baseConfig = {
      headless: formState.headless,
      usePlaywright: formState.usePlaywright,
      timeoutSeconds: formState.timeoutSeconds,
      authProfile: formState.authProfile,
      authBasic: formState.authBasic,
      headersRaw: formState.headersRaw,
      cookiesRaw: formState.cookiesRaw,
      queryRaw: formState.queryRaw,
      loginUrl: formState.loginUrl,
      loginUserSelector: formState.loginUserSelector,
      loginPassSelector: formState.loginPassSelector,
      loginSubmitSelector: formState.loginSubmitSelector,
      loginUser: formState.loginUser,
      loginPass: formState.loginPass,
      extractTemplate: formState.extractTemplate,
      extractValidate: formState.extractValidate,
      preProcessors: formState.preProcessors,
      postProcessors: formState.postProcessors,
      transformers: formState.transformers,
      incremental: formState.incremental,
      maxDepth: formState.maxDepth,
      maxPages: formState.maxPages,
      webhookUrl: formState.webhookUrl,
      webhookEvents: formState.webhookEvents,
      webhookSecret: formState.webhookSecret,
      interceptEnabled: formState.interceptEnabled,
      interceptURLPatterns: formState.interceptURLPatterns,
      interceptResourceTypes: formState.interceptResourceTypes,
      interceptCaptureRequestBody: formState.interceptCaptureRequestBody,
      interceptCaptureResponseBody: formState.interceptCaptureResponseBody,
      interceptMaxBodySize: formState.interceptMaxBodySize,
    };

    if (
      activeTab === "scrape" ||
      activeTab === "crawl" ||
      activeTab === "research"
    ) {
      return {
        ...baseConfig,
        aiExtractEnabled: formState.aiExtractEnabled,
        aiExtractMode: formState.aiExtractMode,
        aiExtractPrompt: formState.aiExtractPrompt,
        aiExtractSchema: formState.aiExtractSchema,
        aiExtractFields: formState.aiExtractFields,
        ...(activeTab === "research"
          ? {
              agenticResearchEnabled: formState.agenticResearchEnabled,
              agenticResearchInstructions:
                formState.agenticResearchInstructions,
              agenticResearchMaxRounds: formState.agenticResearchMaxRounds,
              agenticResearchMaxFollowUpUrls:
                formState.agenticResearchMaxFollowUpUrls,
            }
          : {}),
      };
    }

    return baseConfig;
  }, [activeTab, formState]);

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

  const openAIPreview = useCallback(
    (url?: string) => {
      setAIPreviewInitialURL(url ?? getCurrentUrl());
      setIsAIPreviewOpen(true);
    },
    [getCurrentUrl],
  );

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
          eyebrow: "Operations",
          title: "Jobs",
          description:
            "Monitor live execution and jump directly into the latest results.",
        };
      case "job-detail":
        return {
          eyebrow: route.jobId ? `Job ${route.jobId}` : "Results Explorer",
          title: "Results",
          description:
            "Inspect extracted output first and return to the jobs index only when you need broader queue context.",
        };
      case "new-job":
        return {
          eyebrow: "Submission",
          title: "Create Job",
          description:
            "Launch a scrape, crawl, or research run from one focused workflow.",
        };
      case "templates":
        return {
          eyebrow: "Extraction",
          title: "Templates",
          description:
            "Manage extraction templates and AI-assisted creation without extra dashboard framing.",
        };
      case "automation":
        return {
          eyebrow: "Workflow Orchestration",
          title: "Automation",
          description:
            "Move between batches, chains, watches, exports, and webhook deliveries from one focused automation hub.",
        };
      case "settings":
        return {
          eyebrow: "Runtime Control",
          title: "Settings",
          description:
            "Profiles, schedules, crawl-state inventory, retention, and pipeline tools.",
        };
    }
  }, [route.jobId, route.kind]);

  const showShellSignals =
    route.kind === "jobs" ||
    route.kind === "job-detail" ||
    route.kind === "automation";

  return (
    <div className={`app app--${route.kind}`}>
      <header className="app-shell">
        <div className="app-shell__topbar">
          <div className="app-shell__brand">
            <div className="app-shell__eyebrow">Operation Spartan</div>
            <h1>Spartan Scraper</h1>
            <p>Local-first scraping and automation workbench.</p>
          </div>

          <AppNavigation
            activeRoute={activeRouteForNav}
            onNavigate={navigate}
          />

          <div className="app-shell__toolbar">
            {route.kind !== "new-job" ? (
              <button type="button" onClick={() => navigate("/jobs/new")}>
                Create Job
              </button>
            ) : null}
            <ThemeToggle
              theme={theme}
              resolvedTheme={resolvedTheme}
              onThemeChange={setTheme}
              onToggle={toggleTheme}
            />
          </div>
        </div>

        {showShellSignals ? (
          <section
            className="app-shell__signals"
            aria-label="Live system signals"
          >
            <SignalPill label="Jobs" value={jobsTotal} />
            <SignalPill label="Queued" value={managerStatus?.queued ?? 0} />
            <SignalPill label="Active" value={managerStatus?.active ?? 0} />
          </section>
        ) : null}
      </header>

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
        <>
          <RouteHeader
            eyebrow={routeMeta.eyebrow}
            title={routeMeta.title}
            description={routeMeta.description}
          />

          <div className="route-grid route-grid--jobs">
            <div className="route-primary">
              <section id="jobs">
                <JobList
                  jobs={jobs}
                  failedJobs={failedJobs}
                  error={error}
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
                />
              </section>
            </div>
            <aside className="route-sidebar">
              <section className="panel route-sidebar-panel">
                <div className="route-sidebar-panel__eyebrow">Focus</div>
                <h3>Recent jobs stay primary</h3>
                <p>
                  This route should land directly on the monitoring surface. Use
                  the global create action to queue new work, then reopen a
                  result when you need to drill deeper.
                </p>
                {selectedJobId ? (
                  <div className="route-sidebar-panel__actions">
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => navigate(`/jobs/${selectedJobId}`)}
                    >
                      Open Last Results
                    </button>
                  </div>
                ) : null}
              </section>
              <section className="panel route-sidebar-panel">
                <div className="route-sidebar-panel__eyebrow">Live Signals</div>
                <div className="route-sidebar-panel__stats">
                  <SignalPill
                    label="Loading"
                    value={loading ? "Refreshing" : "Ready"}
                  />
                  <SignalPill
                    label="Headless"
                    value={formState.headless ? "On" : "Off"}
                  />
                  <SignalPill label="Connection" value={connectionState} />
                </div>
              </section>
              <Suspense
                fallback={
                  <div className="loading-placeholder panel">
                    Loading metrics...
                  </div>
                }
              >
                <MetricsDashboard
                  metrics={metrics}
                  connectionState={connectionState}
                />
              </Suspense>
            </aside>
          </div>
        </>
      )}

      {route.kind === "job-detail" && (
        <>
          <RouteHeader
            eyebrow={routeMeta.eyebrow}
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

          <div className="route-grid route-grid--job-detail">
            <div className="route-primary">
              <ResultsContainer resultsState={resultsState} jobs={jobs} />
            </div>
            <aside className="route-sidebar">
              <section className="panel route-sidebar-panel">
                <div className="route-sidebar-panel__eyebrow">Job Context</div>
                <h3>Stay with this result set</h3>
                <p>
                  Keep the default layout focused on the selected run. Return to
                  the jobs index only when you want to compare runs or pivot to
                  another job.
                </p>
                <div className="route-sidebar-panel__stats">
                  <SignalPill
                    label="Result Items"
                    value={resultsState.totalResults}
                  />
                  <SignalPill label="Page" value={resultsState.currentPage} />
                  <SignalPill
                    label="Format"
                    value={resultsState.resultFormat.toUpperCase()}
                  />
                </div>
              </section>
              <section className="panel route-sidebar-panel">
                <div className="route-sidebar-panel__eyebrow">
                  Queue Signals
                </div>
                <div className="route-sidebar-panel__stats">
                  <SignalPill label="Connection" value={connectionState} />
                  <SignalPill
                    label="Queued"
                    value={managerStatus?.queued ?? 0}
                  />
                  <SignalPill
                    label="Active"
                    value={managerStatus?.active ?? 0}
                  />
                </div>
              </section>
            </aside>
          </div>
        </>
      )}

      {route.kind === "new-job" && (
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
              applyPreset={formState.applyPreset}
              savePreset={savePreset}
              getCurrentConfig={getCurrentConfig}
              getCurrentUrl={getCurrentUrl}
              onSelectPreset={handleSelectPreset}
              onOpenAIPreview={openAIPreview}
              onOpenTemplateGenerator={() => setIsAIGeneratorOpen(true)}
            />
          </aside>
        </div>
      )}

      {route.kind === "templates" && (
        <>
          <RouteHeader
            eyebrow={routeMeta.eyebrow}
            title={routeMeta.title}
            description={routeMeta.description}
            actions={
              <>
                <button
                  type="button"
                  className="secondary"
                  onClick={() => openAIPreview()}
                >
                  Preview Extraction with AI
                </button>
                <button
                  type="button"
                  onClick={() => setIsAIGeneratorOpen(true)}
                >
                  Generate Template with AI
                </button>
              </>
            }
          />

          <TemplateManager
            templateNames={templates}
            onTemplatesChanged={() => {
              void refreshTemplates();
            }}
            onOpenAIPreview={() => openAIPreview()}
            onOpenAIGenerator={() => setIsAIGeneratorOpen(true)}
          />
        </>
      )}

      {route.kind === "automation" && (
        <>
          <RouteHeader
            eyebrow={routeMeta.eyebrow}
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
        </>
      )}

      {route.kind === "settings" && (
        <>
          <RouteHeader
            eyebrow={routeMeta.eyebrow}
            title={routeMeta.title}
            description={routeMeta.description}
          />
          <div className="route-grid route-grid--settings">
            <div className="route-primary route-stack">
              <section className="panel">
                <div className="settings-template-callout">
                  <div>
                    <h3>Extraction Templates</h3>
                    <p>
                      Template lifecycle management now lives on the Templates
                      page so existing templates, AI preview, and AI generation
                      stay actionable instead of split across multiple surfaces.
                    </p>
                  </div>
                  <div className="settings-template-callout__actions">
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => navigate("/templates")}
                    >
                      Open Templates
                    </button>
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => openAIPreview()}
                    >
                      Preview Extraction with AI
                    </button>
                    <button
                      type="button"
                      onClick={() => setIsAIGeneratorOpen(true)}
                    >
                      Generate Template with AI
                    </button>
                  </div>
                </div>
              </section>
              <InfoSections
                profiles={profiles}
                schedules={schedules}
                crawlStates={crawlStates}
                crawlStatesPage={crawlStatesPage}
                crawlStatesTotal={crawlStatesTotal}
                crawlStatesPerPage={100}
                onCrawlStatesPageChange={setCrawlStatesPage}
              />
            </div>
            <aside className="route-sidebar">
              <section className="panel route-sidebar-panel">
                <div className="route-sidebar-panel__eyebrow">Inventory</div>
                <div className="route-sidebar-panel__stats">
                  <SignalPill label="Profiles" value={profiles.length} />
                  <SignalPill label="Schedules" value={schedules.length} />
                  <SignalPill label="States" value={crawlStatesTotal} />
                </div>
              </section>
            </aside>
          </div>
          <section className="panel" style={{ marginTop: 16 }}>
            <RenderProfileEditor
              onError={(message) => console.error(message)}
            />
          </section>
          <section className="panel" style={{ marginTop: 16 }}>
            <PipelineJSEditor onError={(message) => console.error(message)} />
          </section>
          <ProxyPoolStatusPanel />
          <RetentionStatusPanel />
        </>
      )}

      <AIExtractPreview
        isOpen={isAIPreviewOpen}
        initialUrl={aiPreviewInitialURL}
        onClose={() => setIsAIPreviewOpen(false)}
      />

      <AITemplateGenerator
        isOpen={isAIGeneratorOpen}
        onClose={() => setIsAIGeneratorOpen(false)}
        onTemplateSaved={() => {
          setIsAIGeneratorOpen(false);
          void refreshTemplates();
        }}
      />

      <div className="footer">Spartan Scraper 1.0 local-first workbench.</div>
    </div>
  );
}
