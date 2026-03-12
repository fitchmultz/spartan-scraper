/**
 * Spartan Scraper Web UI - Balanced 1.0 application shell.
 *
 * Purpose:
 * - Provide the route-based 1.0 shell for the reduced local-first product.
 *
 * Responsibilities:
 * - Route between jobs, templates, automation, and settings views.
 * - Wire shared data hooks, job submission, and result loading.
 * - Remove deleted product surfaces from the top-level navigation.
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
import { Hero } from "./components/Hero";
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
import { AITemplateGenerator } from "./components/AITemplateGenerator";
import { PresetContainer } from "./components/presets/PresetContainer";
import {
  JobSubmissionContainer,
  type JobSubmissionContainerRef,
} from "./components/jobs/JobSubmissionContainer";
import { ResultsContainer } from "./components/results/ResultsContainer";
import { RenderProfileEditor } from "./components/render-profiles";
import { PipelineJSEditor } from "./components/pipeline-js/PipelineJSEditor";
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

const NAV_ITEMS: NavItem[] = [
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
];

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
    <section className="panel" style={{ marginTop: 16 }}>
      <div className="row" style={{ gap: 12, flexWrap: "wrap" }}>
        {NAV_ITEMS.map((item) => {
          const isActive = activeRoute === item.kind;
          return (
            <button
              key={item.path}
              type="button"
              className={isActive ? "" : "secondary"}
              onClick={() => onNavigate(item.path)}
              aria-current={isActive ? "page" : undefined}
            >
              {item.label}
            </button>
          );
        })}
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
          gap: 12,
          marginTop: 16,
        }}
      >
        {NAV_ITEMS.map((item) => (
          <div
            key={item.kind}
            style={{
              padding: 12,
              borderRadius: 10,
              border: "1px solid var(--border)",
              background:
                activeRoute === item.kind ? "var(--bg-alt)" : "var(--bg)",
            }}
          >
            <div style={{ fontWeight: 600, marginBottom: 6 }}>{item.label}</div>
            <div style={{ color: "var(--muted)", fontSize: 14 }}>
              {item.description}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

function PageIntro({
  title,
  description,
  actions,
}: {
  title: string;
  description: string;
  actions?: ReactNode;
}) {
  return (
    <section className="panel" style={{ marginTop: 16 }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          gap: 16,
          flexWrap: "wrap",
          alignItems: "center",
        }}
      >
        <div>
          <h2 style={{ margin: 0 }}>{title}</h2>
          <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>
            {description}
          </p>
        </div>
        {actions}
      </div>
    </section>
  );
}

function ErrorBanner({ message }: { message: string | null }) {
  if (!message) {
    return null;
  }

  return (
    <section className="panel" style={{ marginTop: 16 }}>
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
  const [isAIGeneratorOpen, setIsAIGeneratorOpen] = useState(false);
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
      if (preset.config.url) {
        if (preset.jobType === "scrape") {
          jobSubmissionRef.current?.setScrapeUrl(preset.config.url);
        } else if (preset.jobType === "crawl") {
          jobSubmissionRef.current?.setCrawlUrl(preset.config.url);
        }
      }
      if (preset.config.query) {
        jobSubmissionRef.current?.setResearchQuery(preset.config.query);
      }
    },
    [navigate],
  );

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

  const handleSubmitForm = useCallback(
    async (formType: "scrape" | "crawl" | "research") => {
      navigate("/jobs/new");
      if (formType === "scrape") {
        await jobSubmissionRef.current?.submitScrape();
      } else if (formType === "crawl") {
        await jobSubmissionRef.current?.submitCrawl();
      } else if (formType === "research") {
        await jobSubmissionRef.current?.submitResearch();
      }
    },
    [navigate],
  );

  const activeRouteForNav = route.kind === "job-detail" ? "jobs" : route.kind;

  return (
    <div className="app">
      <Hero
        loading={loading}
        managerStatus={managerStatus}
        jobsCount={jobs.length}
        headless={formState.headless}
        usePlaywright={formState.usePlaywright}
        theme={theme}
        resolvedTheme={resolvedTheme}
        onThemeChange={setTheme}
        onThemeToggle={toggleTheme}
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

      <AppNavigation activeRoute={activeRouteForNav} onNavigate={navigate} />
      <ErrorBanner message={error} />

      <Suspense
        fallback={<div className="loading-placeholder">Loading metrics...</div>}
      >
        <MetricsDashboard metrics={metrics} connectionState={connectionState} />
      </Suspense>

      {(route.kind === "jobs" || route.kind === "job-detail") && (
        <>
          <PageIntro
            title={
              route.kind === "job-detail" && route.jobId
                ? `Job Results: ${route.jobId}`
                : "Jobs"
            }
            description={
              route.kind === "job-detail"
                ? "Inspect artifacts, compare runs, and export supported formats."
                : "Monitor queue activity, browse recent jobs, and open job results."
            }
            actions={
              <button type="button" onClick={() => navigate("/jobs/new")}>
                Create Job
              </button>
            }
          />

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

          {route.kind === "job-detail" && (
            <ResultsContainer resultsState={resultsState} jobs={jobs} />
          )}
        </>
      )}

      {route.kind === "new-job" && (
        <>
          <PageIntro
            title="Create Job"
            description="Submit a scrape, crawl, or research job using the shared local-first engine."
            actions={
              <button
                type="button"
                className="secondary"
                onClick={() => setIsAIGeneratorOpen(true)}
              >
                Generate Template with AI
              </button>
            }
          />

          <PresetContainer
            presets={presets}
            activeTab={activeTab}
            setActiveTab={setActiveTab}
            applyPreset={formState.applyPreset}
            savePreset={savePreset}
            getCurrentConfig={getCurrentConfig}
            getCurrentUrl={getCurrentUrl}
            onSelectPreset={handleSelectPreset}
          />

          <JobSubmissionContainer
            ref={jobSubmissionRef}
            formState={formState}
            onSubmitScrape={handleSubmitScrape}
            onSubmitCrawl={handleSubmitCrawl}
            onSubmitResearch={handleSubmitResearch}
            loading={loading}
            profiles={profiles}
          />
        </>
      )}

      {route.kind === "templates" && (
        <>
          <PageIntro
            title="Templates"
            description="Build extraction templates visually or generate them from a live page with AI."
            actions={
              <button type="button" onClick={() => setIsAIGeneratorOpen(true)}>
                Generate Template with AI
              </button>
            }
          />

          <InfoSections
            profiles={[]}
            schedules={[]}
            templates={templates}
            crawlStates={[]}
            crawlStatesPage={1}
            crawlStatesTotal={0}
            crawlStatesPerPage={100}
            onCrawlStatesPageChange={() => {}}
            onTemplatesChanged={() => {
              void refreshTemplates();
            }}
          />
        </>
      )}

      {route.kind === "automation" && (
        <>
          <PageIntro
            title="Automation"
            description="Coordinate higher-level workflows on top of the core scrape, crawl, and research engines."
          />
          <BatchContainer
            formState={formState}
            profiles={profiles}
            loading={loading}
          />
          <ChainContainer onChainSubmit={refreshJobs} />
          <WatchContainer />
          <ExportScheduleContainer />
          <WebhookDeliveryContainer />
        </>
      )}

      {route.kind === "settings" && (
        <>
          <PageIntro
            title="Settings"
            description="Manage runtime profiles, recurring schedules, crawl state inventory, retention, and pipeline scripts."
          />
          <InfoSections
            profiles={profiles}
            schedules={schedules}
            templates={templates}
            crawlStates={crawlStates}
            crawlStatesPage={crawlStatesPage}
            crawlStatesTotal={crawlStatesTotal}
            crawlStatesPerPage={100}
            onCrawlStatesPageChange={setCrawlStatesPage}
            onTemplatesChanged={() => {
              void refreshTemplates();
            }}
          />
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
