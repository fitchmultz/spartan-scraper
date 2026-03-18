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
import { ActionEmptyState } from "./components/ActionEmptyState";
import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { OnboardingFlow } from "./components/OnboardingFlow";
import { OnboardingNudge } from "./components/OnboardingNudge";
import { RouteHelpPanel } from "./components/RouteHelpPanel";
import { SystemStatusPanel } from "./components/SystemStatusPanel";
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
import { useToast } from "./components/toast";
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
import { ShortcutHint } from "./components/ShortcutHint";
import { TutorialTooltip } from "./components/TutorialTooltip";
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
import { getApiErrorMessage } from "./lib/api-errors";
import { saveJobsViewState } from "./lib/job-monitoring";
import type { OnboardingRouteKey, RouteHelpAction } from "./lib/onboarding";
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
  const toast = useToast();
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
  const previousRouteKeyRef = useRef<OnboardingRouteKey | null>(null);

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
    markRouteVisited,
    visitedRoutes,
  } = useOnboarding();

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
    refreshHealth,
    refreshJobs,
    refreshTemplates,
    setJobsPage,
    setCrawlStatesPage,
    setJobStatusFilter,
  } = appData;

  const { selectedJobId, loadResults } = resultsState;

  const route = useMemo(() => parseRoute(pathname), [pathname]);
  const routeKey = route.kind as OnboardingRouteKey;
  const [routeHelpDefaultExpanded, setRouteHelpDefaultExpanded] =
    useState(false);

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

  useEffect(() => {
    if (previousRouteKeyRef.current === routeKey) {
      return;
    }

    const firstVisit = !visitedRoutes.includes(routeKey);
    setRouteHelpDefaultExpanded(firstVisit);
    if (firstVisit) {
      markRouteVisited(routeKey);
    }
    previousRouteKeyRef.current = routeKey;
  }, [markRouteVisited, routeKey, visitedRoutes]);

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
      } catch (err) {
        toast.update(toastId, {
          tone: "error",
          title: "Failed to cancel job",
          description: getApiErrorMessage(
            err,
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
      } catch (err) {
        toast.update(toastId, {
          tone: "error",
          title: "Failed to delete job",
          description: getApiErrorMessage(
            err,
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

  const activeAutomationSection =
    route.kind === "automation"
      ? (route.automationSection ?? DEFAULT_AUTOMATION_SECTION)
      : DEFAULT_AUTOMATION_SECTION;

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
          navigate("/settings");
          return;
      }
    },
    [jobs, navigate, selectedJobId],
  );

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

  const handleRouteHelpAction = useCallback(
    (actionId: RouteHelpAction["id"]) => {
      switch (actionId) {
        case "create-job":
          navigate("/jobs/new");
          return;
        case "open-templates":
          navigate("/templates");
          return;
        case "open-automation":
          navigate("/automation/batches");
          return;
        case "open-settings":
          navigate("/settings");
          return;
        case "start-tour":
          resetOnboarding();
          return;
      }
    },
    [navigate, resetOnboarding],
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

  const routeHelpPanel = (
    <RouteHelpPanel
      routeKey={routeKey}
      shortcuts={shortcuts}
      isMac={isMac}
      defaultExpanded={routeHelpDefaultExpanded}
      onOpenCommandPalette={openCommandPalette}
      onOpenShortcuts={openHelp}
      onRestartTour={resetOnboarding}
      onAction={handleRouteHelpAction}
    />
  );

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
          isVisible={shouldShowFirstRunHint}
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
        showBeacon={shouldShowFirstRunHint}
        showDelay={500}
      />

      <TutorialTooltip
        target='[data-tour="keyboard-help"]'
        title="Shortcut help is visible now"
        content="Open this anytime to see global shortcuts and a route-specific section for what matters on the current screen."
        position="bottom"
        showBeacon={shouldShowFirstRunHint}
        showDelay={500}
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

      {setupRequired ? (
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
      ) : null}

      {!setupRequired && route.kind === "jobs" && (
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

          {routeHelpPanel}

          <section id="jobs" data-tour="jobs-dashboard">
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
              onCreateJob={() => navigate("/jobs/new")}
              onOpenTemplates={() => navigate("/templates")}
              onOpenAutomation={() => navigate("/automation/batches")}
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

      {!setupRequired && route.kind === "job-detail" && (
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

          {routeHelpPanel}

          <RouteSignals ariaLabel="Result context" items={jobDetailSignals} />

          <div data-tour="job-results">
            <ResultsContainer resultsState={resultsState} jobs={jobs} />
          </div>
        </div>
      )}

      {!setupRequired && route.kind === "new-job" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
          />

          {routeHelpPanel}

          {jobsTotal === 0 && jobStatusFilter === "" ? (
            <ActionEmptyState
              eyebrow="First run"
              title="Start with a single page scrape"
              description="Paste a URL into the form below, keep the defaults, and submit one successful run before moving on to templates or automation."
              actions={[
                {
                  label: "Open templates",
                  onClick: () => navigate("/templates"),
                  tone: "secondary",
                },
                {
                  label: "Restart tour",
                  onClick: resetOnboarding,
                  tone: "secondary",
                },
              ]}
            />
          ) : null}

          <div className="route-grid route-grid--new-job">
            <div className="route-primary route-stack" data-tour="job-wizard">
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

      {!setupRequired && route.kind === "templates" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
          />

          {routeHelpPanel}

          <div data-tour="templates-workspace">
            <TemplateManager
              templateNames={templates}
              onTemplatesChanged={() => {
                void refreshTemplates();
              }}
            />
          </div>
        </div>
      )}

      {!setupRequired && route.kind === "automation" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
            subnav={
              <div data-tour="automation-subnav">
                <AutomationSubnav
                  activeSection={activeAutomationSection}
                  onSectionChange={(section) =>
                    navigate(getAutomationPath(section))
                  }
                />
              </div>
            }
          />
          {routeHelpPanel}
          <section data-tour="automation-hub">
            <AutomationLayout
              activeSection={activeAutomationSection}
              renderSection={renderAutomationSection}
            />
          </section>
        </div>
      )}

      {!setupRequired && route.kind === "settings" && (
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

          {routeHelpPanel}

          <div data-tour="settings-workspace">
            <InfoSections
              profiles={profiles}
              schedules={schedules}
              crawlStates={crawlStates}
              crawlStatesPage={crawlStatesPage}
              crawlStatesTotal={crawlStatesTotal}
              crawlStatesPerPage={100}
              onCrawlStatesPageChange={setCrawlStatesPage}
              onCreateJob={() => navigate("/jobs/new")}
              onOpenAutomation={() => navigate("/automation/batches")}
            />

            <section className="panel">
              <RenderProfileEditor />
            </section>

            <section className="panel">
              <PipelineJSEditor />
            </section>

            <ProxyPoolStatusPanel />
            <RetentionStatusPanel />
          </div>
        </div>
      )}

      <div className="footer">Spartan Scraper 1.0 local-first workbench.</div>
    </div>
  );
}
