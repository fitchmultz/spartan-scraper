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
 *   `/automation`, and `/settings`.
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
import { WatchContainer } from "./components/watches/WatchContainer";
import { ExportScheduleContainer } from "./components/export-schedules/ExportScheduleContainer";
import { WebhookDeliveryContainer } from "./components/webhooks/WebhookDeliveryContainer";
import { RetentionStatusPanel } from "./components/RetentionStatusPanel";
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
}

interface NavItem {
  kind: Exclude<RouteKind, "job-detail">;
  label: string;
  path: string;
  description: string;
}

interface RouteMetaItem {
  label: string;
  value: string;
}

interface RouteMeta {
  eyebrow: string;
  title: string;
  description: string;
  meta: RouteMetaItem[];
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
    path: "/automation",
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
  if (path === "/automation") {
    return { kind: "automation", path };
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

function PageIntro({
  eyebrow,
  title,
  description,
  actions,
  meta,
}: {
  eyebrow?: string;
  title: string;
  description: string;
  actions?: ReactNode;
  meta?: RouteMetaItem[];
}) {
  return (
    <section className="panel route-intro">
      <div className="route-intro__content">
        <div className="route-intro__copy">
          {eyebrow ? (
            <div className="route-intro__eyebrow">{eyebrow}</div>
          ) : null}
          <h2>{title}</h2>
          <p>{description}</p>
          {meta && meta.length > 0 ? (
            <div className="route-intro__meta">
              {meta.map((item) => (
                <div key={item.label} className="route-intro__meta-item">
                  <span>{item.label}</span>
                  <strong>{item.value}</strong>
                </div>
              ))}
            </div>
          ) : null}
        </div>
        {actions ? <div className="route-intro__actions">{actions}</div> : null}
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
    return {
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
  }, [formState]);

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

  const activeRouteForNav = route.kind === "job-detail" ? "jobs" : route.kind;

  const routeMeta = useMemo<RouteMeta>(() => {
    const queuedItems = managerStatus?.queued ?? 0;
    const activeItems = managerStatus?.active ?? 0;

    switch (route.kind) {
      case "jobs":
        return {
          eyebrow: "Operations",
          title: "Jobs",
          description:
            "Queue new work, monitor live execution, and jump directly into the latest results.",
          meta: [
            { label: "Tracked Jobs", value: jobsTotal.toString() },
            { label: "Queued", value: queuedItems.toString() },
            { label: "Active", value: activeItems.toString() },
          ],
        };
      case "job-detail":
        return {
          eyebrow: "Results Explorer",
          title: route.jobId ? `Job Results: ${route.jobId}` : "Job Results",
          description:
            "Inspect extracted output first, then use the jobs index only when you need broader queue context.",
          meta: [
            {
              label: "Format",
              value: resultsState.resultFormat.toUpperCase(),
            },
            {
              label: "Results",
              value: resultsState.totalResults.toString(),
            },
            { label: "Queued", value: queuedItems.toString() },
          ],
        };
      case "new-job":
        return {
          eyebrow: "Submission",
          title: "Create Job",
          description:
            "Launch a scrape, crawl, or research run from one focused workflow.",
          meta: [
            { label: "Profiles", value: profiles.length.toString() },
            { label: "Templates", value: templates.length.toString() },
            { label: "Connection", value: connectionState },
          ],
        };
      case "templates":
        return {
          eyebrow: "Extraction",
          title: "Templates",
          description:
            "Open the template workspace directly, with creation, AI preview, and AI-assisted generation available without scrolling through dashboard chrome.",
          meta: [
            { label: "Templates", value: templates.length.toString() },
            { label: "Built-in", value: "3" },
          ],
        };
      case "automation":
        return {
          eyebrow: "Workflow Orchestration",
          title: "Automation",
          description:
            "Start with the batch runner, then branch into chains, watches, exports, and webhook delivery from a lighter route shell.",
          meta: [
            { label: "Queued", value: queuedItems.toString() },
            { label: "Active", value: activeItems.toString() },
            { label: "Connection", value: connectionState },
          ],
        };
      case "settings":
        return {
          eyebrow: "Runtime Control",
          title: "Settings",
          description:
            "Profiles, schedules, retention, crawl-state inventory, and pipeline tools live here without hiding behind the global dashboard stack.",
          meta: [
            { label: "Profiles", value: profiles.length.toString() },
            { label: "Schedules", value: schedules.length.toString() },
            { label: "Crawl States", value: crawlStatesTotal.toString() },
          ],
        };
    }
  }, [
    connectionState,
    crawlStatesTotal,
    jobsTotal,
    managerStatus,
    profiles.length,
    resultsState.resultFormat,
    resultsState.totalResults,
    route,
    schedules.length,
    templates.length,
  ]);

  return (
    <div className={`app app--${route.kind}`}>
      <header className="app-shell">
        <div className="app-shell__masthead">
          <div className="app-shell__brand">
            <div className="app-shell__eyebrow">Operation Spartan</div>
            <h1>Spartan Scraper</h1>
            <p>{routeMeta.description}</p>
          </div>
          {route.kind !== "new-job" ? (
            <div className="app-shell__signals">
              <SignalPill label="Jobs" value={jobsTotal} />
              <SignalPill label="Queued" value={managerStatus?.queued ?? 0} />
              <SignalPill label="Active" value={managerStatus?.active ?? 0} />
              <SignalPill
                label="Fetcher"
                value={formState.usePlaywright ? "Playwright" : "HTTP"}
              />
              <SignalPill label="Theme" value={resolvedTheme} />
            </div>
          ) : null}
        </div>
        <div className="app-shell__controls">
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

      {(route.kind === "jobs" || route.kind === "job-detail") && (
        <>
          <PageIntro
            eyebrow={routeMeta.eyebrow}
            title={routeMeta.title}
            description={routeMeta.description}
            meta={routeMeta.meta}
            actions={
              route.kind === "job-detail" ? (
                <>
                  <button
                    type="button"
                    className="secondary"
                    onClick={() => navigate("/jobs")}
                  >
                    Back to Jobs
                  </button>
                  <button type="button" onClick={() => navigate("/jobs/new")}>
                    Create Job
                  </button>
                </>
              ) : undefined
            }
          />

          {route.kind === "job-detail" ? (
            <>
              <div className="route-grid route-grid--job-detail">
                <div className="route-primary">
                  <ResultsContainer resultsState={resultsState} jobs={jobs} />
                </div>
                <aside className="route-sidebar">
                  <section className="panel route-sidebar-panel">
                    <div className="route-sidebar-panel__eyebrow">
                      Job Context
                    </div>
                    <h3>Stay on the result you opened</h3>
                    <p>
                      The result explorer is the primary surface here. Use the
                      jobs index below only when you want to compare runs or
                      pivot to another job.
                    </p>
                    <div className="route-sidebar-panel__stats">
                      <SignalPill
                        label="Result Items"
                        value={resultsState.totalResults}
                      />
                      <SignalPill
                        label="Page"
                        value={resultsState.currentPage}
                      />
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

              <section className="route-lower-section">
                <div className="route-section-label">Recent jobs</div>
                <section id="jobs">
                  <JobList
                    jobs={jobs}
                    error={null}
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
              </section>
            </>
          ) : (
            <div className="route-grid route-grid--jobs">
              <div className="route-primary">
                <section id="jobs">
                  <JobList
                    jobs={jobs}
                    error={null}
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
                  <div className="route-sidebar-panel__eyebrow">
                    Above The Fold
                  </div>
                  <h3>Launch or resume work fast</h3>
                  <p>
                    Recent jobs stay dominant on this route, with the primary
                    creation action and live queue state kept nearby instead of
                    hidden behind a shared landing stack.
                  </p>
                  <div className="route-sidebar-panel__actions">
                    <button type="button" onClick={() => navigate("/jobs/new")}>
                      Create Job
                    </button>
                    {selectedJobId ? (
                      <button
                        type="button"
                        className="secondary"
                        onClick={() => navigate(`/jobs/${selectedJobId}`)}
                      >
                        Open Last Results
                      </button>
                    ) : null}
                  </div>
                </section>
                <section className="panel route-sidebar-panel">
                  <div className="route-sidebar-panel__eyebrow">
                    Live Signals
                  </div>
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
          )}
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
          <PageIntro
            eyebrow={routeMeta.eyebrow}
            title={routeMeta.title}
            description={routeMeta.description}
            meta={routeMeta.meta}
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
          <PageIntro
            eyebrow={routeMeta.eyebrow}
            title={routeMeta.title}
            description={routeMeta.description}
            meta={routeMeta.meta}
          />
          <div className="route-grid route-grid--automation">
            <div className="route-primary">
              <BatchContainer
                formState={formState}
                profiles={profiles}
                loading={loading}
              />
            </div>
            <aside className="route-sidebar">
              <section className="panel route-sidebar-panel">
                <div className="route-sidebar-panel__eyebrow">
                  Automation Map
                </div>
                <h3>Start with batches</h3>
                <p>
                  The batch runner is the first action on this route. The other
                  automation surfaces remain one scroll away instead of pushing
                  the entry point down the page.
                </p>
                <div className="route-sidebar-panel__actions">
                  <a href="#batch-forms" className="route-sidebar-link">
                    Batch Runner
                  </a>
                  <a href="#chains" className="route-sidebar-link">
                    Chains
                  </a>
                  <a href="#watches" className="route-sidebar-link">
                    Watches
                  </a>
                  <a href="#export-schedules" className="route-sidebar-link">
                    Exports
                  </a>
                </div>
              </section>
            </aside>
          </div>
          <section className="route-lower-section">
            <div className="route-section-label">
              Additional automation surfaces
            </div>
            <div className="route-stack">
              <div id="chains">
                <ChainContainer onChainSubmit={refreshJobs} />
              </div>
              <div id="watches">
                <WatchContainer />
              </div>
              <div id="export-schedules">
                <ExportScheduleContainer />
              </div>
              <WebhookDeliveryContainer />
            </div>
          </section>
        </>
      )}

      {route.kind === "settings" && (
        <>
          <PageIntro
            eyebrow={routeMeta.eyebrow}
            title={routeMeta.title}
            description={routeMeta.description}
            meta={routeMeta.meta}
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
